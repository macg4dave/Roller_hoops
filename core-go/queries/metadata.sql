-- name: UpsertDeviceMetadata :one
INSERT INTO device_metadata (device_id, owner, location, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (device_id) DO UPDATE
SET owner = EXCLUDED.owner,
    location = EXCLUDED.location,
    notes = EXCLUDED.notes,
    updated_at = now()
RETURNING device_id, owner, location, notes;

-- name: UpsertDeviceMetadataFillBlank :one
INSERT INTO device_metadata (device_id, owner, location, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (device_id) DO UPDATE
SET owner = CASE
              WHEN device_metadata.owner IS NULL OR btrim(device_metadata.owner) = '' THEN EXCLUDED.owner
              ELSE device_metadata.owner
            END,
    location = CASE
                 WHEN device_metadata.location IS NULL OR btrim(device_metadata.location) = '' THEN EXCLUDED.location
                 ELSE device_metadata.location
               END,
    notes = CASE
              WHEN device_metadata.notes IS NULL OR btrim(device_metadata.notes) = '' THEN EXCLUDED.notes
              ELSE device_metadata.notes
            END,
    updated_at = now()
RETURNING device_id, owner, location, notes;
