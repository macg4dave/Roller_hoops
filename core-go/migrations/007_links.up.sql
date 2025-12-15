-- +migrate Up

-- Phase 7: optional physical adjacency facts (LLDP/CDP enrichment can upsert into this table).

CREATE TABLE IF NOT EXISTS links (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  link_key text NOT NULL,
  a_device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  a_interface_id uuid NULL REFERENCES interfaces(id) ON DELETE SET NULL,
  b_device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  b_interface_id uuid NULL REFERENCES interfaces(id) ON DELETE SET NULL,
  link_type text NULL,
  source text NOT NULL, -- "manual" | "lldp" | "cdp"
  observed_at timestamptz NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS links_link_key_uniq ON links (link_key);
CREATE INDEX IF NOT EXISTS links_a_device_id_idx ON links (a_device_id);
CREATE INDEX IF NOT EXISTS links_b_device_id_idx ON links (b_device_id);
CREATE INDEX IF NOT EXISTS links_source_idx ON links (source);
CREATE INDEX IF NOT EXISTS links_observed_at_idx ON links (observed_at DESC);

