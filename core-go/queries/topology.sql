-- name: UpsertLink :exec
INSERT INTO links (
  link_key,
  a_device_id,
  a_interface_id,
  b_device_id,
  b_interface_id,
  link_type,
  source,
  observed_at
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7, $8)
ON CONFLICT (link_key) DO UPDATE
SET a_device_id = EXCLUDED.a_device_id,
    a_interface_id = EXCLUDED.a_interface_id,
    b_device_id = EXCLUDED.b_device_id,
    b_interface_id = EXCLUDED.b_interface_id,
    link_type = EXCLUDED.link_type,
    source = EXCLUDED.source,
    observed_at = EXCLUDED.observed_at,
    updated_at = now();

-- name: UpsertInterfaceByName :one
INSERT INTO interfaces (device_id, name)
VALUES ($1::uuid, $2)
ON CONFLICT (device_id, name) WHERE name IS NOT NULL
DO UPDATE SET updated_at = now()
RETURNING id;

