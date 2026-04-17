package fingerprint

// DeviceFingerprint holds the best-guess identity signals for a device.
type DeviceFingerprint struct {
	OSGuess   string // best-effort OS guess, e.g. "Linux", "Windows", "Cisco IOS"
	MACVendor string // manufacturer from first 3 bytes of MAC (OUI)
	Confidence string // "high", "medium", "low"
}

// GuessOS produces a best-effort OS classification by combining SNMP sysDescr
// parsing, open port heuristics, and MAC OUI vendor signals.
//
// Priority order (first non-empty wins for OS):
//  1. sysDescr OS family (most reliable when SNMP is available)
//  2. Port-based heuristic (passive, always available if scan ran)
//  3. MAC OUI vendor hint (weakest signal, but always available if MAC known)
func GuessOS(sysDescrFamily string, openPorts []int, macAddr string) DeviceFingerprint {
	fp := DeviceFingerprint{}

	// MAC vendor lookup (always try)
	fp.MACVendor = LookupOUIVendor(macAddr)

	// 1. sysDescr is the strongest signal
	if sysDescrFamily != "" {
		fp.OSGuess = sysDescrFamily
		fp.Confidence = "high"
		return fp
	}

	// 2. Port-based heuristic
	if os := guessOSFromPorts(openPorts); os != "" {
		fp.OSGuess = os
		fp.Confidence = "medium"
		return fp
	}

	// 3. MAC vendor as last resort
	if os := guessOSFromVendor(fp.MACVendor); os != "" {
		fp.OSGuess = os
		fp.Confidence = "low"
		return fp
	}

	fp.Confidence = "low"
	return fp
}

func guessOSFromPorts(ports []int) string {
	set := make(map[int]bool, len(ports))
	for _, p := range ports {
		set[p] = true
	}

	// Strong Windows signals
	if set[3389] {
		return "Windows"
	}
	if set[135] && set[445] {
		return "Windows"
	}

	// Apple
	if set[5009] && set[3689] {
		return "macOS"
	}
	if set[62078] { // iOS lockdown
		return "iOS/iPadOS"
	}

	// Network device signals
	if set[179] { // BGP
		return "Network Device"
	}

	// Printer
	if set[9100] && set[515] {
		return "Printer"
	}
	if set[9100] {
		return "Printer"
	}

	// SSH-only is a weak Linux hint but common
	if set[22] && !set[445] && !set[3389] && !set[80] {
		return "Linux"
	}

	return ""
}

func guessOSFromVendor(vendor string) string {
	if vendor == "" {
		return ""
	}
	switch vendor {
	case "Apple":
		return "macOS/iOS"
	case "Microsoft":
		return "Windows"
	case "VMware":
		return "ESXi"
	case "Cisco", "Juniper", "Aruba", "Arista", "MikroTik", "Ubiquiti":
		return "Network Device"
	case "Synology", "QNAP":
		return "NAS"
	case "Raspberry Pi":
		return "Linux"
	default:
		return ""
	}
}
