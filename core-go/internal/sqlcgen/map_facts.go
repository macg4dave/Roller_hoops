package sqlcgen

import "context"

// MapDeviceFacts holds the primary IP and MAC for a device, used to enrich
// map node metadata with the most-recently-observed address facts.
type MapDeviceFacts struct {
	DeviceID   string
	PrimaryIP  *string
	PrimaryMAC *string
}

const listDevicePrimaryFacts = `-- name: ListDevicePrimaryFacts :many
SELECT d.id::text AS device_id,
       (SELECT ia.ip::text
        FROM ip_addresses ia
        LEFT JOIN interfaces i ON i.id = ia.interface_id
        WHERE ia.device_id = d.id OR i.device_id = d.id
        ORDER BY ia.updated_at DESC, ia.ip::text ASC
        LIMIT 1) AS primary_ip,
       (SELECT ma.mac::text
        FROM mac_addresses ma
        LEFT JOIN interfaces i ON i.id = ma.interface_id
        WHERE ma.device_id = d.id OR i.device_id = d.id
        ORDER BY ma.updated_at DESC, ma.mac::text ASC
        LIMIT 1) AS primary_mac
FROM devices d
WHERE d.id::text = ANY($1::text[])
ORDER BY d.id ASC
`

func (q *Queries) ListDevicePrimaryFacts(ctx context.Context, deviceIDs []string) ([]MapDeviceFacts, error) {
	rows, err := q.db.Query(ctx, listDevicePrimaryFacts, deviceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MapDeviceFacts
	for rows.Next() {
		var i MapDeviceFacts
		if err := rows.Scan(&i.DeviceID, &i.PrimaryIP, &i.PrimaryMAC); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
