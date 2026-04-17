# Roller_hoops

Self-hosted network tracker / mapper (Go + Node.js + PostgreSQL), fully containerised.

The map is intended to provide a basic focused network diagram first: select a
device and see useful nearby relationships such as router/switch/peer links with
IP and MAC facts. Larger layered views build on that without defaulting to a
whole-network spaghetti graph.

## Requirements

- **Recommended and fully supported:** Docker + Docker Compose (`docker compose ...`)
- **No local Go, Node.js, npm, PostgreSQL, or migrate CLI is required** for the shared dev stack or validation workflow.
- **Also supported:** run the stack locally without Docker (see [Running locally (no Docker)](#running-locally-no-docker))
- Host port `80/tcp` available (Traefik binds `80:80`; change `docker-compose.yml` if you want a different host port)
- **Optional:** GNU `make` for the convenience targets in `Makefile` (ships with macOS/Linux/WSL; on Windows install via `choco install make` or `scoop install make`, or use WSL)

### Platform support

The project builds and runs on **Windows (Docker Desktop)**, **macOS (Docker Desktop / OrbStack)**, **Debian/Ubuntu**, and **WSL 2**. All validation and build tooling runs inside Docker containers so the host only needs Docker itself.

#### Windows — mapped / dev drives (G:, D:, etc.)

If your workspace is on a non-default drive letter, Docker Desktop may fail to resolve bind-mount paths. Fixes:

1. Open **Docker Desktop → Settings → Resources → File sharing** and add the drive (e.g. `G:\`).
2. Add `COMPOSE_CONVERT_WINDOWS_PATHS=1` to your `.env` file (see `.env.example`).
3. The project's validation tasks use `docker build` (build-context, not bind mounts), so they work regardless of drive letter.

#### Line endings

A `.gitattributes` enforces LF line endings for all source and Docker files. If you cloned the repo before this file existed, run:

```sh
git rm -r --cached . && git reset --hard
```

This re-normalises your working tree so Docker builds don't break from CRLF.

### Installing prerequisites (Debian/Ubuntu)

These are “good enough to get started” commands. For production, pin versions and follow your distro’s guidance.

- Base tools:
  - `sudo apt update`
  - `sudo apt install -y git ca-certificates curl make`

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
  - `sudo dnf install -y git ca-certificates curl make`

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

### Installing prerequisites (macOS)

- Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or OrbStack)
- Install `make` (ships with Xcode Command Line Tools):
  - `xcode-select --install`
- Clone the repo and run `make dev`.

### Installing prerequisites (Windows)

- Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Enable **WSL 2 backend** in Docker Desktop settings (recommended).
- If your workspace is on a non-default drive (G:, D:, etc.), see [Windows — mapped / dev drives](#windows--mapped--dev-drives-g-d-etc) above.
- For `make`, install via `choco install make` or `scoop install make`, or run from WSL.

### Installing prerequisites (WSL 2)

WSL 2 is the recommended way to develop on Windows. Docker Desktop integrates with WSL 2 automatically.

- Inside your WSL distro:
  - `sudo apt update && sudo apt install -y git make`
- Docker Desktop exposes the `docker` and `docker compose` CLI inside WSL automatically.
- Clone the repo inside WSL's filesystem (`~/projects/...`) for best I/O performance. Avoid `/mnt/c/` or `/mnt/g/` paths.

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

- Easiest start:
  - Windows: double-click `scripts/start-windows.cmd` or run `powershell -ExecutionPolicy Bypass -File scripts/start-windows.ps1`
  - macOS: run `sh scripts/start-macos.command`
  - Linux / WSL 2: run `sh scripts/start-linux.sh`
- The startup scripts run Docker Compose in the background, wait for the UI health check, then open <http://localhost/>.
- Start with sample seed data: add `--dev` to the script command, for example `sh scripts/start-linux.sh --dev` or `scripts\start-windows.cmd --dev`.
- Tail logs after the browser opens: add `--logs`.

- Start the full stack: `docker compose up --build` (or `sudo docker compose up --build` if your user can’t access the Docker socket).
- If you use `sudo` for `docker compose`, you’ll also need it for `docker compose logs`, `docker compose down`, etc.
- Optional: copy `.env.example` to `.env` to override local settings like `POSTGRES_PASSWORD`, `AUTH_USERS`, and `DISCOVERY_DEFAULT_SCOPE`.
- Open the UI: <http://localhost/>
- Sign in: <http://localhost/auth/login> (example users live in `.env.example` via `AUTH_USERS`)
- Default quickstart credentials: `admin` / `admin` (also configured in `.env.example`).
- The Go API is **not exposed directly**; the UI proxies `/api/...` requests to `core-go` and enforces auth/roles.

## Compose profiles

- `docker compose up --build` (default) launches Traefik, core-go, ui-node, the database, and runs migrations.
- `docker compose --profile dev up --build` runs the default stack and, once the database is healthy, executes the idempotent SQL in `docker/dev/dev-seed.sql` to populate a sample device, metadata, and related discovery rows.
- `docker compose --profile prod up --build` executes the same stack plus the `prod-readiness` service that waits for both `/healthz` and `/readyz` before exiting successfully, which can be handy for deployment smoke tests.

## Common commands

Commands work on any OS with Docker installed. The `Makefile` wraps them for convenience (run `make help` to see all targets).

| Action | Raw command | Make shortcut |
| --- | --- | --- |
| Start and open UI (Windows) | `scripts\start-windows.cmd` | - |
| Start and open UI (macOS) | `sh scripts/start-macos.command` | - |
| Start and open UI (Linux/WSL) | `sh scripts/start-linux.sh` | - |
| Start stack | `docker compose up --build` | `make up` |
| Start with seed data | `docker compose --profile dev up --build` | `make dev` |
| Stop | `docker compose down` | `make down` |
| Reset DB | `docker compose down -v` | `make reset` |
| Tail logs | `docker compose logs -f --tail=200` | `make logs` |
| Re-run seed | `docker compose --profile dev run --rm dev-seed` | — |

## Docker-only validation

The shared validation workflow runs entirely in Docker, so agents and developers do not need local Go or Node toolchains.

| Check | Raw command | Make shortcut |
| --- | --- | --- |
| Go fmt check | `docker build -f docker/validate/core-go.Dockerfile --target fmtcheck .` | `make go-fmt` |
| Go vet | `docker build -f docker/validate/core-go.Dockerfile --target vet .` | `make go-vet` |
| Go tests | `docker build -f docker/validate/core-go.Dockerfile --target test .` | `make go-test` |
| All Go checks | — | `make go-validate` |
| UI deps | `docker build -f docker/validate/ui-node.Dockerfile --target deps .` | `make ui-deps` |
| UI tests | `docker build -f docker/validate/ui-node.Dockerfile --target test .` | `make ui-test` |
| UI build | `docker build -f docker/validate/ui-node.Dockerfile --target build .` | `make ui-build` |
| All UI checks | — | `make ui-validate` |
| All checks | — | `make validate` |
| Gen OpenAPI types | VS Code task `ui: gen openapi types` | `make gen-types` |
| Smoke test (prod) | `docker compose --profile prod up --build --abort-on-container-exit` | `make smoke` |

## Dev tools container

A combined Go + Node dev shell is available for ad-hoc tooling without installing anything locally:

```sh
make devtools          # build + open interactive shell
# or manually:
docker build -f docker/devtools.Dockerfile -t roller-devtools .
docker run --rm -it -v "$(pwd)":/workspace -w /workspace roller-devtools sh
```

Inside the container you have `go`, `node`, `npm`, `psql`, `migrate`, `sqlc`, `jq`, `curl`, `git`, and `make`.

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

### Docker networking notes

- The default `docker compose up` stack runs `core-go` on a Docker bridge network. It can reach L3 targets, but it cannot see the host ARP cache, so ARP-based discovery will mostly only see the Docker network unless you change the deployment model.
- For Linux-only host-network discovery (higher fidelity ARP/ICMP), use the provided override:
  - `sudo docker compose -f docker-compose.yml -f docker-compose.hostnet.yml up --build`
  - Recommended safety: `CORE_GO_HTTP_ADDR=127.0.0.1:8081 sudo docker compose -f docker-compose.yml -f docker-compose.hostnet.yml up --build` (keeps the unauthenticated Go API bound to loopback)

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

- Agent rules: `AGENTS.md`
- AI backlog / dev runbook: `BACKLOG.md`
- Engineering standards: `docs/engineering-standards.md`
- AI coding control: `docs/ai-coding-control.md`
- VS Code AI workflow: `docs/vscode-ai-workflow.md`
- Roadmap / phases: `docs/roadmap.md`
- Operations runbook (metrics, backups, secrets): `docs/runbooks.md`
- API conventions: `docs/api-contract.md` (canonical spec: `api/openapi.yaml`)

## UI work (Phase 12)

The operator UX is tracked in `docs/roadmap.md` (Phase 12). The UX foundation rules live in:

- `docs/ui-ux.md`
