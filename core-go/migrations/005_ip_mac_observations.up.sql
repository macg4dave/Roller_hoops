-- +migrate Up

CREATE TABLE IF NOT EXISTS ip_observations (
  id bigserial PRIMARY KEY,
  run_id uuid NOT NULL REFERENCES discovery_runs(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  ip inet NOT NULL,
  observed_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ip_observations_run_device_ip_uniq
  ON ip_observations (run_id, device_id, ip);

CREATE INDEX IF NOT EXISTS ip_observations_device_id_observed_at_idx
  ON ip_observations (device_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS mac_observations (
  id bigserial PRIMARY KEY,
  run_id uuid NOT NULL REFERENCES discovery_runs(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  mac macaddr NOT NULL,
  observed_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS mac_observations_run_device_mac_uniq
  ON mac_observations (run_id, device_id, mac);

CREATE INDEX IF NOT EXISTS mac_observations_device_id_observed_at_idx
  ON mac_observations (device_id, observed_at DESC);

