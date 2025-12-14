package snmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Config describes the SNMP enrichment pipeline intentions.
type Config struct {
	Community      string
	Version        string // "2c" (default) | "1" | "3" (unsupported here)
	Port           uint16
	Timeout        time.Duration
	Retries        int
	MaxRepetitions uint32
}

// Target represents a device that can be queried via SNMP.
type Target struct {
	ID      string // optional device ID to decorate
	Address string
}

type SystemInfo struct {
	SysName     *string
	SysDescr    *string
	SysObjectID *string
	SysContact  *string
	SysLocation *string
}

type InterfaceInfo struct {
	IfIndex     int
	Name        *string
	Descr       *string
	Alias       *string
	MAC         *string
	AdminStatus *int32
	OperStatus  *int32
	MTU         *int32
	SpeedBps    *int64
}

// Client wraps a minimal SNMPv2c implementation for enrichment.
type Client struct {
	cfg Config
}

// NewClient constructs the SNMP enrichment client stub.
func NewClient(cfg Config) *Client {
	if strings.TrimSpace(cfg.Community) == "" {
		cfg.Community = "public"
	}
	if strings.TrimSpace(cfg.Version) == "" {
		cfg.Version = "2c"
	}
	if cfg.Port == 0 {
		cfg.Port = 161
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 900 * time.Millisecond
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}
	if cfg.MaxRepetitions == 0 {
		cfg.MaxRepetitions = 10
	}
	return &Client{cfg: cfg}
}

func (c *Client) connect(target Target) (*gosnmp.GoSNMP, error) {
	version := strings.ToLower(strings.TrimSpace(c.cfg.Version))
	var snmpVersion gosnmp.SnmpVersion
	switch version {
	case "2c", "v2c", "":
		snmpVersion = gosnmp.Version2c
	case "1", "v1":
		snmpVersion = gosnmp.Version1
	default:
		return nil, fmt.Errorf("unsupported snmp version %q", c.cfg.Version)
	}

	s := &gosnmp.GoSNMP{
		Target:         target.Address,
		Port:           c.cfg.Port,
		Community:      c.cfg.Community,
		Version:        snmpVersion,
		Timeout:        c.cfg.Timeout,
		Retries:        c.cfg.Retries,
		MaxRepetitions: c.cfg.MaxRepetitions,
	}
	if err := s.Connect(); err != nil {
		return nil, err
	}
	return s, nil
}

const (
	oidSysDescr0    = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID0 = "1.3.6.1.2.1.1.2.0"
	oidSysContact0  = "1.3.6.1.2.1.1.4.0"
	oidSysName0     = "1.3.6.1.2.1.1.5.0"
	oidSysLocation0 = "1.3.6.1.2.1.1.6.0"

	oidIfDescr       = "1.3.6.1.2.1.2.2.1.2"
	oidIfPhysAddress = "1.3.6.1.2.1.2.2.1.6"
	oidIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"
	oidIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
	oidIfMTU         = "1.3.6.1.2.1.2.2.1.4"
	oidIfSpeed       = "1.3.6.1.2.1.2.2.1.5"

	oidIfName      = "1.3.6.1.2.1.31.1.1.1.1"
	oidIfAlias     = "1.3.6.1.2.1.31.1.1.1.18"
	oidIfHighSpeed = "1.3.6.1.2.1.31.1.1.1.15"
)

func pduString(pdu gosnmp.SnmpPDU) (*string, bool) {
	switch v := pdu.Value.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, true
		}
		return &s, true
	case []byte:
		s := strings.TrimSpace(string(v))
		if s == "" {
			return nil, true
		}
		return &s, true
	default:
		return nil, false
	}
}

func pduInt32(pdu gosnmp.SnmpPDU) (*int32, bool) {
	switch v := pdu.Value.(type) {
	case int:
		n := int32(v)
		return &n, true
	case int32:
		n := v
		return &n, true
	case uint:
		n := int32(v)
		return &n, true
	case uint32:
		n := int32(v)
		return &n, true
	case int64:
		n := int32(v)
		return &n, true
	case uint64:
		n := int32(v)
		return &n, true
	default:
		return nil, false
	}
}

func pduInt64(pdu gosnmp.SnmpPDU) (*int64, bool) {
	switch v := pdu.Value.(type) {
	case int:
		n := int64(v)
		return &n, true
	case int32:
		n := int64(v)
		return &n, true
	case uint32:
		n := int64(v)
		return &n, true
	case int64:
		n := v
		return &n, true
	case uint64:
		n := int64(v)
		return &n, true
	default:
		return nil, false
	}
}

func pduMAC(pdu gosnmp.SnmpPDU) (*string, bool) {
	b, ok := pdu.Value.([]byte)
	if !ok || len(b) == 0 {
		return nil, ok
	}
	m := strings.ToLower(net.HardwareAddr(b).String())
	if m == "" || m == "00:00:00:00:00:00" {
		return nil, true
	}
	return &m, true
}

