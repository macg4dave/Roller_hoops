
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
