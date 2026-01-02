# Roller_hoops

Self-hosted network tracker / mapper (Go + Node.js + PostgreSQL), fully containerised.

## Requirements

- **Recommended (no local toolchains):** Docker + Docker Compose v2 (`docker compose ...`)
- **Also supported:** run the stack locally without Docker (see [Running locally (no Docker)](#running-locally-no-docker))
- Host port `80/tcp` available (Traefik binds `80:80`; change `docker-compose.yml` if you want a different host port)

### Installing prerequisites (Debian/Ubuntu)

These are “good enough to get started” commands. For production, pin versions and follow your distro’s guidance.

- Base tools:
  - `sudo apt update`
  - `sudo apt install -y git ca-certificates curl`

- Docker engine + Compose plugin:
  - Option A (simplest; distro packages, versions may lag):
    - `sudo apt install -y docker.io docker-compose-plugin`
  - Option B (Docker CE from Docker’s repo):
    - `sudo apt install -y ca-certificates curl gnupg`
    - `sudo install -m 0755 -d /etc/apt/keyrings`
    - `curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg`
    - `echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null`
    - `sudo apt update`
    - `sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

- Post-install:
  - `sudo systemctl enable --now docker`
  - `sudo usermod -aG docker "$USER"` (then log out/in so `docker` works without sudo)

### Installing prerequisites (Fedora)

- Base tools:
  - `sudo dnf install -y git ca-certificates curl`

- Docker engine + Compose plugin:
  - Option A (distro packages, if available):
    - `sudo dnf install -y docker docker-compose-plugin`
  - Option B (Docker CE from Docker’s repo):
    - `sudo dnf install -y dnf-plugins-core`
    - `sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo`
    - `sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

- Post-install:
  - `sudo systemctl enable --now docker`
  - `sudo usermod -aG docker "$USER"` (then log out/in so `docker` works without sudo)

### Optional (only if running services outside Docker)

- Go `1.24.x` (for `core-go/`)
- Node.js `20.x` + npm (for `ui-node/`)
- PostgreSQL `15+` (Compose uses `postgres:16-alpine`) — only if you don’t use the Compose `db` service

If you want to build/test outside Docker on Ubuntu/Debian:

- Go (the repo uses `go 1.24.x`): install via your preferred version manager (asdf/gimme) or the official tarball (apt’s `golang-go` is often older).
- Node.js 20:
  - `sudo apt install -y nodejs npm` (may be older; for Node 20, use NodeSource or a version manager like nvm/asdf)
- PostgreSQL client/server (optional):
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
  - `traefik:8080` (`internal`) → routes `/api` → `core-go:8081` (used by the UI proxy; not exposed on the host)
  - `core-go:8081` (API + `/metrics`)
  - `ui-node:3000` (Next.js server)
  - `db:5432` (Postgres)

## Configuration (.env)

Compose reads environment variables from `.env` (gitignored). Start with:

- `cp .env.example .env`

For local (no Docker) runs, Next.js reads `ui-node/.env.local` automatically; `core-go` reads env vars from your shell (it does not auto-load `.env`).

Common settings:

- `POSTGRES_PASSWORD`: password for the Compose `db` container (dev default is `postgres`)
- `AUTH_USERS`: comma-separated `username:password:role` entries (example in `.env.example`)
- `AUTH_SESSION_SECRET`: HMAC secret for the `roller_session` cookie (set a real value for production)

## Discovery requirements (network scanning)

The discovery worker can do ARP/ICMP/SNMP and optional port scanning. In Docker, discovery fidelity depends on container networking and privileges (e.g. `CAP_NET_RAW` and/or host networking on Linux). See [docs/discovery-capabilities.md](docs/discovery-capabilities.md) (what works where) and [docs/discovery-deployment.md](docs/discovery-deployment.md) (deployment patterns) before enabling scanning in production.

Discovery runs are scoped. The UI can suggest scopes based on the scanner’s local interfaces; you can also set `DISCOVERY_DEFAULT_SCOPE` to provide a default CIDR/IP when a run omits `scope`.

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

## Running locally (no Docker)

This is optional; the supported “it just works” path is `docker compose up --build`.

Prereqs:

- Go `1.24.x` (for `core-go/`)
- Node.js `20.x` + npm (for `ui-node/`)
- PostgreSQL `15+` running locally (Compose uses `postgres:16-alpine`)
- `migrate` CLI (golang-migrate) for database migrations

### 1) Database (PostgreSQL)

Create a database and a user/password, then set a `DATABASE_URL` that uses TCP (host `localhost`), for example:

- `export DATABASE_URL='postgres://roller:roller@localhost:5432/roller_hoops?sslmode=disable'`

(Debian/Ubuntu example)

- Install + start: `sudo apt-get update && sudo apt-get install -y postgresql`
- Create user + DB:
  - `sudo -u postgres createuser -P roller`
  - `sudo -u postgres createdb -O roller roller_hoops`

### 2) Migrations

In Docker, migrations run via the `migrate` container. Locally, use the `migrate` CLI (golang-migrate):

- Install (requires Go toolchain): `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1`
- Apply: `migrate -path core-go/migrations -database "$DATABASE_URL" up`

Optional: load the dev seed data used by the Compose `dev` profile:

- `psql "$DATABASE_URL" -f docker/dev/dev-seed.sql`

### 3) Run `core-go` (API + worker)

- `cd core-go && HTTP_ADDR=127.0.0.1:8081 LOG_LEVEL=info DATABASE_URL="$DATABASE_URL" go run ./cmd/core-go` (serves on `http://localhost:8081`)
- Health:
  - `curl http://localhost:8081/healthz`
  - `curl http://localhost:8081/readyz`

Tip: `core-go` is intentionally unauthenticated; in normal usage you should call the API through the UI proxy at `http://localhost:3000/api/...`.

If you want to tweak discovery behavior, export `DISCOVERY_*` env vars before starting `core-go` (see `.env.example` for knobs).

### 4) Run `ui-node` (UI + auth proxy)

- `cd ui-node && npm ci`
- Optional: create `ui-node/.env.local` (Next.js loads this automatically):
  - `CORE_GO_BASE_URL=http://localhost:8081` (only needed if you don’t use the default)
  - `AUTH_USERS=admin:admin:admin`
  - `AUTH_SESSION_SECRET=dev-session-secret`
- Run:
  - Dev: `npm run dev` (serves on `http://localhost:3000`)
  - Prod: `npm run build && npm start` (serves on `http://localhost:3000`)
- Open: <http://localhost:3000/> and sign in at <http://localhost:3000/auth/login>
- Default local credentials: `admin` / `admin` (unless overridden via `AUTH_USERS`)

### Troubleshooting (local)

- `GET /readyz` fails: run migrations and verify `DATABASE_URL` points at the right database.
- UI `/api/...` calls fail: ensure `core-go` is running and `CORE_GO_BASE_URL` can reach it.
- Discovery ICMP failures: your OS may require raw-socket permissions for ping (see [docs/discovery-capabilities.md](docs/discovery-capabilities.md)).

## Docs

- Roadmap / phases: `docs/roadmap.md`
- Operations runbook (metrics, backups, secrets): `docs/runbooks.md`
- API conventions: `docs/api-contract.md` (canonical spec: `api/openapi.yaml`)

## UI work (Phase 12)

The operator UX is tracked in `docs/roadmap.md` (Phase 12). The UX foundation rules live in:

- `docs/ui-ux.md`
