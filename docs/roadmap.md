# Roadmap

<!-- markdownlint-disable MD024 -->

---

## High-level goal

A **self-hosted network tracker / mapper** that:

* Stores state historically
* Provides a web UI for viewing and editing metadata
* Runs fully containerised
* Can scale or split later without redesign

---

## Core design choices (locked early)

### Languages

* **Go** → discovery engine + API
* **Node.js (TypeScript)** → web UI

### Infrastructure

* **Docker + docker-compose**
* **Reverse proxy**: Traefik
* **Database**: PostgreSQL (only)
* **Auth**: built-in (handled by UI; Go stays headless)

---

## Service layout (final target)

```text
docker-compose.yml
│
├─ traefik
│
├─ ui-node/
│   └─ Node.js UI service
│
├─ core-go/
│   └─ Go discovery + API service
│
├─ db/
│   └─ PostgreSQL
│
└─ volumes/
    └─ persistent data
```

---

## Current progress snapshot

* Device CRUD live end-to-end via Go API and UI; metadata fields (`owner`, `location`, `notes`) persist in `device_metadata`.
* Discovery endpoints persist runs/logs (`discovery_runs`, `discovery_run_logs`) and return real run ids; the Go worker claims queued runs and performs an ARP scrape + best-effort ICMP sweep (where available) to populate current IP/MAC facts and per-run observations, with optional enrichment (reverse DNS + SNMP) writing `device_name_candidates`, `device_snmp`, interface details, and VLAN PVIDs (`interface_vlans`) when enabled.
* Traefik + docker-compose bring up UI and PostgreSQL with health checks enabled; **core-go stays private on the Docker network**.
* Prometheus metrics (HTTP request counts/durations and discovery run counts/durations) are exposed at `/metrics`, and `docs/runbooks.md` captures the runbook for scraping, backups, migrations, and secrets rotation while we finish Phase 11.
* OpenAPI spec exists at `api/openapi.yaml`, and a **Go contract test prevents drift** between the spec and the chi router.
* Dev DB auth uses a password via env vars (no more `trust` by default); secrets are injected via `.env` (gitignored) or a secret manager.

### Implemented APIs (current)

Canonical contract: `api/openapi.yaml` (served under `/api`).

`core-go` (REST, v1):

```text
GET    /api/v1/devices
POST   /api/v1/devices
GET    /api/v1/devices/{id}
PUT    /api/v1/devices/{id}
GET    /api/v1/devices/{id}/name-candidates

GET    /api/v1/devices/export
POST   /api/v1/devices/import

POST   /api/v1/discovery/run
GET    /api/v1/discovery/status
```

`core-go` (service health; not part of the public API contract):

```text
GET /healthz
GET /readyz
```

`ui-node` (service health):

```text
GET /healthz
```

### Planned APIs (next)

These “API surface” milestones are already described later in this roadmap (keep OpenAPI canonical; prefer adding endpoints/params over UI-side reconstruction):

```text
Phase 9 — historical/diffing (implemented)
GET    /api/v1/devices/changes?since=RFC3339&limit=N
GET    /api/v1/devices/{id}/history?limit=N&cursor=...
GET    /api/v1/discovery/runs
GET    /api/v1/discovery/runs/{id}
GET    /api/v1/discovery/runs/{id}/logs

Phase 11 — ops/telemetry
GET    /metrics

Phase 14+ — map projections (read-only, focus-scoped)
GET    /api/v1/map/{layer}?focusType=device|subnet|vlan|zone&focusId=...
```

---

## Project trackers (source of truth)

* Feature inventory + status: `docs/feature-matrix.md`
* API rules + conventions: `docs/api-contract.md` (canonical spec: `api/openapi.yaml`)
* DB schema intent: `docs/data-model.md` (implemented via migrations in `core-go/migrations/`)
* Service boundaries: `docs/architecture.md`
* Security guardrails: `docs/security.md`
* Network map product direction + mocks: `docs/network_map/network_map_ideas.md` (mocks: `docs/network_map/idea1.png`, `docs/network_map/idea2.png`)

---

## Blockers & risks (current)

* **Auth boundary is enforced via UI sessions**: the UI requires login and issues a signed `roller_session` cookie before any API traffic is allowed; admin vs read-only roles are enforced in the UI (Go remains headless).
* **Discovery inside Docker needs a deployment decision**: ARP/ICMP/SNMP fidelity depends on container networking and capabilities (e.g., `CAP_NET_RAW`, host networking, or a dedicated scanner container deployed on the target network).
* **Production secret injection is a deployment responsibility**: keep secrets out of git, inject via env/Docker secrets/secret manager, and follow `docs/runbooks.md` for rotation expectations.
* **Historical model implemented**: observations, change feed, and run/log APIs now exist; next focus is documenting retention and monitoring query cost.

---

## Roadmap at a glance

| Phase | Status | Summary |
| --- | --- | --- |
| 0 | Done | Stack decisions (Postgres, Traefik, REST) |
| 1 | Done | `core-go` API + persistence baseline |
| 2 | Done | Minimal relational schema + migrations |
| 3 | Done | `ui-node` MVP (CRUD + discovery panel) |
| 4 | Done | Traefik routing; `core-go` kept private |
| 5 | Done | Compose profiles, volumes, health checks |
| 6 | Done | UI polling for “live” feel |
| 7 | Done | Enrichment integrations (mDNS/NetBIOS/ports/IPAM) |
| 8 | Done | Discovery engine v1 (runs/logs + worker + observations) |
| 9 | Done | Historical/diffing APIs (changes feed, history, runs/logs) |
| 10 | Done | Auth + session hardening (roles + audit hooks) |
| 11 | Done | Observability & operations (metrics/runbooks/CI) |
| 12 | Planned | UI workflows for operators |
| 13 | Planned | Map shell + interaction contract |
| 14 | Planned | Projection API + map data model (L3 first) |
| 15 | Planned | Physical/L2/L3 layers |
| 16 | Planned | Services/Security layers + modes |

## Phases

## Phase 0 — Foundations

**Status:** Done

### Goal

Lock the boring defaults early so later phases don’t re-litigate fundamentals.

### Tasks

* [x] Pick DB: **PostgreSQL**
* [x] Pick reverse proxy: **Traefik**
* [x] Pick API style: **REST over HTTP**

### Blockers

* None

### Deliverables

