package snmp

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

type Neighbor struct {
	LocalIfIndex     *int
	RemoteDeviceName *string
	RemotePortName   *string
	RemoteChassisMAC *string
	RemoteMgmtIP     *string
	Source           string // "lldp" | "cdp"
}

const (
	oidLLDPRemChassisID = "1.0.8802.1.1.2.1.4.1.1.5"
	oidLLDPRemPortID    = "1.0.8802.1.1.2.1.4.1.1.7"
	oidLLDPRemPortDesc  = "1.0.8802.1.1.2.1.4.1.1.8"
	oidLLDPRemSysName   = "1.0.8802.1.1.2.1.4.1.1.9"

	oidCDPCacheAddress    = "1.3.6.1.4.1.9.9.23.1.2.1.1.4"
	oidCDPCacheDeviceID   = "1.3.6.1.4.1.9.9.23.1.2.1.1.6"
	oidCDPCacheDevicePort = "1.3.6.1.4.1.9.9.23.1.2.1.1.7"
)

func lastOIDInts(oid string, n int) ([]int, bool) {
	oid = strings.TrimSpace(oid)
	if oid == "" || n <= 0 {
		return nil, false
	}
	parts := strings.Split(oid, ".")
	if len(parts) < n {
		return nil, false
	}
	out := make([]int, 0, n)
	for i := len(parts) - n; i < len(parts); i++ {
		v, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}

func pduBytes(p gosnmp.SnmpPDU) ([]byte, bool) {
	switch v := p.Value.(type) {
	case []byte:
		return v, true
	case string:
		return []byte(v), true
	default:
		return nil, false
	}
}

func parseMgmtIPFromCdAddress(b []byte) (string, bool) {
	if len(b) == 4 {
		return net.IP(b).String(), true
	}
	if len(b) == 16 {
		return net.IP(b).String(), true
	}
	if len(b) >= 6 {
		// Some agents encode as: type(1) len(1) addr(n)
		// (1 = ipv4, 2 = ipv6) is commonly seen.
		t := b[0]
		l := int(b[1])
		if l <= 0 || len(b) < 2+l {
			return "", false
		}
		addr := b[2 : 2+l]
		if t == 1 && l == 4 {
			return net.IP(addr).String(), true
		}
		if t == 2 && l == 16 {
			return net.IP(addr).String(), true
		}
	}
	return "", false
}

func (c *Client) WalkLLDPNeighbors(ctx context.Context, target Target) ([]Neighbor, error) {
	if c == nil {
		return nil, errors.New("snmp client is nil")
	}
	_ = ctx

	s, err := c.connect(target)
	if err != nil {
		return nil, err
	}
	defer s.Conn.Close()

	type key struct {
		LocalPort int
		RemIndex  int
	}
	rows := map[key]*Neighbor{}

	ensure := func(k key) *Neighbor {
		if cur, ok := rows[k]; ok {
			return cur
		}
		n := &Neighbor{Source: "lldp"}
		rows[k] = n
		return n
	}

	walk := func(baseOID string, fn func(k key, p gosnmp.SnmpPDU)) error {
		pdus, err := s.BulkWalkAll(baseOID)
		if err != nil {
			return err
		}
		for _, p := range pdus {
			ints, ok := lastOIDInts(p.Name, 2)
			if !ok {
				continue
			}
			k := key{LocalPort: ints[0], RemIndex: ints[1]}
			fn(k, p)
		}
		return nil
	}

	_ = walk(oidLLDPRemSysName, func(k key, p gosnmp.SnmpPDU) {
		if s, ok := pduString(p); ok {
			n := ensure(k)
			n.RemoteDeviceName = s
			if n.LocalIfIndex == nil {
				i := k.LocalPort
				n.LocalIfIndex = &i
			}
		}
	})
	_ = walk(oidLLDPRemPortDesc, func(k key, p gosnmp.SnmpPDU) {
		if s, ok := pduString(p); ok {
			n := ensure(k)
			n.RemotePortName = s
			if n.LocalIfIndex == nil {
				i := k.LocalPort
				n.LocalIfIndex = &i
			}
		}
	})
	_ = walk(oidLLDPRemPortID, func(k key, p gosnmp.SnmpPDU) {
		if s, ok := pduString(p); ok {
			n := ensure(k)
			if n.RemotePortName == nil || strings.TrimSpace(*n.RemotePortName) == "" {
				n.RemotePortName = s
			}
			if n.LocalIfIndex == nil {
				i := k.LocalPort
				n.LocalIfIndex = &i
			}
		}
	})
	_ = walk(oidLLDPRemChassisID, func(k key, p gosnmp.SnmpPDU) {
		b, ok := pduBytes(p)
		if !ok || len(b) == 0 {
			return
		}
		n := ensure(k)
		if len(b) == 6 {
			m := strings.ToLower(net.HardwareAddr(b).String())
			if m != "" && m != "00:00:00:00:00:00" {
				n.RemoteChassisMAC = &m
			}
		}
		if n.LocalIfIndex == nil {
			i := k.LocalPort
			n.LocalIfIndex = &i
		}
	})

	var out []Neighbor
	for _, n := range rows {
		if n == nil {
			continue
		}
		if n.LocalIfIndex == nil && n.RemoteDeviceName == nil && n.RemoteChassisMAC == nil {
			continue
		}
		out = append(out, *n)
	}
	return out, nil
}

func (c *Client) WalkCDPNeighbors(ctx context.Context, target Target) ([]Neighbor, error) {
	if c == nil {
		return nil, errors.New("snmp client is nil")
	}
	_ = ctx

	s, err := c.connect(target)
	if err != nil {
		return nil, err
	}
	defer s.Conn.Close()

	type key struct {
		IfIndex   int
		DeviceIdx int
	}
	rows := map[key]*Neighbor{}

	ensure := func(k key) *Neighbor {
		if cur, ok := rows[k]; ok {
			return cur
		}
		i := k.IfIndex
		n := &Neighbor{Source: "cdp", LocalIfIndex: &i}
		rows[k] = n
		return n
	}

	walk := func(baseOID string, fn func(k key, p gosnmp.SnmpPDU)) error {
		pdus, err := s.BulkWalkAll(baseOID)
		if err != nil {
			return err
		}
		for _, p := range pdus {
			ints, ok := lastOIDInts(p.Name, 2)
			if !ok {
				continue
			}
			k := key{IfIndex: ints[0], DeviceIdx: ints[1]}
			fn(k, p)
		}
		return nil
	}

	_ = walk(oidCDPCacheDeviceID, func(k key, p gosnmp.SnmpPDU) {
		if s, ok := pduString(p); ok {
			ensure(k).RemoteDeviceName = s
		}
	})
	_ = walk(oidCDPCacheDevicePort, func(k key, p gosnmp.SnmpPDU) {
		if s, ok := pduString(p); ok {
			ensure(k).RemotePortName = s
		}
	})
	_ = walk(oidCDPCacheAddress, func(k key, p gosnmp.SnmpPDU) {
		b, ok := pduBytes(p)
		if !ok || len(b) == 0 {
			return
		}
		if ip, ok := parseMgmtIPFromCdAddress(b); ok {
			n := ensure(k)
			n.RemoteMgmtIP = &ip
		}
	})

	var out []Neighbor
	for _, n := range rows {
		if n == nil {
			continue
		}
		if n.RemoteDeviceName == nil && n.RemoteMgmtIP == nil {
			continue
		}
		out = append(out, *n)
	}
	return out, nil
}
