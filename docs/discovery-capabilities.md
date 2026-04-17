# Discovery capabilities matrix

This document summarizes what parts of **network discovery** are expected to work under different deployment/networking models (Docker bridge vs host networking vs running on a host, etc.), and what privileges/tools each mode typically needs.

It complements:

- `docs/discovery-deployment.md` (deployment patterns + safety)
- `docs/roadmap.md` (project risks/blockers)

## Legend

- **yes**: should work reliably when correctly configured
- **partial**: can work, but depends on environment or has known limitations
- **no**: generally not possible in this deployment model

## Deployment models

| Model | Where `core-go` runs | Network namespace visibility | Typical use |
|---|---|---|---|
| Host (native) | Directly on a Linux host | Host interfaces/routes/ARP | Local dev, small installs |
| Docker (bridge) | Container on default bridge network | Container interfaces/routes/ARP | Safe default; least privilege |
| Docker (host network) | Container with `network_mode: host` (Linux) | Host interfaces/routes/ARP | High fidelity scanning on Linux |
| Dedicated scanner node | VM/container placed on target network | That node’s interfaces/routes/ARP | Production/multi-site and segmented networks |

## Capabilities

| Capability | Host (native) | Docker (bridge) | Docker (host network) | Dedicated scanner node |
|---|---|---|---|---|
| L3 reachability to target subnets | partial | partial | partial | partial |
| ARP-based discovery / ARP cache scrape | yes | no | yes | yes |
| Ping-based discovery (IP-only fallback) | yes | yes (with scope) | yes | yes |
| ICMP ping sweep | partial | partial | partial | partial |
| SNMP polling (UDP/161) | partial | partial | partial | partial |
| Reverse DNS lookups | yes | yes | yes | yes |
| mDNS / NetBIOS name hints | partial | partial | partial | partial |
| TCP port scanning (e.g., `nmap`) | partial | partial | partial | partial |
| VLAN/interface enrichment (via SNMP) | partial | partial | partial | partial |

## Smoke expectations

Use a deliberately small scope first, such as one known responsive host (`192.168.1.10/32`) or a narrow lab range. Do not use an entire site CIDR for a smoke check.

| Model | Smoke command path | Expected pass signals | Expected limitations |
|---|---|---|---|
| Host (native) | Run `core-go` on the host with `DISCOVERY_DEFAULT_SCOPE=<CIDR>` or trigger a scoped run with `{"scope":"<CIDR>","preset":"fast"}`. | Run reaches `succeeded`; logs include `scope targets`, `ping sweep`, `arp scrape`, and `discovery run completed`. | Requires local Go/binary runtime, DB access, host routing, and host ping permissions. |
| Docker (bridge) | Run the normal compose stack with `DISCOVERY_DEFAULT_SCOPE=<CIDR>`, then trigger a scoped run from the UI or inside the `core-go` container. | Run reaches `succeeded`; logs include `ping sweep`. If targets answer ping, IP-only devices can be created. | ARP/MAC visibility is not expected for real LAN devices; `ARP found no in-scope devices; falling back...` is expected when ping responders exist. |
| Docker (host network) | On Linux, run `docker-compose.hostnet.yml` with `CORE_GO_HTTP_ADDR=127.0.0.1:8081` and a narrow `DISCOVERY_DEFAULT_SCOPE`. | Run reaches `succeeded`; logs include `arp scrape` with real host-network ARP visibility when the host has neighbor entries. | Linux-only; the unauthenticated Go API must be bound to loopback or firewalled. |
| Dedicated scanner node | Run the same native or container smoke from a node placed on the target segment. | Same as host/native or host-network, but results should match that segment's routing and L2 visibility. | Each site/segment needs its own scope, credentials, and firewall allowance. |

### Notes on the “partial” rows

Most discovery capabilities are ultimately gated by **reachability** and **policy**, not just code:

- **Routing/firewalls**: if the runtime cannot route to (or is blocked from) a subnet, scans will fail.
- **Privileges**: ICMP often needs raw sockets; on Linux this usually means `CAP_NET_RAW` or running as root (depending on implementation).
- **Tooling**: if port scanning is implemented via an external binary, the runtime needs it installed (container image vs host packages).
- **Name sources**: mDNS/NetBIOS tend to be noisy, often blocked across VLANs, and vary by OS/network.

## Requirements checklist (by capability)

| Capability | Requirements (typical) |
|---|---|
| ARP | Must share the L2 broadcast domain and see the relevant ARP cache; easiest with host network namespace visibility (native or `network_mode: host`). |
| ICMP | Raw socket permission (`CAP_NET_RAW`) and ICMP allowed by target/network policy. |
| SNMP | UDP reachability to targets; credentials/communities; SNMP allowed by policy. |
| Port scan | Reachability + allowed by policy; `nmap` availability if used externally; timeouts and scope controls. |
| Reverse DNS | Working DNS resolution from the runtime; correct search domains / resolvers. |

## Run-log signals

Use the discovery run detail page or `/api/v1/discovery/runs/{id}/logs` to separate deployment limitations from application bugs:

- `scope targets: ...`: the scope parsed and passed the target cap.
- `ping not found in PATH`: install/provide ping in the runtime image or host.
- `ping not permitted (missing CAP_NET_RAW?)`: grant raw-socket permission or use a deployment mode where ping is permitted.
- `ping sweep: attempted=... succeeded=0`: ping ran, but no targets answered or policy blocks ICMP.
- `arp scrape: entries=... devices_seen=...`: ARP was readable; `devices_seen=0` can still be normal in Docker bridge mode.
- `ARP found no in-scope devices; falling back...`: Docker bridge/NAT behavior is likely; expect IP-only discovery.
- `enrichment: targets=... snmp_ok=0`: SNMP is disabled, blocked, miscredentialed, or unsupported by targets.
- `port scan skipped: ...`: active scanning is disabled, not allowlisted, `nmap` is missing, or there are no eligible targets.

## Recommended deployment choices (practical)

- If you want **high-fidelity ARP + better “what’s on my LAN”** results: prefer **Host (native)** or **Docker (host network)** on Linux.
- If you want **safer defaults** (least privilege) and accept lower fidelity: **Docker (bridge)** plus SNMP/DNS-based enrichment can still be useful.
- For **production** or segmented networks: deploy **Dedicated scanner nodes** per site/segment and write observations to Postgres.

## Security and scope (non-negotiables)

- Always require **explicit scope** (CIDRs/interfaces) for discovery runs; never “scan everything” by default.
- Put hard **timeouts** and a max runtime budget on runs.
- Prefer **least privilege**: only add raw-socket capabilities when required.

## Link targets

- Deployment patterns: [docs/discovery-deployment.md](discovery-deployment.md)
- Current blockers/risks: [docs/roadmap.md](roadmap.md)