* A single “default stack” decision set referenced by later phases.

---

## Phase 1 — Go core service (headless) (closed)

**Status:** Done (closed)

### Goal

Ship a headless `core-go` service that owns persistence + API + discovery orchestration.

### Constraints

* `core-go` stays headless (no UI, no auth UX).
* The browser must not call `core-go` directly in v1 (UI is the BFF/proxy).

### Tasks

* [x] Stand up HTTP server (`net/http` + `chi`) with strict JSON decoding.
* [x] Make OpenAPI canonical (`api/openapi.yaml`) and gate drift with a Go contract test.
* [x] Add Postgres access (`sqlc` + `pgx`) and migrations (`golang-migrate`).
* [x] Add structured logging (`zerolog`) and request ID propagation.
* [x] Add service health endpoints (`/healthz`, `/readyz`).

### Blockers

* None (implementation complete); keep `core-go` private and enforce the auth boundary via `ui-node` (Phases 10–11 completed).

### Notes (implementation choices)

* HTTP server: Go stdlib (`net/http`) + `chi` router
* API contract: OpenAPI is canonical (a Go contract test gates drift against the chi router)
* DB access: `sqlc` (PostgreSQL-first) + `pgx`
* Migrations: `golang-migrate`
* Config: env vars only (Docker-friendly)
* Logging: `zerolog` (JSON logs)

Operational defaults (v1):

* Strict JSON decoding for requests (reject unknown fields)
* Health endpoints: `/healthz` and `/readyz`
* Request ID propagation end-to-end (UI → Traefik → Go)

API: see “Implemented APIs (current)” in the progress snapshot at the top of this document.

---

## Phase 2 — Database schema (minimal but future-proof) (closed)

**Status:** Done (closed)

### Goal

Use a boring relational schema that supports both “current state” and later historical/diffing work.

### Tasks

* [x] Create core tables and migrations (devices, interfaces, IPs, MACs, services, user-editable metadata).
* [x] Keep schema relational (avoid JSON blobs as the default).
* [x] Ensure schema direction supports later append-only observations + retention controls (Phase 9).

### Blockers

* Historical model/indexing is a dependency for Phase 9 (not a blocker for this phase, but affects future migrations).

### Deliverables

* DB survives container restarts and migrations are repeatable.

Use **existing relational DBs**, no custom storage.

### Core tables

* devices
* interfaces
* ip_addresses
* services
* mac_addresses
* device_metadata (user-editable)

Avoid:

* JSON blobs everywhere
* Over-normalisation

Deliverable:

* DB survives container restarts
* Core schema supports future historical data (Phases 8+)

---

## Phase 3 — Node.js UI service

**Status:** Done (MVP UI + metadata editing + discovery panel; auth/workflows tracked later)

### Constraint: Node does not do discovery/scanning

### Stack

* Node.js + TypeScript
* Framework: **Next.js** (SSR-first)
* UI: plain HTML forms + minimal CSS (Tailwind optional; avoid heavy client state early)
* API client: typed `fetch` (prefer `openapi-fetch`) + generated TypeScript types from OpenAPI (prefer `openapi-typescript` or `orval`)

API-first rule (so we don’t re-implement server logic in the UI):

* When a UI feature needs a new slice of data, prefer adding a **small new API endpoint** or **optional query params** to an existing endpoint.
* Keep OpenAPI canonical and generate types/clients from it (avoid hand-written DTO drift).

### UI responsibilities

* Forms
* Editing metadata
* Filtering, grouping
* Live view (polling or WebSockets)

### Auth

Start simple:

* Local users
* Session cookies

Rule (v1):

* UI owns authentication and sessions.
* Go stays headless and is not exposed directly to the public internet except via Traefik routing.

Deliverable:

* UI talks only to Go API
* No shared database access

Implementation detail (already in place):

* The UI acts as a BFF/proxy for the browser via `ui-node/app/api/[...path]/route.ts`.
  * Browser calls `/api/...` on the UI.
  * UI proxies to `core-go` on the private Docker network.

### Tasks

* [x] Implement Devices UI (list/detail/edit metadata) backed by Go API.
* [x] Implement Discovery panel (trigger run + status view) backed by Go API.
* [x] Implement UI-as-BFF proxy route to keep `core-go` private.
* [ ] Add auth + sessions (Phase 11).
* [ ] Add richer operator workflows (Phase 10).

### Blockers

* Auth is not implemented yet (Phase 11); mitigate by keeping `core-go` private and routing via UI only.

---

## Phase 4 — Reverse proxy & routing

**Status:** Done (UI exposed; API kept private until auth)

### Goal

Expose a single entrypoint while keeping `core-go` off the public network until auth is real.

### Tasks

* [x] Route `/` → UI, `/api` → Go via Traefik.
* [x] Keep `core-go` internal-only on the Docker network.
* [x] Provide TLS wiring for production (dev runs HTTP).

### Blockers

* Public exposure of `core-go` is blocked on Phase 11 (auth/session hardening).

Use **existing battle-tested infra**

### Traefik

* Routes `/` → Node
* Routes `/api` → Go (internal-only entrypoint; not published to the host)
* Handles TLS (production config; dev runs HTTP)
* Does not replace application auth in v1 (auth remains a UI concern)

Deliverable:

* Single hostname
* Clean separation

---

## Phase 5 — Docker polish

**Status:** Done (closed)

### Goal

Make the stack reproducible and boring to run locally and in production via compose profiles.

### Tasks

* [x] Enforce container rules (one process per container, config via env, DB is the only shared state).
* [x] Add health checks and named Postgres volume.
* [x] Add dev profile seeding (`docker/dev/dev-seed.sql`) for fast local testing.
* [x] Add prod readiness smoke service (`prod-readiness`) to wait on `/healthz` + `/readyz`.

### Blockers

* Production secret injection is a deployment choice; see `docs/runbooks.md` for secret rotation and injection expectations.

### Docker rules

* One process per container
* No shared state except DB
* All config via env vars

### docker-compose

* Profiles for dev / prod
* Named volumes for DB
* Health checks

### Compose profiles

* `docker compose --profile dev up` runs the base stack plus the idempotent SQL loader in `docker/dev/dev-seed.sql`, which seeds a sample device, metadata row, interface, IP, MAC, and service entry for local testing.
* `docker compose --profile prod up` runs the stack plus the `prod-readiness` service, which waits for `/healthz` on the UI and `/readyz` on core-go before exiting, making it useful for deployment smoke tests.

