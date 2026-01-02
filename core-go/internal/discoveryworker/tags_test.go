package discoveryworker

import "testing"

func TestCanonicalizeScanTags(t *testing.T) {
	got := canonicalizeScanTags([]any{" Ports ", "snmp", "banana", 123, "ports", "TOPOLOGY"})
	want := []string{"ports", "snmp", "topology"}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestApplyScanTags_Restores(t *testing.T) {
	w := &Worker{
		nameResolutionEnabled: false,
		snmpEnabled:           false,
		topologyLLDPEnabled:   false,
		topologyCDPEnabled:    false,
		portScanEnabled:       false,
	}

	restore := applyScanTags(w, []string{ScanTagTopology, ScanTagPorts, ScanTagNames})

	if !w.snmpEnabled {
		t.Fatalf("expected snmp enabled")
	}
	if !w.topologyLLDPEnabled || !w.topologyCDPEnabled {
		t.Fatalf("expected topology enabled")
	}
	if !w.portScanEnabled {
		t.Fatalf("expected port scan enabled")
	}
	if !w.nameResolutionEnabled {
		t.Fatalf("expected name resolution enabled")
	}

	restore()

	if w.snmpEnabled || w.topologyLLDPEnabled || w.topologyCDPEnabled || w.portScanEnabled || w.nameResolutionEnabled {
		t.Fatalf("expected restore to reset flags, got %+v", w)
	}
}

