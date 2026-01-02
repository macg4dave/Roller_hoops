# Discovery deployment plan

This document explains how to deploy the **discovery worker** safely and effectively when Roller_hoops runs in Docker.

The discovery engine needs to observe the network (ARP/ICMP/SNMP, etc). In containerized environments, that is **mostly a networking + privileges question**, not an application question.

For a quick “what works where” summary, see [docs/discovery-capabilities.md](discovery-capabilities.md).

## Goals

- Make discovery work on real networks without redesigning the app.
- Keep `core-go` **headless** and private behind Traefik/UI.
- Avoid shipping a container that requires broad privileges by default.
- Keep scope targeting explicit so discovery doesn’t accidentally scan “the internet”.

## Key constraints

- **ICMP ping** generally requires raw sockets.
  - In Linux containers that usually means `CAP_NET_RAW`.
  - Some environments disallow raw sockets entirely (managed Kubernetes, hardened Docker daemon).
- **ARP table scraping** is easiest when the process can see the host network namespace.
  - In Docker, that generally means `network_mode: host` (Linux only) or running discovery on a host/VM directly.
- **SNMP** is just UDP (no raw sockets), but you still need L3 reachability to targets.

## Recommended deployment patterns

### Option A — “Discovery runs on the host network” (simplest, Linux)

Run `core-go` with host networking so it sees the same routing table, ARP cache, and interfaces as the host.

Pros:

- Best fidelity for ARP + ICMP.
- Lowest friction for local lab deployments.


Cons:

- `network_mode: host` is Linux-only.
- The container shares the host network namespace (treat as sensitive).


Operational notes:

- Keep `core-go` **not published** to the internet even though it’s on host networking.
- Traefik should continue to expose only the UI, and the UI calls `core-go` over loopback/host network.

Compose example (local dev):

- `docker-compose.hostnet.yml` shows a working “host network” pattern for Linux.
- Bring it up with: `sudo docker compose -f docker-compose.yml -f docker-compose.hostnet.yml up --build`

### Option B — “Dedicated scanner container / sidecar” (recommended for production)

Deploy a dedicated **scanner** on each target network segment (VM/container) with the required reachability and privileges. That scanner runs `core-go` (or a future dedicated scanner binary) and writes observations to Postgres.

Pros:

- Scales to multiple sites/subnets.
- Least risky operationally: the scanner sits where it belongs.


Cons:

- Requires a deployment decision per network.


Operational notes:

- Treat scanner nodes like network tooling.
- Consider firewall rules to restrict what the scanner can reach.

### Option C — “Grant minimal capabilities to core-go” (ICMP only)

Grant `CAP_NET_RAW` (and only that) to the `core-go` container.

Pros:

- Enables ICMP without host networking.


Cons:

- Still depends on Docker daemon policy.
- Does not help with ARP scraping unless the container can see relevant interfaces.

## Safety & scope controls (must-haves)

- Discovery must be **explicitly scoped**.
  - The API already accepts an optional `scope` hint when triggering a run.
  - Deployment should standardize how scope is provided (UI input + server-side defaults).
- Rate limits / timeouts.
  - All network calls must be bounded by context timeouts.
  - Discovery runs should have a max runtime budget.
- Least privilege.
  - Prefer SNMP/TCP checks where possible.
  - Add raw-socket capabilities only when required.

## What “done” looks like

A deployment is considered correct when:

- The UI can trigger a discovery run.
- The discovery worker transitions `queued → running → (succeeded|failed)`.
- The worker can reach the intended subnet targets with the chosen method.
- `core-go` is still not directly exposed to browsers/public networks.

## Open follow-ups

- Decide the default discovery method order (ARP → ICMP → SNMP) per environment.
- Decide whether production uses host networking (Option A) or dedicated scanners (Option B).
- Document any required Docker/Kubernetes manifests once the worker performs real scanning.