Deliverable:

* `docker compose up`
* System survives reboot

---

## MVP exit criteria (Phases 0-5)

* docker compose brings up Traefik, UI, Go API, and PostgreSQL with named volumes; health checks pass.
* Device CRUD flows work end-to-end (UI → API → DB) with request IDs in logs.
* OpenAPI spec lives in the repo; API behaviour matches it (via codegen or contract tests).
* Migrations are repeatable (`golang-migrate`), and a short doc exists for running them in dev/CI.
* Minimal tests run in CI: Go handler test against Postgres, and a UI smoke that hits the real API.

---

## Phase 6 — Live updates (optional)

**Status:** Done (closed)

### Goal

Keep the UI feeling live without adding tight coupling (WebSockets can come later if needed).

### Tasks

* [x] Poll `/api/v1/devices` and `/api/v1/discovery/status` only when relevant UI pages are visible.
* [x] Keep polling lightweight and resilient to temporary failures.

### Blockers

* None

The UI now polls the Go API (`/api/v1/devices` and `/api/v1/discovery/status`) whenever the devices dashboard or discovery panel is visible. This keeps the operator experience live without coupling the services more tightly, so the last-known state, discovery progress, and device metadata stay up to date with a lightweight polling loop.

---

## Phase 7 — Nice-to-have integrations

**Status:** Done (snapshot + SNMP/VLAN/name-candidates + LLDP/CDP links + service scanning + external inventory import)

Only after core is stable. The intent is to enrich discovered devices by **reusing existing protocols/tools** (SNMP, mDNS, nmap, IPAM APIs), not by building bespoke scanners.

### Tasks

Implemented:

* [x] SNMP enrichment baseline (best-effort; behind enable flags).
* [x] Friendly-name candidates (reverse DNS, mDNS, NetBIOS, SNMP `sysName`) via `GET /api/v1/devices/{id}/name-candidates`.
* [x] Import/export JSON endpoints + UI snapshot workflow.
* [x] LLDP/CDP adjacency via SNMP MIBs (optional; best-effort) → writes to `links` with `source=lldp|cdp`.
* [x] Service/port discovery via `nmap` XML parsing (optional; behind explicit enable flags + allowlists) → writes to `services`.
* [x] External inventory/IPAM import (NetBox/Nautobot) via wrapped payload endpoints.

### Blockers

* **mDNS / NetBIOS resolution** depends on multicast visibility: containers running on bridge networks often cannot join multicast without `CAP_NET_RAW` or host/bridge networking. The workbench needs documentation about how to expose the scanner to the target broadcast domain before enabling this feature.
* **LLDP/CDP adjacency (enrichment)** requires SNMP credentials and device-level support; not all switches expose the right MIBs, so we need allowlists, retry/backoff, and failure isolation so the discovery worker stays healthy even if switches reject requests.
* **Service/port discovery** is an active scan, so we need operator opt-in, clear scope (CIDR/vlan), rate-limits, and allowlists to avoid triggering IDS/IPS responses. Scans should run in a configurable window, obey allowed ports, and provide a cancel path to avoid blocking the worker.
* **External inventory / IPAM syncs** require API tokens, field mapping, conflict resolution, and rate-limit handling for each provider (NetBox, Nautobot, etc.). Without a documented provisioning story (where the tokens live, what fields map to our schema), automation will stall.
* All enrichment paths need **secure secrets** (SNMP community strings, API tokens, port-scan credentials) and proper gates so they remain opt-in (cannot run by default). The team needs a runbook describing where secrets live (`.env`, Docker secrets, or secret manager) and how to rotate them.
* Additional enrichment writes increase DB load (new `device_name_candidates`, `interface_vlans`, `device_snmp`). We need retention/cleanup policies and indexes so the tables stay performant as observations accumulate.

### Notes

* Snapshot tooling is available via `/api/v1/devices/export` and `/api/v1/devices/import`, and the devices UI offers download/upload controls.
* Name candidates now include reverse DNS, mDNS, and NetBIOS results so operators see richer choices in the device detail view.

API surfaced by this phase (implemented):

```text
GET  /api/v1/devices/export
POST /api/v1/devices/import
GET  /api/v1/devices/{id}/name-candidates
POST /api/v1/inventory/netbox/import
POST /api/v1/inventory/nautobot/import
```

---

## Phase 8 — Discovery engine v1 (network scanning)

**Status:**

### Goal

Ship a v1 discovery loop that can be triggered manually, persists runs/logs, and reliably updates “current state” from observations.

Goals:

* Decide discovery scope: start with ARP table scrape + ICMP ping; add read-only SNMP once the loop is solid.
* Job model in Postgres: `discovery_runs` (id, scope, status, started_at, completed_at, stats JSON) and `discovery_run_logs` (structured log lines).
* Worker loop in Go: timer-based plus manual trigger; single worker at first; cancellation and backoff on failure.
* Observations table (append-only) to store IP/MAC/service findings per run; dedupe by stable keys when folding into current state.
* Wire `/api/v1/discovery/run` to enqueue and return a run id; `/api/v1/discovery/status` returns latest run, progress, and last error.

### Tasks

* [x] Persist discovery runs and logs (`discovery_runs`, `discovery_run_logs`).
* [x] Implement a Go worker loop that claims queued runs and reports progress/errors.
* [x] Perform ARP scrape + best-effort ICMP sweep (where available) to generate observations/current state.
* [x] Add optional enrichment (reverse DNS + SNMP) behind enable flags.
* [x] Expose `POST /api/v1/discovery/run` + `GET /api/v1/discovery/status`.

### Blockers

* Deployment choice affects fidelity (Docker networking + required caps vs host networking vs a dedicated scanner container).
* ICMP/SNMP behavior varies by environment; requires explicit runbooks and safe defaults for production.

Deliverable:

* Go service runs alone in Docker, performs a subnet sweep, populates devices/interfaces with timestamps, and returns real discovery status.

---

## Phase 9 — Historical state + diffing

**Status:** Done

### Goal

Make time a first-class feature: every fact can be diffed across runs, and “what changed?” is cheap to query.

### Design constraints (non-negotiable)

