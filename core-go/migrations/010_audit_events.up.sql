-- +migrate Up

-- Phase 10: minimal audit logging (UI-owned auth calls into core-go).

CREATE TABLE IF NOT EXISTS audit_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor text NOT NULL,
  actor_role text NULL,
  action text NOT NULL,
  target_type text NULL,
  target_id uuid NULL,
  details jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_events_created_at_idx
  ON audit_events (created_at DESC);

CREATE INDEX IF NOT EXISTS audit_events_target_idx
  ON audit_events (target_type, target_id);

