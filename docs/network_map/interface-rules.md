# Network map interface & containment rules (nodes-in-nodes)

This document is a **pick-list + contract** for UI rules and interaction patterns for the “nodes in nodes” approach (containers + occupants). Keep what works, delete what doesn’t.

## Goals

- Keep the canvas readable at a glance (no spaghetti).
- Make navigation object-first and reload-safe (URL-driven).
- Preserve determinism across refresh/polling (minimal churn).

## Non-goals (v1)

- A “draw the entire network” topology view.
- Force-graph layouts as the default.
- Representing every possible multi-membership edge case on-canvas.

## Basic diagram floor

The map must still satisfy the ordinary operator expectation of a basic network
diagram. The layered/focused model is a guardrail against clutter, not a reason
to avoid simple topology.

Minimum useful device-focused diagram:

```text
[ Device 1 ] -------- [ Router 1 ]
  IP: 192.168.1.42     IP: 192.168.1.1
  MAC: aa:bb:...       MAC: 11:22:...
```

Rules:

- A selected device should be able to render with nearby connected devices such
  as its router, switch, gateway, or explicitly linked peer when that data is
  known or can be safely inferred.
- Device nodes should expose the basic facts operators expect: display name,
  primary IP, primary MAC, and role/tag when available.
- Links should be visible as links in Physical and focused device diagrams.
  Region membership remains preferred for subnet/VLAN containment, but it must
  not hide a simple "this connects to that" relationship.
- Relationship source/confidence must be visible when relevant:
  `manual`, `lldp`, `cdp`, `gateway-inferred`, `same-subnet`, or `unknown`.
- Inferred relationships must not be presented as certain physical cabling.
- The diagram stays focused and capped; expanding nearby nodes is intentional
  and bounded.

### Implementation status

- **Physical layer device focus** renders the basic diagram: focus device +
  linked peers, each showing display name, primary IP, and primary MAC when
  known.
- The Go API enriches `MapNode.meta` with `primary_ip` and `primary_mac` for
  all device nodes in the physical projection via a batch facts query
  (`ListDevicePrimaryFacts`).
- Link source (e.g., `lldp`, `manual`, `cdp`) is displayed as a prominent
  badge on each link row, separate from link type and link key metadata.
- The batch facts query is a soft dependency: if the query method is not
  available (e.g., in tests without that mock), the diagram renders without
  facts rather than failing.

## Related docs

- `docs/network_map/network_map_ideas.md` (overall product concept)
- `docs/network_map/implementation-stack.md` (UI/layout tooling options)
- `docs/data-model.md` (DB constraints + planned map entities)

## Principles (ideas to keep / cut)

- Regions are **container objects**; nodes are **occupant objects**; “lives inside” is region membership (not edges).
- The canvas renders **one layer at a time** and **one focus at a time** (object-first; empty-by-default).
- Limit visible containment depth (e.g., **max 2 levels** on the canvas); deeper nesting is Inspector drill-down.
- Use **progressive disclosure**: containers summarize by default; expand only when asked.
- Enforce **caps + auto-collapse** (e.g., >25 occupants collapses into a summary tile).
- Prefer **drill-in navigation** over zooming out to “see everything”; provide breadcrumbs.
- Multi-membership without duplication: pick a **primary placement per layer**, show other memberships in Inspector.
- Semantic zoom changes meaning (containers-only → key nodes → full occupants), not just scale.
- Edges are **rare and intentional**; membership is spatial; avoid spaghetti.
- “Go to object” (search/focus picker) is the primary entry point; canvas is not a scavenger hunt.
- Container-level status: show counts (“changed”, “online/offline”, “last discovery”) without expanding.
- Pinned focus keeps the view stable; hover/peek reveals details without expanding.
- Deterministic output: stable IDs, stable ordering, stable layout, minimal churn during polling.
- Visible focus/selection; never rely on color alone.

---

## Terms (shared vocabulary)

- **Layer**: mutually exclusive lens (physical / l2 / l3 / services / security).
- **Focus**: the single object the projection is built around; absence of focus is valid and yields an empty canvas.
- **Container**: an object that can hold occupants (Subnet, VLAN, Zone; later Rack/Site/Cluster).
- **Occupant**: an object that can live inside containers (Device, Interface, Service; later VM/Endpoint).
- **Membership**: “occupant lives in container” relationship (the default way to show structure).

---

## Core interaction contract (non-negotiables)

1. **Empty-by-default**
   - No focus ⇒ empty canvas + guidance.
2. **Object-first**
   - Selecting a layer does not draw “the whole network”.
   - Selecting a focus draws only the focused scope.
   - For a device focus, the focused scope should include the selected device
     and its most useful immediate relationship(s), such as router/switch/peer,
     when those relationships are available.
3. **Inspector is the anchor**
   - The inspector is always visible.
   - Most navigation happens from the inspector (“View in L3”, “Open VLAN”, “Open Subnet”).
