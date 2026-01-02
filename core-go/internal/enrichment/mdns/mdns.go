package mdns

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Candidate holds a friendly name discovered via mDNS/NetBIOS or similar discovery helpers.
type Candidate struct {
	DeviceID string
	Name     string
	Address  string
	Source   string // "mdns" | "netbios" | "reverse_dns" | "manual"
}

// Resolver captures the intent of resolving friendly names through multicast discovery.
type Resolver struct{}

// LookupAddr returns candidate names for a single address.
func (r *Resolver) LookupAddr(ctx context.Context, deviceID, address string) ([]Candidate, error) {
	var (
		errs = make([]error, 0, 3)
		seen = make(map[string]struct{})
		out  []Candidate
	)

	add := func(src, name string) {
		if name == "" {
			return
		}
		normalized := strings.ToLower(name)
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		out = append(out, Candidate{
			DeviceID: deviceID,
			Name:     name,
			Address:  address,
			Source:   src,
		})
	}

	if names, err := reverseDNSLookup(ctx, address); err == nil {
		for _, name := range names {
			add("reverse_dns", name)
		}
	} else {
		errs = append(errs, err)
	}

	if names, err := lookupMDNS(ctx, address); err == nil {
		for _, name := range names {
			add("mdns", name)
		}
	} else {
		errs = append(errs, err)
	}

	if names, err := lookupNetBIOS(ctx, address); err == nil {
		for _, name := range names {
			add("netbios", name)
		}
	} else {
		errs = append(errs, err)
	}

	if len(out) == 0 {
		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}
		return nil, nil
	}
	return out, nil
}

func reverseDNSLookup(ctx context.Context, address string) ([]string, error) {
	names, err := net.DefaultResolver.LookupAddr(ctx, address)
	if err != nil {
		return nil, err
	}
	for i := range names {
		names[i] = strings.TrimSuffix(strings.TrimSpace(names[i]), ".")
	}
	return names, nil
}

func lookupMDNS(ctx context.Context, address string) ([]string, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, fmt.Errorf("mdns: invalid ip %q", address)
	}

	question, err := dns.ReverseAddr(address)
	if err != nil {
		return nil, fmt.Errorf("mdns: reverse: %w", err)
	}

	msg := &dns.Msg{}
	msg.SetQuestion(question, dns.TypePTR)
	msg.RecursionDesired = false

	client := &dns.Client{
		Net:     "udp",
		Timeout: 400 * time.Millisecond,
	}

	server := "224.0.0.251:5353"
	if ip.To4() == nil {
		server = "[ff02::fb]:5353"
	}

	resp, _, err := client.ExchangeContext(ctx, msg, server)
	if err != nil {
		return nil, fmt.Errorf("mdns: exchange: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("mdns: empty response")
	}

	var names []string
	for _, answer := range resp.Answer {
		if ptr, ok := answer.(*dns.PTR); ok {
			candidate := strings.TrimSuffix(strings.TrimSpace(ptr.Ptr), ".")
			if candidate != "" {
				names = append(names, candidate)
			}
		}
		if cname, ok := answer.(*dns.CNAME); ok {
			candidate := strings.TrimSuffix(strings.TrimSpace(cname.Target), ".")
			if candidate != "" {
				names = append(names, candidate)
			}
		}
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("mdns: no PTR/CNAME records for %s", address)
	}
	return names, nil
}