* **OpenAPI is canonical** (`api/openapi.yaml`). The code follows the spec, not the other way around.
* **No UI-side diff reconstruction**. Diffs/history must be server-side and cheap to query.
* **Deterministic output** (stable sorting, stable IDs) so polling + UI diffs don’t churn.
* **Build on what exists**: `ip_observations` and `mac_observations` already exist (Phase 8). Phase 9 extends this model; it doesn’t replace it.

### Shared tasks for Phase 9 (applies to all milestones)

* [x] Pin the v1 **event model**: what constitutes a “change” and how it’s represented (device-level change events vs fact-level changes).
* [x] Decide a **retention strategy** (how long to keep observations/events) and document it.
* [x] Add/verify **indexes** for “since time” queries (high-write, low-latency reads).
* [x] Keep responses **cursor-friendly** (no unbounded lists; enforce limits).
* [x] Add integration tests against Postgres for query correctness and paging stability.

### Blockers

* Schema/index decisions here affect write amplification and query cost; retention/index strategy is now documented and monitored as ingestion grows.
* UI timelines and “changed” overlays depend on these APIs (Phase 10 and Phase 16).

### Milestones (Phase 9)

#### M9.1 — Device change feed (pollable)

**API:** `GET /api/v1/devices/changes?since=RFC3339&limit=N`

Intent: a stable feed that answers “what changed since $t$?” without the UI doing joins.

Tasks:

* Contract
  * [x] Add endpoint + schemas to `api/openapi.yaml`.
  * [x] Define response semantics for `since` (strictly `>` vs `>=`) and document it.
  * [x] Define a stable event payload (recommended fields): `event_id`, `device_id`, `event_at`, `kind`, `summary`.
* Persistence / queries
  * [x] Decide whether change feed is:
    * derived on the fly from observations, **or**
    * persisted as a dedicated events table (recommended for cheap reads and stable paging).
  * [x] Add required migrations (if adding an events table).
  * [x] Add indexes to support `(event_at, event_id)`-style paging.
* Core-go implementation
  * [x] Implement handler and query with hard caps on `limit`.
  * [x] Ensure deterministic ordering when timestamps tie (secondary sort by stable id).
  * [x] Ensure errors follow `docs/api-contract.md` envelope.
* Tests
  * [x] Add Postgres integration tests for:
    * empty feed
    * `since` boundary correctness
    * stable ordering
    * `limit` clamping
* Docs
  * [x] Update “Planned APIs (next)” snapshot if needed.
  * [x] Add a short note about recommended polling cadence + idempotency.

Implementation: `GET /api/v1/devices/changes` now returns cursor-friendly, deterministic events aggregated from observations, metadata, and service scans.

Implementation: `GET /api/v1/devices/changes` now returns cursor-friendly, deterministic events aggregated from observations, metadata, and service scans.

Blockers:

* Requires an agreed “change” definition (device metadata change vs discovered fact changes vs both).
* Needs a paging strategy that remains stable under concurrent discovery writes.

Acceptance criteria:

* Polling the endpoint repeatedly with the same `since` returns a stable, deterministic list.
* Feed is cheap to query (index-backed) and bounded by `limit`.

#### M9.2 — Device history (timeline)

**API:** `GET /api/v1/devices/{id}/history?limit=N&cursor=...`

Intent: a focused timeline suitable for a device detail page.

Tasks:

* Contract
  * [x] Add endpoint + schemas to `api/openapi.yaml`.
  * [x] Define cursor format (opaque string recommended) and ordering (newest-first recommended).
* Persistence / queries
  * [x] Decide history sources:
    * discovery-run events,
    * observation-derived diffs,
    * metadata edits (if/when metadata auditing exists).
  * [x] Add any missing observation tables needed to tell the story (e.g., interface/service observations) without breaking current v1.
* Core-go implementation
  * [x] Implement handler + query that enforces `limit` and supports cursors.
  * [x] Keep payload intentionally small (summary-first; details can be added later).
* Tests
  * [x] Integration tests for cursor paging stability and ordering.
  * [x] 404 behavior for unknown device id.
* Docs
  * [x] Document how history relates to discovery runs (link to M9.3 endpoints).

Implementation: `GET /api/v1/devices/{id}/history` surfaces the same change events for a single device with cursor-based pagination and stable ordering.

Blockers:

* Depends on M9.1 event model decisions (reusing the same event shape is strongly preferred).
* Needs clear stance on whether manual metadata edits appear in the timeline (and when).

Acceptance criteria:

* A device with multiple runs produces a deterministic, paginated timeline.
* The UI can render a useful “what happened” narrative without extra API calls.

#### M9.3 — Discovery run access (timelines + debugging)

**APIs:**

* `GET /api/v1/discovery/runs`
* `GET /api/v1/discovery/runs/{id}`
* `GET /api/v1/discovery/runs/{id}/logs`

Intent: operators can inspect runs and correlate changes to a run id without reading database tables.

Tasks:

* Contract
  * [x] Add endpoints + schemas to `api/openapi.yaml`.
  * [x] Define paging for runs list (`limit`, `cursor`), and for logs (`limit`, `cursor` or `since`).
* Core-go implementation
  * [x] Implement list/get/logs handlers.
  * [x] Ensure log output is bounded and sorted.
* Tests
  * [x] Integration tests for paging and for “run not found”.
* Docs
  * [x] Document how run ids relate to Phase 9 history and change feed.

Implementation: `/api/v1/discovery/runs` with `/runs/{id}` and `/runs/{id}/logs` return paginated runs/log entries with stable cursors.

Blockers:

* Need agreement on log retention and “how much is too much” for API payload sizes.

Acceptance criteria:

* Runs and logs can be inspected entirely via API with stable paging.

Deliverable:

* You can diff any device across runs and show a timeline of changes without hand-written SQL.

---



## Phase 10 — Auth + session hardening

**Status:** Done

### Goal

Make authentication and authorization real, so the system can be exposed safely beyond a dev network while keeping `core-go` headless.

### Tasks

* [x] Implement a UI-owned login page (`/auth/login`) that validates credentials from `AUTH_USERS` (or back-compat `AUTH_USERNAME` / `AUTH_PASSWORD`).
* [x] Issue signed HTTP-only session cookies (`roller_session`, 24h TTL) backed by `AUTH_SESSION_SECRET` and scoped to the UI domain.
* [x] Add roles (admin vs read-only) and enforce authorization in the UI before calling Go (proxy routes + server actions).
* [x] Add account lifecycle (password change, admin reset/recovery flow; requires `AUTH_USERS_FILE` to persist updates).
* [x] Add minimal audit logging via Go API (UI never writes core tables directly).
* [x] Decide whether to use Auth.js/NextAuth vs a minimal custom session implementation (v1 uses a minimal custom signed-cookie session).

