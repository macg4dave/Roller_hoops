package discoveryworker

import (
	"context"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"roller_hoops/core-go/internal/enrichment/mdns"
	"roller_hoops/core-go/internal/enrichment/snmp"
	"roller_hoops/core-go/internal/enrichment/vlan"
	"roller_hoops/core-go/internal/naming"
	"roller_hoops/core-go/internal/sqlcgen"
	"roller_hoops/core-go/internal/tagging"
)

type enrichmentTarget struct {
	DeviceID string
	IP       netip.Addr
}

func (w *Worker) runEnrichment(ctx context.Context, targets []enrichmentTarget) map[string]any {
	if w == nil || w.q == nil {
		return nil
	}
	if !w.nameResolutionEnabled && !w.snmpEnabled {
		return nil
	}
	if len(targets) == 0 {
		return map[string]any{
			"targets":       0,
			"snmp_ok":       0,
			"names_written": 0,
			"vlans_written": 0,
			"links_written": 0,
		}
	}

	if w.enrichMaxTargets > 0 && len(targets) > w.enrichMaxTargets {
		targets = targets[:w.enrichMaxTargets]
	}

	resolver := &mdns.Resolver{}

	var snmpClient *snmp.Client
	var vlanCollector *vlan.Collector
	if w.snmpEnabled {
		snmpClient = snmp.NewClient(snmp.Config{
			Community: w.snmpCommunity,
			Version:   w.snmpVersion,
			Port:      w.snmpPort,
			Timeout:   w.snmpTimeout,
			Retries:   w.snmpRetries,
		})
		vlanCollector = vlan.NewCollector(snmpClient)
	}

	var snmpOK int32
	var namesWritten int32
	var vlansWritten int32
	var linksWritten int32

	snmpAttempted := sync.Map{}
	nameAttempted := sync.Map{}

	jobs := make(chan enrichmentTarget)
	wg := sync.WaitGroup{}

	worker := func() {
		defer wg.Done()
		for t := range jobs {
			if ctx.Err() != nil {
				return
			}

			ipStr := t.IP.String()
			ipPtr := &ipStr
			deviceCandidates := make([]naming.Candidate, 0, 8)
			deviceNames := make([]string, 0, 8)
			upsertSuggestions := func(deviceID string, suggestions []tagging.Suggestion) {
				for _, s := range suggestions {
					if s.Evidence == nil {
						s.Evidence = map[string]any{}
					}
					s.Evidence["ip"] = ipStr
					_ = w.q.UpsertDeviceTag(ctx, sqlcgen.UpsertDeviceTagParams{
						DeviceID:   deviceID,
						Tag:        s.Tag,
						Source:     "auto",
						Confidence: int32(s.Confidence),
						Evidence:   s.Evidence,
					})
				}
			}

			if w.snmpEnabled && snmpClient != nil {
				if _, loaded := snmpAttempted.LoadOrStore(t.DeviceID, struct{}{}); loaded {
					// SNMP enrichment (including display name selection) should run once per device.
					continue
				}

				if w.nameResolutionEnabled {
					if _, loaded := nameAttempted.LoadOrStore(t.DeviceID, struct{}{}); !loaded {
						nameCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
						cands, err := resolver.LookupAddr(nameCtx, t.DeviceID, ipStr)
						cancel()
						if err != nil && len(cands) == 0 {
							w.log.Debug().Err(err).Str("ip", ipStr).Msg("name resolution failed")
						}
						if len(cands) > 0 {
							for _, c := range cands {
								stored, _, _, ok := naming.NormalizeCandidate(c.Source, c.Name)
								if !ok {
									continue
								}
								deviceCandidates = append(deviceCandidates, naming.Candidate{Name: stored, Source: c.Source})
								deviceNames = append(deviceNames, stored)
								if err := w.q.InsertDeviceNameCandidate(ctx, sqlcgen.InsertDeviceNameCandidateParams{
									DeviceID: t.DeviceID,
									Name:     stored,
									Source:   c.Source,
									Address:  ipPtr,
								}); err == nil {
									atomic.AddInt32(&namesWritten, 1)
								}
							}
						}
					}
				}

				target := snmp.Target{ID: t.DeviceID, Address: ipStr}
				system, err := snmpClient.GetSystem(ctx, target)
				now := time.Now()
				if err != nil {
					msg := err.Error()
					_ = w.q.UpsertDeviceSNMP(ctx, sqlcgen.UpsertDeviceSNMPParams{
						DeviceID:      t.DeviceID,
						Address:       ipPtr,
						LastSuccessAt: nil,
						LastError:     &msg,
					})

					if displayName, ok := naming.ChooseBestDisplayName(deviceCandidates); ok {
						_, _ = w.q.SetDeviceDisplayNameIfUnset(ctx, sqlcgen.SetDeviceDisplayNameIfUnsetParams{
							ID:          t.DeviceID,
							DisplayName: displayName,
						})
					}
					upsertSuggestions(t.DeviceID, tagging.MergeSuggestions(tagging.SuggestFromNames(deviceNames)))

					continue
				}

				atomic.AddInt32(&snmpOK, 1)
				_ = w.q.UpsertDeviceSNMP(ctx, sqlcgen.UpsertDeviceSNMPParams{
					DeviceID:      t.DeviceID,
					Address:       ipPtr,
					SysName:       system.SysName,
					SysDescr:      system.SysDescr,
					SysObjectID:   system.SysObjectID,
					SysContact:    system.SysContact,
					SysLocation:   system.SysLocation,
					LastSuccessAt: &now,
					LastError:     nil,
				})

				if system.SysName != nil && strings.TrimSpace(*system.SysName) != "" {
					stored, _, _, ok := naming.NormalizeCandidate("snmp", *system.SysName)
					if ok {
						deviceCandidates = append(deviceCandidates, naming.Candidate{Name: stored, Source: "snmp"})
						deviceNames = append(deviceNames, stored)
						_ = w.q.InsertDeviceNameCandidate(ctx, sqlcgen.InsertDeviceNameCandidateParams{
							DeviceID: t.DeviceID,
							Name:     stored,
							Source:   "snmp",
							Address:  ipPtr,
						})
					}
				}

				if system.SysDescr != nil && strings.TrimSpace(*system.SysDescr) != "" {
					upsertSuggestions(t.DeviceID, tagging.MergeSuggestions(
						tagging.SuggestFromSNMP(*system.SysDescr),
						tagging.SuggestFromNames(deviceNames),
					))
				} else {
					upsertSuggestions(t.DeviceID, tagging.MergeSuggestions(tagging.SuggestFromNames(deviceNames)))
				}

				if displayName, ok := naming.ChooseBestDisplayName(deviceCandidates); ok {
					_, _ = w.q.SetDeviceDisplayNameIfUnset(ctx, sqlcgen.SetDeviceDisplayNameIfUnsetParams{
						ID:          t.DeviceID,
						DisplayName: displayName,
					})
				}

				ifaces, err := snmpClient.WalkInterfaces(ctx, target)
				if err != nil {
					continue
				}

				ifIndexToInterfaceID := make(map[int]string, len(ifaces))
				for ifIndex, info := range ifaces {
					interfaceID, err := w.q.UpsertInterfaceFromSNMP(ctx, sqlcgen.UpsertInterfaceFromSNMPParams{
						DeviceID:    t.DeviceID,
						Ifindex:     int32(ifIndex),
						Name:        info.Name,
						Descr:       info.Descr,
						Alias:       info.Alias,
						MAC:         info.MAC,
						AdminStatus: info.AdminStatus,
						OperStatus:  info.OperStatus,
						MTU:         info.MTU,
						SpeedBps:    info.SpeedBps,
					})
					if err != nil || interfaceID == "" {
						continue
					}
					ifIndexToInterfaceID[ifIndex] = interfaceID

					if info.MAC != nil && *info.MAC != "" {
						_ = w.q.UpsertDeviceMAC(ctx, sqlcgen.UpsertDeviceMACParams{
							DeviceID: t.DeviceID,
							MAC:      *info.MAC,
						})
						_ = w.q.UpsertInterfaceMAC(ctx, sqlcgen.UpsertInterfaceMACParams{
							DeviceID:    t.DeviceID,
							InterfaceID: interfaceID,
							MAC:         *info.MAC,
						})
						_, _ = w.q.LinkDeviceMACToInterface(ctx, sqlcgen.LinkDeviceMACToInterfaceParams{
							DeviceID:    t.DeviceID,
							MAC:         *info.MAC,
							InterfaceID: interfaceID,
						})
					}
				}

				if vlanCollector != nil && len(ifIndexToInterfaceID) > 0 {
					pvidByIfIndex, err := vlanCollector.CollectPVIDByIfIndex(ctx, target)
					if err != nil {
						continue
					}
					for ifIndex, vlanID := range pvidByIfIndex {
						interfaceID := ifIndexToInterfaceID[ifIndex]
						if interfaceID == "" || vlanID <= 0 {
							continue
						}
						if err := w.q.UpsertInterfaceVLAN(ctx, sqlcgen.UpsertInterfaceVLANParams{
							InterfaceID: interfaceID,
							VlanID:      int32(vlanID),
							Role:        "pvid",
							Source:      "snmp",
						}); err == nil {
							atomic.AddInt32(&vlansWritten, 1)
						}
					}
				}

				if (w.topologyLLDPEnabled || w.topologyCDPEnabled) && allowedByAllowlist(t.IP, w.topologyAllowlist) {
					var neighbors []snmp.Neighbor
					if w.topologyLLDPEnabled {
						if ns, err := snmpClient.WalkLLDPNeighbors(ctx, target); err == nil {
							neighbors = append(neighbors, ns...)
						}
					}
					if w.topologyCDPEnabled {
						if ns, err := snmpClient.WalkCDPNeighbors(ctx, target); err == nil {
							neighbors = append(neighbors, ns...)
						}
					}

					now := time.Now()
					linkType := "ethernet"
					observedAt := &now

					for _, n := range neighbors {
						if ctx.Err() != nil {
							return
						}

						remoteDeviceID := ""
						if n.RemoteChassisMAC != nil && *n.RemoteChassisMAC != "" {
							id, err := w.q.FindDeviceIDByMAC(ctx, *n.RemoteChassisMAC)
							if err == nil {
								remoteDeviceID = id
							}
						}
						if remoteDeviceID == "" && n.RemoteMgmtIP != nil && *n.RemoteMgmtIP != "" {
							id, err := w.q.FindDeviceIDByIP(ctx, *n.RemoteMgmtIP)
							if err == nil {
								remoteDeviceID = id
							}
						}

						if remoteDeviceID == "" {
							display := n.RemoteDeviceName
							created, err := w.q.CreateDevice(ctx, display)
							if err != nil {
								continue
							}
							remoteDeviceID = created.ID
						}
						if remoteDeviceID == "" || remoteDeviceID == t.DeviceID {
							continue
						}

						if n.RemoteChassisMAC != nil && *n.RemoteChassisMAC != "" {
							_ = w.q.UpsertDeviceMAC(ctx, sqlcgen.UpsertDeviceMACParams{
								DeviceID: remoteDeviceID,
								MAC:      *n.RemoteChassisMAC,
							})
						}
						if n.RemoteDeviceName != nil && strings.TrimSpace(*n.RemoteDeviceName) != "" {
							stored, display, score, ok := naming.NormalizeCandidate(n.Source, strings.TrimSpace(*n.RemoteDeviceName))
							if !ok || score < 70 {
								continue
							}
							_ = w.q.InsertDeviceNameCandidate(ctx, sqlcgen.InsertDeviceNameCandidateParams{
								DeviceID: remoteDeviceID,
								Name:     stored,
								Source:   n.Source,
								Address:  nil,
							})
							if display != "" {
								_, _ = w.q.SetDeviceDisplayNameIfUnset(ctx, sqlcgen.SetDeviceDisplayNameIfUnsetParams{
									ID:          remoteDeviceID,
									DisplayName: display,
								})
							}
						}

						var localInterfaceID *string
						if n.LocalIfIndex != nil {
							ifaceID := ifIndexToInterfaceID[*n.LocalIfIndex]
							if ifaceID != "" {
								localInterfaceID = &ifaceID
							}
						}

						var remoteInterfaceID *string
						if n.RemotePortName != nil && strings.TrimSpace(*n.RemotePortName) != "" {
							ifaceID, err := w.q.UpsertInterfaceByName(ctx, sqlcgen.UpsertInterfaceByNameParams{
								DeviceID: remoteDeviceID,
								Name:     strings.TrimSpace(*n.RemotePortName),
							})
							if err == nil && ifaceID != "" {
								remoteInterfaceID = &ifaceID
							}
						}

						aDev, aIf, bDev, bIf := canonicalizeLinkEndpoints(t.DeviceID, localInterfaceID, remoteDeviceID, remoteInterfaceID)
						linkKey := makeLinkKey(n.Source, aDev, aIf, bDev, bIf)
						if err := w.q.UpsertLink(ctx, sqlcgen.UpsertLinkParams{
							LinkKey:      linkKey,
							ADeviceID:    aDev,
							AInterfaceID: aIf,
							BDeviceID:    bDev,
							BInterfaceID: bIf,
							LinkType:     &linkType,
							Source:       n.Source,
							ObservedAt:   observedAt,
						}); err == nil {
							atomic.AddInt32(&linksWritten, 1)
						}
					}
				}
			}

			// If SNMP is disabled, still attempt to auto-name from reverse DNS / mDNS / NetBIOS.
			if w.nameResolutionEnabled {
				if _, loaded := nameAttempted.LoadOrStore(t.DeviceID, struct{}{}); loaded {
					continue
				}
				nameCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
				cands, err := resolver.LookupAddr(nameCtx, t.DeviceID, ipStr)
				cancel()
				if err != nil && len(cands) == 0 {
					w.log.Debug().Err(err).Str("ip", ipStr).Msg("name resolution failed")
				}
				if len(cands) > 0 {
					for _, c := range cands {
						stored, _, _, ok := naming.NormalizeCandidate(c.Source, c.Name)
						if !ok {
							continue
						}
						deviceCandidates = append(deviceCandidates, naming.Candidate{Name: stored, Source: c.Source})
						deviceNames = append(deviceNames, stored)
						if err := w.q.InsertDeviceNameCandidate(ctx, sqlcgen.InsertDeviceNameCandidateParams{
							DeviceID: t.DeviceID,
							Name:     stored,
							Source:   c.Source,
							Address:  ipPtr,
						}); err == nil {
							atomic.AddInt32(&namesWritten, 1)
						}
					}
				}
				if displayName, ok := naming.ChooseBestDisplayName(deviceCandidates); ok {
					_, _ = w.q.SetDeviceDisplayNameIfUnset(ctx, sqlcgen.SetDeviceDisplayNameIfUnsetParams{
						ID:          t.DeviceID,
						DisplayName: displayName,
					})
				}
				upsertSuggestions(t.DeviceID, tagging.MergeSuggestions(tagging.SuggestFromNames(deviceNames)))
			}
		}
	}

	workers := w.enrichWorkers
	if workers <= 0 {
		workers = 8
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, t := range targets {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return map[string]any{
				"targets":       len(targets),
				"snmp_ok":       int(snmpOK),
				"names_written": int(namesWritten),
				"vlans_written": int(vlansWritten),
				"links_written": int(linksWritten),
				"canceled":      true,
			}
		case jobs <- t:
		}
	}
	close(jobs)
	wg.Wait()

	return map[string]any{
		"targets":       len(targets),
		"snmp_ok":       int(snmpOK),
		"names_written": int(namesWritten),
		"vlans_written": int(vlansWritten),
		"links_written": int(linksWritten),
	}
}

