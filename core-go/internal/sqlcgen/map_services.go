package sqlcgen

import (
	"context"
	"time"
)

type MapService struct {
	ID         string
	DeviceID   string
	Protocol   *string
	Port       *int32
	Name       *string
	State      *string
	Source     *string
	ObservedAt time.Time
}

const listServicesForDevice = `-- name: ListServicesForDevice :many
SELECT id,
       device_id,
       protocol,
       port,
       name,
       state,
       source,
       observed_at
FROM services
WHERE device_id = $1::uuid
ORDER BY observed_at DESC, protocol ASC NULLS LAST, port ASC NULLS LAST, name ASC NULLS LAST, id ASC
LIMIT $2;
`

func (q *Queries) ListServicesForDevice(ctx context.Context, deviceID string, limit int32) ([]MapService, error) {
	rows, err := q.db.Query(ctx, listServicesForDevice, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MapService
	for rows.Next() {
		var i MapService
		if err := rows.Scan(&i.ID, &i.DeviceID, &i.Protocol, &i.Port, &i.Name, &i.State, &i.Source, &i.ObservedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getServiceByID = `-- name: GetServiceByID :one
SELECT id,
       device_id,
       protocol,
       port,
       name,
       state,
       source,
       observed_at
FROM services
WHERE id = $1::uuid;
`

func (q *Queries) GetServiceByID(ctx context.Context, serviceID string) (MapService, error) {
	row := q.db.QueryRow(ctx, getServiceByID, serviceID)
	var i MapService
	err := row.Scan(&i.ID, &i.DeviceID, &i.Protocol, &i.Port, &i.Name, &i.State, &i.Source, &i.ObservedAt)
	return i, err
}