### Blockers

* Public exposure of `core-go` (and broader adoption) is blocked on shipping this phase and the associated role/authorization work.

Deliverable:

* Auth no longer relies on stubs; sign-in/out and session cookies work end-to-end, even though finer-grained RBAC is still Phase 11.

Implementation notes:

* `/auth/login` renders the form, and `/api/auth/login` + `/api/auth/logout` handle credential validation plus signed `roller_session` cookies.
* Sessions are signed with `AUTH_SESSION_SECRET`, last 24 hours, and are marked `HttpOnly; SameSite=Lax` for CSRF resilience on the same hostname.
* Configure users via `AUTH_USERS` (or back-compat `AUTH_USERNAME` / `AUTH_PASSWORD`), and optionally set `AUTH_USERS_FILE` to enable password changes; the UI enforces the login before any `/api` request is proxied.

---

## Phase 11 — Observability & operations

**Status:** Done

### Goal

Make the stack operable by someone who didn’t write it: metrics, runbooks, backups, and CI confidence.

### Tasks

* [x] Add metrics (`GET /metrics`) and decide on internal scrape/routing posture.
* [x] Add runbooks: backup/restore (`pg_dump`), migrations, secret rotation, seeding, and document them in `docs/runbooks.md`.
* [x] Add CI coverage: Go unit/integration (Postgres), UI smoke against real API, OpenAPI contract drift gate.
* [x] Add basic SLO monitoring approach (health endpoints + uptime alert stubs).

### Blockers

* None (runbooks document secret injection and rotation expectations).

### Notes

* Metrics: Prometheus endpoints from Go (HTTP request/duration + discovery run counters/durations) and Traefik access logs/metrics.
* Tracing/logging: request IDs end-to-end; structured logs on stdout; optional OpenTelemetry spans for discovery runs.
* Runbooks: `docs/runbooks.md` details `docker compose` snippets for backup/restore (`pg_dump`), migrations (`golang-migrate up`), rotating secrets, seeding a dev stack, and configuring `/metrics` scrapes.
* Testing: CI jobs for Go unit/integration (with Postgres container), UI smoke against `next dev` + API, and a contract test to keep OpenAPI in sync.
* SLOs: health endpoints monitored by Traefik + simple uptime check; webhook/email alert stubs.

Deliverable:

* Operators have metrics, logs, runbooks, and tests so the stack can be run by someone who did not write it.

API/telemetry endpoints (planned):

* `GET /metrics` on `core-go` (Prometheus scrape target; internal network)
* Traefik metrics/access logs as configured per deployment

---

## Phase 12 — UI workflows for operators

**Status:** Planned

### Goal

Make day-to-day operation possible without curl: discovery, triage, and metadata updates are fast and safe.

### Foundations (read this first)

* `docs/ui-ux.md` — Phase 12 UX principles, page anatomy, design primitives, and operator workflow patterns.

### Milestones (Phase 12)

#### M12.1 — UI foundation (app shell + primitives)

Tasks:

* [x] Establish a consistent app shell (header/nav + content layout) used by all pages.
* [x] Add a small internal set of UI primitives (buttons/inputs/badges/cards/alerts/skeletons).
* [x] Add global empty/loading/error handling patterns (no blank screens).
* [x] Ensure read-only role UX is obvious (disabled actions explain why).

Acceptance criteria:

* All primary routes have consistent layout, typography, and spacing.
* Every screen has a non-jarring loading state and a friendly empty state.

#### M12.2 — Devices list v2 (triage-first)

Tasks:

* [ ] Filters: online/offline/changed + quick clear/reset.
* [ ] Search: server-backed search with URL-stored state.
* [ ] Sorting: stable sort options (explicit, never “magical”).
* [ ] Pagination: cursor-based paging with deterministic ordering.
* [ ] Row actions: open device, copy ID/IP, quick metadata affordances.

Acceptance criteria:

* Operators can find and open a device in a few seconds.
* Filter/sort/search state survives refresh and can be shared as a link.

#### M12.3 — Device detail v2 (facts + timeline)

Tasks:

* [ ] Sections (tabs or cards): Overview, Facts (IPs/MACs/interfaces/services), Metadata, History.
* [ ] History/timeline UX built directly on Phase 9 endpoints (no UI-side diff reconstruction).
* [ ] Clear “last seen / last changed” indicators.

Acceptance criteria:

* Operators can answer: “what is this device?”, “what changed?”, and “what’s the current truth?” quickly.

#### M12.4 — Discovery UX v2 (confidence + debugging)

Tasks:

* [ ] One obvious “Run discovery” action with clear status feedback.
* [ ] Add a discovery runs list and a run detail page (including logs) using existing APIs.
* [ ] Make failures actionable (surface last error + link to logs).

Acceptance criteria:

* Operators can trigger discovery and diagnose failures without leaving the UI.

#### M12.5 — Operator-grade polish (accessibility + resilience)

Tasks:

* [ ] Accessibility pass: keyboard nav, focus ring, contrast, reduced motion.
* [ ] Error resilience: retries, “last updated” stamps for live panels, and stable polling.
* [ ] Performance guardrails: avoid unnecessary client JS; keep interactions snappy.

Acceptance criteria:

* The UI is comfortable to use for long sessions (no “death by papercuts”).

### Blockers

* None (Phase 9 APIs exist; Phase 10 auth and Phase 11 operations are complete).

Deliverable:

* An operator can sign in, launch discovery, watch progress, inspect a device, and edit metadata without using curl.

API preference for Phase 12 UX:

* Prefer optional query params on `GET /api/v1/devices` (or small dedicated endpoints) for:
  * server-side filtering (`status=online|offline|changed`)
  * search (`q=...`)
  * sorting (`sort=last_seen_desc`)
  * pagination (`limit`, `cursor`)
* Avoid pulling “all devices” to the browser just to re-filter/re-sort.

---

## Phase 13 — Network map v1 (Layered Explorer shell)

**Status:** Planned

### Goal

