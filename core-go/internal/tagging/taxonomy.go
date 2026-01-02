package tagging

import (
	"sort"
	"strings"
)

const (
	TagRouter      = "router"
	TagSwitch      = "switch"
	TagAccessPoint = "access_point"
	TagFirewall    = "firewall"
	TagPrinter     = "printer"
	TagServer      = "server"
	TagWorkstation = "workstation"
	TagNAS         = "nas"
	TagCamera      = "camera"
	TagVMHost      = "vm_host"
	TagIoT         = "iot"
)

var allTags = []string{
	TagRouter,
	TagSwitch,
	TagAccessPoint,
	TagFirewall,
	TagPrinter,
	TagServer,
	TagWorkstation,
	TagNAS,
	TagCamera,
	TagVMHost,
	TagIoT,
}

func AllTags() []string {
	out := make([]string, len(allTags))
	copy(out, allTags)
	return out
}

func IsValidTag(tag string) bool {
	tag = NormalizeTag(tag)
	for _, t := range allTags {
		if t == tag {
			return true
		}
	}
	return false
}

func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

func NormalizeTagList(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		t := NormalizeTag(raw)
		if t == "" {
			continue
		}
		if !IsValidTag(t) {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
