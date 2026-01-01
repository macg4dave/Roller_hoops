package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"roller_hoops/core-go/internal/sqlcgen"
)

const (
	mapMaxRegions = 8
	mapMaxNodes   = 120
	mapMaxEdges   = 80
)

type mapProjection struct {
	Layer      string         `json:"layer"`
	Focus      *mapFocus      `json:"focus,omitempty"`
	Guidance   *string        `json:"guidance,omitempty"`
	Regions    []mapRegion    `json:"regions"`
	Nodes      []mapNode      `json:"nodes"`
	Edges      []mapEdge      `json:"edges"`
	Inspector  *mapInspector  `json:"inspector,omitempty"`
	Truncation mapTruncation  `json:"truncation"`
}

type mapFocus struct {
	Type  string  `json:"type"`
	ID    string  `json:"id"`
	Label *string `json:"label,omitempty"`
}

type mapTruncation struct {
	Regions mapTruncationMetric `json:"regions"`
	Nodes   mapTruncationMetric `json:"nodes"`
	Edges   mapTruncationMetric `json:"edges"`
}

type mapTruncationMetric struct {
	Returned  int     `json:"returned"`
	Limit     int     `json:"limit"`
	Truncated bool    `json:"truncated"`
	Total     *int    `json:"total,omitempty"`
	Warning   *string `json:"warning,omitempty"`
}

type mapRegion struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Label          string         `json:"label"`
	ParentRegionID *string        `json:"parent_region_id,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
}

type mapNode struct {
	ID              string         `json:"id"`
	Kind            string         `json:"kind"`
	Label           *string        `json:"label,omitempty"`
	PrimaryRegionID *string        `json:"primary_region_id,omitempty"`
	RegionIDs       []string       `json:"region_ids"`
	Meta            map[string]any `json:"meta,omitempty"`
}

type mapEdge struct {
	ID    string         `json:"id"`
	Kind  string         `json:"kind"`
	From  string         `json:"from"`
	To    string         `json:"to"`
	Label *string        `json:"label,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
}

type mapInspector struct {
	Title         string                   `json:"title"`
	Identity      []mapInspectorField      `json:"identity"`
	Status        []mapInspectorField      `json:"status"`
	Relationships []mapInspectorRelation   `json:"relationships"`
}

type mapInspectorField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type mapInspectorRelation struct {
	Label     string `json:"label"`
	Layer     string `json:"layer"`
	FocusType string `json:"focus_type"`
	FocusID   string `json:"focus_id"`
}

