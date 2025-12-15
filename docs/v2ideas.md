# Phase 7–8 step-by-step build list

This file is a working checklist for implementing the Phase 7 (“nice-to-have integrations”) and Phase 8 (“discovery engine v1”) items from `docs/roadmap.md`.

## Phase 8 — Discovery engine v1 (network scanning)

### Step 1 — Observations (IP/MAC) + fold into current state
- [x] Add `ip_observations` + `mac_observations` tables (append-only per run).
- [x] Worker writes observations for every discovered IP/MAC.
- [x] Worker upserts “current state” into `ip_addresses` / `mac_addresses` (already exists).

### Step 2 — Scope limits + runtime budget
- [x] Refuse huge scopes by default (e.g. >1024 targets) unless explicitly overridden.
- [x] Record scope + limits in `discovery_runs.stats`.
- [x] Make the worker runtime/ping budgets configurable via `DISCOVERY_*` env vars and apply backoff when claim/run fails.

### Step 3 — ICMP sweep (best-effort in Docker)
- [x] For an explicit scope, run a bounded ICMP “ping sweep” to stimulate ARP population.
- [x] Degrade gracefully if ICMP is unavailable (no `ping`, missing caps).
- [x] Ship `ping` in the `core-go` image (still best-effort depending on container caps/policy).

### Step 4 — SNMP read-only enrichment (optional, Phase 7/8 overlap)
- [x] Decide approach: Go SNMP library (`gosnmp`) + stub client under `core-go/internal/enrichment/snmp` to capture config, target discovery, and future data mapping.
- [x] Store enrichments in normalized columns (sysName/sysDescr etc.) + link to interfaces (schema/migration work pending).

  > Implemented: `core-go/migrations/006_enrichment_snmp_names_vlans.up.sql` adds `device_snmp` plus SNMP-derived interface columns and uniqueness on `(device_id, ifindex)`.

  > Implemented: discovery worker can optionally query SNMP (`DISCOVERY_SNMP_ENABLED=true`) to populate `device_snmp`, create/update `interfaces`, and associate interface MACs.

## Phase 7 — Nice-to-have integrations

### Step 5 — mDNS / NetBIOS name resolution

- [x] Decide approach: start with Go's `net`/`net/mdns` exploration; scaffolding lives in `core-go/internal/enrichment/mdns` to surface resolved candidate names.
- [x] Store “friendly name” candidates and a chosen display name strategy (schema + UI integration to follow once names settle).
- [x] Resolve names via reverse DNS, mDNS reverse lookups, and NetBIOS node-status so operators see richer name candidates before picking a display name.

  > Implemented: `device_name_candidates` stores candidates (reverse DNS + SNMP sysName today) and the worker auto-sets `devices.display_name` only if unset.

  > Implemented: discovery worker now also captures mDNS and NetBIOS candidates before falling back to SNMP in `core-go/internal/enrichment/mdns`.

  > Implemented: UI shows candidates in the device detail pane and can apply one as the display name (`ui-node/app/devices/DeviceNameCandidatesPanel.tsx`).

### Step 6 — VLAN / switch port mapping

- [x] Add `core-go/internal/enrichment/vlan` stub that outlines VLAN/switch-port mapping plans and the bridge-MIB dependency.
- [x] Populate via SNMP bridge-MIB where available (the walk helper will reuse the SNMP enrichment client once the schema is final).

  > Implemented: worker uses bridge/q-bridge tables (`dot1dBasePortIfIndex` + `dot1qPvid`) to populate `interface_vlans` (role=`pvid`, source=`snmp`) when SNMP is enabled.

  > Stored in: `core-go/migrations/006_enrichment_snmp_names_vlans.up.sql` (`interface_vlans`).

### Step 7 — Import/export JSON (UI-owned workflow)

- [x] Define export format (versioned) and document it; current JSON round-trips existing `Device` resources.
- [x] Add core-go endpoints to export/import (and UI download/upload controls) — done, tracked via `core-go/httpapi` and `ui-node/app/devices/ImportExportPanel.tsx`.
- [x] UI download/upload workflow.

### Step 8 — LLDP/CDP adjacency (SNMP; best-effort)

- [x] Add `links` table to store physical adjacency with `source=lldp|cdp`.
- [x] Collect neighbors via SNMP LLDP-MIB / CISCO-CDP-MIB (best-effort; behind allowlists) and upsert into `links`.

### Step 9 — Service/port discovery (active scan; opt-in)

- [x] Add optional `nmap`-driven scan stage (connect scan + XML parsing) behind explicit enable flags and CIDR allowlists.
- [x] Upsert open ports into `services` with `source=nmap` and `observed_at`.

### Step 10 — External inventory import (IPAM/inventory feeds)

- [x] Add wrapped payload import endpoints for NetBox/Nautobot and map into `devices`, `device_metadata`, and `ip_addresses`.

## Phase 9 — Historical/diffing APIs

- [x] Aggregate `ip_observations`, `mac_observations`, metadata edits, and services into a stable `events` feed.
- [x] Expose `GET /api/v1/devices/changes` and `GET /api/v1/devices/{id}/history` with cursoring and limits.
- [x] Surface `GET /api/v1/discovery/runs`, `/runs/{id}`, and `/runs/{id}/logs` so operators can inspect runs + logs via API.
  
Implemented: change feed + history now power the device detail timeline, while the run/log endpoints support run inspection without CLI access.