func lastOIDIndexInt(oid string) (int, bool) {
	oid = strings.TrimSpace(oid)
	if oid == "" {
		return 0, false
	}
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0, false
	}
	last := parts[len(parts)-1]
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (c *Client) GetSystem(ctx context.Context, target Target) (SystemInfo, error) {
	if c == nil {
		return SystemInfo{}, errors.New("snmp client is nil")
	}
	_ = ctx

	s, err := c.connect(target)
	if err != nil {
		return SystemInfo{}, err
	}
	defer s.Conn.Close()

	pkt, err := s.Get([]string{oidSysName0, oidSysDescr0, oidSysObjectID0, oidSysContact0, oidSysLocation0})
	if err != nil {
		return SystemInfo{}, err
	}

	var out SystemInfo
	for _, v := range pkt.Variables {
		switch v.Name {
		case oidSysName0:
			out.SysName, _ = pduString(v)
		case oidSysDescr0:
			out.SysDescr, _ = pduString(v)
		case oidSysObjectID0:
			out.SysObjectID, _ = pduString(v)
		case oidSysContact0:
			out.SysContact, _ = pduString(v)
		case oidSysLocation0:
			out.SysLocation, _ = pduString(v)
		}
	}
	return out, nil
}

func (c *Client) WalkIntTable(ctx context.Context, target Target, baseOID string) (map[int]int, error) {
	if c == nil {
		return nil, errors.New("snmp client is nil")
	}
	_ = ctx

	s, err := c.connect(target)
	if err != nil {
		return nil, err
	}
	defer s.Conn.Close()

	pdus, err := s.BulkWalkAll(baseOID)
	if err != nil {
		return nil, err
	}

	out := make(map[int]int, len(pdus))
	for _, p := range pdus {
		idx, ok := lastOIDIndexInt(p.Name)
		if !ok {
			continue
		}
		if v, ok := pduInt32(p); ok && v != nil {
			out[idx] = int(*v)
		}
	}
	return out, nil
}

func (c *Client) WalkInterfaces(ctx context.Context, target Target) (map[int]InterfaceInfo, error) {
	if c == nil {
		return nil, errors.New("snmp client is nil")
	}
	_ = ctx

	s, err := c.connect(target)
	if err != nil {
		return nil, err
	}
	defer s.Conn.Close()

	out := make(map[int]InterfaceInfo)
	ensure := func(idx int) InterfaceInfo {
		if cur, ok := out[idx]; ok {
			return cur
		}
		ii := InterfaceInfo{IfIndex: idx}
		out[idx] = ii
		return ii
	}
	save := func(idx int, ii InterfaceInfo) {
		out[idx] = ii
	}

	walk := func(baseOID string, handle func(idx int, p gosnmp.SnmpPDU)) error {
		pdus, err := s.BulkWalkAll(baseOID)
		if err != nil {
			return err
		}
		for _, p := range pdus {
			idx, ok := lastOIDIndexInt(p.Name)
			if !ok {
				continue
			}
			handle(idx, p)
		}
		return nil
	}

	if err := walk(oidIfName, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if s, ok := pduString(p); ok {
			ii.Name = s
		}
		save(idx, ii)
	}); err != nil {
		return nil, err
	}
	_ = walk(oidIfDescr, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if s, ok := pduString(p); ok {
			ii.Descr = s
		}
		save(idx, ii)
	})
	_ = walk(oidIfAlias, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if s, ok := pduString(p); ok {
			ii.Alias = s
		}
		save(idx, ii)
	})
	_ = walk(oidIfPhysAddress, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if m, ok := pduMAC(p); ok {
			ii.MAC = m
		}
		save(idx, ii)
	})
	_ = walk(oidIfAdminStatus, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if n, ok := pduInt32(p); ok {
			ii.AdminStatus = n
		}
		save(idx, ii)
	})
	_ = walk(oidIfOperStatus, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if n, ok := pduInt32(p); ok {
			ii.OperStatus = n
		}
		save(idx, ii)
	})
	_ = walk(oidIfMTU, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if n, ok := pduInt32(p); ok {
			ii.MTU = n
		}
		save(idx, ii)
	})
	_ = walk(oidIfSpeed, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if n, ok := pduInt64(p); ok {
			ii.SpeedBps = n
		}
		save(idx, ii)
	})
	_ = walk(oidIfHighSpeed, func(idx int, p gosnmp.SnmpPDU) {
		ii := ensure(idx)
		if n, ok := pduInt64(p); ok && n != nil {
			// ifHighSpeed is in Mbps.
			bps := (*n) * 1_000_000
			ii.SpeedBps = &bps
		}
		save(idx, ii)
	})

	return out, nil
}
