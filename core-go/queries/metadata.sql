-- name: UpsertDeviceMetadata :one
INSERT INTO device_metadata (device_id, owner, location, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (device_id) DO UPDATE
SET owner = EXCLUDED.owner,
    location = EXCLUDED.location,
    notes = EXCLUDED.notes,
    updated_at = now()
RETURNING device_id, owner, location, notes;
