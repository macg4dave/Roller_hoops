package vlan

import (
	"context"
	"fmt"

	"roller_hoops/core-go/internal/enrichment/snmp"
)

// PortMapping describes the VLAN and switch-port relationship we plan to capture.
type PortMapping struct {
	DeviceID string
	Switch   string
	Port     string
	VLAN     int
}

// Collector is a stub for future VLAN bridge-MIB walks.
type Collector struct {
	snmp *snmp.Client
}

func NewCollector(client *snmp.Client) *Collector {
	return &Collector{snmp: client}
}

const (
	oidDot1dBasePortIfIndex = "1.3.6.1.2.1.17.1.4.1.2"
	oidDot1qPvid            = "1.3.6.1.2.1.17.7.1.4.5.1.1"
)

// CollectPVIDByIfIndex maps ifIndex -> VLAN ID (PVID) using bridge/q-bridge MIB tables.
func (c *Collector) CollectPVIDByIfIndex(ctx context.Context, target snmp.Target) (map[int]int, error) {
	if c == nil || c.snmp == nil {
		return nil, fmt.Errorf("snmp client not configured")
	}

	// dot1dBasePortIfIndex (index: dot1dBasePort -> value: ifIndex)
	basePortToIfIndex, err := c.snmp.WalkIntTable(ctx, target, oidDot1dBasePortIfIndex)
	if err != nil {
		return nil, err
	}
	// dot1qPvid (index: dot1dBasePort -> value: vlan id)
	basePortToPVID, err := c.snmp.WalkIntTable(ctx, target, oidDot1qPvid)
	if err != nil {
		return nil, err
	}

	out := make(map[int]int)
	for basePort, ifIndex := range basePortToIfIndex {
		pvid, ok := basePortToPVID[basePort]
		if !ok || pvid <= 0 {
			continue
		}
		out[ifIndex] = pvid
	}
	return out, nil
}

// Collect fetches switch-port mappings via SNMP bridge-MIB; it currently returns PVID-based mappings only.
func (c *Collector) Collect(ctx context.Context, switchIP string) ([]PortMapping, error) {
	target := snmp.Target{Address: switchIP}
	pvidByIfIndex, err := c.CollectPVIDByIfIndex(ctx, target)
	if err != nil {
		return nil, err
	}

	out := make([]PortMapping, 0, len(pvidByIfIndex))
	for ifIndex, vlanID := range pvidByIfIndex {
		out = append(out, PortMapping{
			Switch: switchIP,
			Port:   fmt.Sprintf("ifIndex:%d", ifIndex),
			VLAN:   vlanID,
		})
	}
	return out, nil
}
