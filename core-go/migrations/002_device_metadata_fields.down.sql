-- +migrate Down

ALTER TABLE device_metadata
  DROP COLUMN IF EXISTS notes,
  DROP COLUMN IF EXISTS location,
  DROP COLUMN IF EXISTS owner;

