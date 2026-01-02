# Roller_hoops

Self-hosted network tracker / mapper (Go + Node.js + PostgreSQL), fully containerised.

## Requirements

- **Recommended (no local toolchains):** Docker + Docker Compose v2 (`docker compose ...`)
- Host port `80/tcp` available (Traefik binds `80:80`; change `docker-compose.yml` if you want a different host port)

### Installing prerequisites (Ubuntu/Debian examples)

These are “good enough to get started” commands. For production, pin versions and follow your distro’s Docker/Postgres guidance.

- Base tools:
  - `sudo apt update`
  - `sudo apt install -y git ca-certificates curl`
- Docker engine + Compose plugin:
  - `sudo apt install -y docker.io docker-compose-plugin`
  - `sudo usermod -aG docker "$USER"` (then log out/in so `docker` works without sudo)
  - `sudo systemctl enable --now docker`

### Optional (only if running services outside Docker)

- Go `1.24.x` (for `core-go/`)
- Node.js `20.x` + npm (for `ui-node/`)
- PostgreSQL `16.x` (if you don’t use the Compose `db` service)

If you want to build/test outside Docker on Ubuntu/Debian:

- Go (the repo uses `go 1.24.x`): install via your preferred version manager (asdf/gimme) or the official tarball (apt’s `golang-go` is often older).
- Node.js 20:
  - `sudo apt install -y nodejs npm` (may be older; for Node 20, use NodeSource or a version manager like nvm/asdf)
- PostgreSQL 16 client/server (optional):
  - `sudo apt install -y postgresql-client`
  - `sudo apt install -y postgresql` (if you want a local server instead of Compose)

## Quickstart (dev)

- Start the full stack: `docker compose up --build`
- Optional: copy `.env.example` to `.env` to override local settings like `POSTGRES_PASSWORD`.
- Open the UI: <http://localhost/>
- Sign in: <http://localhost/auth/login> (example users live in `.env.example` via `AUTH_USERS`)
- Default quickstart credentials: `admin` / `admin` (also configured in `.env.example`).
- The Go API is **not exposed directly**; the UI proxies `/api/...` requests to `core-go` and enforces auth/roles.

## Compose profiles

- `docker compose up --build` (default) launches Traefik, core-go, ui-node, the database, and runs migrations.
- `docker compose --profile dev up --build` runs the default stack and, once the database is healthy, executes the idempotent SQL in `docker/dev/dev-seed.sql` to populate a sample device, metadata, and related discovery rows.
- `docker compose --profile prod up --build` executes the same stack plus the `prod-readiness` service that waits for both `/healthz` and `/readyz` before exiting successfully, which can be handy for deployment smoke tests.

## Common commands

- Tail logs: `docker compose logs -f --tail=200`
- Stop: `docker compose down`
- Reset DB (dev only): `docker compose down -v`
- Re-run seed (dev profile): `docker compose --profile dev run --rm dev-seed`

## Services (responsibilities)

- `core-go` (Go): REST API + persistence + discovery worker. No HTML/UI.
- `ui-node` (Next.js): UI rendering + workflows + auth/sessions. No DB access.
- `db` (Postgres): the only database.
- `traefik`: routes `/` → UI (core-go stays private).

## Ports

- Host-exposed:
  - `80/tcp` → Traefik `web` → `ui-node:3000` (UI)
- Container/network-only (not published to the host by default):
  - `traefik:8080` (`internal`) → routes `/api` → `core-go:8081`
  - `core-go:8081` (API + `/metrics`)
  - `ui-node:3000` (Next.js server)
  - `db:5432` (Postgres)

## Configuration (.env)

Compose reads environment variables from `.env` (gitignored). Start with:

- `cp .env.example .env`

Common settings:

- `POSTGRES_PASSWORD`: password for the Compose `db` container (dev default is `postgres`)
- `AUTH_USERS`: comma-separated `username:password:role` entries (example in `.env.example`)
- `AUTH_SESSION_SECRET`: HMAC secret for the `roller_session` cookie (set a real value for production)

## Discovery requirements (network scanning)

The discovery worker can do ARP/ICMP/SNMP and optional port scanning. In Docker, discovery fidelity depends on container networking and privileges (e.g. `CAP_NET_RAW` and/or host networking on Linux). See `docs/discovery-deployment.md` before enabling scanning in production.

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

- Devices (REST, v1):

  - `GET /api/v1/devices` (search/filter/sort + cursor pagination)
  - `POST /api/v1/devices`
  - `GET /api/v1/devices/{id}`
  - `PUT /api/v1/devices/{id}`
  - `GET /api/v1/devices/{id}/name-candidates`
  - `GET /api/v1/devices/{id}/facts`
  - `GET /api/v1/devices/export`
  - `POST /api/v1/devices/import`

- History & runs:

  - `GET /api/v1/devices/changes?since=RFC3339&limit=N`
  - `GET /api/v1/devices/{id}/history?limit=N&cursor=...`
  - `POST /api/v1/discovery/run`
  - `GET /api/v1/discovery/status`
  - `GET /api/v1/discovery/runs`
  - `GET /api/v1/discovery/runs/{id}`
  - `GET /api/v1/discovery/runs/{id}/logs`

- Audit:

  - `GET /api/v1/audit/events`

- Map (read-only projection):

  - `GET /api/v1/map/{layer}?focusType=device|subnet|vlan|zone|service&focusId=...`

- Observability:

  - `GET /metrics` (Prometheus scrape target; intended for internal routing)

- External inventory import (optional):

  - `POST /api/v1/inventory/netbox/import`
  - `POST /api/v1/inventory/nautobot/import`

The canonical API contract is in `api/openapi.yaml` (`servers: /api`).

## Authentication (UI-owned)

The UI enforces authentication before proxying any `/api/...` requests to `core-go`.

- Configure users via `AUTH_USERS` (format: `username:password:role`).
- Optional: set `AUTH_USERS_FILE` to a writable path to enable password changes and admin resets via the `/auth/account` page.

## Building and running without Docker (advanced)

This is optional; the supported “it just works” path is `docker compose up --build`.

- `core-go`:
  - Requires `DATABASE_URL` (example: `postgres://postgres:postgres@localhost:5432/roller_hoops?sslmode=disable`)
  - Run: `cd core-go && go run ./cmd/core-go` (uses `HTTP_ADDR` default `:8081`)
- `ui-node`:
  - Install deps: `cd ui-node && npm ci`
  - Run: `cd ui-node && npm run dev` (serves on `http://localhost:3000`)
  - Set `CORE_GO_BASE_URL=http://localhost:8081` if you’re not running behind Traefik

## Docs

- Roadmap / phases: `docs/roadmap.md`
- Operations runbook (metrics, backups, secrets): `docs/runbooks.md`
- API conventions: `docs/api-contract.md` (canonical spec: `api/openapi.yaml`)

## UI work (Phase 12)

The operator UX is tracked in `docs/roadmap.md` (Phase 12). The UX foundation rules live in:

- `docs/ui-ux.md`
