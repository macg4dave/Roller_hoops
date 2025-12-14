# Feature matrix

This matrix is the source of truth for what exists in the system. No orphan code.

Status values: `planned` | `in-progress` | `complete` | `deprecated`

| Feature | Description | Owning service | API endpoints | DB tables | Status |
|---|---|---|---|---|---|
| Devices API | CRUD for devices (headless) | core-go | `/api/v1/devices` | `devices` | complete |
| Device metadata | User-editable metadata stored in DB | core-go (store), ui-node (edit) | `/api/v1/devices/{id}` | `device_metadata` | in-progress |
| Discovery run | Trigger a discovery pass | core-go | `/api/v1/discovery/run` | `discovery_runs`, `discovery_run_logs` | in-progress |
| Discovery status | Report last run + current status | core-go | `/api/v1/discovery/status` | `discovery_runs` | in-progress |
| UI editing workflows | Forms for editing metadata and browsing | ui-node | (calls Go API) | none (no DB access) | in-progress |
| Authentication & sessions | Local users + session cookies in UI | ui-node | (N/A) | none | planned |
| Reverse proxy routing | `/` → UI, `/api` → Go | traefik | (N/A) | none | complete |
| Docker compose bootstrap | `docker compose up` works with health checks | (all) | (N/A) | (all) | complete |
