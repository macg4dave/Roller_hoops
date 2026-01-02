package tagging

import (
	"sort"
	"strings"
)

type Suggestion struct {
	Tag        string
	Confidence int
	Evidence   map[string]any
}

func MergeSuggestions(groups ...[]Suggestion) []Suggestion {
	byTag := make(map[string]Suggestion)

	for _, group := range groups {
		for _, s := range group {
			tag := NormalizeTag(s.Tag)
			if !IsValidTag(tag) {
				continue
			}
			if s.Confidence <= 0 {
				continue
			}

			existing, ok := byTag[tag]
			if !ok || s.Confidence > existing.Confidence {
				s.Tag = tag
				byTag[tag] = s
				continue
			}
			if ok && s.Evidence != nil {
				if existing.Evidence == nil {
					existing.Evidence = map[string]any{}
				}
				for k, v := range s.Evidence {
					existing.Evidence[k] = v
				}
				byTag[tag] = existing
			}
		}
	}

	out := make([]Suggestion, 0, len(byTag))
	for _, v := range byTag {
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

func SuggestFromNames(names []string) []Suggestion {
	var out []Suggestion
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}

		tokens := tokenize(name)
		matches := func(set ...string) bool {
			for _, t := range tokens {
				for _, candidate := range set {
					if t == candidate {
						return true
					}
				}
			}
			return false
		}

		add := func(tag string, match string) {
			out = append(out, Suggestion{
				Tag:        tag,
				Confidence: 70,
				Evidence: map[string]any{
					"signal": "name",
					"name":   raw,
					"match":  match,
				},
			})
		}

		switch {
		case matches("ap", "wap", "unifi", "eap", "wlan", "wireless"):
			add(TagAccessPoint, "ap")
		case matches("sw", "switch"):
			add(TagSwitch, "switch")
		case matches("gw", "router", "gateway", "edge"):
			add(TagRouter, "router")
		case matches("fw", "firewall", "pfsense", "opnsense", "fortigate", "fortinet", "paloalto", "panos", "asa"):
			add(TagFirewall, "firewall")
		case matches("printer", "hp", "brother", "epson", "canon"):
			add(TagPrinter, "printer")
		case matches("nas", "synology", "qnap", "truenas", "freenas"):
			add(TagNAS, "nas")
		case matches("esxi", "vmware", "proxmox", "pve", "hyperv", "xen"):
			add(TagVMHost, "vm_host")
		case matches("cam", "camera", "nvr", "dvr"):
			add(TagCamera, "camera")
		case matches("iot"):
			add(TagIoT, "iot")
		}
	}
	return out
}

func SuggestFromSNMP(sysDescr string) []Suggestion {
	descr := strings.ToLower(strings.TrimSpace(sysDescr))
	if descr == "" {
		return nil
	}

	add := func(tag string, match string, confidence int) Suggestion {
		return Suggestion{
			Tag:        tag,
			Confidence: confidence,
			Evidence: map[string]any{
				"signal":    "snmp",
				"match":     match,
				"sys_descr": truncate(sysDescr, 240),
			},
		}
	}

	var out []Suggestion
	switch {
	case strings.Contains(descr, "access point") || strings.Contains(descr, "wireless"):
		out = append(out, add(TagAccessPoint, "access_point", 90))
	case strings.Contains(descr, "switch"):
		out = append(out, add(TagSwitch, "switch", 90))
	case strings.Contains(descr, "router") || strings.Contains(descr, "routing"):
		out = append(out, add(TagRouter, "router", 88))
	}

	if strings.Contains(descr, "firewall") || strings.Contains(descr, "pfsense") || strings.Contains(descr, "opnsense") ||
		strings.Contains(descr, "fortigate") || strings.Contains(descr, "pan-os") || strings.Contains(descr, "palo alto") {
		out = append(out, add(TagFirewall, "firewall", 90))
	}

	if strings.Contains(descr, "vmware esxi") || strings.Contains(descr, "proxmox") || strings.Contains(descr, "hyper-v") {
		out = append(out, add(TagVMHost, "vm_host", 88))
	}

	if strings.Contains(descr, "synology") || strings.Contains(descr, "qnap") || strings.Contains(descr, "truenas") || strings.Contains(descr, "freenas") {
		out = append(out, add(TagNAS, "nas", 86))
	}

	if strings.Contains(descr, "printer") {
		out = append(out, add(TagPrinter, "printer", 82))
	}

	return out
}

func SuggestFromOpenPorts(openPorts []int32) []Suggestion {
	if len(openPorts) == 0 {
		return nil
	}
	set := make(map[int32]struct{}, len(openPorts))
	for _, p := range openPorts {
		set[p] = struct{}{}
	}

	has := func(p int32) bool {
		_, ok := set[p]
		return ok
	}

	evidencePorts := func(ports ...int32) map[string]any {
		out := make([]int32, 0, len(ports))
		for _, p := range ports {
			if has(p) {
				out = append(out, p)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		return map[string]any{"signal": "ports", "ports": out}
	}

	var out []Suggestion
	if has(9100) || has(515) || has(631) {
		out = append(out, Suggestion{Tag: TagPrinter, Confidence: 85, Evidence: evidencePorts(9100, 515, 631)})
	}
	if has(554) || has(8554) {
		out = append(out, Suggestion{Tag: TagCamera, Confidence: 82, Evidence: evidencePorts(554, 8554)})
	}
	if has(53) && (has(67) || has(68)) {
		out = append(out, Suggestion{Tag: TagRouter, Confidence: 80, Evidence: evidencePorts(53, 67, 68)})
	}
	if has(2049) || has(3260) {
		out = append(out, Suggestion{Tag: TagNAS, Confidence: 78, Evidence: evidencePorts(2049, 3260)})
	}
	return out
}

func tokenize(value string) []string {
	var out []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, buf.String())
		buf.Reset()
	}

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			buf.WriteRune(r)
		case r >= '0' && r <= '9':
			buf.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return out
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:1]
	}
	return value[:limit-1] + "…"
}

