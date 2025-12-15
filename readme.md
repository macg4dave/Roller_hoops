# Roller_hoops

Self-hosted network tracker / mapper (Go + Node.js + PostgreSQL), fully containerised.

## Quickstart (dev)

- Start the full stack: `docker compose up --build`
- Optional: copy `.env.example` to `.env` to override local settings like `POSTGRES_PASSWORD`.
- Open the UI: <http://localhost/>
- Sign in: <http://localhost/auth/login> (example users live in `.env.example` via `AUTH_USERS`)
- Default quickstart credentials: `admin` / `admin` (also configured in `.env.example`).
- The Go API is **not exposed directly to browsers**; ui-node calls it over the internal Docker network (via Traefik’s internal-only entrypoint).

## Compose profiles

- `docker compose up --build` (default) launches Traefik, core-go, ui-node, the database, and runs migrations.
- `docker compose --profile dev up --build` runs the default stack and, once the database is healthy, executes the idempotent SQL in `docker/dev/dev-seed.sql` to populate a sample device, metadata, and related discovery rows.
- `docker compose --profile prod up --build` executes the same stack plus the `prod-readiness` service that waits for both `/healthz` and `/readyz` before exiting successfully, which can be handy for deployment smoke tests.

## Services (responsibilities)

- `core-go` (Go): REST API + persistence + discovery worker. No HTML/UI.
- `ui-node` (Next.js): UI rendering + workflows + auth/sessions. No DB access.
- `db` (Postgres): the only database.
- `traefik`: routes `/` → UI (core-go stays private).

## Health checks

- Go (core-go):

  - `GET /healthz` (liveness)
  - `GET /readyz` (readiness, checks DB)

- UI (ui-node):

  - `GET /healthz`

## Migrations

Migrations are applied automatically by the `migrate` service when you run `docker compose up`.

Migration sources live in:

- `core-go/migrations/`
- See `docs/migrations.md` for manual steps and how to add new files.

## Request IDs

The system propagates `X-Request-ID` end-to-end (UI → API). If a request id is not provided upstream, the UI generates one for outbound API calls.

## What’s implemented right now

- Devices:

  - `GET /api/v1/devices`
  - `GET /api/v1/devices/{id}`
  - `POST /api/v1/devices`
  - `PUT /api/v1/devices/{id}`

- History and run inspection:

  - `GET /api/v1/devices/changes`
  - `GET /api/v1/devices/{id}/history`
  - `GET /api/v1/discovery/runs`
  - `GET /api/v1/discovery/runs/{id}`
  - `GET /api/v1/discovery/runs/{id}/logs`

- Device metadata:

  - Optional `owner`, `location`, `notes` persisted in `device_metadata`
  - Available on device responses; UI create form captures metadata

- Discovery scaffolding:

  - `POST /api/v1/discovery/run` returns a real run id and persists into `discovery_runs` + logs
  - `GET /api/v1/discovery/status` surfaces the latest run status (UI shows it and can trigger runs)

- Observability:

  - `GET /metrics` (Prometheus scrape target; intended for internal routing)

- External inventory import (optional):

  - `POST /api/v1/inventory/netbox/import`
  - `POST /api/v1/inventory/nautobot/import`

The canonical API contract is in `api/openapi.yaml`.

## Authentication (UI-owned)

The UI enforces authentication before proxying any `/api/...` requests to `core-go`.

- Configure users via `AUTH_USERS` (format: `username:password:role`).
- Optional: set `AUTH_USERS_FILE` to a writable path to enable password changes and admin resets via the `/auth/account` page.

## UI work (Phase 12)

The operator UX is tracked in `docs/roadmap.md` (Phase 12). The UX foundation rules live in:

- `docs/ui-ux.md`
