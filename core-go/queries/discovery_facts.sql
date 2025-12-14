-- name: FindDeviceIDByMAC :one
SELECT device_id
FROM mac_addresses
WHERE mac = $1::macaddr
  AND device_id IS NOT NULL
ORDER BY created_at ASC
LIMIT 1;

-- name: FindDeviceIDByIP :one
SELECT device_id
FROM ip_addresses
WHERE ip = $1::inet
  AND device_id IS NOT NULL
ORDER BY created_at ASC
LIMIT 1;

-- name: UpsertDeviceIP :exec
INSERT INTO ip_addresses (device_id, ip)
VALUES ($1::uuid, $2::inet)
ON CONFLICT (device_id, ip) WHERE device_id IS NOT NULL
DO UPDATE SET updated_at = now();

-- name: UpsertDeviceMAC :exec
INSERT INTO mac_addresses (device_id, mac)
VALUES ($1::uuid, $2::macaddr)
ON CONFLICT (device_id, mac) WHERE device_id IS NOT NULL
DO UPDATE SET updated_at = now();
