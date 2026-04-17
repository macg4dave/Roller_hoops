# Runbooks

This directory-level guide collects the day-to-day operations runbook that lives alongside Phase 11’s observability and operations work.

## Observability

- `/metrics` exposes Prometheus-friendly metrics from `core-go`. The Go service publishes the following gauges/counters under the `roller` namespace:
  - `roller_http_requests_total` (by `method`, `path`, `status`)
  - `roller_http_request_duration_seconds` (same labels, `DefBuckets`)
  - `roller_discovery_runs_total`
  - `roller_discovery_run_duration_seconds`
- Use the Prometheus job config or Traefik’s internal metrics to scrape `http://core-go:8081/metrics` (or `http://localhost:8081/metrics` on the host) on your internal network. Health checks and probes should continue to hit `/healthz` and `/readyz`.
- Example: `curl -s http://localhost:8081/metrics | grep roller_http_request_duration_seconds`
- Logs already include structured request metadata and `X-Request-ID`; pair the two with the request ID envelope on the UI if you need to trace a user action into Go.

## Backups & restores

- **Backup** (recommended via Postgres container):
  - `docker compose exec -T db sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U postgres -d roller_hoops' > /tmp/roller-backup.sql`
- **Restore**:
  - `cat /tmp/roller-backup.sql | docker compose exec -T db sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -U postgres -d roller_hoops'`
  - Always stop writes (or run in maintenance window) when restoring to avoid conflicts.
  - For large dumps, stream directly from a mounted volume or object store.

## Migrations

- Apply latest migrations: `docker compose run --rm migrate`
- Check applied versions:
  - `docker compose exec -T db sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -U postgres -d roller_hoops -c \"select * from schema_migrations order by version;\"'`

## Data retention cleanup

No automatic retention job runs today. Use this manual cleanup during a
maintenance window after taking a backup.

Default retention windows:

| Table | Window |
| --- | --- |
| `ip_observations` | 90 days, preserving the latest row per `(device_id, ip)` |
| `mac_observations` | 90 days, preserving the latest row per `(device_id, mac)` |
| `discovery_run_logs` | 30 days |
| `audit_events` | 365 days |

Dry-run counts:

```sql
WITH ranked_ip AS (
  SELECT id,
         row_number() OVER (PARTITION BY device_id, ip ORDER BY observed_at DESC, id DESC) AS rn
  FROM ip_observations
),
ranked_mac AS (
  SELECT id,
         row_number() OVER (PARTITION BY device_id, mac ORDER BY observed_at DESC, id DESC) AS rn
  FROM mac_observations
)
SELECT 'ip_observations' AS table_name, count(*) AS rows_to_delete
FROM ip_observations io
JOIN ranked_ip r ON r.id = io.id
WHERE r.rn > 1 AND io.observed_at < now() - interval '90 days'
UNION ALL
SELECT 'mac_observations', count(*)
FROM mac_observations mo
JOIN ranked_mac r ON r.id = mo.id
WHERE r.rn > 1 AND mo.observed_at < now() - interval '90 days'
UNION ALL
SELECT 'discovery_run_logs', count(*)
FROM discovery_run_logs
WHERE created_at < now() - interval '30 days'
UNION ALL
SELECT 'audit_events', count(*)
FROM audit_events
WHERE created_at < now() - interval '365 days';
```

Cleanup SQL:

```sql
BEGIN;

WITH ranked_ip AS (
  SELECT id,
         row_number() OVER (PARTITION BY device_id, ip ORDER BY observed_at DESC, id DESC) AS rn
  FROM ip_observations
)
DELETE FROM ip_observations io
USING ranked_ip r
WHERE r.id = io.id
  AND r.rn > 1
  AND io.observed_at < now() - interval '90 days';

WITH ranked_mac AS (
  SELECT id,
         row_number() OVER (PARTITION BY device_id, mac ORDER BY observed_at DESC, id DESC) AS rn
  FROM mac_observations
)
DELETE FROM mac_observations mo
USING ranked_mac r
WHERE r.id = mo.id
  AND r.rn > 1
  AND mo.observed_at < now() - interval '90 days';

DELETE FROM discovery_run_logs
WHERE created_at < now() - interval '30 days';

DELETE FROM audit_events
WHERE created_at < now() - interval '365 days';

COMMIT;
```

After a large cleanup, run:

```sql
VACUUM (ANALYZE) ip_observations;
VACUUM (ANALYZE) mac_observations;
VACUUM (ANALYZE) discovery_run_logs;
VACUUM (ANALYZE) audit_events;
```

## Secrets rotation

- `AUTH_SESSION_SECRET`: rotate to invalidate all `roller_session` cookies. Update the `.env` (gitignored) or your secrets store and restart `ui-node`.
- `POSTGRES_PASSWORD`, `DATABASE_URL`, `AUTH_USERS`: treat these as injected secrets. Rotate them via your secret manager and restart (or send SIGHUP) the affected services.
- `AUTH_USERS_FILE` (optional): if in use, update the file, invalidate the cache, and warn operators about the new credentials.

## Seeds & dev fixtures

- Local dev seed (profile): `docker compose --profile dev up --build`
  - This runs the `dev-seed` service once the DB is healthy.
