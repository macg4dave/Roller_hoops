SELECT d.id,
       d.display_name,
       m.owner,
       m.location,
       m.notes
FROM devices d
LEFT JOIN device_metadata m ON m.device_id = d.id
ORDER BY d.created_at DESC;

-- name: GetDevice :one
SELECT d.id,
       d.display_name,
       m.owner,
       m.location,
       m.notes
FROM devices d
LEFT JOIN device_metadata m ON m.device_id = d.id
WHERE d.id = $1;

-- name: CreateDevice :one
WITH inserted AS (
  INSERT INTO devices (display_name)
  VALUES ($1)
  RETURNING id, display_name
)
SELECT i.id,
       i.display_name,
       m.owner,
       m.location,
       m.notes
FROM inserted i
LEFT JOIN device_metadata m ON m.device_id = i.id;

-- name: UpdateDevice :one
WITH updated AS (
  UPDATE devices
  SET display_name = $2,
      updated_at = now()
  WHERE id = $1
  RETURNING id, display_name
)
SELECT u.id,
       u.display_name,
       m.owner,
       m.location,
       m.notes
FROM updated u
LEFT JOIN device_metadata m ON m.device_id = u.id;