func allowedByAllowlist(ip netip.Addr, allowlist []netip.Prefix) bool {
	if !ip.IsValid() || len(allowlist) == 0 {
		return false
	}
	for _, p := range allowlist {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func canonicalizeLinkEndpoints(aDev string, aIf *string, bDev string, bIf *string) (string, *string, string, *string) {
	if aDev < bDev {
		return aDev, aIf, bDev, bIf
	}
	if aDev > bDev {
		return bDev, bIf, aDev, aIf
	}

	aIfKey := ""
	if aIf != nil {
		aIfKey = *aIf
	}
	bIfKey := ""
	if bIf != nil {
		bIfKey = *bIf
	}
	if aIfKey <= bIfKey {
		return aDev, aIf, bDev, bIf
	}
	return bDev, bIf, aDev, aIf
}

func makeLinkKey(source string, aDev string, aIf *string, bDev string, bIf *string) string {
	aIfKey := "-"
	if aIf != nil && strings.TrimSpace(*aIf) != "" {
		aIfKey = strings.TrimSpace(*aIf)
	}
	bIfKey := "-"
	if bIf != nil && strings.TrimSpace(*bIf) != "" {
		bIfKey = strings.TrimSpace(*bIf)
	}
	return strings.TrimSpace(source) + ":" + aDev + ":" + aIfKey + ":" + bDev + ":" + bIfKey
}
