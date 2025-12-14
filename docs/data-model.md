
# Data model (PostgreSQL)

This document is the source of truth for the database schema.

Constraints:

- PostgreSQL only
- Migrations are mandatory (planned tool: `golang-migrate`)
- Avoid unstructured JSON unless explicitly justified

## Extensions

- `pgcrypto` (for `gen_random_uuid()`)

## General column conventions

Most tables should include:

- `id uuid primary key default gen_random_uuid()`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()` (maintained by application or trigger; TBD)

## Core tables (v1 scope)

These tables are listed in `docs/roadmap.md`.

### `devices`

Purpose: one row per discovered or manually-created device.

Minimum columns (v1):

- `id` (uuid)
- `display_name` (text, nullable)

Additional discovery-derived fields are **TBD** and will be added incrementally.

### `interfaces`

Purpose: network interfaces for a device.

Minimum columns (v1):

- `id` (uuid)
- `device_id` (uuid, foreign key → `devices.id`)
- `name` (text, nullable)
- `ifindex` (integer, nullable)

Enrichment-derived columns (Phase 7/8):

- `descr` (text, nullable; SNMP `ifDescr`)
- `alias` (text, nullable; SNMP `ifAlias`)
- `mac` (macaddr, nullable; SNMP `ifPhysAddress`)
- `admin_status` (integer, nullable; SNMP `ifAdminStatus`)
- `oper_status` (integer, nullable; SNMP `ifOperStatus`)
- `mtu` (integer, nullable; SNMP `ifMtu`)
- `speed_bps` (bigint, nullable; SNMP `ifSpeed`/`ifHighSpeed`)

### `ip_addresses`

Purpose: IP addresses observed on interfaces/devices.

Minimum columns (v1):

- `id` (uuid)
- `device_id` (uuid, foreign key → `devices.id`, nullable if later normalized via interface)
- `interface_id` (uuid, foreign key → `interfaces.id`, nullable)
- `ip` (inet)

Constraints (v1):

- at least one of `device_id` or `interface_id` must be present
- uniqueness is enforced per parent (`(device_id, ip)` and `(interface_id, ip)` via partial unique indexes)

### `mac_addresses`

Purpose: MAC addresses observed on interfaces/devices.

Minimum columns (v1):

- `id` (uuid)
- `device_id` (uuid, foreign key → `devices.id`, nullable)
- `interface_id` (uuid, foreign key → `interfaces.id`, nullable)
- `mac` (macaddr)

Constraints (v1):

- at least one of `device_id` or `interface_id` must be present
- uniqueness is enforced per parent (`(device_id, mac)` and `(interface_id, mac)` via partial unique indexes)

### `services`

Purpose: discovered services (ports/protocols) on a device.

Minimum columns (v1):

- `id` (uuid)
- `device_id` (uuid, foreign key → `devices.id`)
- `protocol` (text, nullable; when present: `tcp` or `udp`)
- `port` (integer, nullable; when present: 1–65535)
- `name` (text, nullable)

### `device_metadata`

Purpose: user-editable metadata separate from discovery facts.

Minimum columns (v1):

- `id` (uuid)
- `device_id` (uuid, foreign key → `devices.id`, unique)

Metadata fields are **TBD** and will be added as UI workflows are implemented.

## Enrichment (Phase 7/8)

These tables store optional “facts” discovered via read-only enrichments (SNMP, name resolution, etc.).

### `device_snmp`

Purpose: latest SNMP “system group” snapshot per device (best-effort).

Minimum columns:

- `device_id` (uuid, primary key, foreign key → `devices.id`)
- `address` (inet, nullable; address used for the last successful/attempted query)
- `sys_name` (text, nullable)
- `sys_descr` (text, nullable)
- `sys_object_id` (text, nullable)
- `sys_contact` (text, nullable)
- `sys_location` (text, nullable)
- `last_success_at` (timestamptz, nullable)
- `last_error` (text, nullable)

### `device_name_candidates`

Purpose: store candidate human-friendly names found via enrichment sources (e.g. reverse DNS, SNMP).

Minimum columns:

- `id` (bigserial)
- `device_id` (uuid, foreign key → `devices.id`)
- `name` (text)
- `source` (text)
- `address` (inet, nullable)
- `observed_at` (timestamptz)

### `interface_vlans`

Purpose: store VLAN membership observations per interface (v1 primarily stores PVID via SNMP bridge/q-bridge MIB).

Minimum columns:

- `id` (bigserial)
- `interface_id` (uuid, foreign key → `interfaces.id`)
- `vlan_id` (integer)
- `role` (text; `pvid` | `tagged` | `untagged`)
- `source` (text; `snmp`)
- `observed_at` (timestamptz)

## Observations (Phase 8+)

These tables are append-only logs keyed by `discovery_runs.id`. They enable history/diffing later (Phase 9+) while keeping “current state” in the core tables (`ip_addresses`, `mac_addresses`, etc).

### `ip_observations`

Purpose: record that an IP was observed on a device during a discovery run.

Minimum columns (v1):

- `id` (bigserial)
- `run_id` (uuid, foreign key → `discovery_runs.id`)
- `device_id` (uuid, foreign key → `devices.id`)
- `ip` (inet)
- `observed_at` (timestamptz)

### `mac_observations`

Purpose: record that a MAC was observed on a device during a discovery run.

Minimum columns (v1):

- `id` (bigserial)
- `run_id` (uuid, foreign key → `discovery_runs.id`)
- `device_id` (uuid, foreign key → `devices.id`)
- `mac` (macaddr)
- `observed_at` (timestamptz)
