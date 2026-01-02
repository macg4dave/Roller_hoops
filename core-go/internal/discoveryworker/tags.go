package discoveryworker

import (
	"sort"
	"strings"
)

const (
	ScanTagPorts    = "ports"
	ScanTagSNMP     = "snmp"
	ScanTagTopology = "topology"
	ScanTagNames    = "names"
)

func canonicalizeScanTags(value any) []string {
	var raw []string

	switch v := value.(type) {
	case []string:
		raw = v
	case []any:
		for _, entry := range v {
			if s, ok := entry.(string); ok {
				raw = append(raw, s)
			}
		}
	case string:
		raw = []string{v}
	default:
		return nil
	}

	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, entry := range raw {
		s := strings.ToLower(strings.TrimSpace(entry))
		if s == "" {
			continue
		}
		switch s {
		case ScanTagPorts, ScanTagSNMP, ScanTagTopology, ScanTagNames:
		default:
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func applyScanTags(w *Worker, tags []string) func() {
	if w == nil || len(tags) == 0 {
		return func() {}
	}

	prev := struct {
		nameResolutionEnabled bool
		snmpEnabled           bool
		topologyLLDPEnabled   bool
		topologyCDPEnabled    bool
		portScanEnabled       bool
	}{
		nameResolutionEnabled: w.nameResolutionEnabled,
		snmpEnabled:           w.snmpEnabled,
		topologyLLDPEnabled:   w.topologyLLDPEnabled,
		topologyCDPEnabled:    w.topologyCDPEnabled,
		portScanEnabled:       w.portScanEnabled,
	}

	for _, tag := range tags {
		switch tag {
		case ScanTagPorts:
			w.portScanEnabled = true
		case ScanTagSNMP:
			w.snmpEnabled = true
		case ScanTagTopology:
			w.snmpEnabled = true
			w.topologyLLDPEnabled = true
			w.topologyCDPEnabled = true
		case ScanTagNames:
			w.nameResolutionEnabled = true
		}
	}

	return func() {
		w.nameResolutionEnabled = prev.nameResolutionEnabled
		w.snmpEnabled = prev.snmpEnabled
		w.topologyLLDPEnabled = prev.topologyLLDPEnabled
		w.topologyCDPEnabled = prev.topologyCDPEnabled
		w.portScanEnabled = prev.portScanEnabled
	}
}
