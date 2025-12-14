-- +migrate Up

-- Enrichment storage: SNMP facts, friendly-name candidates, and VLAN/port mappings.

-- devices: SNMP system group facts (per-device latest snapshot).
CREATE TABLE IF NOT EXISTS device_snmp (
  device_id uuid PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
  address inet NULL,
  sys_name text NULL,
  sys_descr text NULL,
  sys_object_id text NULL,
  sys_contact text NULL,
  sys_location text NULL,
  last_success_at timestamptz NULL,
  last_error text NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS device_snmp_sys_name_idx ON device_snmp (sys_name);

-- devices: name candidates (append-only-ish; dedupe on device/source/name/address).
CREATE TABLE IF NOT EXISTS device_name_candidates (
  id bigserial PRIMARY KEY,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  name text NOT NULL,
  source text NOT NULL,
  address inet NULL,
  observed_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS device_name_candidates_device_source_name_addr_uniq
  ON device_name_candidates (device_id, source, name, address);

CREATE INDEX IF NOT EXISTS device_name_candidates_device_observed_at_idx
  ON device_name_candidates (device_id, observed_at DESC);

-- interfaces: SNMP-derived columns (kept on the interface row for now).
ALTER TABLE interfaces
  ADD COLUMN IF NOT EXISTS descr text NULL,
  ADD COLUMN IF NOT EXISTS alias text NULL,
  ADD COLUMN IF NOT EXISTS mac macaddr NULL,
  ADD COLUMN IF NOT EXISTS admin_status integer NULL,
  ADD COLUMN IF NOT EXISTS oper_status integer NULL,
  ADD COLUMN IF NOT EXISTS mtu integer NULL,
  ADD COLUMN IF NOT EXISTS speed_bps bigint NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'interfaces_device_id_ifindex_uniq'
  ) THEN
    ALTER TABLE interfaces
      ADD CONSTRAINT interfaces_device_id_ifindex_uniq UNIQUE (device_id, ifindex);
  END IF;
END $$;

-- VLAN / switch-port mapping: store current per-interface VLAN membership observations.
CREATE TABLE IF NOT EXISTS interface_vlans (
  id bigserial PRIMARY KEY,
  interface_id uuid NOT NULL REFERENCES interfaces(id) ON DELETE CASCADE,
  vlan_id integer NOT NULL,
  role text NOT NULL,   -- "pvid" | "tagged" | "untagged" (v1 primarily stores pvid)
  source text NOT NULL, -- "snmp"
  observed_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS interface_vlans_interface_role_uniq
  ON interface_vlans (interface_id, role);

CREATE INDEX IF NOT EXISTS interface_vlans_vlan_id_idx
  ON interface_vlans (vlan_id);