func (h *Handler) handleGetMapProjection(w http.ResponseWriter, r *http.Request) {
	layer := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "layer")))
	if !isValidMapLayer(layer) {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid layer", map[string]any{"layer": layer})
		return
	}

	q := r.URL.Query()
	depth, err := parseMapDepthParam(q.Get("depth"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid depth", map[string]any{"error": err.Error()})
		return
	}

	limitHint, err := parseLimitParam(q.Get("limit"), mapMaxNodes, 500)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}

	regionLimit := min(mapMaxRegions, limitHint)
	nodeLimit := min(mapMaxNodes, limitHint)
	edgeLimit := min(mapMaxEdges, limitHint)

	focusTypeRaw := strings.TrimSpace(q.Get("focusType"))
	focusIDRaw := strings.TrimSpace(q.Get("focusId"))

	if focusTypeRaw == "" && focusIDRaw == "" {
		guidance := "No focus selected. Provide focusType and focusId to render a scoped projection."
		resp := emptyMapProjection(layer, &guidance, regionLimit, nodeLimit, edgeLimit)
		sortMapProjection(&resp)
		h.writeJSON(w, http.StatusOK, resp)
		return
	}

	if focusTypeRaw == "" || focusIDRaw == "" {
		h.writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"focusType and focusId must be provided together",
			map[string]any{"focusType": focusTypeRaw, "focusId": focusIDRaw},
		)
		return
	}

	focusType := strings.ToLower(focusTypeRaw)
	if !isValidMapFocusType(focusType) {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid focusType", map[string]any{"focusType": focusType})
		return
	}

	var focusID string
	var focusLabel *string
	var focusNode *mapNode
	var inspector *mapInspector
	var l3AllSubnets []string
	var l3Subnets []string
	var l3SubnetsTruncated bool
	var l3PeerRegions map[string]map[string]struct{}
	var l3PeerLabels map[string]*string

	switch focusType {
	case "device":
		if !h.ensureDeviceQueries(w) {
			return
		}
		ctx := r.Context()
		deviceRow, err := h.devices.GetDevice(ctx, focusIDRaw)
		if err != nil {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": focusIDRaw})
			case isInvalidUUID(err):
				h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": focusIDRaw})
			default:
				h.log.Error().Err(err).Str("device_id", focusIDRaw).Msg("map projection device lookup failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to load device", nil)
			}
			return
		}

		focusID = deviceRow.ID
		focusLabel = deviceRow.DisplayName
		focusNode = &mapNode{
			ID:        deviceRow.ID,
			Kind:      "device",
			Label:     deviceRow.DisplayName,
			RegionIDs: []string{},
			Meta: map[string]any{
				"device_id": deviceRow.ID,
			},
		}

		title := deviceRow.ID
		if deviceRow.DisplayName != nil && strings.TrimSpace(*deviceRow.DisplayName) != "" {
			title = strings.TrimSpace(*deviceRow.DisplayName)
		}
		identity := []mapInspectorField{
			{Label: "Type", Value: "Device"},
			{Label: "ID", Value: deviceRow.ID},
		}
		if deviceRow.DisplayName != nil && strings.TrimSpace(*deviceRow.DisplayName) != "" {
			identity = append(identity, mapInspectorField{Label: "Display name", Value: strings.TrimSpace(*deviceRow.DisplayName)})
		}
		if deviceRow.Owner != nil && strings.TrimSpace(*deviceRow.Owner) != "" {
			identity = append(identity, mapInspectorField{Label: "Owner", Value: strings.TrimSpace(*deviceRow.Owner)})
		}
		if deviceRow.Location != nil && strings.TrimSpace(*deviceRow.Location) != "" {
			identity = append(identity, mapInspectorField{Label: "Location", Value: strings.TrimSpace(*deviceRow.Location)})
		}

		status := []mapInspectorField{
			{Label: "Layer", Value: layer},
			{Label: "Projection", Value: "scaffolding"},
		}

		if layer == "l3" && depth > 0 {
			status[1].Value = "l3 (device focus)"

			deviceIPs, err := h.devices.ListDeviceIPs(ctx, deviceRow.ID)
			if err != nil {
				h.log.Error().Err(err).Str("device_id", deviceRow.ID).Msg("list device IPs for map projection failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l3 projection", nil)
				return
			}

			ipStrings := make([]string, 0, len(deviceIPs))
			for _, row := range deviceIPs {
				ipStrings = append(ipStrings, row.IP)
			}
			l3AllSubnets = deriveL3SubnetIDs(ipStrings)
			l3Subnets = l3AllSubnets
			if len(l3Subnets) > regionLimit {
				l3Subnets = l3Subnets[:regionLimit]
			}
			l3SubnetsTruncated = len(l3AllSubnets) > len(l3Subnets)

			peerLister, ok := h.devices.(interface {
				ListDevicePeersInCIDR(ctx context.Context, cidr string, excludeDeviceID string, limit int32) ([]sqlcgen.MapDevicePeer, error)
			})
			if ok && len(l3Subnets) > 0 {
				l3PeerRegions = make(map[string]map[string]struct{})
				l3PeerLabels = make(map[string]*string)
				peerQueryLimit := int32(nodeLimit)
				for _, subnet := range l3Subnets {
					peers, err := peerLister.ListDevicePeersInCIDR(ctx, subnet, deviceRow.ID, peerQueryLimit)
					if err != nil {
						h.log.Error().Err(err).Str("device_id", deviceRow.ID).Str("subnet", subnet).Msg("list l3 peers failed")
						h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l3 projection", nil)
						return
					}
					for _, peer := range peers {
						regions := l3PeerRegions[peer.ID]
						if regions == nil {
							regions = make(map[string]struct{})
							l3PeerRegions[peer.ID] = regions
						}
						regions[subnet] = struct{}{}
						if _, exists := l3PeerLabels[peer.ID]; !exists {
							l3PeerLabels[peer.ID] = peer.DisplayName
						}
					}
				}
			}

			if len(l3Subnets) > 0 {
				if l3SubnetsTruncated {
					status = append(status, mapInspectorField{
						Label: "Subnets",
						Value: fmt.Sprintf("%d of %d", len(l3Subnets), len(l3AllSubnets)),
					})
				} else {
					status = append(status, mapInspectorField{Label: "Subnets", Value: strconv.Itoa(len(l3Subnets))})
				}
				primary := l3Subnets[0]
				status = append(status, mapInspectorField{Label: "Primary subnet", Value: primary})
				if len(l3Subnets) > 1 || l3SubnetsTruncated {
					alsoParts := []string{}
					if len(l3Subnets) > 1 {
						alsoParts = append(alsoParts, l3Subnets[1:]...)
					}
					alsoIn := strings.Join(alsoParts, ", ")
					if l3SubnetsTruncated {
						omitted := len(l3AllSubnets) - len(l3Subnets)
						if omitted > 0 {
							if alsoIn == "" {
								alsoIn = fmt.Sprintf("(+%d more)", omitted)
							} else {
								alsoIn = alsoIn + fmt.Sprintf(" (+%d more)", omitted)
							}
						}
					}
					if alsoIn != "" {
						status = append(status, mapInspectorField{Label: "Also in", Value: alsoIn})
					}
				}
			} else {
				status = append(status, mapInspectorField{Label: "Subnets", Value: "none"})
			}
		}

		inspector = &mapInspector{
			Title:         title,
			Identity:      identity,
			Status:        status,
			Relationships: buildMapInspectorRelationships(focusType, focusID),
		}

	case "subnet":
		prefix, err := netip.ParsePrefix(focusIDRaw)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_id", "subnet id must be a CIDR prefix", map[string]any{"id": focusIDRaw})
			return
		}
		canonical := prefix.Masked().String()
		focusID = canonical
		label := canonical
		focusLabel = &label

		projection := "scaffolding (no regions/nodes yet)"
		if layer == "l3" {
			projection = "l3 (subnet focus)"
		}
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "Subnet"},
				{Label: "CIDR", Value: label},
			},
			Status: []mapInspectorField{
				{Label: "Layer", Value: layer},
				{Label: "Projection", Value: projection},
			},
			Relationships: buildMapInspectorRelationships(focusType, focusID),
		}

	case "vlan":
		vlanID, err := strconv.Atoi(focusIDRaw)
		if err != nil || vlanID <= 0 || vlanID > 4094 {
			h.writeError(w, http.StatusBadRequest, "invalid_id", "vlan id must be an integer between 1 and 4094", map[string]any{"id": focusIDRaw})
			return
		}
		focusID = strconv.Itoa(vlanID)
		label := "VLAN " + focusID
		focusLabel = &label
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "VLAN"},
				{Label: "ID", Value: focusID},
			},
			Status: []mapInspectorField{
				{Label: "Layer", Value: layer},
				{Label: "Projection", Value: "scaffolding (no regions/nodes yet)"},
			},
			Relationships: buildMapInspectorRelationships(focusType, focusID),
		}

	case "zone":
		focusID = focusIDRaw
		label := focusIDRaw
		focusLabel = &label
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "Zone"},
				{Label: "ID", Value: focusID},
			},
			Status: []mapInspectorField{
				{Label: "Layer", Value: layer},
				{Label: "Projection", Value: "scaffolding (no regions/nodes yet)"},
			},
			Relationships: buildMapInspectorRelationships(focusType, focusID),
		}

	case "service":
		focusID = focusIDRaw
		label := focusIDRaw
		focusLabel = &label
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "Service"},
				{Label: "ID", Value: focusID},
			},
			Status: []mapInspectorField{
				{Label: "Layer", Value: layer},
				{Label: "Projection", Value: "scaffolding (no regions/nodes yet)"},
			},
			Relationships: buildMapInspectorRelationships(focusType, focusID),
		}
	}

	resp := emptyMapProjection(layer, nil, regionLimit, nodeLimit, edgeLimit)
	resp.Focus = &mapFocus{Type: focusType, ID: focusID, Label: focusLabel}
	resp.Inspector = inspector

	if layer == "l3" && focusType == "device" && depth > 0 {
		// Regions: derived subnets from the focused device's IP facts.
		if focusNode != nil {
			focusNode.RegionIDs = append([]string{}, l3Subnets...)
			if len(l3Subnets) > 0 {
				primary := l3Subnets[0]
				focusNode.PrimaryRegionID = &primary
			}
		}

		resp.Regions = make([]mapRegion, 0, len(l3Subnets))
		for _, subnet := range l3Subnets {
			resp.Regions = append(resp.Regions, mapRegion{ID: subnet, Kind: "subnet", Label: subnet})
		}
		resp.Truncation.Regions.Returned = len(resp.Regions)
		resp.Truncation.Regions.Truncated = l3SubnetsTruncated
		totalRegions := len(l3AllSubnets)
		resp.Truncation.Regions.Total = &totalRegions
		if l3SubnetsTruncated {
			warning := fmt.Sprintf("Subnet cap hit: showing %d of %d.", len(resp.Regions), len(l3AllSubnets))
			resp.Truncation.Regions.Warning = &warning
		}

		peerIDs := make([]string, 0, len(l3PeerRegions))
		for id := range l3PeerRegions {
			peerIDs = append(peerIDs, id)
		}
		sort.Strings(peerIDs)

		maxPeers := nodeLimit - 1
		if maxPeers < 0 {
			maxPeers = 0
		}
		nodesTruncated := len(peerIDs) > maxPeers
		peerIDsIncluded := peerIDs
		if nodesTruncated {
			peerIDsIncluded = peerIDs[:maxPeers]
			warning := fmt.Sprintf("Node cap hit: showing %d of >%d devices.", 1+len(peerIDsIncluded), nodeLimit)
			resp.Truncation.Nodes.Warning = &warning
		}

		resp.Nodes = make([]mapNode, 0, 1+len(peerIDsIncluded))
		if focusNode != nil {
			resp.Nodes = append(resp.Nodes, *focusNode)
		}
		for _, peerID := range peerIDsIncluded {
			regionSet := l3PeerRegions[peerID]
			regionIDs := make([]string, 0, len(regionSet))
			for regionID := range regionSet {
				regionIDs = append(regionIDs, regionID)
			}
			sort.Strings(regionIDs)

			n := mapNode{ID: peerID, Kind: "device", Label: l3PeerLabels[peerID], RegionIDs: regionIDs}
			if len(regionIDs) > 0 {
				primary := regionIDs[0]
				n.PrimaryRegionID = &primary
			}
			resp.Nodes = append(resp.Nodes, n)
		}
		resp.Truncation.Nodes.Returned = len(resp.Nodes)
		resp.Truncation.Nodes.Truncated = nodesTruncated
		if !nodesTruncated {
			totalNodes := len(resp.Nodes)
			resp.Truncation.Nodes.Total = &totalNodes
		}

		// Optional, bounded connectors: focus → peer edges (no mesh).
		edgesTotal := len(peerIDsIncluded)
		resp.Truncation.Edges.Total = &edgesTotal

		edgePeerIDs := peerIDsIncluded
		edgesTruncated := len(edgePeerIDs) > edgeLimit
		if edgesTruncated {
			edgePeerIDs = edgePeerIDs[:edgeLimit]
			warning := fmt.Sprintf("Edge cap hit: showing %d of %d.", len(edgePeerIDs), edgesTotal)
			resp.Truncation.Edges.Warning = &warning
		}
		resp.Edges = make([]mapEdge, 0, len(edgePeerIDs))
		for _, peerID := range edgePeerIDs {
			resp.Edges = append(resp.Edges, mapEdge{
				ID:   fmt.Sprintf("peer:%s:%s", focusID, peerID),
				Kind: "peer",
				From: focusID,
				To:   peerID,
			})
		}
		resp.Truncation.Edges.Returned = len(resp.Edges)
		resp.Truncation.Edges.Truncated = edgesTruncated

		if resp.Inspector != nil {
			peerCount := len(resp.Nodes)
			if peerCount > 0 {
				peerCount = peerCount - 1
			}
			resp.Inspector.Status = append(resp.Inspector.Status, mapInspectorField{Label: "Peers", Value: strconv.Itoa(peerCount)})
		}

		if len(l3AllSubnets) == 0 {
			guidance := "No subnet regions derived (no IP facts). Run discovery or add IP facts to render an L3 projection."
			resp.Guidance = &guidance
		} else if resp.Truncation.Regions.Truncated || resp.Truncation.Nodes.Truncated || resp.Truncation.Edges.Truncated {
			guidance := "Projection truncated: some subnets/nodes/edges were capped for readability."
			resp.Guidance = &guidance
		}
	} else if layer == "l3" && focusType == "subnet" {
		resp.Regions = []mapRegion{{ID: focusID, Kind: "subnet", Label: focusID}}
		resp.Truncation.Regions.Returned = len(resp.Regions)
		totalRegions := 1
		resp.Truncation.Regions.Total = &totalRegions

		if depth > 0 {
			if !h.ensureDeviceQueries(w) {
				return
			}
			ctx := r.Context()
			subnetLister, ok := h.devices.(interface {
				ListDevicesInCIDR(ctx context.Context, cidr string, limit int32) ([]sqlcgen.MapDevicePeer, error)
			})
			if !ok {
				h.log.Error().Msg("map subnet projection query missing")
				h.writeError(w, http.StatusInternalServerError, "internal_error", "map subnet projection not supported", nil)
				return
			}

			queryLimit := int32(nodeLimit + 1)
			members, err := subnetLister.ListDevicesInCIDR(ctx, focusID, queryLimit)
			if err != nil {
				h.log.Error().Err(err).Str("subnet", focusID).Msg("list subnet members failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l3 projection", nil)
				return
			}

			nodesTruncated := len(members) > nodeLimit
			membersIncluded := members
			if nodesTruncated {
				membersIncluded = members[:nodeLimit]
				warning := fmt.Sprintf("Node cap hit: showing %d of >%d devices.", len(membersIncluded), nodeLimit)
				resp.Truncation.Nodes.Warning = &warning
			}

			primary := focusID
			resp.Nodes = make([]mapNode, 0, len(membersIncluded))
			for _, member := range membersIncluded {
				resp.Nodes = append(resp.Nodes, mapNode{
					ID:              member.ID,
					Kind:            "device",
					Label:           member.DisplayName,
					PrimaryRegionID: &primary,
					RegionIDs:       []string{focusID},
				})
			}

			resp.Truncation.Nodes.Returned = len(resp.Nodes)
			resp.Truncation.Nodes.Truncated = nodesTruncated
			if !nodesTruncated {
				totalNodes := len(resp.Nodes)
				resp.Truncation.Nodes.Total = &totalNodes
			}

			if resp.Inspector != nil {
				resp.Inspector.Status = append(resp.Inspector.Status, mapInspectorField{Label: "Devices", Value: strconv.Itoa(len(resp.Nodes))})
			}

			if len(resp.Nodes) == 0 {
				guidance := "No devices observed in this subnet yet. Run discovery or add IP facts to populate membership."
				resp.Guidance = &guidance
			} else if resp.Truncation.Nodes.Truncated {
				guidance := "Projection truncated: some devices were capped for readability."
				resp.Guidance = &guidance
			}
		}
	} else if focusNode != nil {
		resp.Nodes = []mapNode{*focusNode}
		resp.Truncation.Nodes.Returned = len(resp.Nodes)
	}

	sortMapProjection(&resp)
	h.writeJSON(w, http.StatusOK, resp)
}

