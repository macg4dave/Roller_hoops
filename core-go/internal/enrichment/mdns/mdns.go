package mdns

import (
	"context"
	"net"
	"strings"
)

// Candidate holds a friendly name discovered via mDNS/NetBIOS or similar discovery helpers.
type Candidate struct {
	DeviceID string
	Name     string
	Address  string
	Source   string // "mdns" | "netbios" | "reverse_dns" | "manual"
}

// Resolver captures the intent of resolving friendly names through multicast discovery.
type Resolver struct {
}

// LookupAddr returns candidate names for a single address.
//
// NOTE: This is currently implemented as best-effort reverse DNS lookup (PTR). On many LANs that
// effectively includes mDNS-style names when the resolver stack is configured accordingly.
func (r *Resolver) LookupAddr(ctx context.Context, deviceID, address string) ([]Candidate, error) {
	names, err := net.DefaultResolver.LookupAddr(ctx, address)
	if err != nil {
		return nil, err
	}

	out := make([]Candidate, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(strings.TrimSuffix(raw, "."))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Candidate{
			DeviceID: deviceID,
			Name:     name,
			Address:  address,
			Source:   "reverse_dns",
		})
	}
	return out, nil
}
