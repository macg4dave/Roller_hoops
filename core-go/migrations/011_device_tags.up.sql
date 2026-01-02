-- +migrate Up

-- Phase 10: device tagging / classification (auto + manual).

CREATE TABLE IF NOT EXISTS device_tags (
  id bigserial PRIMARY KEY,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  tag text NOT NULL,
  source text NOT NULL, -- "auto" | "manual"
  confidence integer NOT NULL DEFAULT 50,
  evidence jsonb NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'device_tags_source_chk'
  ) THEN
    ALTER TABLE device_tags
      ADD CONSTRAINT device_tags_source_chk
      CHECK (source IN ('auto', 'manual'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS device_tags_device_tag_source_uniq
  ON device_tags (device_id, tag, source);

CREATE INDEX IF NOT EXISTS device_tags_device_id_idx
  ON device_tags (device_id);

CREATE INDEX IF NOT EXISTS device_tags_tag_idx
  ON device_tags (tag);