func lookupNetBIOS(ctx context.Context, address string) ([]string, error) {
	if net.ParseIP(address) == nil {
		return nil, fmt.Errorf("netbios: invalid ip %q", address)
	}

	req, txID, err := buildNetBIOSNodeStatusRequest()
	if err != nil {
		return nil, fmt.Errorf("netbios: build request: %w", err)
	}

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", net.JoinHostPort(address, "137"))
	if err != nil {
		return nil, fmt.Errorf("netbios: dial: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	}

	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("netbios: write: %w", err)
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("netbios: read: %w", err)
	}

	names, err := parseNetBIOSNodeStatusResponse(buf[:n], txID)
	if err != nil {
		return nil, fmt.Errorf("netbios: parse: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("netbios: no names in response")
	}
	return names, nil
}

func buildNetBIOSNodeStatusRequest() ([]byte, uint16, error) {
	txID, err := randomUint16()
	if err != nil {
		return nil, 0, err
	}

	req := make([]byte, 50)
	binary.BigEndian.PutUint16(req[0:], txID)
	binary.BigEndian.PutUint16(req[4:], 1)

	req[12] = 0x20
	copy(req[13:], encodeNetBIOSName("*"))
	req[45] = 0
	binary.BigEndian.PutUint16(req[46:], 0x0021)
	binary.BigEndian.PutUint16(req[48:], 0x0001)

	return req, txID, nil
}

func encodeNetBIOSName(name string) []byte {
	padded := strings.ToUpper(name)
	if len(padded) > 15 {
		padded = padded[:15]
	}
	padded += strings.Repeat(" ", 16-len(padded))

	encoded := make([]byte, 32)
	for i := 0; i < 16; i++ {
		ch := padded[i]
		high := (ch >> 4) & 0x0F
		low := ch & 0x0F
		encoded[2*i] = 'A' + high
		encoded[2*i+1] = 'A' + low
	}
	return encoded
}

func parseNetBIOSNodeStatusResponse(resp []byte, expectedTxID uint16) ([]string, error) {
	if len(resp) < 12 {
		return nil, fmt.Errorf("netbios: response too short")
	}
	if binary.BigEndian.Uint16(resp[0:2]) != expectedTxID {
		return nil, fmt.Errorf("netbios: transaction mismatch")
	}
	answerCount := int(binary.BigEndian.Uint16(resp[6:8]))
	if answerCount == 0 {
		return nil, fmt.Errorf("netbios: no answers")
	}

	offset := 12
	var err error
	offset, err = skipDomainName(resp, offset)
	if err != nil {
		return nil, err
	}
	offset += 4

	offset, err = skipDomainName(resp, offset)
	if err != nil {
		return nil, err
	}

	if offset+10 > len(resp) {
		return nil, fmt.Errorf("netbios: answer header truncated")
	}

	rtype := binary.BigEndian.Uint16(resp[offset:])
	offset += 2
	offset += 2
	offset += 4
	rdLen := int(binary.BigEndian.Uint16(resp[offset:]))
	offset += 2

	if offset+rdLen > len(resp) {
		return nil, fmt.Errorf("netbios: truncated data")
	}

	if rtype != 0x0021 {
		return nil, fmt.Errorf("netbios: expected NBSTAT, got %d", rtype)
	}

	data := resp[offset : offset+rdLen]
	if len(data) < 1 {
		return nil, fmt.Errorf("netbios: empty nbstat payload")
	}
	numNames := int(data[0])
	payload := data[1:]
	if 1+numNames*18+6 > len(data) {
		return nil, fmt.Errorf("netbios: nbstat payload too short")
	}

	names := make([]string, 0, numNames)
	preferred := make([]string, 0, numNames)
	fallback := make([]string, 0, numNames)
	seen := make(map[string]struct{}, numNames)
	pos := 0
	for i := 0; i < numNames; i++ {
		if pos+18 > len(payload) {
			break
		}
		raw := strings.TrimSpace(string(payload[pos : pos+15]))
		suffix := payload[pos+15]
		flags := binary.BigEndian.Uint16(payload[pos+16 : pos+18])
		pos += 18

		if raw == "" {
			continue
		}

		normalized := strings.ToLower(raw)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}

		// Group names are often things like WORKGROUP or BROWSE lists, not device identities.
		isGroup := flags&0x8000 != 0
		if isGroup {
			continue
		}

		switch suffix {
		case 0x20:
			preferred = append(preferred, raw)
		case 0x00:
			fallback = append(fallback, raw)
		default:
			// Ignore other suffixes (often services/groups).
		}
	}

	names = append(names, preferred...)
	names = append(names, fallback...)
	return names, nil
}

func skipDomainName(msg []byte, offset int) (int, error) {
	for offset < len(msg) {
		length := int(msg[offset])
		offset++
		if length == 0 {
			return offset, nil
		}
		if length&0xC0 == 0xC0 {
			if offset >= len(msg) {
				return 0, fmt.Errorf("netbios: truncated pointer")
			}
			offset++
			return offset, nil
		}
		offset += length
		if offset > len(msg) {
			return 0, fmt.Errorf("netbios: name overflow")
		}
	}
	return 0, fmt.Errorf("netbios: name overflow")
}

func randomUint16() (uint16, error) {
	var buf [2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf[:]), nil
}
