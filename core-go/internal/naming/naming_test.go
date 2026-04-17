package naming

import "testing"

func TestNormalizeCandidate(t *testing.T) {
	stored, display, score, ok := NormalizeCandidate("reverse_dns", "Router.Home.ARPA.")
	if !ok {
		t.Fatalf("expected ok")
	}
	if stored != "router.home.arpa" {
		t.Fatalf("expected stored lowercased without trailing dot, got %q", stored)
	}
	if display != "router" {
		t.Fatalf("expected display hostname label, got %q", display)
	}
	if score < 70 {
		t.Fatalf("expected score >= 70, got %d", score)
	}
}

func TestChooseBestDisplayName_PrefersHigherSignal(t *testing.T) {
	name, ok := ChooseBestDisplayName([]Candidate{
		{Name: "router.local", Source: "mdns"},
		{Name: "core-switch-1", Source: "snmp"},
	})
	if !ok {
		t.Fatalf("expected ok")
	}
	if name != "core-switch-1" {
		t.Fatalf("expected snmp to win, got %q", name)
	}
}

func TestChooseBestDisplayName_RejectsGarbage(t *testing.T) {
	name, ok := ChooseBestDisplayName([]Candidate{
		{Name: "4.3.2.1.in-addr.arpa", Source: "reverse_dns"},
		{Name: "__MSBROWSE__", Source: "netbios"},
	})
	if ok {
		t.Fatalf("expected ok=false, got name=%q", name)
	}
}

func TestFallbackDisplayName(t *testing.T) {
	tests := []struct {
		ip, id, want string
	}{
		{"10.0.1.5", "abcdef01-2345-6789-abcd-ef0123456789", "device-10.0.1.5"},
		{"", "abcdef01-2345-6789-abcd-ef0123456789", "device-abcdef01"},
		{"", "short", "device-short"},
		{"", "", "device-unknown"},
		{"  ", "  ", "device-unknown"},
	}
	for _, tt := range tests {
		got := FallbackDisplayName(tt.ip, tt.id)
		if got != tt.want {
			t.Errorf("FallbackDisplayName(%q, %q) = %q, want %q", tt.ip, tt.id, got, tt.want)
		}
	}
}
