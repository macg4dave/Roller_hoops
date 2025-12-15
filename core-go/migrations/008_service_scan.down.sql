-- +migrate Down

DROP INDEX IF EXISTS services_device_observed_at_idx;
DROP INDEX IF EXISTS services_device_protocol_port_uniq;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'services_state_chk'
  ) THEN
    ALTER TABLE services
      DROP CONSTRAINT services_state_chk;
  END IF;
END $$;

ALTER TABLE services
  DROP COLUMN IF EXISTS observed_at,
  DROP COLUMN IF EXISTS source,
  DROP COLUMN IF EXISTS state;

