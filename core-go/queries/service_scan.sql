-- name: UpsertServiceFromScan :exec
INSERT INTO services (
  device_id,
  protocol,
  port,
  name,
  state,
  source,
  observed_at
)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
ON CONFLICT (device_id, protocol, port) WHERE protocol IS NOT NULL AND port IS NOT NULL
DO UPDATE
SET name = EXCLUDED.name,
    state = EXCLUDED.state,
    source = EXCLUDED.source,
    observed_at = EXCLUDED.observed_at,
    updated_at = now();