4. **URL-driven state**
   - Layer + focus + mode are encoded in the URL (`layer`, `focusType`, `focusId`, `mode`).
   - Deep links are stable and reload-safe.
   - `focusType=subnet` uses a CIDR string focus id (canonical form, e.g. `10.0.1.0/24`).
5. **Deterministic presentation**
   - Stable IDs, stable ordering, stable layout.
   - Polling must not reshuffle the view while the user is interacting.
6. **Mode-driven actions**
   - Mode controls which actions and overlays are available on the canvas.
   - Modes: `explore` (default), `build`, `secure`, `operate`.
   - Invalid or missing mode resolves to `explore`.
   - Mode persists across layer and focus changes.
   - Build mode does not expose write actions until Build-mode APIs exist.
   - See `docs/ui-ux.md` § "Map modes" for full semantics.

---

## Map projection shape (assumptions)

The UI should treat the map as a projection payload, not an editable source of truth.

- `regions[]`: container objects (zone/subnet/vlan/etc.)
- `nodes[]`: occupant objects (devices/services/etc.)
- optional `edges[]`: only when a layer explicitly calls for it (see “Edges policy”)
- `inspector`: render-ready identity/status/relationships for the focused object (UI renders from this block without extra fetches in v1).
- Nodes express membership via region IDs (not via a dense edge mesh).

---

## The “nodes in nodes” model (how to keep it clean)

### 1) Containers should usually be regions

Use `regions[]` for container objects:

- L3: `subnet` regions (derived first; optional persistence later; until we store prefix lengths, default to `/24` for IPv4 and `/64` for IPv6 when deriving CIDRs).
- L2: `vlan` regions (derived from `interface_vlans` first; optional VLAN metadata later).
- Security: `zone` regions (likely curated/manual).
- Physical (later): `rack`/`site` regions (curated/manual).

Place `nodes[]` inside regions to express membership.

### 2) Devices remain devices (server/router/switch are “kinds”)

Treat “server”, “router”, “switch”, “firewall” as a **device kind/role**:

- A single `device` object type stays canonical for inventory/history/metadata.
- The map uses kind/role to pick iconography, labels, and filters.

Avoid separate “router table” / “server table” unless there is a hard lifecycle difference (usually there isn’t).

### 3) Bounded nesting depth

Render at most **two containment levels** on the canvas, for example:

- `Zone → Subnet → Devices`
- `VLAN → Devices`
- `Subnet → Devices → Services` (if services are shown as children)

Anything deeper is shown by:

- collapsing a container into a summary tile, and/or
- a drill-in action (new focus) rather than drawing deeper nesting.

---

## Progressive disclosure patterns (recommended)

### Container summary tiles (default state)

When a container is present, show:

- title (e.g., `10.0.1.0/24`, `VLAN 20`, `Zone: Prod`)
- occupant count (total + key subsets)
- status rollup (changed/offline counts, last discovery time)

Do not show all occupants until expanded or drilled into.

### Expand vs drill-in (two different tools)

- **Expand**: reveals a limited set of occupants “in place” (still within the current focus scope).
- **Drill-in**: changes focus to the container itself (new URL) and redraws a bigger view of that container.

Rule of thumb:

- Expand is for “peek”.
- Drill-in is for “work”.

### Auto-collapse thresholds (tuning knob)

Pick defaults like:

- `maxRegionsVisible`: 8
- `maxOccupantsPerRegionExpanded`: 25
- `maxNodesTotal`: 120
- `maxEdgesTotal`: 80

When caps are hit:

- collapse to summaries
- show “Showing 25 of 140 devices” with a drill-in suggestion

### Pin + peek

- **Pin focus**: prevents the canvas from reflowing due to polling updates; updates flow into the Inspector first.
- **Peek** (hover/focus): shows a small card with identity + key facts; never triggers a re-render.

---

## Multi-membership without chaos

Reality: a device can belong to multiple containers (multiple VLANs, multiple subnets over time, multiple zones if misconfigured).

Rules to keep visuals clean:

- Pick one **primary container placement per layer** (deterministic).
  - Example L3: choose the most recently observed IP’s subnet, or smallest subnet, or a stable sort rule.
- Show additional memberships in the Inspector as chips/links:
  - “Also in: VLAN 20, VLAN 30”
  - “Also has IPs in: 10.0.2.0/24”
- If a device truly must appear twice (rare), do it only via a user action (“show duplicates”), not by default.

---

## Edges policy (avoid spaghetti)

Default stance: **membership is spatial**, edges are optional.

Allowed edges (examples):

- router ↔ subnet (gateway relationship, if known)
- firewall ↔ zone boundary (policy context)
- service → service dependency (Services layer only)
- physical link edges (Physical layer only; still bounded)
- focused device ↔ router/switch/peer where the relationship is explicit or
  clearly marked as inferred

Disallowed by default:

- “connect every device in a subnet to every other device”
- generic L2/L3 mesh edges

If edges exist:

- hard cap them
- route them cleanly
- show labels on hover only

---

## Semantic zoom (meaning, not pixels)

