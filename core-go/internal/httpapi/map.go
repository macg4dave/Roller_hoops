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
	var l2AllVLANs []string
	var l2VLANs []string
	var l2VLANsTruncated bool
	var l2PeerRegions map[string]map[string]struct{}
	var l2PeerLabels map[string]*string
	var physicalLinks []sqlcgen.MapDeviceLinkPeer
	var physicalLinksTruncated bool

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

			if len(deviceIPs) > 0 {
				ips := append([]sqlcgen.DeviceIP(nil), deviceIPs...)
				sort.SliceStable(ips, func(i, j int) bool {
					if ips[i].UpdatedAt.Equal(ips[j].UpdatedAt) {
						return ips[i].IP < ips[j].IP
					}
					return ips[i].UpdatedAt.After(ips[j].UpdatedAt)
				})
				primary := strings.TrimSpace(ips[0].IP)
				if primary != "" {
					identity = append(identity, mapInspectorField{Label: "Primary IP", Value: primary})
				}
				status = append(status, mapInspectorField{Label: "IP facts", Value: strconv.Itoa(len(deviceIPs))})
			} else {
				status = append(status, mapInspectorField{Label: "IP facts", Value: "0"})
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

		if layer == "l2" && depth > 0 {
			status[1].Value = "l2 (device focus)"
			status = append(status, mapInspectorField{Label: "Membership model", Value: "PVID only"})

			pvidLister, ok := h.devices.(interface {
				ListDevicePVIDs(ctx context.Context, deviceID string) ([]int32, error)
			})
			if !ok {
				h.log.Error().Msg("map l2 projection query missing")
				h.writeError(w, http.StatusInternalServerError, "internal_error", "map l2 projection not supported", nil)
				return
			}

			pvids, err := pvidLister.ListDevicePVIDs(ctx, deviceRow.ID)
			if err != nil {
				h.log.Error().Err(err).Str("device_id", deviceRow.ID).Msg("list device pvids for map projection failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l2 projection", nil)
				return
			}

			vlanSeen := make(map[int32]struct{})
			vlanSorted := make([]int, 0, len(pvids))
			for _, vlanID := range pvids {
				if vlanID <= 0 {
					continue
				}
				if _, exists := vlanSeen[vlanID]; exists {
					continue
				}
				vlanSeen[vlanID] = struct{}{}
				vlanSorted = append(vlanSorted, int(vlanID))
			}
			sort.Ints(vlanSorted)

			l2AllVLANs = make([]string, 0, len(vlanSorted))
			for _, vlanID := range vlanSorted {
				l2AllVLANs = append(l2AllVLANs, strconv.Itoa(vlanID))
			}

			l2VLANs = l2AllVLANs
			if len(l2VLANs) > regionLimit {
				l2VLANs = l2VLANs[:regionLimit]
			}
			l2VLANsTruncated = len(l2AllVLANs) > len(l2VLANs)

			peerLister, ok := h.devices.(interface {
				ListDevicePeersInVLAN(ctx context.Context, vlanID int32, excludeDeviceID string, limit int32) ([]sqlcgen.MapDevicePeer, error)
			})
			if ok && len(l2VLANs) > 0 {
				l2PeerRegions = make(map[string]map[string]struct{})
				l2PeerLabels = make(map[string]*string)
				peerQueryLimit := int32(nodeLimit)
				for _, vlanIDStr := range l2VLANs {
					vlanIDInt, err := strconv.Atoi(vlanIDStr)
					if err != nil {
						continue
					}
					peers, err := peerLister.ListDevicePeersInVLAN(ctx, int32(vlanIDInt), deviceRow.ID, peerQueryLimit)
					if err != nil {
						h.log.Error().Err(err).Str("device_id", deviceRow.ID).Str("vlan", vlanIDStr).Msg("list l2 peers failed")
						h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l2 projection", nil)
						return
					}
					for _, peer := range peers {
						regions := l2PeerRegions[peer.ID]
						if regions == nil {
							regions = make(map[string]struct{})
							l2PeerRegions[peer.ID] = regions
						}
						regions[vlanIDStr] = struct{}{}
						if _, exists := l2PeerLabels[peer.ID]; !exists {
							l2PeerLabels[peer.ID] = peer.DisplayName
						}
					}
				}
			}

			if len(l2VLANs) > 0 {
				if l2VLANsTruncated {
					status = append(status, mapInspectorField{
						Label: "VLANs",
						Value: fmt.Sprintf("%d of %d", len(l2VLANs), len(l2AllVLANs)),
					})
				} else {
					status = append(status, mapInspectorField{Label: "VLANs", Value: strconv.Itoa(len(l2VLANs))})
				}
				primary := l2VLANs[0]
				status = append(status, mapInspectorField{Label: "Primary VLAN", Value: primary})
				if len(l2VLANs) > 1 || l2VLANsTruncated {
					alsoParts := []string{}
					if len(l2VLANs) > 1 {
						alsoParts = append(alsoParts, l2VLANs[1:]...)
					}
					alsoIn := strings.Join(alsoParts, ", ")
					if l2VLANsTruncated {
						omitted := len(l2AllVLANs) - len(l2VLANs)
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
				status = append(status, mapInspectorField{Label: "VLANs", Value: "none"})
			}
		}

		if layer == "physical" && depth > 0 {
			status[1].Value = "physical (device focus)"

			linkLister, ok := h.devices.(interface {
				ListDeviceLinkPeers(ctx context.Context, deviceID string, limit int32) ([]sqlcgen.MapDeviceLinkPeer, error)
			})
			if !ok {
				h.log.Error().Msg("map physical projection query missing")
				h.writeError(w, http.StatusInternalServerError, "internal_error", "map physical projection not supported", nil)
				return
			}

			maxPeers := nodeLimit - 1
			if maxPeers < 0 {
				maxPeers = 0
			}
			if edgeLimit < maxPeers {
				maxPeers = edgeLimit
			}
			queryLimit := int32(maxPeers + 1)
			links, err := linkLister.ListDeviceLinkPeers(ctx, deviceRow.ID, queryLimit)
			if err != nil {
				h.log.Error().Err(err).Str("device_id", deviceRow.ID).Msg("list device physical links failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build physical projection", nil)
				return
			}

			physicalLinksTruncated = len(links) > maxPeers
			physicalLinks = links
			if physicalLinksTruncated {
				physicalLinks = links[:maxPeers]
			}

			if len(physicalLinks) > 0 {
				if physicalLinksTruncated {
					status = append(status, mapInspectorField{Label: "Links", Value: fmt.Sprintf("%d of >%d", len(physicalLinks), maxPeers)})
				} else {
					status = append(status, mapInspectorField{Label: "Links", Value: strconv.Itoa(len(physicalLinks))})
				}
			} else {
				status = append(status, mapInspectorField{Label: "Links", Value: "none"})
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

		family := "IPv4"
		if prefix.Addr().Is6() {
			family = "IPv6"
		}
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "Subnet"},
				{Label: "CIDR", Value: label},
				{Label: "Family", Value: family},
				{Label: "Prefix bits", Value: strconv.Itoa(prefix.Bits())},
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

		projection := "scaffolding (no regions/nodes yet)"
		if layer == "l2" {
			projection = "l2 (vlan focus)"
		}

		status := []mapInspectorField{
			{Label: "Layer", Value: layer},
			{Label: "Projection", Value: projection},
		}
		if layer == "l2" {
			status = append(status, mapInspectorField{Label: "Membership model", Value: "PVID only"})
		}
		inspector = &mapInspector{
			Title: label,
			Identity: []mapInspectorField{
				{Label: "Type", Value: "VLAN"},
				{Label: "ID", Value: focusID},
			},
			Status:        status,
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

			for _, subnet := range l3Subnets {
				resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
					Label:     "Open subnet " + subnet,
					Layer:     "l3",
					FocusType: "subnet",
					FocusID:   subnet,
				})
			}

			for _, peerID := range peerIDsIncluded {
				label := peerID
				if peerLabel := l3PeerLabels[peerID]; peerLabel != nil {
					if trimmed := strings.TrimSpace(*peerLabel); trimmed != "" {
						label = trimmed
					}
				}

				resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
					Label:     "Open device " + label,
					Layer:     "l3",
					FocusType: "device",
					FocusID:   peerID,
				})
			}
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

				for _, node := range resp.Nodes {
					label := node.ID
					if node.Label != nil {
						if trimmed := strings.TrimSpace(*node.Label); trimmed != "" {
							label = trimmed
						}
					}

					resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
						Label:     "Open device " + label,
						Layer:     "l3",
						FocusType: "device",
						FocusID:   node.ID,
					})
				}
			}

			if len(resp.Nodes) == 0 {
				guidance := "No devices observed in this subnet yet. Run discovery or add IP facts to populate membership."
				resp.Guidance = &guidance
			} else if resp.Truncation.Nodes.Truncated {
				guidance := "Projection truncated: some devices were capped for readability."
				resp.Guidance = &guidance
			}
		}
	} else if layer == "l2" && focusType == "device" && depth > 0 {
		if focusNode != nil {
			focusNode.RegionIDs = append([]string{}, l2VLANs...)
			if len(l2VLANs) > 0 {
				primary := l2VLANs[0]
				focusNode.PrimaryRegionID = &primary
			}
		}

		resp.Regions = make([]mapRegion, 0, len(l2VLANs))
		for _, vlanID := range l2VLANs {
			resp.Regions = append(resp.Regions, mapRegion{ID: vlanID, Kind: "vlan", Label: "VLAN " + vlanID})
		}
		resp.Truncation.Regions.Returned = len(resp.Regions)
		resp.Truncation.Regions.Truncated = l2VLANsTruncated
		totalRegions := len(l2AllVLANs)
		resp.Truncation.Regions.Total = &totalRegions
		if l2VLANsTruncated {
			warning := fmt.Sprintf("VLAN cap hit: showing %d of %d.", len(resp.Regions), len(l2AllVLANs))
			resp.Truncation.Regions.Warning = &warning
		}

		peerIDs := make([]string, 0, len(l2PeerRegions))
		for id := range l2PeerRegions {
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
			regionSet := l2PeerRegions[peerID]
			regionIDs := make([]string, 0, len(regionSet))
			for regionID := range regionSet {
				regionIDs = append(regionIDs, regionID)
			}
			sort.SliceStable(regionIDs, func(i, j int) bool {
				li, errI := strconv.Atoi(regionIDs[i])
				rj, errJ := strconv.Atoi(regionIDs[j])
				switch {
				case errI == nil && errJ == nil:
					if li == rj {
						return regionIDs[i] < regionIDs[j]
					}
					return li < rj
				case errI == nil:
					return true
				case errJ == nil:
					return false
				default:
					return regionIDs[i] < regionIDs[j]
				}
			})

			n := mapNode{ID: peerID, Kind: "device", Label: l2PeerLabels[peerID], RegionIDs: regionIDs}
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

		if resp.Inspector != nil {
			peerCount := len(resp.Nodes)
			if peerCount > 0 {
				peerCount = peerCount - 1
			}
			resp.Inspector.Status = append(resp.Inspector.Status, mapInspectorField{Label: "Peers", Value: strconv.Itoa(peerCount)})

			for _, vlanID := range l2VLANs {
				resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
					Label:     "Open VLAN " + vlanID,
					Layer:     "l2",
					FocusType: "vlan",
					FocusID:   vlanID,
				})
			}

			for _, peerID := range peerIDsIncluded {
				label := peerID
				if peerLabel := l2PeerLabels[peerID]; peerLabel != nil {
					if trimmed := strings.TrimSpace(*peerLabel); trimmed != "" {
						label = trimmed
					}
				}

				resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
					Label:     "Open device " + label,
					Layer:     "l2",
					FocusType: "device",
					FocusID:   peerID,
				})
			}
		}

		if len(l2AllVLANs) == 0 {
			guidance := "No VLAN regions derived (no PVID facts). Run discovery with VLAN enrichment to render an L2 projection."
			resp.Guidance = &guidance
		} else if resp.Truncation.Regions.Truncated || resp.Truncation.Nodes.Truncated {
			guidance := "Projection truncated: some VLANs/nodes were capped for readability."
			resp.Guidance = &guidance
		}
	} else if layer == "l2" && focusType == "vlan" {
		resp.Regions = []mapRegion{{ID: focusID, Kind: "vlan", Label: "VLAN " + focusID}}
		resp.Truncation.Regions.Returned = len(resp.Regions)
		totalRegions := 1
		resp.Truncation.Regions.Total = &totalRegions

		if depth > 0 {
			if !h.ensureDeviceQueries(w) {
				return
			}
			ctx := r.Context()
			vlanLister, ok := h.devices.(interface {
				ListDevicesInVLAN(ctx context.Context, vlanID int32, limit int32) ([]sqlcgen.MapDevicePeer, error)
			})
			if !ok {
				h.log.Error().Msg("map vlan membership query missing")
				h.writeError(w, http.StatusInternalServerError, "internal_error", "map vlan projection not supported", nil)
				return
			}

			vlanIDInt, err := strconv.Atoi(focusID)
			if err != nil {
				h.writeError(w, http.StatusBadRequest, "invalid_id", "vlan id must be an integer between 1 and 4094", map[string]any{"id": focusID})
				return
			}

			queryLimit := int32(nodeLimit + 1)
			members, err := vlanLister.ListDevicesInVLAN(ctx, int32(vlanIDInt), queryLimit)
			if err != nil {
				h.log.Error().Err(err).Str("vlan", focusID).Msg("list vlan members failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to build l2 projection", nil)
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

				for _, node := range resp.Nodes {
					label := node.ID
					if node.Label != nil {
						if trimmed := strings.TrimSpace(*node.Label); trimmed != "" {
							label = trimmed
						}
					}

					resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
						Label:     "Open device " + label,
						Layer:     "l2",
						FocusType: "device",
						FocusID:   node.ID,
					})
				}
			}

			if len(resp.Nodes) == 0 {
				guidance := "No devices observed in this VLAN yet. Run discovery with VLAN enrichment to populate membership."
				resp.Guidance = &guidance
			} else if resp.Truncation.Nodes.Truncated {
				guidance := "Projection truncated: some devices were capped for readability."
				resp.Guidance = &guidance
			}
		}
	} else if layer == "physical" && focusType == "device" && depth > 0 {
		maxPeers := nodeLimit - 1
		if maxPeers < 0 {
			maxPeers = 0
		}
		if edgeLimit < maxPeers {
			maxPeers = edgeLimit
		}

		linksTruncated := physicalLinksTruncated
		linksIncluded := physicalLinks
		if len(linksIncluded) > maxPeers {
			linksIncluded = linksIncluded[:maxPeers]
			linksTruncated = true
		}

		peerIDs := make([]string, 0, len(linksIncluded))
		peerLabels := make(map[string]*string, len(linksIncluded))
		for _, link := range linksIncluded {
			peerIDs = append(peerIDs, link.PeerDeviceID)
			if _, exists := peerLabels[link.PeerDeviceID]; !exists {
				peerLabels[link.PeerDeviceID] = link.PeerDisplayName
			}
		}
		sort.Strings(peerIDs)

		nodesTruncated := linksTruncated
		peerIDsIncluded := peerIDs
		if nodesTruncated && len(peerIDsIncluded) > maxPeers {
			peerIDsIncluded = peerIDsIncluded[:maxPeers]
		}

		resp.Nodes = make([]mapNode, 0, 1+len(peerIDsIncluded))
		if focusNode != nil {
			resp.Nodes = append(resp.Nodes, *focusNode)
		}
		for _, peerID := range peerIDsIncluded {
			resp.Nodes = append(resp.Nodes, mapNode{
				ID:        peerID,
				Kind:      "device",
				Label:     peerLabels[peerID],
				RegionIDs: []string{},
			})
		}
		resp.Truncation.Nodes.Returned = len(resp.Nodes)
		resp.Truncation.Nodes.Truncated = nodesTruncated
		if nodesTruncated {
			warning := fmt.Sprintf("Node cap hit: showing %d of >%d devices.", len(resp.Nodes), nodeLimit)
			resp.Truncation.Nodes.Warning = &warning
		} else {
			totalNodes := len(resp.Nodes)
			resp.Truncation.Nodes.Total = &totalNodes
		}

		resp.Edges = make([]mapEdge, 0, len(linksIncluded))
		for _, link := range linksIncluded {
			linkType := ""
			if link.LinkType != nil {
				linkType = strings.TrimSpace(*link.LinkType)
			}
			meta := map[string]any{
				"link_key": link.LinkKey,
				"source":   link.Source,
			}
			if linkType != "" {
				meta["link_type"] = linkType
			}
			resp.Edges = append(resp.Edges, mapEdge{
				ID:   "link:" + link.LinkID,
				Kind: "link",
				From: focusID,
				To:   link.PeerDeviceID,
				Meta: meta,
			})
		}
		resp.Truncation.Edges.Returned = len(resp.Edges)
		resp.Truncation.Edges.Truncated = linksTruncated
		if linksTruncated {
			warning := fmt.Sprintf("Edge cap hit: showing %d of >%d.", len(resp.Edges), edgeLimit)
			resp.Truncation.Edges.Warning = &warning
		} else {
			totalEdges := len(resp.Edges)
			resp.Truncation.Edges.Total = &totalEdges
		}

		if resp.Inspector != nil {
			neighborCount := len(resp.Nodes)
			if neighborCount > 0 {
				neighborCount = neighborCount - 1
			}
			resp.Inspector.Status = append(resp.Inspector.Status, mapInspectorField{Label: "Neighbors", Value: strconv.Itoa(neighborCount)})

			for _, peerID := range peerIDsIncluded {
				label := peerID
				if peerLabel := peerLabels[peerID]; peerLabel != nil {
					if trimmed := strings.TrimSpace(*peerLabel); trimmed != "" {
						label = trimmed
					}
				}

				resp.Inspector.Relationships = append(resp.Inspector.Relationships, mapInspectorRelation{
					Label:     "Open device " + label,
					Layer:     "physical",
					FocusType: "device",
					FocusID:   peerID,
				})
			}
		}

		if len(linksIncluded) == 0 {
			guidance := "No physical links known yet. Add manual links or enable LLDP/CDP enrichment to render adjacency."
			resp.Guidance = &guidance
		} else if resp.Truncation.Nodes.Truncated || resp.Truncation.Edges.Truncated {
			guidance := "Projection truncated: some links/nodes were capped for readability."
			resp.Guidance = &guidance
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