func emptyMapProjection(layer string, guidance *string, regionLimit, nodeLimit, edgeLimit int) mapProjection {
	return mapProjection{
		Layer:    layer,
		Guidance: guidance,
		Regions:  []mapRegion{},
		Nodes:    []mapNode{},
		Edges:    []mapEdge{},
		Truncation: mapTruncation{
			Regions: mapTruncationMetric{Returned: 0, Limit: regionLimit, Truncated: false},
			Nodes:   mapTruncationMetric{Returned: 0, Limit: nodeLimit, Truncated: false},
			Edges:   mapTruncationMetric{Returned: 0, Limit: edgeLimit, Truncated: false},
		},
	}
}

func isValidMapLayer(layer string) bool {
	switch layer {
	case "physical", "l2", "l3", "services", "security":
		return true
	default:
		return false
	}
}

func isValidMapFocusType(focusType string) bool {
	switch focusType {
	case "device", "subnet", "vlan", "zone", "service":
		return true
	default:
		return false
	}
}

func buildMapInspectorRelationships(focusType, focusID string) []mapInspectorRelation {
	return []mapInspectorRelation{
		{Label: "View in Physical", Layer: "physical", FocusType: focusType, FocusID: focusID},
		{Label: "View in L2", Layer: "l2", FocusType: focusType, FocusID: focusID},
		{Label: "View in L3", Layer: "l3", FocusType: focusType, FocusID: focusID},
		{Label: "View in Services", Layer: "services", FocusType: focusType, FocusID: focusID},
		{Label: "View in Security", Layer: "security", FocusType: focusType, FocusID: focusID},
	}
}

