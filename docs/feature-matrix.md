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
| Discovery worker | Executes discovery runs (currently stub scanning w/ discovery worker loop that transitions queued→running→succeeded/failed) | core-go | (uses existing discovery endpoints) | (TBD: observations/current-state tables) | in-progress |
| Historical observations + diffing | Append-only observations + derived “current state” | core-go | (TBD) | `*_observations` (TBD), `devices`, `interfaces`, `ip_addresses`, `services`, `mac_addresses` | planned |
| UI device list + create | Browse devices and create new devices | ui-node | (calls Go API) | none (no DB access) | complete |
| UI discovery panel | Trigger runs + show latest status | ui-node | (calls Go API) | none (no DB access) | in-progress |
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
| CI pipeline | Automated tests + drift checks in CI | (repo) | (N/A) | none | planned |
