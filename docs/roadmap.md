# Roadmap

Here’s a **clean, realistic roadmap** that explicitly **reuses existing, proven components**, runs fully in **Docker**, and keeps Go and Node doing what they’re best at. No reinventing wheels.

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

- Device CRUD live end-to-end via Go API and UI; metadata fields (`owner`, `location`, `notes`) persist in `device_metadata`.
- Discovery endpoints now persist runs/logs (`discovery_runs`, `discovery_run_logs`) and return real run ids; UI can trigger a run and see the latest status (stubbed worker for now).
- Traefik + docker-compose bring up UI, API, and PostgreSQL with health checks enabled.

---

## Phases

## Phase 0 — Foundations

* Pick DB: **PostgreSQL**
* Pick reverse proxy: **Traefik**
* Pick API style: **REST over HTTP**

---

## Phase 1 — Go core service (headless)

### Constraint: Do not build a UI yet

### Responsibilities

* Normalisation
* Persistence
* API

### Components (existing, proven)

* HTTP server: Go stdlib (`net/http`) + `chi` router
* API contract: OpenAPI is canonical; Go server stubs are generated (no hand-rolled drift)
* DB access: `sqlc` (PostgreSQL-first) + `pgx`
* Migrations: `golang-migrate`
* Config: env vars only (Docker-friendly)
* Logging: `zerolog` (JSON logs)

Operational defaults (v1):

* Strict JSON decoding for requests (reject unknown fields)
* Health endpoints: `/healthz` and `/readyz`
* Request ID propagation end-to-end (UI → Traefik → Go)

### API (example)

```text
GET    /api/v1/devices
GET    /api/v1/devices/{id}
POST   /api/v1/devices
PUT    /api/v1/devices/{id}

POST   /api/v1/discovery/run
GET    /api/v1/discovery/status
```

---

## Phase 2 — Database schema (minimal but future-proof)

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
* Historical data works

---

## Phase 3 — Node.js UI service

### Constraint: Node does not touch the network

### Stack

* Node.js + TypeScript
* Framework: **Next.js** (SSR-first)
* UI: plain HTML forms + Tailwind (progressive enhancement; avoid heavy client state early)
* API client: typed `fetch` + generated TypeScript types from OpenAPI

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

---

## Phase 4 — Reverse proxy & routing

Use **existing battle-tested infra**

### Traefik

* Routes `/` → Node
* Routes `/api` → Go
* Handles TLS
* Does not replace application auth in v1 (auth remains a UI concern)

Deliverable:

* Single hostname
* Clean separation

---

## Phase 5 — Docker polish

### Docker rules

* One process per container
* No shared state except DB
* All config via env vars

### docker-compose

* Profiles for dev / prod
* Named volumes for DB
* Health checks

Deliverable:

* `docker compose up`
* System survives reboot

---

## MVP exit criteria (Phases 0-5)

* docker compose brings up Traefik, UI, Go API, and PostgreSQL with named volumes; health checks pass.
* Device CRUD flows work end-to-end (UI → API → DB) with request IDs in logs.
* OpenAPI spec lives in the repo; Go server + Node client are generated from it.
* Migrations are repeatable (`golang-migrate`), and a short doc exists for running them in dev/CI.
* Minimal tests run in CI: Go handler test against Postgres, and a UI smoke that hits the real API.

---

## Phase 6 — Live updates (optional)

Two clean options:

### Option A — Polling

* UI polls `/api/devices`
* Simple
* Good enough for most networks

### Option B — WebSockets

* Go emits events
* Node fans out to browsers

Still no tight coupling.

---

## Phase 7 — Nice-to-have integrations (later)

Only after core is stable.

* SNMP enrichment
* VLAN / switch port mapping
* Export to NetBox
* Read-only LDAP auth
* Multi-site support
* Extra Prometheus exporters/dashboards for network-specific metrics (beyond baseline ops)

---

## Phase 8 — Discovery engine v1 (network scanning)

Goals:

* Decide discovery scope: start with ARP table scrape + ICMP ping; add read-only SNMP once the loop is solid.
* Job model in Postgres: `discovery_runs` (id, scope, status, started_at, completed_at, stats JSON) and `discovery_run_logs` (structured log lines).
* Worker loop in Go: timer-based plus manual trigger; single worker at first; cancellation and backoff on failure.
* Observations table (append-only) to store IP/MAC/service findings per run; dedupe by stable keys when folding into current state.
* Wire `/api/v1/discovery/run` to enqueue and return a run id; `/api/v1/discovery/status` returns latest run, progress, and last error.

Deliverable:

* Go service runs alone in Docker, performs a subnet sweep, populates devices/interfaces with timestamps, and returns real discovery status.

---

## Phase 9 — Historical state + diffing

* Append-only observation log keyed by run id (`device_observations`, `interface_observations`, `service_observations`) with `observed_at`.
* Derived "current" tables/views updated from latest observation; keep previous snapshot addressable.
* State transitions captured (`online`, `offline`, `changed`) with an events table and run id references.
* Retention knobs: keep raw observations for N days; keep rollups forever; indexes to keep queries fast.
* API: list devices changed since a timestamp; fetch the history for a device; expose last run status and error.

Deliverable:

* You can diff any device across runs and show a timeline of changes without hand-written SQL.

---

## Phase 10 — UI workflows for operators

* Device list: filters (online/offline/changed), search by display name/IP/MAC, sort by last seen.
* Device detail: IPs, MACs, interfaces, services, metadata, change timeline; deep linkable.
* Metadata editing: inline forms with optimistic UI and rollback on failure; uses typed client generated from OpenAPI.
* Discovery UX: trigger a run, show queued/running/done with progress and errors; reuse polling/WebSocket choice from Phase 6.
* Error and resilience: empty states, loading states, and friendly failure surfaces in UI.

Deliverable:

* An operator can sign in, launch discovery, watch progress, inspect a device, and edit metadata without using curl.

---

## Phase 11 — Auth + session hardening

* Auth stays in the UI layer: local users stored in a dedicated schema/table owned by the UI service (still separate from core data access).
* Passwords hashed (argon2id or bcrypt), session cookies signed/encrypted, CSRF protection on form posts.
* Roles: admin vs read-only; authorization enforced in the UI layer before calling the Go API.
* Account lifecycle: change password flow and a manual admin reset/one-time token flow for recovery.
* Audit: minimal audit log of user actions stored via the Go API so the UI never writes to core tables directly.

Deliverable:

* Auth no longer relies on stubs; sign-in/out and role-based access work end-to-end with secure cookies and hashed credentials.

---

## Phase 12 — Observability & operations

* Metrics: Prometheus endpoints from Go (HTTP + DB latency, discovery duration) and Traefik access logs/metrics.
* Tracing/logging: request IDs end-to-end; structured logs on stdout; optional OpenTelemetry spans for discovery runs.
* Runbooks: `docker compose` snippets for backup/restore (`pg_dump`), migrations (`golang-migrate up`), rotating secrets, and seeding a dev stack.
* Testing: CI jobs for Go unit/integration (with Postgres container), UI smoke against `next dev` + API, and a contract test to keep OpenAPI in sync.
* SLOs: health endpoints monitored by Traefik + simple uptime check; webhook/email alert stubs.

Deliverable:

* Operators have metrics, logs, backups, and tests so the stack can be run by someone who did not write it.

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

* Lock the OpenAPI spec (devices, discovery runs, observations) and regenerate the Go/TS clients from it.
* Add migrations + sqlc queries for `discovery_runs` and observations (Phase 8 prep) with a happy-path handler test.
* Add a docker-compose dev profile with seeded data and CI health checks (lint + tests + migrate).
* Ship the first UI smoke (create/list device) hitting the Go API to guard regressions.

## Open decisions before Phase 8

* Default discovery scope order (ARP + ICMP first; SNMP once stable; mDNS/NetBIOS optional).
* Run cadence and max runtime budget per subnet.
* Retention window for raw observations vs rollups; defaults for trimming old runs.

## Definition of done for discovery (Phases 8-10)

* Discovery endpoints return real run ids and status, and populate device/interface/service data.
* Device history/diffs are visible via API and UI (timeline).
* Operators can trigger discovery and edit metadata without CLI access.
