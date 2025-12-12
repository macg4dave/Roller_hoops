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
* **Database**: PostgreSQL
* **Auth**: built-in

---

## Service layout (final target)

```
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
│   └─ PostgreSQL or MySQL
│
└─ volumes/
    └─ persistent data
```

---

# Roadmap

## Phase 0 — Foundations

* Pick DB: **PostgreSQL** 
* Pick reverse proxy: **Traefik**
* Pick API style: **REST over HTTP**

---

## Phase 1 — Go core service (headless)

**Do not build a UI yet**

### Responsibilities

* Normalisation
* Persistence
* API

### Components (existing, proven)

* HTTP server: Go stdlib
* DB access: `sqlc` or `gorm`
* Migrations: `golang-migrate`
* Config: env vars only (Docker-friendly)
* Logging: `zap` or `zerolog`

### API (example)

```
GET    /api/devices
GET    /api/devices/{id}
POST   /api/devices
PUT    /api/devices/{id}

POST   /api/discovery/run
GET    /api/discovery/status
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

**Node does not touch the network**

### Stack

* Node.js + TypeScript
* Framework: **Next.js** or **Fastify + SSR**
* UI: plain HTML + Tailwind *or* React (your choice)
* API client: typed fetch

### Responsibilities

* Forms
* Editing metadata
* Filtering, grouping
* Live view (polling or WebSockets)

### Auth

Start simple:

* Local users
* Session cookies

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
* Handles auth later if needed

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

* Prometheus metrics
* SNMP enrichment
* VLAN / switch port mapping
* Export to NetBox
* Read-only LDAP auth
* Multi-site support

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

* I can turn this into a **Git repo layout**
* Write a **docker-compose.yml skeleton**
* Define the **initial DB schema**
* Sketch the **API contract** so it never breaks later

## Furture work

* Discovers and tracks devices
* Decide discovery scope:

  * ARP
  * ICMP ping
  * SNMP (read-only)
  * mDNS / NetBIOS (optional)

* Network discovery workers

### Discovery model

* Workers run on timers
* Results deduplicated
* State transitions tracked (online/offline/changed)

Deliverable:

* Go service runs alone in Docker
* Can curl it and get real data