Zoom levels can change what is rendered:

- Zoomed out: containers only (no occupant nodes), with counts.
- Mid: containers + “top N” occupants (e.g., gateways, changed devices).
- Zoomed in: full occupant nodes + optional edges for that layer.

This preserves clarity while still letting operators explore.

## Layout rules (deterministic, low-churn)

- Deterministic ordering:
  - stable sort keys for regions/nodes
  - stable IDs for all objects
- Deterministic placement:
  - region order consistent across refresh
  - node placement inside a region stable unless membership changes
- Polling stability:
  - do not reorder regions/nodes while the user is interacting
  - update “last updated” timestamps and Inspector first; reflow only when needed

---

## Selection & input rules

- Clicking a region selects the region; clicking a node selects the node; selection is always visible.
- Hover is optional affordance (peek), never required to discover critical state.
- Never rely on color alone for state (changed/offline/selected); pair with icon/text.

---

## v1 must-haves (minimum product bar)

- Empty state: “Pick a layer + focus” guidance and a focus picker/search.
- Basic focused network diagram: selecting a device can show that device linked
  to its router/switch/peer with IP and MAC facts visible when known.
- Stable deep links: `layer` + `focusType` + `focusId` reload to the same view.
- Clear “Expand” vs “Drill-in” affordances (and consistent behavior).
- Deterministic layout and stable ordering (no jitter across polling).
- Hard caps with honest messaging when truncated (and a path to drill in).
- Inspector shows “also in…” memberships and provides layer navigation links.
- Accessible selection (visible outline) and predictable interactions.

---

## Layer → containers → occupants (default policy)

This is the default mapping for what “lives inside what”. It can evolve, but should stay simple.

| Layer | Containers (regions) | Occupants (nodes) | Notes |
| --- | --- | --- | --- |
| Security | Zones | Devices (later Services) | Zones are manual-first. v1 has no inter-zone edges. |
| L2 | VLANs | Devices (or Interfaces in drill-in) | Membership derived from `interface_vlans`. |
| L3 | Subnets | Devices | Membership derived from observed IPs; pick a primary per device. |
| Services | Service groups (optional) | Services | Prefer service→service edges only here. |
| Physical (later) | Sites/Racks | Devices | Physical links are the only “default edge” layer. |

---

## Security layer contract (v1)

The Security layer renders manual zones as regions with devices as occupants.

### Model decisions (resolved by T004)

- **Manual zones only** in v1. No auto-derived zones from tags, subnets, or
  discovery data. Zones are operator-curated truth.
- **`zones` + `device_zones`** tables store zone definitions and membership.
  Schema defined in `docs/data-model.md`.
- **No inter-zone edges** in v1. No `zone_policies` table. Future
  enhancement when the zone model is validated.
- **Device tags remain separate** from zones. Tags (`device_tags` table) are
  for classification/role; zones are for security grouping. They are orthogonal.
- **Write endpoints** live under `/api/v1/topology/zones`, not under
  `/api/v1/map/`. The map projection is read-only.

### Projection rules

- Zone focus (`focusType=zone`, `focusId={zone_uuid}`): render the zone as a
  single region with member devices as occupant nodes. Inspector shows zone
  name, description, member count.
- Device focus (`focusType=device`, `focusId={device_uuid}`): show which
  zones the device belongs to as regions. Inspector includes zone memberships
  in cross-layer navigation.
- No focus: return guidance message suggesting a zone or device focus.
- Empty zones are valid and render as empty regions.
- Multi-zone devices appear in each zone region they belong to.
- Caps: same `regionLimit`/`nodeLimit` rules as other layers.

### Build-mode requirements

- Zone CRUD and membership management go through `/api/v1/topology/zones`
  endpoints (see `docs/api-contract.md`).
- Until auth/roles are implemented, zone writes are open but audit-logged
  via the existing `audit_events` table.
- The UI should distinguish Build mode from Explore mode in the zone view
  (show add/edit/remove controls only in Build mode).

---

## Gaps / decisions to make (capture before implementation)

1. **Primary membership rules (per layer)**:
   - L3: smallest-subnet vs most-recently-seen vs “most-stable over time” tie-breaker.
   - L2: per-interface VLANs imply multiple memberships; define when we show “device in VLAN” vs “interface in VLAN”.
2. **Derived vs curated membership**:
   - When (if ever) can a user override derived membership on the map?
3. **Selection model**:
   - Do we need multi-select (shift-click) for bulk actions in Build mode, or is v1 single-select only?
4. **Region identity**:
   - How we generate stable IDs for derived containers (e.g., subnet keying, VLAN keying) so URLs remain stable.
5. **Update semantics**:
   - How to present “data changed” without reflowing the canvas (badge + “apply updates” action vs live update).

---

## Notes (useful reminders)

- Prefer drill-in navigation (breadcrumbs + back/forward) over zooming out to “see everything”.
- Gateways/routers are still `device` nodes; treat “gateway” as a kind/role to highlight.
