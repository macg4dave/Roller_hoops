package discoveryworker

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"roller_hoops/core-go/internal/sqlcgen"
)

type nmapRun struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Ports []nmapPort `xml:"ports>port"`
}

type nmapPort struct {
	Protocol string     `xml:"protocol,attr"`
	PortID   int        `xml:"portid,attr"`
	State    nmapState  `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name string `xml:"name,attr"`
}

type portScanTarget struct {
	DeviceID string
	IP       string
}

func (w *Worker) runPortScan(ctx context.Context, targets []enrichmentTarget) map[string]any {
	if w == nil || w.q == nil || !w.portScanEnabled {
		return nil
	}
	if len(w.portScanAllowlist) == 0 || len(w.portScanPorts) == 0 {
		return map[string]any{"enabled": true, "available": false, "reason": "no_allowlist_or_ports"}
	}

	nmapPath, err := exec.LookPath("nmap")
	if err != nil {
		return map[string]any{"enabled": true, "available": false, "reason": "nmap_not_found"}
	}

	seenDevice := map[string]struct{}{}
	scanTargets := make([]portScanTarget, 0, len(targets))
	for _, t := range targets {
		if _, ok := seenDevice[t.DeviceID]; ok {
			continue
		}
		if !allowedByAllowlist(t.IP, w.portScanAllowlist) {
			continue
		}
		seenDevice[t.DeviceID] = struct{}{}
		scanTargets = append(scanTargets, portScanTarget{DeviceID: t.DeviceID, IP: t.IP.String()})
		if w.portScanMaxTargets > 0 && len(scanTargets) >= w.portScanMaxTargets {
			break
		}
	}
	if len(scanTargets) == 0 {
		return map[string]any{"enabled": true, "available": true, "targets": 0, "services_written": 0}
	}

	ports := make([]string, 0, len(w.portScanPorts))
	for _, p := range w.portScanPorts {
		if p <= 0 || p > 65535 {
			continue
		}
		ports = append(ports, strconv.Itoa(p))
	}
	if len(ports) == 0 {
		return map[string]any{"enabled": true, "available": true, "targets": len(scanTargets), "services_written": 0}
	}
	portArg := strings.Join(ports, ",")

	var attempted int32
	var succeeded int32
	var servicesWritten int32

	jobs := make(chan portScanTarget)
	wg := sync.WaitGroup{}

	worker := func() {
		defer wg.Done()
		for t := range jobs {
			if ctx.Err() != nil {
				return
			}
			atomic.AddInt32(&attempted, 1)

			scanCtx, cancel := context.WithTimeout(ctx, w.portScanTimeout)
			out, err := exec.CommandContext(scanCtx, nmapPath,
				"-oX", "-",
				"-Pn",
				"-sT",
				"--host-timeout", w.portScanTimeout.String(),
				"--max-retries", "1",
				"--open",
				"-p", portArg,
				t.IP,
			).Output()
			cancel()
			if err != nil {
				continue
			}
			atomic.AddInt32(&succeeded, 1)

			var run nmapRun
			if err := xml.Unmarshal(out, &run); err != nil {
				continue
			}

			now := time.Now()
			source := "nmap"
			state := "open"

			for _, h := range run.Hosts {
				for _, p := range h.Ports {
					if strings.ToLower(p.State.State) != "open" {
						continue
					}
					proto := strings.ToLower(strings.TrimSpace(p.Protocol))
					if proto != "tcp" && proto != "udp" {
						continue
					}
					if p.PortID <= 0 || p.PortID > 65535 {
						continue
					}
					var name *string
					if s := strings.TrimSpace(p.Service.Name); s != "" {
						name = &s
					}

					if err := w.q.UpsertServiceFromScan(ctx, sqlcgen.UpsertServiceFromScanParams{
						DeviceID:   t.DeviceID,
						Protocol:   proto,
						Port:       int32(p.PortID),
						Name:       name,
						State:      &state,
						Source:     &source,
						ObservedAt: now,
					}); err == nil {
						atomic.AddInt32(&servicesWritten, 1)
					}
				}
			}
		}
	}

	workers := w.portScanWorkers
	if workers <= 0 {
		workers = 4
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, t := range scanTargets {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return map[string]any{
				"enabled":         true,
				"available":       true,
				"targets":         len(scanTargets),
				"attempted":       int(attempted),
				"succeeded":       int(succeeded),
				"services_written": int(servicesWritten),
				"canceled":        true,
			}
		case jobs <- t:
		}
	}
	close(jobs)
	wg.Wait()

	return map[string]any{
		"enabled":         true,
		"available":       true,
		"targets":         len(scanTargets),
		"attempted":       int(attempted),
		"succeeded":       int(succeeded),
		"services_written": int(servicesWritten),
		"ports":           portArg,
		"timeout":         w.portScanTimeout.String(),
	}
}

func (w *Worker) portScanLogMessage(stats map[string]any) string {
	if stats == nil {
		return ""
	}
	if avail, ok := stats["available"].(bool); ok && !avail {
		if reason, ok := stats["reason"].(string); ok && reason != "" {
			return fmt.Sprintf("port scan skipped: %s", reason)
		}
		return "port scan skipped"
	}
	return fmt.Sprintf("port scan: targets=%v attempted=%v succeeded=%v services=%v", stats["targets"], stats["attempted"], stats["succeeded"], stats["services_written"])
}

