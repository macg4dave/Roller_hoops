package discoveryworker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/metrics"
	"roller_hoops/core-go/internal/sqlcgen"
)

// Queries is the minimal DB interface the discovery worker needs.
//
// NOTE: core-go uses sqlc for DB access. *sqlcgen.Queries satisfies this.
type Queries interface {
	ClaimNextDiscoveryRun(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error)
	UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
	CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	FindDeviceIDByMAC(ctx context.Context, mac string) (string, error)
	FindDeviceIDByIP(ctx context.Context, ip string) (string, error)
	UpsertDeviceIP(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error
	UpsertDeviceMAC(ctx context.Context, arg sqlcgen.UpsertDeviceMACParams) error
	InsertIPObservation(ctx context.Context, arg sqlcgen.InsertIPObservationParams) error
	InsertMACObservation(ctx context.Context, arg sqlcgen.InsertMACObservationParams) error
	InsertDeviceNameCandidate(ctx context.Context, arg sqlcgen.InsertDeviceNameCandidateParams) error
	SetDeviceDisplayNameIfUnset(ctx context.Context, arg sqlcgen.SetDeviceDisplayNameIfUnsetParams) (int64, error)
	UpsertDeviceSNMP(ctx context.Context, arg sqlcgen.UpsertDeviceSNMPParams) error
	UpsertInterfaceFromSNMP(ctx context.Context, arg sqlcgen.UpsertInterfaceFromSNMPParams) (string, error)
	UpsertInterfaceByName(ctx context.Context, arg sqlcgen.UpsertInterfaceByNameParams) (string, error)
	UpsertInterfaceMAC(ctx context.Context, arg sqlcgen.UpsertInterfaceMACParams) error
	LinkDeviceMACToInterface(ctx context.Context, arg sqlcgen.LinkDeviceMACToInterfaceParams) (int64, error)
	UpsertInterfaceVLAN(ctx context.Context, arg sqlcgen.UpsertInterfaceVLANParams) error
	UpsertLink(ctx context.Context, arg sqlcgen.UpsertLinkParams) error
	UpsertServiceFromScan(ctx context.Context, arg sqlcgen.UpsertServiceFromScanParams) error
}

type Worker struct {
	log                   zerolog.Logger
	q                     Queries
	pollInterval          time.Duration
	runDelay              time.Duration
	maxRuntime            time.Duration
	arpTablePath          string
	maxTargets            int
	pingTimeout           time.Duration
	pingWorkers           int
	enrichMaxTargets      int
	enrichWorkers         int
	nameResolutionEnabled bool
	snmpEnabled           bool
	snmpCommunity         string
	snmpVersion           string
	snmpTimeout           time.Duration
	snmpRetries           int
	snmpPort              uint16
	topologyLLDPEnabled   bool
	topologyCDPEnabled    bool
	topologyAllowlist     []netip.Prefix
	portScanEnabled       bool
	portScanAllowlist     []netip.Prefix
	portScanPorts         []int
	portScanWorkers       int
	portScanTimeout       time.Duration
	portScanMaxTargets    int
	metrics               *metrics.Metrics
}

type Options struct {
	PollInterval          time.Duration
	RunDelay              time.Duration
	MaxRuntime            time.Duration
	ARPTablePath          string
	MaxTargets            int
	PingTimeout           time.Duration
	PingWorkers           int
	EnrichMaxTargets      int
	EnrichWorkers         int
	NameResolutionEnabled bool
	SNMPEnabled           bool
	SNMPCommunity         string
	SNMPVersion           string
	SNMPTimeout           time.Duration
	SNMPRetries           int
	SNMPPort              uint16
	TopologyLLDPEnabled   bool
	TopologyCDPEnabled    bool
	TopologyAllowlist     []netip.Prefix
	PortScanEnabled       bool
	PortScanAllowlist     []netip.Prefix
	PortScanPorts         []int
	PortScanWorkers       int
	PortScanTimeout       time.Duration
	PortScanMaxTargets    int
}

func New(log zerolog.Logger, q Queries, opts Options, m *metrics.Metrics) *Worker {
	pi := opts.PollInterval
	if pi <= 0 {
		pi = 400 * time.Millisecond
	}
	rd := opts.RunDelay
	if rd < 0 {
		rd = 0
	}
	mr := opts.MaxRuntime
	if mr <= 0 {
		mr = 30 * time.Second
	}
	arpPath := opts.ARPTablePath
	if strings.TrimSpace(arpPath) == "" {
		arpPath = "/proc/net/arp"
	}
	maxTargets := opts.MaxTargets
	if maxTargets <= 0 {
		maxTargets = 1024
	}
	pingTimeout := opts.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = 800 * time.Millisecond
	}
	pingWorkers := opts.PingWorkers
	if pingWorkers <= 0 {
		pingWorkers = 16
	}

	enrichMaxTargets := opts.EnrichMaxTargets
	if enrichMaxTargets <= 0 {
		enrichMaxTargets = 64
	}
	enrichWorkers := opts.EnrichWorkers
	if enrichWorkers <= 0 {
		enrichWorkers = 8
	}

	snmpTimeout := opts.SNMPTimeout
	if snmpTimeout <= 0 {
		snmpTimeout = 900 * time.Millisecond
	}
	snmpCommunity := strings.TrimSpace(opts.SNMPCommunity)
	if snmpCommunity == "" {
		snmpCommunity = "public"
	}
	snmpVersion := strings.TrimSpace(opts.SNMPVersion)
	if snmpVersion == "" {
		snmpVersion = "2c"
	}
	snmpRetries := opts.SNMPRetries
	if snmpRetries < 0 {
		snmpRetries = 0
	}
	snmpPort := opts.SNMPPort
	if snmpPort == 0 {
		snmpPort = 161
	}

	portScanWorkers := opts.PortScanWorkers
	if portScanWorkers <= 0 {
		portScanWorkers = 4
	}
	portScanTimeout := opts.PortScanTimeout
	if portScanTimeout <= 0 {
		portScanTimeout = 3 * time.Second
	}
	portScanMaxTargets := opts.PortScanMaxTargets
	if portScanMaxTargets <= 0 {
		portScanMaxTargets = 24
	}

	return &Worker{
		log:                   log,
		q:                     q,
		pollInterval:          pi,
		runDelay:              rd,
		maxRuntime:            mr,
		arpTablePath:          arpPath,
		maxTargets:            maxTargets,
		pingTimeout:           pingTimeout,
		pingWorkers:           pingWorkers,
		enrichMaxTargets:      enrichMaxTargets,
		enrichWorkers:         enrichWorkers,
		nameResolutionEnabled: opts.NameResolutionEnabled,
		snmpEnabled:           opts.SNMPEnabled,
		snmpCommunity:         snmpCommunity,
		snmpVersion:           snmpVersion,
		snmpTimeout:           snmpTimeout,
		snmpRetries:           snmpRetries,
		snmpPort:              snmpPort,
		topologyLLDPEnabled:   opts.TopologyLLDPEnabled,
		topologyCDPEnabled:    opts.TopologyCDPEnabled,
		topologyAllowlist:     opts.TopologyAllowlist,
		portScanEnabled:       opts.PortScanEnabled,
		portScanAllowlist:     opts.PortScanAllowlist,
		portScanPorts:         opts.PortScanPorts,
		portScanWorkers:       portScanWorkers,
		portScanTimeout:       portScanTimeout,
		portScanMaxTargets:    portScanMaxTargets,
		metrics:               m,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.q == nil {
		return
	}

	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()

	var consecutiveFailures int
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		for {
			processed, err := w.runOnce(ctx)
			if err != nil {
				consecutiveFailures++
				break
			}
			consecutiveFailures = 0
			if !processed {
				break
			}
		}

		timer.Reset(backoffDuration(w.pollInterval, consecutiveFailures))
	}
}

