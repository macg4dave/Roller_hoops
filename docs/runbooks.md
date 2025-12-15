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

1. Trigger a run with `POST /api/v1/discovery/run` (UI discovery panel or `curl`).
2. Watch `/api/v1/discovery/status` to see the latest run.
3. Inspect logs via `/api/v1/discovery/runs/{id}` and `/api/v1/discovery/runs/{id}/logs` for any errors.
4. If a run fails repeatedly, look at `/var/log` inside the worker container for network issues (ARP, ping, SNMP).

## Post-deploy sanity

- Ensure `/metrics` returns `200 OK` and Prom metrics scrape successfully.
- Confirm `docker compose logs --tail 50 core-go` show structured logs with request IDs.
- Run `go test ./...` locally before shipping to keep the contract gate healthy.

## Monitoring / SLO stubs

Minimum checks (start here before adding heavier tooling):

- **Uptime**: probe `ui-node` `GET /healthz` and `core-go` `GET /readyz` every 30–60s.
- **Latency**: alert if `roller_http_request_duration_seconds` p95 grows beyond your local baseline.
- **Discovery health**: alert if discovery runs fail repeatedly (watch `roller_discovery_runs_total` growth + `discovery_runs.last_error` via logs/API).

Example ad-hoc check (no dependencies):

- `curl -fsS http://localhost/healthz && curl -fsS http://localhost:8081/readyz`

Script stub:

- `UI_URL=http://localhost/healthz CORE_READY_URL=http://localhost:8081/readyz ./docker/ops/uptime-check.sh`
