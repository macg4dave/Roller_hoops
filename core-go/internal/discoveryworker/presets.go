package discoveryworker

import (
	"strings"
	"time"
)

const (
	ScanPresetFast   = "fast"
	ScanPresetNormal = "normal"
	ScanPresetDeep   = "deep"
)

func canonicalizeScanPreset(value any) string {
	switch v := value.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "" {
			return ScanPresetNormal
		}
		switch s {
		case ScanPresetFast, ScanPresetNormal, ScanPresetDeep:
			return s
		default:
			return ScanPresetNormal
		}
	default:
		return ScanPresetNormal
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func maxDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a > b {
		return a
	}
	return b
}

func applyScanPreset(w *Worker, preset string) func() {
	if w == nil {
		return func() {}
	}

	prev := struct {
		maxRuntime          time.Duration
		maxTargets          int
		pingTimeout         time.Duration
		pingWorkers         int
		enrichMaxTargets    int
		enrichWorkers       int
		snmpEnabled         bool
		topologyLLDPEnabled bool
		topologyCDPEnabled  bool
		portScanEnabled     bool
		portScanWorkers     int
		portScanTimeout     time.Duration
		portScanMaxTargets  int
	}{
		maxRuntime:          w.maxRuntime,
		maxTargets:          w.maxTargets,
		pingTimeout:         w.pingTimeout,
		pingWorkers:         w.pingWorkers,
		enrichMaxTargets:    w.enrichMaxTargets,
		enrichWorkers:       w.enrichWorkers,
		snmpEnabled:         w.snmpEnabled,
		topologyLLDPEnabled: w.topologyLLDPEnabled,
		topologyCDPEnabled:  w.topologyCDPEnabled,
		portScanEnabled:     w.portScanEnabled,
		portScanWorkers:     w.portScanWorkers,
		portScanTimeout:     w.portScanTimeout,
		portScanMaxTargets:  w.portScanMaxTargets,
	}

	switch preset {
	case ScanPresetFast:
		w.maxRuntime = minDuration(w.maxRuntime, 15*time.Second)
		w.maxTargets = minInt(w.maxTargets, 256)
		w.pingTimeout = minDuration(w.pingTimeout, 400*time.Millisecond)
		w.pingWorkers = minInt(w.pingWorkers, 16)
		w.enrichMaxTargets = minInt(w.enrichMaxTargets, 32)
		w.enrichWorkers = minInt(w.enrichWorkers, 4)
		w.snmpEnabled = false
		w.topologyLLDPEnabled = false
		w.topologyCDPEnabled = false
		w.portScanEnabled = false
	case ScanPresetDeep:
		w.maxRuntime = maxDuration(w.maxRuntime, 2*time.Minute)
		w.maxTargets = maxInt(w.maxTargets, 4096)
		w.pingTimeout = maxDuration(w.pingTimeout, 1500*time.Millisecond)
		w.pingWorkers = maxInt(w.pingWorkers, 32)
		w.enrichMaxTargets = maxInt(w.enrichMaxTargets, 256)
		w.enrichWorkers = maxInt(w.enrichWorkers, 16)
		w.snmpEnabled = true
		w.topologyLLDPEnabled = true
		w.topologyCDPEnabled = true
		w.portScanEnabled = true
		w.portScanWorkers = maxInt(w.portScanWorkers, 8)
		w.portScanTimeout = maxDuration(w.portScanTimeout, 5*time.Second)
		w.portScanMaxTargets = maxInt(w.portScanMaxTargets, 64)
	default:
		// normal: preserve configured values
	}

	return func() {
		w.maxRuntime = prev.maxRuntime
		w.maxTargets = prev.maxTargets
		w.pingTimeout = prev.pingTimeout
		w.pingWorkers = prev.pingWorkers
		w.enrichMaxTargets = prev.enrichMaxTargets
		w.enrichWorkers = prev.enrichWorkers
		w.snmpEnabled = prev.snmpEnabled
		w.topologyLLDPEnabled = prev.topologyLLDPEnabled
		w.topologyCDPEnabled = prev.topologyCDPEnabled
		w.portScanEnabled = prev.portScanEnabled
		w.portScanWorkers = prev.portScanWorkers
		w.portScanTimeout = prev.portScanTimeout
		w.portScanMaxTargets = prev.portScanMaxTargets
	}
}