func backoffDuration(base time.Duration, failures int) time.Duration {
	if base <= 0 {
		base = 400 * time.Millisecond
	}
	if failures <= 0 {
		return base
	}

	// Exponential-ish backoff: base * 2^failures, capped.
	if failures > 6 {
		failures = 6
	}
	d := base * time.Duration(1<<failures)
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}

func (w *Worker) runOnce(ctx context.Context) (bool, error) {
	// Claim a run.
	run, err := w.q.ClaimNextDiscoveryRun(ctx, map[string]any{
		"stage": "running",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		w.log.Error().Err(err).Msg("discovery worker failed to claim next run")
		return false, err
	}

	if w.metrics != nil {
		w.metrics.IncDiscoveryRun()
	}
	start := time.Now()
	defer func() {
		if w.metrics != nil {
			w.metrics.ObserveDiscoveryRunDuration(time.Since(start))
		}
	}()

	w.log.Info().Str("run_id", run.ID).Msg("discovery run claimed")

	preset := ScanPresetNormal
	if run.Stats != nil {
		preset = canonicalizeScanPreset(run.Stats["preset"])
	}
	restorePreset := applyScanPreset(w, preset)
	defer restorePreset()

	// Execute (ARP scrape to seed IP/MAC facts; other methods are Phase 8+).
	execCtx, cancel := context.WithTimeout(ctx, w.maxRuntime)
	defer cancel()

	if err := w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: "discovery run started",
	}); err != nil {
		w.log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to write discovery start log")
	}

	if err := w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: fmt.Sprintf("scan preset: %s", preset),
	}); err != nil {
		w.log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to write preset log")
	}

	if w.runDelay > 0 {
		t := time.NewTimer(w.runDelay)
		select {
		case <-execCtx.Done():
			t.Stop()
			_ = w.failRun(execCtx, run.ID, execCtx.Err().Error(), map[string]any{
				"stage": "failed",
				"scope": safeScopeString(run.Scope),
			})
			return true, execCtx.Err()
		case <-t.C:
		}
	}

	scopePrefix, err := parseDiscoveryScope(run.Scope)
	if err != nil {
		_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
			RunID:   run.ID,
			Level:   "error",
			Message: "invalid discovery scope: " + err.Error(),
		})
		_ = w.failRun(execCtx, run.ID, "invalid discovery scope", map[string]any{
			"stage": "failed",
			"scope": safeScopeString(run.Scope),
		})
		return true, err
	}

	var ping pingSweepResult
	var scopeTargets int
	if scopePrefix != nil {
		if count, err := countScopeTargets(*scopePrefix, w.maxTargets); err != nil {
			_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
				RunID:   run.ID,
				Level:   "error",
				Message: err.Error(),
			})
			_ = w.failRun(execCtx, run.ID, err.Error(), map[string]any{
				"stage":       "failed",
				"scope":       scopePrefix.String(),
				"max_targets": w.maxTargets,
			})
			return true, err
		} else {
			scopeTargets = count
			_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
				RunID:   run.ID,
				Level:   "info",
				Message: fmt.Sprintf("scope targets: %d (max=%d)", count, w.maxTargets),
			})
		}

		var pingErr error
		ping, pingErr = w.pingSweep(execCtx, *scopePrefix)
		if pingErr != nil {
			_ = w.failRun(execCtx, run.ID, pingErr.Error(), map[string]any{
				"stage": "failed",
				"scope": scopePrefix.String(),
			})
			return true, pingErr
		}
		if ping.Attempted > 0 {
			_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
				RunID:   run.ID,
				Level:   "info",
				Message: fmt.Sprintf("ping sweep: attempted=%d succeeded=%d", ping.Attempted, ping.Succeeded),
			})
		}
	}

	result, err := w.scrapeARP(execCtx, run.ID, scopePrefix)
	if err != nil {
		_ = w.failRun(execCtx, run.ID, err.Error(), map[string]any{
			"stage": "failed",
			"scope": scopePrefixOrNil(scopePrefix),
		})
		return true, err
	}

	_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: fmt.Sprintf("arp scrape: entries=%d devices_seen=%d devices_created=%d", result.ARPEntries, result.DevicesSeen, result.DevicesCreated),
	})

	enrichmentStats := w.runEnrichment(execCtx, result.Targets)
	if enrichmentStats != nil {
		_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
			RunID:   run.ID,
			Level:   "info",
			Message: fmt.Sprintf("enrichment: targets=%v snmp_ok=%v names=%v vlans=%v links=%v", enrichmentStats["targets"], enrichmentStats["snmp_ok"], enrichmentStats["names_written"], enrichmentStats["vlans_written"], enrichmentStats["links_written"]),
		})
	}

	portScanStats := w.runPortScan(execCtx, result.Targets)
	if msg := w.portScanLogMessage(portScanStats); msg != "" {
		_ = w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
			RunID:   run.ID,
			Level:   "info",
			Message: msg,
		})
	}

	completedAt := time.Now()
	stats := map[string]any{
		"stage":             "completed",
		"preset":            preset,
		"method":            discoveryMethod(ping),
		"scope":             scopePrefixOrNil(scopePrefix),
		"scope_targets":     scopeTargets,
		"max_targets":       w.maxTargets,
		"runtime_budget_ms": int(w.maxRuntime.Milliseconds()),
		"ping_available":    ping.Available,
		"ping_attempted":    ping.Attempted,
		"ping_succeeded":    ping.Succeeded,
		"arp_entries":       result.ARPEntries,
		"devices_seen":      result.DevicesSeen,
		"devices_created":   result.DevicesCreated,
	}
	if enrichmentStats != nil {
		stats["enrichment"] = enrichmentStats
	}
	if portScanStats != nil {
		stats["port_scan"] = portScanStats
	}
	if _, err := w.q.UpdateDiscoveryRun(execCtx, sqlcgen.UpdateDiscoveryRunParams{
		ID:          run.ID,
		Status:      "succeeded",
		Stats:       stats,
		CompletedAt: &completedAt,
		LastError:   nil,
	}); err != nil {
		w.log.Error().Err(err).Str("run_id", run.ID).Msg("failed to mark discovery run succeeded")

		msg := err.Error()
		_ = w.failRun(execCtx, run.ID, msg, map[string]any{
			"stage": "failed",
			"scope": scopePrefixOrNil(scopePrefix),
		})
		return true, err
	}

	if err := w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: "discovery run completed",
	}); err != nil {
		w.log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to write discovery completion log")
	}

	return true, nil
}

