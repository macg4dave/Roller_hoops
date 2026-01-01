package sqlcgen

import (
	"context"
)

const listDevicePVIDs = `-- name: ListDevicePVIDs :many
SELECT DISTINCT iv.vlan_id
FROM interfaces i
JOIN interface_vlans iv ON iv.interface_id = i.id AND iv.role = 'pvid'
WHERE i.device_id = $1::uuid
ORDER BY iv.vlan_id ASC;
`

func (q *Queries) ListDevicePVIDs(ctx context.Context, deviceID string) ([]int32, error) {
	rows, err := q.db.Query(ctx, listDevicePVIDs, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []int32
	for rows.Next() {
		var vlanID int32
		if err := rows.Scan(&vlanID); err != nil {
			return nil, err
		}
		items = append(items, vlanID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDevicesInVLAN = `-- name: ListDevicesInVLAN :many
SELECT DISTINCT d.id,
                d.display_name
FROM interface_vlans iv
JOIN interfaces i ON i.id = iv.interface_id
JOIN devices d ON d.id = i.device_id
WHERE iv.role = 'pvid'
  AND iv.vlan_id = $1
ORDER BY d.id ASC
LIMIT $2;
`

const listDevicePeersInVLAN = `-- name: ListDevicePeersInVLAN :many
SELECT DISTINCT d.id,
                d.display_name
FROM interface_vlans iv
JOIN interfaces i ON i.id = iv.interface_id
JOIN devices d ON d.id = i.device_id
WHERE iv.role = 'pvid'
  AND iv.vlan_id = $1
  AND d.id <> $2::uuid
ORDER BY d.id ASC
LIMIT $3;
`

func (q *Queries) ListDevicesInVLAN(ctx context.Context, vlanID int32, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listDevicesInVLAN, vlanID, limit)
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

func (q *Queries) ListDevicePeersInVLAN(ctx context.Context, vlanID int32, excludeDeviceID string, limit int32) ([]MapDevicePeer, error) {
	rows, err := q.db.Query(ctx, listDevicePeersInVLAN, vlanID, excludeDeviceID, limit)
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