Land the stable layout + interaction contract from the mocks, without committing to a “global topology” view.

### Mock-driven UI contract (what the screenshots actually imply)

The two mock screens in `docs/network_map/idea1.png` and `docs/network_map/idea2.png` anchor the v1 UX:

* **Persistent 3-pane layout**
  * Left: layer chooser (Physical / L2 / L3 / Services / Security)
  * Center: canvas (dark theme + subtle depth)
  * Right: inspector (identity + status + relationships)
* **Object-first rendering**
  * Default canvas is intentionally empty (hint text).
  * Selecting a layer does not imply “draw everything”; you still need a focus.
* **Inspector as the anchor**
  * Inspector shows identity fields + a relationship action like “View L3” / “View in Physical”.
* **No spaghetti edges**
  * The mock uses a small number of intentional connectors and mostly relies on grouping/regions.

Non-negotiables (from `docs/network_map/network_map_ideas.md` + mocks):

* 3-pane layout is constant: **Layer panel (left)** / **Canvas (center)** / **Inspector (right)**
* Only **one layer active** at a time; switching layers **fully re-renders** the canvas
* Object-first: nothing renders by default; user selects a layer and a focus object/scope
* “Stacked regions”, not wire soup (e.g., subnets as rounded regions; labels on hover)

### Milestones (Phase 13)

#### M13.1 — Route + layout (shell)

Tasks:

* UI routing
  * [ ] Add `/map` route (SSR-friendly, no client-only dependencies required to show the shell).
  * [ ] Render the constant 3-pane layout (Layer panel / Canvas / Inspector).
* Empty-state contract
  * [ ] No focus ⇒ empty canvas + instructional hint text (match mock philosophy).
* DX + maintainability
  * [ ] Keep state local to the route (avoid cross-app global state early).

Blockers:

* None (mock data is explicitly allowed here).

Acceptance criteria:

* Visiting `/map` renders the 3-pane layout instantly and shows an empty-state hint.

#### M13.2 — Layer switching contract (URL-driven)

Tasks:

* URL contract
  * [ ] Encode layer in the URL (e.g., `/map?layer=l3`).
  * [ ] Validate unknown layers (fall back to empty-state or a friendly error).
* Rendering
  * [ ] Switching layers clears canvas state; each layer owns its own projection/render config.

Blockers:

* Needs a stable list of layer names that matches `docs/api-contract.md`.

Acceptance criteria:

* Deep-linking to `/map?layer=l3` selects L3 and reloading preserves it.

#### M13.3 — Focus contract (object-first)

Tasks:

* URL contract
  * [ ] Encode focus in URL (`focusType`, `focusId`).
  * [ ] Define “no focus” behavior as valid, not an error.
* UI affordances
  * [ ] Provide a minimal focus picker entry point (can be a stub/search box for now).

Blockers:

* Focus types must align with Phase 14 query params (so UI doesn’t drift).

Acceptance criteria:

* `/map?layer=l3&focusType=device&focusId=...` is a stable deep link.

#### M13.4 — Inspector contract (always-on)

Tasks:

* Inspector structure
  * [ ] Implement sections: Identity / Status / Relationships.
  * [ ] Relationship actions exist as stubs (e.g., “View in L3”, “View in Physical”).
* Cross-layer navigation
  * [ ] “View in …” actions update URL (layer + focus) without losing focus if possible.

Blockers:

* Relationship action targets must match the Phase 13/14 URL contract.

Acceptance criteria:

* Inspector remains visible and stays in sync with the focused object.

Acceptance criteria:

* The UI matches the mock’s interaction philosophy: constant layout, mutually-exclusive layers, empty-by-default.
* The inspector is always visible and stays in sync with focus.
* Deep links are stable (layer/focus stored in URL).
* No API changes required yet; mock data is acceptable for this phase.

Deliverable:

* UI has the layered map shell and interaction patterns, even with mocked data.

---

## Phase 14 — Map data model + API projections (layer-aware)

**Status:** Planned

### Goal

Make the map a projection of structured objects, not a hand-drawn diagram.

### Data model (incremental, minimal v1)

* Reuse existing core entities: `devices`, `interfaces`, IP/MAC facts, services.
* Add the smallest set of new entities needed for projections:
  * `subnets` (+ membership derived from IPs)
  * `vlans` (+ interface association; reuse/extend existing VLAN/PVID enrichment tables)
  * `links` (physical adjacency; start as user-entered, later LLDP/CDP enrichment)
  * `zones` (security grouping; manual tags to start)

### API (projection-first)

Add read endpoints that return a **render-ready projection** for a given layer and focus:

* `GET /api/v1/map/{layer}` with query params like `focusType=device|subnet|vlan|zone`, `focusId=...`, `scope=...`
* Response contains:
  * `regions[]` (e.g., subnets/zones as rounded containers)
  * `nodes[]` (devices/interfaces/services)
  * `edges[]` (layer-defined relationships only)
  * `inspector` payload for the focused object

Rules:

* No “entire network graph” endpoint in v1.
* Default response for no focus: empty graph + guidance message.
* OpenAPI is canonical; add contract tests for projections (shape + required fields).

### Blockers

* Requires the Phase 13 URL/state contract (layer + focus) to be stable so the UI and API match.
* Subnet/VLAN derivation needs predictable rules (CIDR source of truth, membership from observed IPs, and stable IDs).
* Requires careful output stability (sorted nodes/edges, stable IDs) so UI diffs don’t churn.

Deliverable:

* Selecting a focus object in the UI produces a real (small) graph for L3 from live data.

### Milestones (Phase 14)

#### M14.1 — Projection schema pinned (OpenAPI-first)

Tasks:

* Contract
  * [ ] Define a `MapProjection` schema in `api/openapi.yaml` (canonical).
  * [ ] Define shared types: `Region`, `Node`, `Edge`, `Inspector`.
  * [ ] Define error behavior for invalid layer/focus (per `docs/api-contract.md`).
* Stability rules
  * [ ] Document sorting rules (e.g., sort regions/nodes/edges by stable id).
  * [ ] Document “no focus” behavior: return an empty projection + guidance (200 OK).
* Tests
  * [ ] Add contract coverage so router cannot drift (keep the existing drift-gate philosophy).
* UI wiring (scaffolding)
  * [ ] Ensure UI types are generated from OpenAPI (no hand-written DTOs).