func (w *Worker) failRun(ctx context.Context, runID string, errMsg string, stats map[string]any) error {
	if stats == nil {
		stats = map[string]any{}
	}
	stats["stage"] = "failed"
	stats["max_targets"] = w.maxTargets
	stats["runtime_budget_ms"] = int(w.maxRuntime.Milliseconds())
	stats["ping_timeout_ms"] = int(w.pingTimeout.Milliseconds())
	stats["ping_workers"] = w.pingWorkers

	// If the provided context is already canceled/deadlined, still try to mark the run failed
	// with a short background context so we don't leave it stuck in "running".
	if ctx == nil || ctx.Err() != nil {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		ctx = bg
	}

	completedAt := time.Now()
	lastErr := errMsg
	_, err := w.q.UpdateDiscoveryRun(ctx, sqlcgen.UpdateDiscoveryRunParams{
		ID:          runID,
		Status:      "failed",
		Stats:       stats,
		CompletedAt: &completedAt,
		LastError:   &lastErr,
	})
	if err != nil {
		w.log.Error().Err(err).Str("run_id", runID).Msg("failed to mark discovery run failed")
		return err
	}

	_ = w.q.InsertDiscoveryRunLog(ctx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   runID,
		Level:   "error",
		Message: "discovery run failed: " + errMsg,
	})

	return nil
}

