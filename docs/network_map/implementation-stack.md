# Network map implementation stack (don’t reinvent wheels)

This doc lists **existing, proven libraries/tools** we can adopt for the Network Map work (Phases 13–16) so we don’t build bespoke graph tooling, layout engines, or type systems.

Constraints (from repo docs):

- **core-go owns truth and state** (DB + REST API); ui-node never touches Postgres.
- **OpenAPI is canonical** (`api/openapi.yaml`), and a drift gate already exists.
- Prefer **boring, deterministic** visuals (regions > edges; no spaghetti graphs).

---

## Default recommendation (v1)

If we want the fastest path to something that matches the mocks **without a dependency explosion**:

- **Renderer**: SVG (React) with pan/zoom
- **Layout**: deterministic region layout (no force graph)
- **Types**: generate TS types from OpenAPI (no hand-rolled DTO drift)
- **Validation**: Zod for UI-side parsing of API responses

You can still swap in a heavier graph library later if needed.

---

## UI canvas / graph rendering options (Node/Next)

 React Flow (recommended if we want “batteries included”)

- Project: <https://reactflow.dev/>
- What it solves:
  - node/edge rendering
  - dragging, selection, minimap, controls
  - pan/zoom
  - React ergonomics
- Good for:
  - quick iteration to a working map UI
  - interactive editing (Build mode) later
- Watch-outs:
  - it is node/edge centered; our design is **region-first** (subnet/VLAN/zone regions). We can still model regions as parent nodes (group nodes) but it requires discipline.
  - deterministic layout is still on us (or via ELK/Dagre).

Recommended pairing:

- Layout engine: **ELK** via `elkjs` (see below)

Cytoscape.js (recommended if we want mature graph features)

- Project: <https://js.cytoscape.org/>
- What it solves:
  - mature graph model + layouts
  - styling and interaction
- Good for:
  - complex graph behavior over time
  - built-in layout options
- Watch-outs:
  - less “React-native” than React Flow; you embed and bridge
  - region-first UI still requires custom layering/grouping

---

## Layout engines (deterministic, not force-graph chaos)

### ELK (Eclipse Layout Kernel) via `elkjs` (strong recommendation)

- Project: <https://www.eclipse.org/elk/>
- JS package: `elkjs`
- What it solves:
  - deterministic layouts (layered, tree, etc.)
  - works well with React Flow and generic node graphs
- Good for:
  - Physical layer “tree-ish” layouts
  - clean edge routing

### Dagre

- Project: <https://github.com/dagrejs/dagre>
- What it solves:
  - simple directed graph layout
- Good for:
  - quick, predictable layouts for smaller graphs

### D3 layout primitives (for region-first)

- `d3-hierarchy` (tree, cluster) — <https://github.com/d3/d3-hierarchy>
- `d3-force` (only if constrained) — <https://github.com/d3/d3-force>

Recommendation:

- Prefer **hierarchy** (Physical) and **simple packing/rows** (inside regions) rather than force graphs.

---

## OpenAPI tooling (avoid hand-written clients/types)

### TypeScript types from OpenAPI (recommended)

- `openapi-typescript` — <https://github.com/openapi-ts/openapi-typescript>
  - generates TS types from `api/openapi.yaml`
  - keeps UI types aligned with the contract

### Typed fetch client generation

- `orval` — <https://orval.dev/>
  - generates a full client (fetch/axios/react-query)

### Go OpenAPI codegen (optional)

If we want to generate server types/models (without changing chi routing):

- `oapi-codegen` — <https://github.com/oapi-codegen/oapi-codegen>
- `ogen` — <https://github.com/ogen-go/ogen>

Guidance:

- Given the existing chi + contract test, start by generating **types/models only** if needed, not a new server framework.

---

## Validation (don’t trust API responses implicitly)

Even with typed clients, runtime validation prevents weirdness:

- Zod — <https://github.com/colinhacks/zod>
  - validate incoming projection payloads
  - especially helpful for map projections (regions/nodes/edges)

---

## Auth/session tooling (UI-owned)

Phase 11 owns auth, but if we don’t want to build session plumbing from scratch:

- Auth.js / NextAuth — <https://authjs.dev/>
  - handles session cookies and providers
  - can be used with credentials-based auth

- `bcrypt` (Node)

---

## Go-side helpers (projection building)

