package naming

import (
	"sort"
	"strings"
)

type Candidate struct {
	Name   string
	Source string
}

type normalizedCandidate struct {
	Source      string
	StoredName  string
	DisplayName string
	Score       int
}

func NormalizeCandidate(source, rawName string) (storedName string, displayName string, score int, ok bool) {
	source = strings.ToLower(strings.TrimSpace(source))
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", "", 0, false
	}
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return "", "", 0, false
	}

	stored := name
	switch source {
	case "reverse_dns", "mdns":
		stored = strings.ToLower(stored)
	}

	display := stored
	if strings.Contains(display, ".") && !strings.ContainsAny(display, " \t") {
		parts := strings.SplitN(display, ".", 2)
		if len(parts) > 0 && parts[0] != "" {
			display = parts[0]
		}
	}

	s := scoreCandidate(source, stored, display)
	if s < 0 {
		return stored, display, s, false
	}

	return stored, display, s, true
}

func ChooseBestDisplayName(candidates []Candidate) (string, bool) {
	best := normalizedCandidate{Score: -1_000_000}

	for _, c := range candidates {
		stored, display, score, ok := NormalizeCandidate(c.Source, c.Name)
		if !ok {
			continue
		}
		// Require a minimum quality bar before auto-setting a display name.
		if score < 70 {
			continue
		}
		next := normalizedCandidate{
			Source:      c.Source,
			StoredName:  stored,
			DisplayName: display,
			Score:       score,
		}
		if betterCandidate(next, best) {
			best = next
		}
	}

	if best.Score < 70 || strings.TrimSpace(best.DisplayName) == "" {
		return "", false
	}
	return best.DisplayName, true
}

func SortCandidatesForDisplay(candidates []Candidate) []Candidate {
	type scored struct {
		orig       Candidate
		normalized normalizedCandidate
		ok         bool
	}

	scoredList := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		stored, display, score, ok := NormalizeCandidate(c.Source, c.Name)
		scoredList = append(scoredList, scored{
			orig: c,
			normalized: normalizedCandidate{
				Source:      c.Source,
				StoredName:  stored,
				DisplayName: display,
				Score:       score,
			},
			ok: ok,
		})
	}

	sort.SliceStable(scoredList, func(i, j int) bool {
		ai := scoredList[i]
		aj := scoredList[j]
		if ai.ok != aj.ok {
			return ai.ok
		}
		if ai.normalized.Score != aj.normalized.Score {
			return ai.normalized.Score > aj.normalized.Score
		}
		if ai.normalized.DisplayName != aj.normalized.DisplayName {
			return ai.normalized.DisplayName < aj.normalized.DisplayName
		}
		return ai.normalized.StoredName < aj.normalized.StoredName
	})

	out := make([]Candidate, 0, len(scoredList))
	for _, item := range scoredList {
		out = append(out, item.orig)
	}
	return out
}

func betterCandidate(a, b normalizedCandidate) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	// Prefer shorter display names after scoring (tends to avoid noisy FQDNs when equal).
	if len(a.DisplayName) != len(b.DisplayName) {
		return len(a.DisplayName) < len(b.DisplayName)
	}
	// Stable tie-breaker.
	if a.DisplayName != b.DisplayName {
		return a.DisplayName < b.DisplayName
	}
	return a.StoredName < b.StoredName
}

func scoreCandidate(source, stored, display string) int {
	normalized := strings.ToLower(stored)
	if looksGarbage(normalized) {
		return -1
	}

	base := 50
	switch source {
	case "dhcp":
		base = 95
	case "reverse_dns":
		base = 90
	case "snmp":
		base = 88
	case "lldp", "cdp":
		base = 86
	case "mdns":
		base = 80
	case "netbios":
		base = 78
	case "manual":
		base = 70
	}

	// Penalize very short labels.
	if len(display) < 2 {
		base -= 50
	}

	// Prefer hostname-like values without spaces for auto display names.
	if strings.ContainsAny(display, " \t") {
		base -= 25
	}

	// Prefer something that looks like a hostname label.
	if !looksHostnameLabel(display) {
		base -= 20
	}

	if strings.HasSuffix(normalized, ".local") || strings.HasSuffix(normalized, ".localdomain") {
		base -= 5
	}

	return base
}

func looksHostnameLabel(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func looksGarbage(normalized string) bool {
	if normalized == "" {
		return true
	}
	if strings.Contains(normalized, "in-addr.arpa") || strings.Contains(normalized, "ip6.arpa") {
		return true
	}
	switch normalized {
	case "workgroup", "mshome", "__msbrowse__", "localdomain", "localhost":
		return true
	}
	return false
}