type arpScrapeResult struct {
	ARPEntries     int
	DevicesSeen    int
	DevicesCreated int
	Targets        []enrichmentTarget
}

type pingSweepResult struct {
	Attempted int
	Succeeded int
	Available bool
}

func safeScopeString(scope *string) *string {
	if scope == nil {
		return nil
	}
	s := strings.TrimSpace(*scope)
	if s == "" {
		return nil
	}
	return &s
}

func scopePrefixOrNil(p *netip.Prefix) *string {
	if p == nil {
		return nil
	}
	s := p.String()
	return &s
}

func discoveryMethod(ping pingSweepResult) string {
	if ping.Attempted > 0 || ping.Available {
		return "arp+icmp"
	}
	return "arp"
}

func parseDiscoveryScope(scope *string) (*netip.Prefix, error) {
	if scope == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*scope)
	if s == "" {
		return nil, nil
	}

	if p, err := netip.ParsePrefix(s); err == nil {
		return &p, nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		p := netip.PrefixFrom(a, a.BitLen())
		return &p, nil
	}

	return nil, fmt.Errorf("scope must be a CIDR prefix or a single IP (got %q)", s)
}

func countScopeTargets(p netip.Prefix, maxTargets int) (int, error) {
	p = p.Masked()

	if p.Addr().Is4() {
		bits := p.Bits()
		if bits < 0 || bits > 32 {
			return 0, fmt.Errorf("invalid scope bits: %d", bits)
		}
		hostBits := 32 - bits
		if hostBits >= 31 {
			return 0, fmt.Errorf("scope too large (/%d); max targets is %d", bits, maxTargets)
		}
		count := 1 << hostBits
		if count > maxTargets {
			return 0, fmt.Errorf("scope too large (%d targets); max targets is %d", count, maxTargets)
		}
		return count, nil
	}

	// Keep IPv6 tightly scoped until we have better iterators/limits.
	if p.Addr().Is6() {
		if p.Bits() < 128 {
			return 0, fmt.Errorf("ipv6 scope must be a single IP (/128) for now")
		}
		return 1, nil
	}

	return 0, fmt.Errorf("unsupported scope address family")
}

type arpEntry struct {
	IP  netip.Addr
	MAC string
}