Blockers:

* Requires agreement on stable IDs for non-device focus types (subnet/vlan/zone identifiers).

Acceptance criteria:

* The projection response shape is pinned and can be consumed by UI without guessing.

#### M14.2 — L3 projection (device focus) from live data

**API:** `GET /api/v1/map/l3?focusType=device&focusId=...`

Tasks:

* Projection rules
  * [ ] Regions = subnets derived from the focused device’s IP facts.
  * [ ] Nodes = focused device + peers in those subnets.
  * [ ] Edges = region membership and a small number of intentional connectors (no mesh).
* Core-go implementation
  * [ ] Implement handler + SQL for IP→subnet grouping and peer selection.
  * [ ] Enforce hard limits (node/edge caps) to prevent pathological graphs.
  * [ ] Ensure deterministic ordering.
* Tests
  * [ ] Integration tests using seeded data (subnet grouping + deterministic output).

Blockers:

* Needs predictable subnet derivation rules (CIDR boundaries and identifier choice).

Acceptance criteria:

* One focused device yields a small, readable L3 projection with stable output.

#### M14.3 — L3 projection (subnet focus)

**API:** `GET /api/v1/map/l3?focusType=subnet&focusId=...`

Tasks:

* Contract
  * [ ] Define `subnet` focus identifier format (recommend: CIDR string unless/until a `subnets` table exists).
* Core-go implementation
  * [ ] Return devices observed in that subnet, bounded and deterministic.
* Tests
  * [ ] Validate 400 for invalid CIDR and 404 vs empty semantics (choose and document).

Blockers:

* If we later introduce a `subnets` table, we must preserve backwards compatibility or version the focus id.

Acceptance criteria:

* Subnet focus deep link renders a consistent region with member devices.

#### M14.4 — Inspector payload (no extra round trips)

Tasks:

* Contract
  * [ ] Define an `inspector` block that is render-ready (identity/status/relationships).
* Core-go implementation
  * [ ] Populate inspector from existing tables in the same request.
* UI
  * [ ] Render inspector from the projection response only (no additional fetches for v1).

Blockers:

* Requires agreement on what is “must-have” vs “nice-to-have” in inspector.

Acceptance criteria:

* UI can render the inspector reliably from `MapProjection.inspector` alone.

Acceptance criteria:

* No endpoint returns a “whole network” graph.
* Responses are stable and diff-friendly (sorted output, stable IDs).
* A single focused device can render a subnet region view that resembles the L3 mock: a few regions, nodes inside, minimal connectors.

---

## Phase 15 — Layer implementations (Physical / L2 / L3)

**Status:** Planned

### Goal

Ship the first three layers that map directly to discovered data, matching the mock mental model.

Layers:

* **Physical**: devices + `links` (manual first), optional interface-level drilldown in Inspector.
* **L2 (VLANs)**: VLAN regions + device/interface membership (start with PVID; add trunk/tagged later).
* **L3 (Subnets)**: subnet regions + device membership based on IPs; show routing devices as “connectors”.

### Tasks

* Canvas renderer with “stacked regions” (soft rounded containers) and node placement per region.
* Minimal, deterministic layout rules first (avoid force-graph chaos); semantic zoom deferred.
* Inspector shows identity/status/relationships and “View in …” links between layers.

### Blockers

* Physical adjacency is manual-first unless LLDP/CDP enrichment is enabled (Phase 7).
* L2 accuracy depends on VLAN enrichment coverage (PVID exists today; trunks/tagged may require more data).

Deliverable:

* A user can click a device, view its L3 subnet relationships, and jump to Physical/L2 views via Inspector links.

Milestones (match mock intent before “smart” layout):

#### M15.1 — L3 “stacked regions” renderer (UI)

Tasks:

* Rendering
  * [ ] Render region containers (subnets) as rounded “stacked regions”.
  * [ ] Place nodes deterministically within regions (avoid force graphs in v1).
* Interaction
  * [ ] Hover highlights and labels-on-hover.
  * [ ] Click selects focus and updates Inspector.

Blockers:

* Depends on Phase 14 L3 projection being stable enough to render.

Acceptance criteria:

* L3 view resembles the mock philosophy: regions first, minimal connectors, readable.

#### M15.2 — Physical v1 projection + renderer

Tasks:

* Data model
  * [ ] Add `links` table via migration if/when Build mode editing begins (manual-first).
* API
  * [ ] Add `GET /api/v1/map/physical` projection (read-only first).
* UI
  * [ ] Render a small adjacency/tree view (no “everything at once”).

Blockers:

* Physical adjacency is manual-first unless LLDP/CDP enrichment lands.

Acceptance criteria:

* A curated set of links produces a stable, readable physical view.

#### M15.3 — L2 v1 projection + renderer

Tasks:

* API
  * [ ] Add `GET /api/v1/map/l2` projection.
  * [ ] Use PVID-only membership from `interface_vlans` first.
* UI
  * [ ] Render VLAN regions with membership.

Blockers:

* L2 accuracy depends on VLAN enrichment coverage.

Acceptance criteria:

* L2 view shows VLAN groupings without implying trunks/tagged membership yet.

#### M15.4 — Cross-layer navigation (Inspector)

Tasks:

* UI
  * [ ] Preserve focus when switching layers where possible.
  * [ ] Define fallbacks when a focus type does not exist in the target layer.

Blockers:

* Requires consistent focus identifiers across layers.

Acceptance criteria:

* “View in …” actions feel predictable and don’t strand the user.

Acceptance criteria:

* L3 view shows subnet regions and device membership; no edge explosion.
* Physical view shows a small adjacency view that can be edited/curated manually.
* L2 view shows VLAN membership based on existing `interface_vlans` PVID facts.
* The inspector allows jumping between layers without losing context.

---

## Phase 16 — Layer implementations (Services / Security) + modes

**Status:** Planned

### Goal

Add the two “meaning” layers and the product modes from the spec without mixing layers.

Layers:

* **Services**: service nodes grouped by host; dependencies optional and can start as user-entered.
* **Security**: zones as regions; policies/flows as edges (manual first).

Modes (top bar in the spec):

* **Explore**: read-only, minimal chrome
* **Build**: allows editing links/regions/tags (writes to metadata tables via Go API)
* **Secure**: focuses on zones/policies only (hides unrelated objects)
* **Operate**: overlays status + last-seen + changes (pairs with Phase 9/10 diffing)