func parseMapDepthParam(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("invalid value")
	}
	if parsed < 0 {
		return 0, errors.New("must be non-negative")
	}
	if parsed > 3 {
		parsed = 3
	}
	return parsed, nil
}

func sortMapProjection(p *mapProjection) {
	sort.SliceStable(p.Regions, func(i, j int) bool { return p.Regions[i].ID < p.Regions[j].ID })
	sort.SliceStable(p.Nodes, func(i, j int) bool { return p.Nodes[i].ID < p.Nodes[j].ID })
	sort.SliceStable(p.Edges, func(i, j int) bool { return p.Edges[i].ID < p.Edges[j].ID })
}

func deriveL3SubnetIDs(ips []string) []string {
	seen := make(map[string]struct{})
	for _, raw := range ips {
		prefix, ok := deriveL3SubnetID(raw)
		if !ok {
			continue
		}
		seen[prefix] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func deriveL3SubnetID(ip string) (string, bool) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", false
	}

	var addr netip.Addr
	var bits int
	if p, err := netip.ParsePrefix(ip); err == nil {
		addr = p.Addr()
		bits = p.Bits()
	} else if a, err := netip.ParseAddr(ip); err == nil {
		addr = a
		bits = addr.BitLen()
	} else {
		return "", false
	}

	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() || addr.IsMulticast() || addr.IsLinkLocalUnicast() {
		return "", false
	}

	defaultBits := 24
	if addr.Is6() {
		defaultBits = 64
	}

	// Most stored inet values have a full-length mask; treat those as "unknown prefix" and apply defaults.
	if bits <= 0 || bits >= addr.BitLen() {
		bits = defaultBits
	}
	prefix := netip.PrefixFrom(addr, bits).Masked()
	return prefix.String(), true
}
