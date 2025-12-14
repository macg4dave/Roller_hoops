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
| Historical observations + diffing | Append-only observations + derived “current state” | core-go | (TBD) | `*_observations` (TBD), `devices`, `interfaces`, `ip_addresses`, `services`, `mac_addresses` | planned |
| UI device list + create | Browse devices and create new devices | ui-node | (calls Go API) | none (no DB access) | complete |
| UI discovery panel | Trigger runs + show latest status with live polling updates | ui-node | (calls Go API) | none (no DB access) | complete |
| UI metadata editing | Edit `owner`/`location`/`notes` after creation via inline form + server action | ui-node | (calls Go API) | none (no DB access) | complete |
| Authentication & sessions | Local users + session cookies in UI | ui-node | (N/A) | none | planned |
| Protect `/api` | Prevent browser-direct access to Go API (UI-as-BFF) | traefik + ui-node | `/api/...` (internal only via Traefik internal entrypoint) | none | complete |
| Reverse proxy routing | `/` → UI (core-go stays private) | traefik | (N/A) | none | complete |
| Docker compose bootstrap | `docker compose up` works with health checks | (all) | (N/A) | (all) | complete |
| Health + readiness | `/healthz` and `/readyz` across services | core-go + ui-node | `/healthz`, `/readyz` | none | complete |
| Request ID propagation | End-to-end `X-Request-ID` correlation | core-go + ui-node | (all) | none | complete |
| Strict JSON decoding | Reject unknown JSON fields | core-go | (all JSON endpoints) | none | complete |
| OpenAPI spec | Canonical API contract file | (repo) | (N/A) | none | complete |
| OpenAPI drift gate | Contract test comparing `api/openapi.yaml` to chi routes | core-go | (N/A) | none | complete |
| SNMP enrichment | Enrich devices/interfaces with SNMP sysName/sysDescr and interface facts (best-effort, opt-in) so operators see richer metadata without manual entry | core-go | (via discovery worker; no dedicated endpoint) | `device_snmp`, `interfaces`, `mac_addresses` | complete |
| VLAN / switch port mapping | Map switch interfaces to VLAN IDs (PVID via bridge/q-bridge MIB; best-effort, opt-in) | core-go | (via discovery worker; no dedicated endpoint) | `interface_vlans`, `interfaces` | complete |
| mDNS / NetBIOS resolution | Turn up friendly names via name-resolution helpers (reverse DNS today; mDNS/NetBIOS future) and store candidates for selection | core-go | `/api/v1/devices/{id}/name-candidates` | `device_name_candidates`, `devices` | complete |
| Import/export JSON | Export or import the device catalog and metadata; Go exposes `/api/v1/devices/export` and `/api/v1/devices/import` while the UI offers snapshot download/upload controls. | core-go + ui-node | `/api/v1/devices/export`, `/api/v1/devices/import` | none | complete |
| CI pipeline | Automated tests + drift checks in CI | (repo) | (N/A) | none | planned |
| Network map UI shell | `/map` route with constant 3-pane layout (Layer panel / Canvas / Inspector), empty-by-default canvas, deep-linkable layer + focus in URL, and inspector-driven cross-layer navigation stubs. | ui-node | (calls Go API later) | none | planned |
| Map projection API (base) | Projection-first read endpoints returning render-ready `regions[]/nodes[]/edges[]` + `inspector` for a focused object; **no global graph** endpoints. | core-go | `/api/v1/map/{layer}` (planned), starting with `/api/v1/map/l3` | (derived from existing tables; no new tables required for L3 v1) | planned |
| Map projection: L3 (Subnets) | Subnet regions derived from IP facts; device membership + minimal relationships for a focused device/subnet. | core-go + ui-node | `/api/v1/map/l3` (planned) | `ip_addresses`, `devices` (+ observations later) | planned |
| Map projection: L2 (VLANs) | VLAN regions and membership based on SNMP-derived VLAN facts (start with PVID). | core-go + ui-node | `/api/v1/map/l2` (planned) | `interface_vlans`, `interfaces`, `devices` | planned |
| Map projection: Physical | Physical adjacency projection based on curated/manual links initially, with future LLDP/CDP enrichment possible. | core-go + ui-node | `/api/v1/map/physical` (planned) | (planned) `links` | planned |
| Map projection: Services | Services view grouping by host from discovered services; optional manual dependencies as explicit edges. | core-go + ui-node | `/api/v1/map/services` (planned) | `services` (+ planned `service_dependencies`) | planned |
| Map projection: Security | Zones as regions with manual policies/flows as edges; rendered only in Security layer/mode. | core-go + ui-node | `/api/v1/map/security` (planned) | (planned) `zones`, `zone_policies` | planned |
| Map editing (Build mode) | Author curated truth (links/zones/service deps) through Go APIs; UI never touches DB directly. | core-go + ui-node | (TBD; planned write endpoints under `/api/v1/map/...` or `/api/v1/topology/...`) | (planned) `links`, `zones`, `service_dependencies` | planned |
