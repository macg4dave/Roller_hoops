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
- Keyboard-first interactions and visible focus; never rely on color alone.

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
3. **Inspector is the anchor**
   - The inspector is always visible.
   - Most navigation happens from the inspector (“View in L3”, “Open VLAN”, “Open Subnet”).
4. **URL-driven state**
   - Layer + focus are encoded in the URL (`layer`, `focusType`, `focusId`).
   - Deep links are stable and reload-safe.
5. **Deterministic presentation**
   - Stable IDs, stable ordering, stable layout.
   - Polling must not reshuffle the view while the user is interacting.

---

## Map projection shape (assumptions)

The UI should treat the map as a projection payload, not an editable source of truth.

- `regions[]`: container objects (zone/subnet/vlan/etc.)
- `nodes[]`: occupant objects (devices/services/etc.)
- optional `edges[]`: only when a layer explicitly calls for it (see “Edges policy”)
- Nodes express membership via region IDs (not via a dense edge mesh).

---

## The “nodes in nodes” model (how to keep it clean)

### 1) Containers should usually be regions

Use `regions[]` for container objects:

- L3: `subnet` regions (derived first; optional persistence later).
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

## Selection & input rules (must work without a mouse)

- Keyboard navigation must allow moving focus between regions and nodes, opening Inspector details, and drilling in/back via shortcuts.
- Clicking a region selects the region; clicking a node selects the node; selection is always visible.
- Hover is optional affordance (peek), never required to discover critical state.
- Never rely on color alone for state (changed/offline/selected); pair with icon/text.

---

## v1 must-haves (minimum product bar)

- Empty state: “Pick a layer + focus” guidance and a focus picker/search.
- Stable deep links: `layer` + `focusType` + `focusId` reload to the same view.
- Clear “Expand” vs “Drill-in” affordances (and consistent behavior).
- Deterministic layout and stable ordering (no jitter across polling).
- Hard caps with honest messaging when truncated (and a path to drill in).
- Inspector shows “also in…” memberships and provides layer navigation links.
- Accessible selection + keyboard focus (visible outline, predictable tab order).

---

## Layer → containers → occupants (default policy)

This is the default mapping for what “lives inside what”. It can evolve, but should stay simple.

| Layer | Containers (regions) | Occupants (nodes) | Notes |
| --- | --- | --- | --- |
| Security | Zones | Devices (later Services) | Zones are likely curated/manual. |
| L2 | VLANs | Devices (or Interfaces in drill-in) | Membership derived from `interface_vlans`. |
| L3 | Subnets | Devices | Membership derived from observed IPs; pick a primary per device. |
| Services | Service groups (optional) | Services | Prefer service→service edges only here. |
| Physical (later) | Sites/Racks | Devices | Physical links are the only “default edge” layer. |

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
