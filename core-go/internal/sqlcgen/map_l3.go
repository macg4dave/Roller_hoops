package sqlcgen

import "context"

type MapDevicePeer struct {
	ID          string
	DisplayName *string
}

const listDevicesInCIDR = `-- name: ListDevicesInCIDR :many
SELECT DISTINCT d.id,
                d.display_name
FROM ip_addresses ia
LEFT JOIN interfaces i ON i.id = ia.interface_id
JOIN devices d ON d.id = COALESCE(ia.device_id, i.device_id)
WHERE ia.ip << $1::cidr
ORDER BY d.id ASC
LIMIT $2;
`

const listDevicePeersInCIDR = `-- name: ListDevicePeersInCIDR :many
SELECT DISTINCT d.id,
                d.display_name
FROM ip_addresses ia
LEFT JOIN interfaces i ON i.id = ia.interface_id
JOIN devices d ON d.id = COALESCE(ia.device_id, i.device_id)
WHERE ia.ip << $1::cidr
  AND d.id <> $2::uuid
ORDER BY d.id ASC
LIMIT $3;
`

func (q *Queries) ListDevicesInCIDR(ctx context.Context, cidr string, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listDevicesInCIDR, cidr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MapDevicePeer
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

func (q *Queries) ListDevicePeersInCIDR(ctx context.Context, cidr string, excludeDeviceID string, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listDevicePeersInCIDR, cidr, excludeDeviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MapDevicePeer
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
