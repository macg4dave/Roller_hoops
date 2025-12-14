-- +migrate Up

ALTER TABLE device_metadata
  ADD COLUMN owner text NULL,
  ADD COLUMN location text NULL,
  ADD COLUMN notes text NULL;

