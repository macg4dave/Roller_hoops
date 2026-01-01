# Feature matrix

This matrix is the source of truth for what exists in the system. No orphan code.

Status values: `planned` | `in-progress` | `complete` | `deprecated`

| Feature | Description | Owning service | API endpoints | DB tables | Status |
|---|---|---|---|---|---|
| Core schema (interfaces/IPs/MACs/services) | Normalized tables exist for discovery facts | core-go | (TBD) | `interfaces`, `ip_addresses`, `mac_addresses`, `services` | complete |
| Devices API | CRUD for devices (headless) | core-go | `/api/v1/devices` | `devices` | complete |
| Device metadata | User-editable metadata stored in DB | core-go (store), ui-node (edit) | `/api/v1/devices/{id}` | `device_metadata` | complete |
| Discovery run | Trigger a discovery pass | core-go | `/api/v1/discovery/run` | `discovery_runs`, `discovery_run_logs` | complete |
| Discovery status | Report last run + current status | core-go | `/api/v1/discovery/status` | `discovery_runs` | complete |
| Discovery worker | Executes discovery runs (queued→running→succeeded/failed) with a bounded ICMP sweep (best-effort) + ARP scrape to upsert discovered IP/MAC facts | core-go | (uses existing discovery endpoints) | `discovery_runs`, `discovery_run_logs`, `devices`, `ip_addresses`, `mac_addresses` | complete |
| Discovery observations (IP/MAC) | Append-only IP/MAC observations per run, used later for history + diffing | core-go | (uses existing discovery endpoints) | `ip_observations`, `mac_observations` | complete |
| Device change feed | Cursorable change feed derived from observations/metadata/services; exposed via `/api/v1/devices/changes`. | core-go | `/api/v1/devices/changes` | `ip_observations`, `mac_observations`, `device_metadata`, `services` | complete |
| Device history timeline | Device-focused timeline endpoint powering history overlays. | core-go | `/api/v1/devices/{id}/history` | `ip_observations`, `mac_observations`, `device_metadata`, `services` | complete |
| Discovery runs/logs explorer | Paginated discovery run list + logs, including run details and cursor-friendly log history. | core-go | `/api/v1/discovery/runs`, `/api/v1/discovery/runs/{id}`, `/api/v1/discovery/runs/{id}/logs` | `discovery_runs`, `discovery_run_logs` | complete |
| Historical observations + diffing | Change feed + per-device timeline + run/log inspection backed by observations and existing fact tables; supports cursor paging and bounded responses. | core-go | `/api/v1/devices/changes`, `/api/v1/devices/{id}/history`, `/api/v1/discovery/runs` | `ip_observations`, `mac_observations`, `services`, `device_metadata`, `devices`, `discovery_runs`, `discovery_run_logs` | complete |
| UI device list + create | Browse devices and create new devices | ui-node | (calls Go API) | none (no DB access) | complete |
| UI discovery panel | Trigger runs + show latest status with live polling updates | ui-node | (calls Go API) | none (no DB access) | complete |
| UI metadata editing | Edit `owner`/`location`/`notes` after creation via inline form + server action | ui-node | (calls Go API) | none (no DB access) | complete |
| Authentication & sessions | Local login + roles (admin/read-only) + signed session cookie (`roller_session`) enforced before proxying to Go | ui-node | `/auth/login`, `/api/auth/login`, `/api/auth/logout`, `/auth/account` | none | complete |
| Protect `/api` | Prevent browser-direct access to Go API (UI-as-BFF) | traefik + ui-node | `/api/...` (internal only via Traefik internal entrypoint) | none | complete |
| Reverse proxy routing | `/` → UI (core-go stays private) | traefik | (N/A) | none | complete |
| Docker compose bootstrap | `docker compose up` works with health checks | (all) | (N/A) | (all) | complete |
| Health + readiness | `/healthz` and `/readyz` across services | core-go + ui-node | `/healthz`, `/readyz` | none | complete |
| Observability metrics | Prometheus scrape endpoint exposing HTTP and discovery metrics (`roller_http_*`, `roller_discovery_*`) | core-go | `/metrics` | none | complete |
| Request ID propagation | End-to-end `X-Request-ID` correlation | core-go + ui-node | (all) | none | complete |
| Strict JSON decoding | Reject unknown JSON fields | core-go | (all JSON endpoints) | none | complete |
| OpenAPI spec | Canonical API contract file | (repo) | (N/A) | none | complete |
| OpenAPI drift gate | Contract test comparing `api/openapi.yaml` to chi routes | core-go | (N/A) | none | complete |
| SNMP enrichment | Enrich devices/interfaces with SNMP sysName/sysDescr and interface facts (best-effort, opt-in) so operators see richer metadata without manual entry | core-go | (via discovery worker; no dedicated endpoint) | `device_snmp`, `interfaces`, `mac_addresses` | complete |
| VLAN / switch port mapping | Map switch interfaces to VLAN IDs (PVID via bridge/q-bridge MIB; best-effort, opt-in) | core-go | (via discovery worker; no dedicated endpoint) | `interface_vlans`, `interfaces` | complete |
| mDNS / NetBIOS resolution | Turn up friendly names via name-resolution helpers (reverse DNS, mDNS, NetBIOS, SNMP sysName candidates) and store them for selection | core-go | `/api/v1/devices/{id}/name-candidates` | `device_name_candidates`, `devices` | complete |
| Import/export JSON | Export or import the device catalog and metadata; Go exposes `/api/v1/devices/export` and `/api/v1/devices/import` while the UI offers snapshot download/upload controls. | core-go + ui-node | `/api/v1/devices/export`, `/api/v1/devices/import` | none | complete |
| LLDP/CDP adjacency enrichment | Best-effort switch neighbor discovery via SNMP LLDP-MIB and CISCO-CDP-MIB; upserts physical links for later Physical-layer projection. | core-go | (via discovery worker; no dedicated endpoint) | `links`, `interfaces` | complete |
| Service/port discovery | Optional active scan via `nmap` (XML parsing) to upsert open ports/services per device, behind explicit enable flags and allowlists. | core-go | (via discovery worker; no dedicated endpoint) | `services` | complete |
| External inventory import (NetBox/Nautobot) | Import devices from upstream inventory/IPAM exports and backfill `display_name`, metadata, and primary IPs without clobbering existing curated fields. | core-go | `/api/v1/inventory/netbox/import`, `/api/v1/inventory/nautobot/import` | `devices`, `device_metadata`, `ip_addresses` | complete |
| CI pipeline | GitHub Actions runs Go tests (with Postgres), UI build/typegen drift gate, and a docker-compose smoke test (auth-aware). | (repo) | (N/A) | none | complete |
| UI foundation (Phase 12) | Consistent app shell + small internal UI primitives (buttons/inputs/badges/cards/alerts/skeletons) + global empty/loading/error patterns | ui-node | (calls Go API) | none | complete |
| UI device list v2 (Phase 12) | Operator-grade device table: server-backed search, filters, stable sorting, cursor paging, shareable URL state, and row actions (open/copy ID/copy IP) | ui-node | (calls Go API) | none | complete |
| UI device detail v2 (Phase 12) | Device detail with facts (IPs/MACs/interfaces/services), metadata editing, and history timeline UX powered by Phase 9 endpoints | ui-node | (calls Go API) | none | complete |
| UI discovery runs explorer (Phase 12) | Discovery run list, detail, and logs views backed by Phase 9 discovery APIs plus real-time failure/status signals | ui-node | (calls Go API) | none | complete |
| UI polish & accessibility (Phase 12) | Focus/selection styles, contrast, reduced motion, and resilient polling/loading states across the UI | ui-node | (calls Go API) | none | complete |
| Network map UI shell | `/map` route with constant 3-pane layout (Layer panel / Canvas / Inspector), empty-by-default canvas, deep-linkable layer + focus in URL, and inspector-driven cross-layer navigation stubs. | ui-node | (calls Go API later) | none | complete |
| Map projection API (base) | Projection-first read endpoints returning render-ready `regions[]/nodes[]/edges[]` + `inspector` for a focused object; **no global graph** endpoints. | core-go | `/api/v1/map/{layer}` (scaffolding; starting with `/api/v1/map/l3`) | (derived from existing tables; no new tables required for L3 v1) | complete |
| Map projection: L3 (Subnets) | Subnet regions derived from IP facts; **device + subnet focus are live** (no global graphs). | core-go + ui-node | `/api/v1/map/l3` | `ip_addresses`, `devices` (+ observations later) | complete |
| Map projection: L2 (VLANs) | VLAN regions and membership based on SNMP-derived VLAN facts (start with PVID). | core-go + ui-node | `/api/v1/map/l2` (planned) | `interface_vlans`, `interfaces`, `devices` | planned |
| Map projection: Physical | Physical adjacency projection based on curated/manual links initially, with future LLDP/CDP enrichment possible. | core-go + ui-node | `/api/v1/map/physical` (planned) | (planned) `links` | planned |
| Map projection: Services | Services view grouping by host from discovered services; optional manual dependencies as explicit edges. | core-go + ui-node | `/api/v1/map/services` (planned) | `services` (+ planned `service_dependencies`) | planned |
| Map projection: Security | Zones as regions with manual policies/flows as edges; rendered only in Security layer/mode. | core-go + ui-node | `/api/v1/map/security` (planned) | (planned) `zones`, `zone_policies` | planned |
| Map editing (Build mode) | Author curated truth (links/zones/service deps) through Go APIs; UI never touches DB directly. | core-go + ui-node | (TBD; planned write endpoints under `/api/v1/map/...` or `/api/v1/topology/...`) | (planned) `links`, `zones`, `service_dependencies` | planned |