### CIDR/subnet computations

- Prefer Go stdlib `net/netip` for IP parsing/manipulation.
- If we need CIDR containment indexing:
  - `cidranger` — <https://github.com/yl2chen/cidranger>

### Graph utilities (optional)

If we end up needing generic graph ops in Go (not required for v1):

- `gonum/graph` — <https://github.com/gonum/gonum/tree/master/graph>

---

## Discovery & enrichment tooling (Go) (don’t build scanners)

The map layers (especially **L2** and **Physical**) get dramatically easier if we reuse existing discovery tooling rather than writing our own probe stacks.

### LLDP/CDP adjacency (for Physical layer)

Goal: populate `links` with `source=lldp|cdp` where possible, without building a custom “topology protocol”.

Practical approach:

- Query neighbors via **SNMP**:
  - LLDP: standard **LLDP-MIB** tables (vendor-neutral)
  - CDP: **CISCO-CDP-MIB** (Cisco-heavy environments)

Implementation notes:

- Keep it best-effort and additive: a failed poll should not fail a discovery run.
- Model as observations (timestamped) and upsert a “current best” link set if desired.
- Optional: use a MIB parser to avoid hardcoding OIDs:
  - `gosmi` — <https://github.com/sleepinggenius2/gosmi>

### Port/service discovery (for Services layer)

Avoid writing our own TCP connect scanner / OS fingerprinting. If we want “what ports are open”:

- `nmap` (binary) — parse XML output into `services`
  - Go wrapper (optional): `Ullaakut/nmap` — <https://github.com/Ullaakut/nmap>
- `masscan` (binary) — very fast “is port open” sweeps (good for broad scopes; careful with ops impact)

Guidance:

- Start with **nmap** in “safe defaults” mode (small port list, timeouts capped).
- Run scanners behind an explicit enable flag + allowlist scope (see `docs/discovery-deployment.md`).

### Packet capture (optional; only if we really need it)

If we ever need passive discovery (DHCP, ARP, etc.), don’t implement pcap parsing:

- `gopacket` — <https://github.com/google/gopacket>

### mDNS / Bonjour (friendly names)

If/when we add mDNS name candidates (Phase 7+ idea), don’t hand-roll multicast DNS:

- `zeroconf` — <https://github.com/grandcat/zeroconf>

---

## Practical picks per layer (v1)

- **L3 (Subnets)**: SVG-first region renderer + simple deterministic placement inside regions.
- **L2 (VLANs)**: same renderer; regions keyed by VLAN id; membership from `interface_vlans`.
- **Physical**: ELK/Dagre hierarchy layout or a simple tree layout (d3-hierarchy) if done in UI.

---

## External sources (import instead of inventing IPAM)

If the goal is “map what exists”, many orgs already have an inventory/IPAM source of truth.
Instead of rebuilding that whole workflow, treat it as an import feed.

Good candidates:

- NetBox — <https://github.com/netbox-community/netbox>
- Nautobot — <https://github.com/nautobot/nautobot>

Integration pattern (fits this repo’s constraints):

- core-go runs a periodic sync job (or manual “import now”) and writes into our tables.
- UI stays consistent: it still talks only to core-go.

---

## UI data fetching/state (so we don’t build client plumbing)

The map is projection-driven and focus-scoped, which benefits from caching + request de-duping.

- `@tanstack/react-query` — <https://tanstack.com/query/latest>
  - caching per `(layer, focusType, focusId)`
  - background refetch (optional “Operate” overlays later)
- `openapi-fetch` — <https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch>
  - typed `fetch` client from the `openapi-typescript` types (no bespoke client glue)

---

## UI interaction primitives (don’t build these from scratch)

- Command palette / global focus:
  - `cmdk` — <https://cmdk.paco.me/>
- Fuzzy search (devices/subnets/services):
  - `Fuse.js` — <https://www.fusejs.io/>
- Tooltips/popovers (canvas hover, inspector affordances):
  - Floating UI — <https://floating-ui.com/>

---

## Decision checklist (so we pick once)

Before adding dependencies:

1.  **graph library-first** with interaction features
2. Read-only projections from data model (no graph library; SVG + deterministic layout)
3. we need layout complexity now to help updates

one coherent bundle:
- React Flow bundle: `reactflow` + `elkjs` (or `dagre`)
