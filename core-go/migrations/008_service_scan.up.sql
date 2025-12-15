-- +migrate Up

-- Phase 7: optional service/port discovery enrichment (nmap, etc).

ALTER TABLE services
  ADD COLUMN IF NOT EXISTS state text NULL,
  ADD COLUMN IF NOT EXISTS source text NULL,
  ADD COLUMN IF NOT EXISTS observed_at timestamptz NOT NULL DEFAULT now();

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'services_state_chk'
  ) THEN
    ALTER TABLE services
      ADD CONSTRAINT services_state_chk
      CHECK (state IS NULL OR state IN ('open', 'closed'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS services_device_protocol_port_uniq
  ON services (device_id, protocol, port)
  WHERE protocol IS NOT NULL AND port IS NOT NULL;

CREATE INDEX IF NOT EXISTS services_device_observed_at_idx
  ON services (device_id, observed_at DESC);

