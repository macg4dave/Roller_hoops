-- +migrate Up

-- Phase 16: security zones / regions for the Security layer.

CREATE TABLE IF NOT EXISTS zones (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text UNIQUE NOT NULL,
  description text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS device_zones (
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  zone_id uuid NOT NULL REFERENCES zones(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS device_zones_device_zone_uniq
  ON device_zones (device_id, zone_id);

CREATE INDEX IF NOT EXISTS device_zones_zone_id_idx
  ON device_zones (zone_id);
