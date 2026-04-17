-- name: ListDevices :many
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

-- name: GetDeviceSummaryTimestamps :one
SELECT
    (
        SELECT ia.ip::text
        FROM ip_addresses ia
        LEFT JOIN interfaces i2 ON i2.id = ia.interface_id
        WHERE ia.device_id = $1::uuid OR i2.device_id = $1::uuid
        ORDER BY ia.updated_at DESC, ia.ip::text ASC
        LIMIT 1
    ) AS primary_ip,
    (
        SELECT MAX(ts)
        FROM (
            VALUES
                ((SELECT MAX(ia.updated_at) FROM ip_addresses ia LEFT JOIN interfaces i ON i.id = ia.interface_id WHERE ia.device_id = $1::uuid OR i.device_id = $1::uuid)),
                ((SELECT MAX(ma.updated_at) FROM mac_addresses ma LEFT JOIN interfaces i ON i.id = ma.interface_id WHERE ma.device_id = $1::uuid OR i.device_id = $1::uuid)),
                ((SELECT MAX(s.observed_at) FROM services s WHERE s.device_id = $1::uuid)),
                ((SELECT MAX(ds.last_success_at) FROM device_snmp ds WHERE ds.device_id = $1::uuid))
        ) v(ts)
    ) AS last_seen_at,
    (
        SELECT MAX(ts)
        FROM (
            VALUES
                ((SELECT d.updated_at FROM devices d WHERE d.id = $1::uuid)),
                ((SELECT MAX(dm.updated_at) FROM device_metadata dm WHERE dm.device_id = $1::uuid)),
                ((SELECT MAX(ia.created_at) FROM ip_addresses ia LEFT JOIN interfaces i ON i.id = ia.interface_id WHERE ia.device_id = $1::uuid OR i.device_id = $1::uuid)),
                ((SELECT MAX(ma.created_at) FROM mac_addresses ma LEFT JOIN interfaces i ON i.id = ma.interface_id WHERE ma.device_id = $1::uuid OR i.device_id = $1::uuid)),
                ((SELECT MAX(s.created_at) FROM services s WHERE s.device_id = $1::uuid)),
                ((SELECT MAX(ds.updated_at) FROM device_snmp ds WHERE ds.device_id = $1::uuid)),
                ((SELECT MAX(iv.observed_at) FROM interface_vlans iv JOIN interfaces i ON i.id = iv.interface_id WHERE i.device_id = $1::uuid)),
                ((SELECT MAX(COALESCE(l.observed_at, l.updated_at)) FROM links l WHERE l.a_device_id = $1::uuid OR l.b_device_id = $1::uuid))
        ) v(ts)
    ) AS last_change_at;

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
