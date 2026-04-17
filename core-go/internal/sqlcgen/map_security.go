package sqlcgen

import "context"

// MapSecurityZone is a lightweight zone projection for the Security layer.
type MapSecurityZone struct {
	ID          string
	Name        string
	Description *string
	MemberCount int64
}

const getMapZone = `-- name: GetMapZone :one
SELECT z.id,
       z.name,
       z.description,
       COUNT(dz.device_id)::bigint AS member_count
FROM zones z
LEFT JOIN device_zones dz ON dz.zone_id = z.id
WHERE z.id = $1::uuid
GROUP BY z.id, z.name, z.description
`

func (q *Queries) GetMapZone(ctx context.Context, zoneID string) (MapSecurityZone, error) {
	row := q.db.QueryRow(ctx, getMapZone, zoneID)
	var i MapSecurityZone
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.MemberCount)
	return i, err
}

const listZonesForDevice = `-- name: ListZonesForDevice :many
SELECT z.id,
       z.name,
       z.description,
       COUNT(dz_all.device_id)::bigint AS member_count
FROM device_zones dz
JOIN zones z ON z.id = dz.zone_id
LEFT JOIN device_zones dz_all ON dz_all.zone_id = z.id
WHERE dz.device_id = $1::uuid
GROUP BY z.id, z.name, z.description
ORDER BY z.name ASC, z.id ASC
LIMIT $2
`

func (q *Queries) ListZonesForDevice(ctx context.Context, deviceID string, limit int32) ([]MapSecurityZone, error) {
	rows, err := q.db.Query(ctx, listZonesForDevice, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MapSecurityZone, 0)
	for rows.Next() {
		var i MapSecurityZone
		if err := rows.Scan(&i.ID, &i.Name, &i.Description, &i.MemberCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDevicesInZone = `-- name: ListDevicesInZone :many
SELECT d.id,
       d.display_name
FROM device_zones dz
JOIN devices d ON d.id = dz.device_id
WHERE dz.zone_id = $1::uuid
ORDER BY d.id ASC
LIMIT $2
`

func (q *Queries) ListDevicesInZone(ctx context.Context, zoneID string, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listDevicesInZone, zoneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MapDevicePeer, 0)
	for rows.Next() {
		var i MapDevicePeer
		if err := rows.Scan(&i.ID, &i.DisplayName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listZonePeers = `-- name: ListZonePeers :many
SELECT d.id,
       d.display_name
FROM device_zones dz
JOIN devices d ON d.id = dz.device_id
WHERE dz.zone_id = $1::uuid
  AND d.id <> $2::uuid
ORDER BY d.id ASC
LIMIT $3
`

func (q *Queries) ListZonePeers(ctx context.Context, zoneID string, excludeDeviceID string, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listZonePeers, zoneID, excludeDeviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MapDevicePeer, 0)
	for rows.Next() {
		var i MapDevicePeer
		if err := rows.Scan(&i.ID, &i.DisplayName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
