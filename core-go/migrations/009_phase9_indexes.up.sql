-- +migrate Up

-- Phase 9: index support for change feeds and cursor paging.

-- device change feeds (since queries across all devices)
CREATE INDEX IF NOT EXISTS ip_observations_observed_at_idx
  ON ip_observations (observed_at DESC);

CREATE INDEX IF NOT EXISTS mac_observations_observed_at_idx
  ON mac_observations (observed_at DESC);

CREATE INDEX IF NOT EXISTS device_metadata_updated_at_idx
  ON device_metadata (updated_at DESC);

CREATE INDEX IF NOT EXISTS devices_updated_at_idx
  ON devices (updated_at DESC);

CREATE INDEX IF NOT EXISTS services_observed_at_idx
  ON services (observed_at DESC);

CREATE INDEX IF NOT EXISTS interface_vlans_observed_at_idx
  ON interface_vlans (observed_at DESC);

-- discovery run paging
CREATE INDEX IF NOT EXISTS discovery_runs_started_at_id_idx
  ON discovery_runs (started_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS discovery_run_logs_run_created_id_idx
  ON discovery_run_logs (run_id, created_at DESC, id DESC);

