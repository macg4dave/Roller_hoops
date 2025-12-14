-- +migrate Up

CREATE TABLE IF NOT EXISTS discovery_runs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  scope text NULL,
  stats jsonb NOT NULL DEFAULT '{}'::jsonb,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz NULL,
  last_error text NULL
);

CREATE INDEX IF NOT EXISTS discovery_runs_started_at_idx ON discovery_runs (started_at DESC);

CREATE TABLE IF NOT EXISTS discovery_run_logs (
  id bigserial PRIMARY KEY,
  run_id uuid NOT NULL REFERENCES discovery_runs(id) ON DELETE CASCADE,
  level text NOT NULL DEFAULT 'info',
  message text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS discovery_run_logs_run_id_created_at_idx
  ON discovery_run_logs (run_id, created_at);

