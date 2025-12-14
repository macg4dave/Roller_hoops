-- name: InsertIPObservation :exec
INSERT INTO ip_observations (run_id, device_id, ip)
VALUES ($1::uuid, $2::uuid, $3::inet)
ON CONFLICT (run_id, device_id, ip) DO NOTHING;

-- name: InsertMACObservation :exec
INSERT INTO mac_observations (run_id, device_id, mac)
VALUES ($1::uuid, $2::uuid, $3::macaddr)
ON CONFLICT (run_id, device_id, mac) DO NOTHING;

