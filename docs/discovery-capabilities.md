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
| ICMP ping sweep | partial | partial | partial | partial |
| SNMP polling (UDP/161) | partial | partial | partial | partial |
| Reverse DNS lookups | yes | yes | yes | yes |
| mDNS / NetBIOS name hints | partial | partial | partial | partial |
| TCP port scanning (e.g., `nmap`) | partial | partial | partial | partial |
| VLAN/interface enrichment (via SNMP) | partial | partial | partial | partial |

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
