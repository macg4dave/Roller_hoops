-- +migrate Up

-- Phase 2: minimal but future-proof core schema details (no API impact yet).

-- interfaces
ALTER TABLE interfaces
  ADD COLUMN IF NOT EXISTS name text NULL,
  ADD COLUMN IF NOT EXISTS ifindex integer NULL;

CREATE INDEX IF NOT EXISTS interfaces_device_id_idx ON interfaces (device_id);
CREATE UNIQUE INDEX IF NOT EXISTS interfaces_device_id_name_uniq
  ON interfaces (device_id, name)
  WHERE name IS NOT NULL;

-- ip_addresses
CREATE INDEX IF NOT EXISTS ip_addresses_ip_idx ON ip_addresses (ip);
CREATE INDEX IF NOT EXISTS ip_addresses_device_id_idx ON ip_addresses (device_id);
CREATE INDEX IF NOT EXISTS ip_addresses_interface_id_idx ON ip_addresses (interface_id);

CREATE UNIQUE INDEX IF NOT EXISTS ip_addresses_device_id_ip_uniq
  ON ip_addresses (device_id, ip)
  WHERE device_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ip_addresses_interface_id_ip_uniq
  ON ip_addresses (interface_id, ip)
  WHERE interface_id IS NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'ip_addresses_device_or_interface_chk'
  ) THEN
    ALTER TABLE ip_addresses
      ADD CONSTRAINT ip_addresses_device_or_interface_chk
      CHECK (device_id IS NOT NULL OR interface_id IS NOT NULL);
  END IF;
END $$;

-- mac_addresses
CREATE INDEX IF NOT EXISTS mac_addresses_mac_idx ON mac_addresses (mac);
CREATE INDEX IF NOT EXISTS mac_addresses_device_id_idx ON mac_addresses (device_id);
CREATE INDEX IF NOT EXISTS mac_addresses_interface_id_idx ON mac_addresses (interface_id);

CREATE UNIQUE INDEX IF NOT EXISTS mac_addresses_device_id_mac_uniq
  ON mac_addresses (device_id, mac)
  WHERE device_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mac_addresses_interface_id_mac_uniq
  ON mac_addresses (interface_id, mac)
  WHERE interface_id IS NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'mac_addresses_device_or_interface_chk'
  ) THEN
    ALTER TABLE mac_addresses
      ADD CONSTRAINT mac_addresses_device_or_interface_chk
      CHECK (device_id IS NOT NULL OR interface_id IS NOT NULL);
  END IF;
END $$;

-- services
ALTER TABLE services
  ADD COLUMN IF NOT EXISTS protocol text NULL,
  ADD COLUMN IF NOT EXISTS port integer NULL,
  ADD COLUMN IF NOT EXISTS name text NULL;

CREATE INDEX IF NOT EXISTS services_device_id_idx ON services (device_id);
CREATE INDEX IF NOT EXISTS services_port_idx ON services (port);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'services_protocol_chk'
  ) THEN
    ALTER TABLE services
      ADD CONSTRAINT services_protocol_chk
      CHECK (protocol IS NULL OR protocol IN ('tcp', 'udp'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'services_port_range_chk'
  ) THEN
    ALTER TABLE services
      ADD CONSTRAINT services_port_range_chk
      CHECK (port IS NULL OR (port >= 1 AND port <= 65535));
  END IF;
END $$;

