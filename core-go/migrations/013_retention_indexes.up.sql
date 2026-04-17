-- +migrate Up

-- Supports retention cleanup of old discovery logs across all runs.
CREATE INDEX IF NOT EXISTS discovery_run_logs_created_at_idx
  ON discovery_run_logs (created_at DESC);

