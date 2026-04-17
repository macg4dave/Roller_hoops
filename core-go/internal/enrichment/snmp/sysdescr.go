package snmp

import (
	"regexp"
	"strings"
)

// ParsedSysDescr holds structured fields extracted from a sysDescr string.
type ParsedSysDescr struct {
	OSFamily  string // e.g. "Linux", "IOS", "IOS-XE", "NX-OS", "JunOS", "Windows", "ArubaOS"
	OSVersion string // best-effort version string
}

var patterns = []struct {
	family string
	re     *regexp.Regexp
}{
	// Cisco families — order matters: specific first
	{"IOS-XE", regexp.MustCompile(`(?i)IOS-XE Software.*Version\s+(\S+)`)},
	{"IOS-XR", regexp.MustCompile(`(?i)IOS XR Software.*Version\s+(\S+)`)},
	{"NX-OS", regexp.MustCompile(`(?i)NX-OS.*version\s+(\S+)`)},
	{"IOS", regexp.MustCompile(`(?i)Cisco IOS Software.*Version\s+(\S+)`)},
	{"IOS", regexp.MustCompile(`(?i)Cisco Internetwork Operating System.*Version\s+(\S+)`)},
	// Juniper
	{"JunOS", regexp.MustCompile(`(?i)Juniper.*JUNOS\s+(\S+)`)},
	// Aruba / HP networking
	{"ArubaOS", regexp.MustCompile(`(?i)ArubaOS[- ]*(?:\(MODEL[^)]*\))?,?\s*Version\s+(\S+)`)},
	{"ArubaOS-CX", regexp.MustCompile(`(?i)ArubaOS-CX\s+(\S+)`)},
	{"ProCurve", regexp.MustCompile(`(?i)ProCurve.*revision\s+(\S+)`)},
	{"Comware", regexp.MustCompile(`(?i)Comware.*Version\s+(\S+)`)},
	// Fortinet
	{"FortiOS", regexp.MustCompile(`(?i)FortiGate.*v(\S+)`)},
	{"FortiOS", regexp.MustCompile(`(?i)Forti\S+\s+v(\S+)`)},
	// Palo Alto
	{"PanOS", regexp.MustCompile(`(?i)Palo Alto.*PAN-OS\s+(\S+)`)},
	// MikroTik
	{"RouterOS", regexp.MustCompile(`(?i)RouterOS\s+(\S+)`)},
	{"RouterOS", regexp.MustCompile(`(?i)MikroTik.*(\d+\.\d+\S*)`)},
	// Ubiquiti
	{"EdgeOS", regexp.MustCompile(`(?i)EdgeOS\s+v?(\S+)`)},
	{"UniFi", regexp.MustCompile(`(?i)UniFi.*?(\d+\.\d+\.\d+\S*)`)},
	// Dell / Force10
	{"FTOS", regexp.MustCompile(`(?i)Force10.*FTOS\s+\S*\s*(\S+)`)},
	{"Dell Networking", regexp.MustCompile(`(?i)Dell.*Networking.*OS.*(\d+\.\d+\S*)`)},
	// VMware
	{"ESXi", regexp.MustCompile(`(?i)VMware ESXi?\s+(\S+)`)},
	// pfSense / OPNsense
	{"pfSense", regexp.MustCompile(`(?i)pfSense.*?(\d+\.\d+\S*)`)},
	{"OPNsense", regexp.MustCompile(`(?i)OPNsense.*?(\d+\.\d+\S*)`)},
	// Synology / QNAP
	{"DSM", regexp.MustCompile(`(?i)Synology.*DSM\s+(\S+)`)},
	{"Synology", regexp.MustCompile(`(?i)Synology\s+(\S+)`)},
	{"QTS", regexp.MustCompile(`(?i)QNAP.*QTS\s+(\S+)`)},
	// Generic OS — keep last
	{"Linux", regexp.MustCompile(`(?i)Linux\s+\S+\s+(\S+)`)},
	{"Windows", regexp.MustCompile(`(?i)Hardware:.*Windows\s+Version\s+(\S+)`)},
	{"Windows", regexp.MustCompile(`(?i)Windows.*Build\s+(\S+)`)},
	{"FreeBSD", regexp.MustCompile(`(?i)FreeBSD\s+(\S+)`)},
	{"NetBSD", regexp.MustCompile(`(?i)NetBSD\s+(\S+)`)},
	{"OpenBSD", regexp.MustCompile(`(?i)OpenBSD\s+(\S+)`)},
	{"SunOS", regexp.MustCompile(`(?i)SunOS\s+\S+\s+(\S+)`)},
	{"AIX", regexp.MustCompile(`(?i)IBM.*AIX.*(\d+\.\d+)`)},
}

// ParseSysDescr extracts OS family and version from a sysDescr string.
// Returns a zero value if nothing could be parsed.
func ParseSysDescr(raw string) ParsedSysDescr {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedSysDescr{}
	}
	for _, p := range patterns {
		m := p.re.FindStringSubmatch(raw)
		if m != nil {
			ver := ""
			if len(m) > 1 {
				ver = strings.TrimRight(m[1], ",;")
			}
			return ParsedSysDescr{OSFamily: p.family, OSVersion: ver}
		}
	}
	return ParsedSysDescr{}
}
