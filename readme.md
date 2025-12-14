# Roller_hoops

Self-hosted network tracker / mapper (Go + Node.js + PostgreSQL), fully containerised.

## Quickstart (dev)

- Start the full stack: `docker compose up --build`
- Open the UI: <http://localhost/>
- API (via Traefik): <http://localhost/api/v1/devices>

## Services (responsibilities)

- `core-go` (Go): REST API + persistence (and later discovery). No HTML/UI.
- `ui-node` (Next.js): UI rendering + workflows (and later auth/sessions). No DB access.
- `db` (Postgres): the only database.
- `traefik`: routes `/` → UI and `/api` → Go.

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

- Device CRUD API (minimal):

  - `GET /api/v1/devices`
  - `GET /api/v1/devices/{id}`
  - `POST /api/v1/devices`
  - `PUT /api/v1/devices/{id}`

- Device metadata:

  - Optional `owner`, `location`, `notes` persisted in `device_metadata`
  - Available on device responses; UI create form captures metadata

- Discovery scaffolding:

  - `POST /api/v1/discovery/run` returns a real run id and persists into `discovery_runs` + logs
  - `GET /api/v1/discovery/status` surfaces the latest run status (UI shows it and can trigger runs)

The canonical API contract is in `api/openapi.yaml`.