- Manual re-seed (dev only): `docker compose --profile dev run --rm dev-seed`
- The UI can import snapshots through `/api/v1/devices/import`.

## Discovery run checklist

1. Pick a narrow, allowed scope first. Use a single known host (`192.168.1.10/32`) or a small lab CIDR before trying a `/24`.
2. Trigger a run from the UI Discovery panel or with `POST /api/v1/discovery/run`.
3. Watch `/api/v1/discovery/status` to see the latest run.
4. Inspect logs via the Discovery run detail page or `/api/v1/discovery/runs/{id}/logs`.
5. Confirm the run reaches `succeeded` or fails with an actionable message.

### Docker bridge smoke

Use this for the normal compose deployment:

```powershell
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
docker compose --profile dev up -d --build
docker compose exec -T core-go wget -qO- --header='Content-Type: application/json' --post-data='{"scope":"<CIDR>","preset":"fast"}' http://127.0.0.1:8081/api/v1/discovery/run
docker compose exec -T core-go wget -qO- http://127.0.0.1:8081/api/v1/discovery/status
docker compose logs --tail 100 core-go
```

Expected: ping-based discovery can work, but real LAN MAC addresses are not expected. `ARP found no in-scope devices; falling back...` is normal when the bridge can ping targets but cannot see their ARP entries.

### Linux host-network smoke

Use this when ARP/MAC visibility matters on a Linux Docker host:

```powershell
$env:CORE_GO_HTTP_ADDR="127.0.0.1:8081"
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml --profile dev up -d --build
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8081/api/v1/discovery/run -ContentType 'application/json' -Body '{"scope":"<CIDR>","preset":"fast"}'
Invoke-RestMethod -Uri http://127.0.0.1:8081/api/v1/discovery/status
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml logs --tail 100 core-go
```

Expected: logs include `scope targets`, `ping sweep`, `arp scrape`, and `discovery run completed`. Keep `CORE_GO_HTTP_ADDR=127.0.0.1:8081` so the unauthenticated Go API is not exposed off-host.

### Native-host smoke

Use this when running `core-go` outside Docker:

```powershell
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml up -d db migrate
Set-Location core-go
$env:DATABASE_URL="postgres://postgres:postgres@127.0.0.1:15432/roller_hoops?sslmode=disable"
$env:HTTP_ADDR="127.0.0.1:8081"
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
go run ./cmd/core-go
```

Then trigger a run from a second shell:

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8081/api/v1/discovery/run -ContentType 'application/json' -Body '{"scope":"<CIDR>","preset":"fast"}'
Invoke-RestMethod -Uri http://127.0.0.1:8081/api/v1/discovery/status
```

Expected: host routing and host ARP behavior apply. If logs say `ping not permitted (missing CAP_NET_RAW?)` or `ping not found in PATH`, fix the host runtime before changing application code.

### Discovery failure triage

- `invalid discovery scope`: use CIDR or a single IP only.
- `scope too large`: reduce the smoke scope or intentionally adjust `DISCOVERY_MAX_TARGETS`.
- `ping not found in PATH`: install/provide `ping` in the runtime.
- `ping not permitted (missing CAP_NET_RAW?)`: grant raw-socket permission or switch deployment mode.
- `ping sweep: attempted=... succeeded=0`: check routing, firewall policy, target liveness, and ICMP filtering.
- `arp scrape: entries=0`: generate traffic to known targets, or use host/native deployment if MAC visibility is required.
- `ARP found no in-scope devices; falling back...`: Docker bridge/NAT behavior; expect IP-only devices.
- `enrichment: targets=... snmp_ok=0`: check `DISCOVERY_SNMP_ENABLED`, credentials, UDP/161, target SNMP support, and allowlists.
- `port scan skipped: ...`: check `DISCOVERY_PORT_SCAN_ENABLED`, `DISCOVERY_PORT_SCAN_ALLOWLIST`, target eligibility, and `nmap` availability.

## Post-deploy sanity

- Ensure `/metrics` returns `200 OK` and Prom metrics scrape successfully.
- Confirm `docker compose logs --tail 50 core-go` show structured logs with request IDs.
- Run the Docker-backed validation checks before shipping to keep the contract gate healthy:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`

## Docker-only developer workflow

- Shared VS Code validation tasks no longer require local Go or Node.js installs.
- The compose stack no longer relies on host bind mounts for Traefik config, migrations, or dev seed SQL, which makes Windows mapped-drive usage more reliable.
- To refresh generated UI API types without a local Node install, use the `ui: gen openapi types` task.

## Monitoring / SLO stubs

Minimum checks (start here before adding heavier tooling):

- **Uptime**: probe `ui-node` `GET /healthz` and `core-go` `GET /readyz` every 30–60s.
- **Latency**: alert if `roller_http_request_duration_seconds` p95 grows beyond your local baseline.
- **Discovery health**: alert if discovery runs fail repeatedly (watch `roller_discovery_runs_total` growth + `discovery_runs.last_error` via logs/API).

Example ad-hoc check (no dependencies):

- `curl -fsS http://localhost/healthz && curl -fsS http://localhost:8081/readyz`

Script stub:

- `UI_URL=http://localhost/healthz CORE_READY_URL=http://localhost:8081/readyz ./docker/ops/uptime-check.sh`
