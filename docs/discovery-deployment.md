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
  - On **Docker bridge** networking, `/proc/net/arp` only contains entries for the Docker-internal network (gateway, other containers). Real device MACs are not visible because all external traffic routes through Docker's NAT.
  - When ARP is not available, the discovery worker **falls back to ping-based discovery** (IP-only, no MACs). This still discovers devices by IP and allows SNMP/DNS enrichment, but cannot identify hardware-level identity.
- **SNMP** is just UDP (no raw sockets), but you still need L3 reachability to targets.

## Windows / Docker Desktop limitations

Docker Desktop for Windows (and macOS) runs containers inside a Linux VM (WSL2 or Hyper-V). This means:

- **`network_mode: host`** connects to the **VM's** network, not the Windows host network. It does not help with real-network discovery.
- The container's `/proc/net/arp` will never show your real LAN neighbors.
- Ping sweep can still reach targets if Docker's NAT routes to them, and the discovery worker will fall back to ping-based device creation (IP-only, no MAC).

### Recommended approach for Windows

1. **Set `DISCOVERY_DEFAULT_SCOPE`** in `.env` to your real subnet (e.g., `192.168.1.0/24`). This ensures the ping sweep targets real network hosts instead of only using ARP.
2. **Enable SNMP** (`DISCOVERY_SNMP_ENABLED=true`) so enrichment can gather device names, interfaces, and VLANs even without MAC-level ARP visibility.
3. **Accept IP-only discovery**: devices will be discovered by responding IP address. MAC addresses will not be available unless a future dedicated scanner component runs natively on the host.
4. **Alternative**: run `core-go` natively on the Windows host (outside Docker) while keeping Postgres and the UI in Docker. This gives full ARP + ICMP fidelity but requires a local Go toolchain.

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
- Recommended safety: set `CORE_GO_HTTP_ADDR=127.0.0.1:8081` so the unauthenticated Go API binds only to loopback.

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

## Deployment smoke matrix

Use these checks when validating a new deployment mode or debugging "discovery found nothing" reports. Replace `<CIDR>` with a narrow allowed scope before running any command.

### Docker bridge smoke

This is the default compose mode and the safest first smoke because `core-go` stays private on the Docker network.

```powershell
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
docker compose --profile dev up -d --build
```

Trigger a scoped fast run from the UI Discovery panel, or from inside the private `core-go` container:

```powershell
docker compose exec -T core-go wget -qO- --header='Content-Type: application/json' --post-data='{"scope":"<CIDR>","preset":"fast"}' http://127.0.0.1:8081/api/v1/discovery/run
docker compose exec -T core-go wget -qO- http://127.0.0.1:8081/api/v1/discovery/status
docker compose logs --tail 100 core-go
```

Expected result:

- The run reaches `succeeded` or fails with an actionable message.
- `ping sweep: attempted=...` appears when a scope is provided.
- Real LAN MAC addresses are not expected. If ping finds devices but ARP cannot see them, logs should include `ARP found no in-scope devices; falling back...` and created devices will be IP-only.

### Linux host-network smoke

Use this when ARP/MAC visibility matters and the Docker host is Linux.

```powershell
$env:CORE_GO_HTTP_ADDR="127.0.0.1:8081"
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml --profile dev up -d --build
```

In a POSIX shell, the equivalent start command is:

```sh
CORE_GO_HTTP_ADDR=127.0.0.1:8081 DISCOVERY_DEFAULT_SCOPE=<CIDR> \
  sudo docker compose -f docker-compose.yml -f docker-compose.hostnet.yml --profile dev up -d --build
```

Trigger and inspect directly through the loopback-bound Go API:

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8081/api/v1/discovery/run -ContentType 'application/json' -Body '{"scope":"<CIDR>","preset":"fast"}'
Invoke-RestMethod -Uri http://127.0.0.1:8081/api/v1/discovery/status
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml logs --tail 100 core-go
```

Expected result:

- The run reaches `succeeded`.
- Logs include `scope targets`, `ping sweep`, `arp scrape`, and `discovery run completed`.
- `arp scrape` can only report what the Linux host can see in its neighbor/ARP table. Generate normal traffic to known targets first if the table is empty.

### Native-host smoke

Use this when testing outside Docker or when Docker Desktop cannot provide the needed network namespace. This path requires a local Go toolchain or prebuilt `core-go` binary.

Start Postgres and migrations with a host port for the native process:

```powershell
docker compose -f docker-compose.yml -f docker-compose.hostnet.yml up -d db migrate
```

Run `core-go` natively from the repo:

```powershell
Set-Location core-go
$env:DATABASE_URL="postgres://postgres:postgres@127.0.0.1:15432/roller_hoops?sslmode=disable"
$env:HTTP_ADDR="127.0.0.1:8081"
$env:DISCOVERY_DEFAULT_SCOPE="<CIDR>"
go run ./cmd/core-go
```

In another shell, trigger and inspect:

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8081/api/v1/discovery/run -ContentType 'application/json' -Body '{"scope":"<CIDR>","preset":"fast"}'
Invoke-RestMethod -Uri http://127.0.0.1:8081/api/v1/discovery/status
```

Expected result:

- The run reaches `succeeded` with host-level routing and ARP behavior.
- If logs say `ping not permitted (missing CAP_NET_RAW?)`, fix host ping permissions or run in a mode where ICMP is permitted.
- If logs say `ping not found in PATH`, install/provide `ping` for the native runtime.

### Optional enrichment smoke

Only run this against explicitly allowed targets with known credentials:

```powershell
$env:DISCOVERY_SNMP_ENABLED="true"
$env:DISCOVERY_SNMP_COMMUNITY="<community>"
$env:DISCOVERY_TOPOLOGY_ALLOWLIST="<CIDR>"
```

Use the `snmp` or `topology` scan tag from the UI. Expected logs include `enrichment: targets=... snmp_ok=...`. If `snmp_ok=0`, check UDP/161 reachability, credentials, target SNMP support, and allowlists before changing code.

For port-scan smoke, keep active scanning disabled unless the target is explicitly approved:

```powershell
$env:DISCOVERY_PORT_SCAN_ENABLED="true"
$env:DISCOVERY_PORT_SCAN_ALLOWLIST="<CIDR>"
$env:DISCOVERY_PORT_SCAN_PORTS="22,80,443"
```

Expected logs include either `port scan: targets=...` or a clear `port scan skipped: ...` reason.

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
