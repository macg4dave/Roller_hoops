package fingerprint

import "strings"

// LookupOUIVendor returns the manufacturer name for a MAC address using
// the first 3 octets (OUI). Returns "" if unknown.
// MAC can be in any common format: "aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF", "aabb.ccdd.eeff".
func LookupOUIVendor(mac string) string {
	prefix := normalizeOUI(mac)
	if prefix == "" {
		return ""
	}
	if v, ok := ouiTable[prefix]; ok {
		return v
	}
	return ""
}

func normalizeOUI(mac string) string {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if mac == "" {
		return ""
	}
	// Remove common separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	if len(mac) < 6 {
		return ""
	}
	return mac[:6]
}

// ouiTable maps OUI prefixes (6 uppercase hex chars) to vendor names.
// This is a curated subset of the IEEE OUI database covering the most common
// vendors seen in enterprise and home networks.
var ouiTable = map[string]string{
	// Apple
	"3C22FB": "Apple", "A4B197": "Apple", "F0D1A9": "Apple",
	"AC87A3": "Apple", "F4F15A": "Apple", "DC2B2A": "Apple",
	"28CF51": "Apple", "2C3361": "Apple", "60FACD": "Apple",
	"A860B6": "Apple", "B8E856": "Apple", "6C4008": "Apple",
	"A4D1D2": "Apple", "58B035": "Apple", "7CD1C3": "Apple",
	"F0B479": "Apple", "685B35": "Apple", "A8968A": "Apple",
	"D0817A": "Apple", "0C3021": "Apple", "8866A5": "Apple",
	// Cisco
	"000C29": "VMware", // often mis-labelled; this is VMware
	"0050C5": "Cisco", "F4CF21": "Cisco", "2C3124": "Cisco",
	"00259C": "Cisco", "D0D0FD": "Cisco", "B838DF": "Cisco",
	"F0EFAB": "Cisco", "FC5B39": "Cisco", "F87B20": "Cisco",
	"0026CB": "Cisco", "64F69D": "Cisco", "D46D50": "Cisco",
	// Juniper
	"0019E2": "Juniper", "002688": "Juniper", "F01C2D": "Juniper",
	"5C4527": "Juniper", "88E0F3": "Juniper", "44F477": "Juniper",
	// Aruba / HPE
	"D8C7C8": "Aruba", "204C03": "Aruba", "6C8BD5": "Aruba",
	"000F20": "Aruba", "20A6CD": "Aruba", "9C1C12": "Aruba",
	// Arista
	"001C73": "Arista", "28991A": "Arista", "444CA8": "Arista",
	// Dell
	"B083FE": "Dell", "D4BE41": "Dell", "F8BC12": "Dell",
	"246E96": "Dell", "5CF9DD": "Dell", "509A4C": "Dell",
	// HP
	"308D99": "HP", "3C4A92": "HP", "6C3BE5": "HP",
	"D0BF9C": "HP", "80CE62": "HP", "1CC1DE": "HP",
	// Lenovo
	"F0761C": "Lenovo", "7C7A91": "Lenovo", "E8F408": "Lenovo",
	// Microsoft
	"000D3A": "Microsoft", "7CB27D": "Microsoft", "28187B": "Microsoft",
	"C8D9D2": "Microsoft", "602233": "Microsoft", "0015B2": "Microsoft",
	// Intel
	"3C970E": "Intel", "A4BADB": "Intel", "48B02D": "Intel",
	"64006A": "Intel", "0021F4": "Intel", "0024D7": "Intel",
	"001B21": "Intel", "F8E43B": "Intel", "000E0C": "Intel",
	// Broadcom
	"001018": "Broadcom", "C4346B": "Broadcom",
	// Realtek
	"00E04C": "Realtek", "525400": "Realtek",
	// VMware
	"005056": "VMware", "000569": "VMware",
	// MikroTik
	"D4CA6D": "MikroTik", "E4D328": "MikroTik", "2CCFFC": "MikroTik",
	"6C3B6B": "MikroTik", "48A98A": "MikroTik", "CC2DE0": "MikroTik",
	// Ubiquiti
	"24A43C": "Ubiquiti", "802AA8": "Ubiquiti", "F09FC2": "Ubiquiti",
	"FC2203": "Ubiquiti", "B4FBE4": "Ubiquiti", "68D79A": "Ubiquiti",
	"18E829": "Ubiquiti", "2483A8": "Ubiquiti", "78452A": "Ubiquiti",
	// Fortinet
	"085B0E": "Fortinet", "00090F": "Fortinet", "70BE8B": "Fortinet",
	// Palo Alto
	"D4F4BE": "Palo Alto", "00862B": "Palo Alto", "0024AC": "Palo Alto",
	// Synology
	"001132": "Synology", "0011D8": "Synology",
	// QNAP
	"0008F8": "QNAP", "2476F5": "QNAP",
	// Raspberry Pi Foundation
	"B827EB": "Raspberry Pi", "D83ADD": "Raspberry Pi", "DC2632": "Raspberry Pi",
	"E45F01": "Raspberry Pi",
	// TP-Link
	"F09F09": "TP-Link", "1C3BF3": "TP-Link", "EC086B": "TP-Link",
	"5C628B": "TP-Link", "AC84C6": "TP-Link",
	// Netgear
	"A42B8C": "Netgear", "E0469A": "Netgear", "28C68E": "Netgear",
	"6038E0": "Netgear", "B03956": "Netgear",
	// Asus
	"FCAAE1": "ASUS", "043D98": "ASUS", "AC9E17": "ASUS",
	// Huawei
	"D0D04B": "Huawei", "88CEFA": "Huawei", "C8D15E": "Huawei",
	// Samsung
	"F4428F": "Samsung", "0C14D5": "Samsung", "8CB8A5": "Samsung",
	// Amazon
	"40B4CD": "Amazon", "FCA183": "Amazon", "687D6B": "Amazon",
	// Sonos
	"B8E937": "Sonos", "5CAAFD": "Sonos", "347E5C": "Sonos",
	// Google
	"3C5AB4": "Google", "A47733": "Google", "F4F5D8": "Google",
	// Supermicro
	"0025E7": "Supermicro", "0CC47A": "Supermicro",
}