func parseProcNetARP(content string) ([]arpEntry, error) {
	s := bufio.NewScanner(strings.NewReader(content))

	// Header line: "IP address       HW type     Flags       HW address            Mask     Device"
	if !s.Scan() {
		return nil, nil
	}

	var out []arpEntry
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		ipStr := fields[0]
		flagsStr := fields[2]
		macStr := strings.ToLower(fields[3])

		// Require a "complete" ARP entry.
		flags, err := strconv.ParseInt(flagsStr, 0, 64)
		if err != nil || flags&0x2 == 0 {
			continue
		}

		if macStr == "00:00:00:00:00:00" {
			continue
		}
		if _, err := net.ParseMAC(macStr); err != nil {
			continue
		}

		ip, err := netip.ParseAddr(ipStr)
		if err != nil {
			continue
		}
		out = append(out, arpEntry{IP: ip, MAC: macStr})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *Worker) scrapeARP(ctx context.Context, runID string, scope *netip.Prefix) (arpScrapeResult, error) {
	if w == nil {
		return arpScrapeResult{}, nil
	}

	content, err := os.ReadFile(w.arpTablePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return arpScrapeResult{}, nil
		}
		return arpScrapeResult{}, err
	}

	entries, err := parseProcNetARP(string(content))
	if err != nil {
		return arpScrapeResult{}, err
	}

	var result arpScrapeResult
	seenTargets := make(map[string]struct{})

	for _, e := range entries {
		if scope != nil && !scope.Contains(e.IP) {
			continue
		}
		result.ARPEntries++

		deviceID, err := w.q.FindDeviceIDByMAC(ctx, e.MAC)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return result, err
		}
		if errors.Is(err, pgx.ErrNoRows) {
			deviceID, err = w.q.FindDeviceIDByIP(ctx, e.IP.String())
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return result, err
			}
		}
		if deviceID == "" {
			row, err := w.q.CreateDevice(ctx, nil)
			if err != nil {
				return result, err
			}
			deviceID = row.ID
			result.DevicesCreated++
		}

		if err := w.q.UpsertDeviceMAC(ctx, sqlcgen.UpsertDeviceMACParams{
			DeviceID: deviceID,
			MAC:      e.MAC,
		}); err != nil {
			return result, err
		}
		if err := w.q.UpsertDeviceIP(ctx, sqlcgen.UpsertDeviceIPParams{
			DeviceID: deviceID,
			IP:       e.IP.String(),
		}); err != nil {
			return result, err
		}
		if runID != "" {
			if err := w.q.InsertMACObservation(ctx, sqlcgen.InsertMACObservationParams{
				RunID:    runID,
				DeviceID: deviceID,
				MAC:      e.MAC,
			}); err != nil {
				return result, err
			}
			if err := w.q.InsertIPObservation(ctx, sqlcgen.InsertIPObservationParams{
				RunID:    runID,
				DeviceID: deviceID,
				IP:       e.IP.String(),
			}); err != nil {
				return result, err
			}
		}
		result.DevicesSeen++

		key := deviceID + "|" + e.IP.String()
		if _, ok := seenTargets[key]; !ok {
			seenTargets[key] = struct{}{}
			result.Targets = append(result.Targets, enrichmentTarget{
				DeviceID: deviceID,
				IP:       e.IP,
			})
		}
	}

	return result, nil
}

func (w *Worker) pingSweep(ctx context.Context, scope netip.Prefix) (pingSweepResult, error) {
	scope = scope.Masked()

	pingPath, err := exec.LookPath("ping")
	if err != nil {
		return pingSweepResult{Available: false}, nil
	}
	result := pingSweepResult{Available: true}

	targetCount, err := countScopeTargets(scope, w.maxTargets)
	if err != nil {
		return result, err
	}
	if targetCount <= 0 {
		return result, nil
	}

	var attempted int32
	var succeeded int32

	jobs := make(chan netip.Addr, w.pingWorkers*2)
	wg := sync.WaitGroup{}

	worker := func() {
		defer wg.Done()
		for ip := range jobs {
			atomic.AddInt32(&attempted, 1)

			pingCtx, cancel := context.WithTimeout(ctx, w.pingTimeout)
			cmd := exec.CommandContext(pingCtx, pingPath, "-c", "1", "-W", "1", ip.String())
			cmd.Stdout = nil
			cmd.Stderr = nil
			err := cmd.Run()
			cancel()

			if err == nil {
				atomic.AddInt32(&succeeded, 1)
			}
		}
	}

	for i := 0; i < w.pingWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	// Only implement an iterator for small IPv4 scopes.
	if scope.Addr().Is4() {
		ip := scope.Addr()
		for scope.Contains(ip) {
			select {
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				result.Attempted = int(attempted)
				result.Succeeded = int(succeeded)
				return result, ctx.Err()
			case jobs <- ip:
				ip = ip.Next()
			}
		}
	}

	close(jobs)
	wg.Wait()

	result.Attempted = int(attempted)
	result.Succeeded = int(succeeded)
	return result, nil
}