Deliverable:

* Services and Security layers exist with a clean inspector-driven workflow, plus initial mode gating.

### Blockers

* Operate overlays depend on Phase 9/10 history + change feed being available.
* Services dependencies and security policies start manual-first; needs clear data model + UX so operators don’t create inconsistent truth.

Milestones:

#### M16.1 — Modes UI (Explore / Build / Secure / Operate)

Tasks:

* UI
  * [ ] Add a top bar mode selector.
  * [ ] Define what each mode changes (actions enabled + rendering density).
* Safety
  * [ ] Ensure mode does not bypass layer separation (no blending layers).

Blockers:

* Build and Operate modes depend on Phase 11 auth/roles to be safe by default.

Acceptance criteria:

* Mode switching is instant, URL-deep-linkable (recommended), and deterministic.

#### M16.2 — Services v1 (projection + renderer)

Tasks:

* API
  * [ ] Add `GET /api/v1/map/services` projection using existing `services` facts.
* UI
  * [ ] Render services grouped by host/device.
* Data model (optional, later)
  * [ ] If adding dependencies, model as explicit edges (manual-first) and gate writes behind Build mode.

Blockers:

* Active port discovery (Phase 7) changes the completeness of this layer; keep it best-effort and clearly labeled.

Acceptance criteria:

* Services view remains readable and bounded; no attempt at “full dependency graph” in v1.

#### M16.3 — Security v1 (zones + policies)

Tasks:

* Data model
  * [ ] Add `zones` (+ membership) migrations when Build mode editing begins.
* API
  * [ ] Add `GET /api/v1/map/security` projection.
* UI
  * [ ] Render zones as regions; show only zone-level edges.

Blockers:

* Needs a clear UX contract so operators don’t create inconsistent truth.

Acceptance criteria:

* Security layer is zone-first and intentionally simplified.

#### M16.4 — Operate overlays (history-aware)

Tasks:

* Data
  * [ ] Use Phase 9 APIs to power “last seen” and “changed” overlays.
* UI
  * [ ] Add overlays without turning the map into a monitoring dashboard.
  * [ ] Provide a clear legend and allow toggling overlays.

Blockers:

* Depends on Phase 9 history/change feed being implemented and performant.

Acceptance criteria:

* Operate mode can show change/recency without visual noise.

Acceptance criteria:

* Explore is clean/read-only.
* Build can author truth (links/zones/dependencies) via Go APIs (no UI DB access).
* Secure hides non-security objects.
* Operate can overlay state without turning into a monitoring dashboard.

---

## Open decisions (network map)

* **Renderer**: SVG (simple, accessible) vs Canvas/WebGL (performance) vs React Flow (speed of delivery).
* **Layout strategy**: deterministic region layout (recommended for v1) vs force-directed.
* **Editing model**: what becomes user-authored truth (`links`, `zones`, `service deps`) vs discovered truth.
* **Search/focus**: how users pick the first object (global search, devices list, or “pick subnet”).
* **Time axis**: whether the map timeline is built directly on Phase 9 observations/events or starts as “last seen” only.

Related decision support:

* `docs/network_map/implementation-stack.md` — curated libraries/tools to avoid reinventing graph/layout/type plumbing.

Suggested defaults (to keep v1 boring and shippable):

* Renderer: **SVG first** (predictable, inspectable, accessible); consider Canvas/WebGL only when performance demands it.
* Layout: **deterministic region layout** (no force-directed graph in v1).
* Editing: manual truth for `links`/`zones`/`service_deps`, discovered truth for device/interface/IP/service facts.
* Time axis: collapsed placeholder that starts with “last seen” (timeline scrubber comes after Phase 9 APIs exist).

Preferred off-the-shelf picks (so we don’t build our own plumbing):

* Map UI: SVG-first; if we need “batteries included”, use `reactflow` + `elkjs` (or `dagre`) for deterministic layout
* API types/client: `openapi-typescript` + `openapi-fetch` (or `orval`) to avoid hand-written DTO drift
* Data fetching/cache: `@tanstack/react-query` keyed by `(layer, focusType, focusId)`
* Runtime validation: `zod` (especially for projection payloads)
* UI primitives: `cmdk` (command palette), `Fuse.js` (search), `floating-ui` (tooltips/popovers)

---

## What you are *explicitly not* building

* A custom web server
* A custom database
* A custom auth system
* A custom reverse proxy
* A JS backend doing network scanning

## Strong advice (based on experience)

1. **Write the Go API as if Node doesn’t exist**
2. **Never let Node touch the DB**
3. **Use boring tech everywhere**
4. **Expose APIs early, even if unused**

If you want next:

* Push time/history into the product (Phase 9), then wire the operator UX on top (Phase 12).
* Ship auth hardening + operations (Phases 10–11) before exposing anything beyond a trusted network.
* Start the map track with the UI shell contract (Phase 13), then pin the projection schema (M14.1) and ship L3 first (M14.2).

## Next milestone checklist

* [x] M9.1 — `GET /api/v1/devices/changes` (change feed)
* [x] M9.2 — `GET /api/v1/devices/{id}/history` (device timeline)
* [x] M9.3 — discovery run listing + logs (`/api/v1/discovery/runs...`)
* [x] Phase 10 — auth + sessions + roles
* [x] Phase 11 — metrics + runbooks + CI smoke
* [ ] Phase 12 — operator UX foundations + workflows
* [ ] M13.1 — `/map` route + 3-pane shell (mock data OK)
* [ ] M14.1 — `MapProjection` schema pinned in `api/openapi.yaml`
* [ ] M14.2 — L3 projection (device focus) from live data

## Definition of done for discovery (Phases 8-10)

* Discovery endpoints return real run ids and status, and populate device/interface/service data.
* Device history/diffs are visible via API and UI (timeline).
* Operators can trigger discovery and edit metadata without CLI access.

## Definition of done for network map (Phases 13-16)

* The UI matches the mock interaction contract: constant 3-pane layout, mutually-exclusive layers, object-first rendering.
* L3 view can render subnet regions + device membership from real data, and Inspector can cross-navigate between layers.
* Physical/L2 views work with a minimal, deterministic layout; no uncontrolled “spaghetti graph” screens.
* Services/Security views can start manual-first, but are served via projection endpoints (no direct DB access from UI).
