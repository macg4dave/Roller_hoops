# Architecture

## System overview

Roller_hoops is a **Go + Node.js + PostgreSQL** system that runs fully in Docker.

- **Traefik** routes traffic
  - `/` → **ui-node** (Node.js / Next.js)
- **core-go** is not exposed directly to browsers; it stays private on the Docker network.
- **core-go** owns network discovery, normalisation, persistence, and the REST API.
- **ui-node** owns HTML rendering, user workflows, and authentication UI/sessions.
- **PostgreSQL** is the only database.

## Service responsibilities

### core-go (Go)

Owns:

- Network discovery
- Polling/scheduling
- Normalisation
- Database access
- REST API (`/api/v1/...`)
- WebSockets (later; optional)

Forbidden:

- HTML rendering
- UI workflows
- Direct user interaction
- Accessing UI session state

### ui-node (Node.js / Next.js)

Owns:

- UI rendering (SSR-first)
- Forms and workflows
- Authentication UI and sessions
- Calling the Go API

Forbidden:

- Network scanning
- Direct database access
- Re-implementing Go business rules

## Routing and trust boundaries

- Only Traefik publishes ports to the host.
- `core-go` and `db` are private to the internal Docker network.
- `ui-node` is the only component that accepts browser sessions.

This makes **ui-node the BFF** (backend-for-frontend): browser traffic only hits the UI, and the UI performs server-side calls to the Go API via `CORE_GO_BASE_URL` (wired in Docker Compose to a Traefik internal-only entrypoint).

## API contract source of truth

The API contract is defined in OpenAPI:

- Canonical spec: `api/openapi.yaml`
- Documentation: `docs/api-contract.md`

Generated code (Go server stubs, TypeScript types) is **derived** from OpenAPI and should not diverge.

## Configuration

- All configuration is via **environment variables**.
- No secrets are committed to this repository.

## Discovery deployment

Discovery (ARP/ICMP/SNMP, etc.) depends on Docker networking and container capabilities. Operational guidance lives in:

- `docs/discovery-deployment.md`

## Observability

Baseline expectations:

- Structured JSON logs across services
- A request ID is propagated end-to-end
- Health endpoints:
  - `GET /healthz` (liveness)
  - `GET /readyz` (readiness)
