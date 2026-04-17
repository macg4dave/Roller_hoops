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
	{"IOS-XE", regexp.MustCompile(`(?i)IOS-XE Software.*Version\s+(\S+)`)},
	{"IOS-XR", regexp.MustCompile(`(?i)IOS XR Software.*Version\s+(\S+)`)},
	{"NX-OS", regexp.MustCompile(`(?i)NX-OS.*version\s+(\S+)`)},
	{"IOS", regexp.MustCompile(`(?i)Cisco IOS Software.*Version\s+(\S+)`)},
	{"JunOS", regexp.MustCompile(`(?i)Juniper.*JUNOS\s+(\S+)`)},
	{"ArubaOS", regexp.MustCompile(`(?i)ArubaOS[- ]*(?:\(MODEL[^)]*\))?,?\s*Version\s+(\S+)`)},
	{"FortiOS", regexp.MustCompile(`(?i)FortiGate.*v(\S+)`)},
	{"PanOS", regexp.MustCompile(`(?i)Palo Alto.*PAN-OS\s+(\S+)`)},
	{"Linux", regexp.MustCompile(`(?i)Linux\s+\S+\s+(\S+)`)},
	{"Windows", regexp.MustCompile(`(?i)Windows.*Version\s+(\S+)`)},
	{"FreeBSD", regexp.MustCompile(`(?i)FreeBSD\s+(\S+)`)},
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
