package snmp

import "testing"

func TestParseSysDescr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		family  string
		version string
	}{
		{"empty", "", "", ""},
		{"cisco ios", "Cisco IOS Software, C2960 Software Version 15.0(2)SE", "IOS", "15.0(2)SE"},
		{"ios-xe", "Cisco IOS-XE Software, Version 17.03.04a", "IOS-XE", "17.03.04a"},
		{"nx-os", "Cisco NX-OS(tm) n5000, Software (n5000-uk9), version 7.3(5)N1(1)", "NX-OS", "7.3(5)N1(1)"},
		{"junos", "Juniper Networks, Inc. srx240h2, JUNOS 15.1X49-D150.2", "JunOS", "15.1X49-D150.2"},
		{"linux", "Linux server1 5.15.0-91-generic #101-Ubuntu SMP", "Linux", "5.15.0-91-generic"},
		{"windows", "Hardware: Intel64 Windows Version 6.3 (Build 9600)", "Windows", "6.3"},
		{"unknown", "Some random device firmware v1.0", "", ""},
		{"fortios", "FortiGate-60E v7.2.4,build1396,230131", "FortiOS", "7.2.4,build1396,230131"},
		{"mikrotik", "RouterOS 7.13.2", "RouterOS", "7.13.2"},
		{"mikrotik long", "MikroTik RouterBOARD 3011 7.12", "RouterOS", "7.12"},
		{"ubiquiti edgeos", "EdgeOS v2.0.9-hotfix.4", "EdgeOS", "2.0.9-hotfix.4"},
		{"esxi", "VMware ESXi 7.0.3 build-20328353", "ESXi", "7.0.3"},
		{"pfsense", "pfSense 2.7.2-RELEASE", "pfSense", "2.7.2-RELEASE"},
		{"opnsense", "OPNsense 24.1", "OPNsense", "24.1"},
		{"synology dsm", "Synology NAS DSM 7.2-64570", "DSM", "7.2-64570"},
		{"procurve", "ProCurve J9729A 2920-48G-POE+ Switch revision WB.16.10.0023", "ProCurve", "WB.16.10.0023"},
		{"freebsd", "FreeBSD 14.0-RELEASE-p6 amd64", "FreeBSD", "14.0-RELEASE-p6"},
		{"openbsd", "OpenBSD 7.4 GENERIC#1234 amd64", "OpenBSD", "7.4"},
		{"sunos", "SunOS myhost 5.11 11.4.0.15.0", "SunOS", "5.11"},
		{"ios old", "Cisco Internetwork Operating System Software IOS Version 12.2(55)SE", "IOS", "12.2(55)SE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSysDescr(tt.input)
			if got.OSFamily != tt.family {
				t.Errorf("OSFamily = %q, want %q", got.OSFamily, tt.family)
			}
			if got.OSVersion != tt.version {
				t.Errorf("OSVersion = %q, want %q", got.OSVersion, tt.version)
			}
		})
	}
}
