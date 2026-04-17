package fingerprint

import "testing"

func TestLookupOUIVendor(t *testing.T) {
	tests := []struct {
		mac    string
		vendor string
	}{
		{"3C:22:FB:AA:BB:CC", "Apple"},
		{"3c:22:fb:aa:bb:cc", "Apple"},
		{"3c22.fbaa.bbcc", "Apple"},
		{"00-50-56-12-34-56", "VMware"},
		{"D4:CA:6D:11:22:33", "MikroTik"},
		{"B8:27:EB:00:00:01", "Raspberry Pi"},
		{"FF:FF:FF:FF:FF:FF", ""},
		{"", ""},
		{"00:00:00", ""},
	}
	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			got := LookupOUIVendor(tt.mac)
			if got != tt.vendor {
				t.Errorf("LookupOUIVendor(%q) = %q, want %q", tt.mac, got, tt.vendor)
			}
		})
	}
}

func TestGuessOS(t *testing.T) {
	tests := []struct {
		name       string
		sysDescr   string
		ports      []int
		mac        string
		wantOS     string
		wantConf   string
		wantVendor string
	}{
		{
			name: "sysDescr wins",
			sysDescr: "IOS-XE",
			ports: []int{22, 443},
			mac: "3C:22:FB:AA:BB:CC",
			wantOS: "IOS-XE",
			wantConf: "high",
			wantVendor: "Apple",
		},
		{
			name: "ports windows",
			sysDescr: "",
			ports: []int{135, 445, 3389},
			mac: "",
			wantOS: "Windows",
			wantConf: "medium",
		},
		{
			name: "ports linux ssh only",
			sysDescr: "",
			ports: []int{22},
			mac: "",
			wantOS: "Linux",
			wantConf: "medium",
		},
		{
			name: "ports printer",
			sysDescr: "",
			ports: []int{9100, 515, 80},
			mac: "",
			wantOS: "Printer",
			wantConf: "medium",
		},
		{
			name: "vendor fallback apple",
			sysDescr: "",
			ports: nil,
			mac: "3C:22:FB:AA:BB:CC",
			wantOS: "macOS/iOS",
			wantConf: "low",
			wantVendor: "Apple",
		},
		{
			name: "vendor fallback raspberry pi",
			sysDescr: "",
			ports: nil,
			mac: "B8:27:EB:00:00:01",
			wantOS: "Linux",
			wantConf: "low",
			wantVendor: "Raspberry Pi",
		},
		{
			name: "no signal",
			sysDescr: "",
			ports: nil,
			mac: "",
			wantOS: "",
			wantConf: "low",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := GuessOS(tt.sysDescr, tt.ports, tt.mac)
			if fp.OSGuess != tt.wantOS {
				t.Errorf("OSGuess = %q, want %q", fp.OSGuess, tt.wantOS)
			}
			if fp.Confidence != tt.wantConf {
				t.Errorf("Confidence = %q, want %q", fp.Confidence, tt.wantConf)
			}
			if tt.wantVendor != "" && fp.MACVendor != tt.wantVendor {
				t.Errorf("MACVendor = %q, want %q", fp.MACVendor, tt.wantVendor)
			}
		})
	}
}
