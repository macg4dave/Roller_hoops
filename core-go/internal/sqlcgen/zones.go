package sqlcgen

import (
	"context"
	"time"
)

// Zone is a manually curated security grouping used by the Security map layer.
type Zone struct {
	ID          string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ZoneMember is a lightweight device projection for zone membership APIs.
type ZoneMember struct {
	DeviceID    string
	DisplayName *string
}

type CreateZoneParams struct {
	Name        string
	Description *string
}

type UpdateZoneParams struct {
	ID          string
	Name        string
	Description *string
}

type AddZoneMemberParams struct {
	ZoneID   string
	DeviceID string
}

type RemoveZoneMemberParams struct {
	ZoneID   string
	DeviceID string
}

type ReplaceZoneMembersParams struct {
	ZoneID    string
	DeviceIDs []string
}

const listZones = `-- name: ListZones :many
SELECT id,
       name,
       description,
       created_at,
       updated_at
FROM zones
ORDER BY name ASC, id ASC
`

func (q *Queries) ListZones(ctx context.Context) ([]Zone, error) {
	rows, err := q.db.Query(ctx, listZones)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Zone, 0)
	for rows.Next() {
		var i Zone
		if err := rows.Scan(&i.ID, &i.Name, &i.Description, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getZone = `-- name: GetZone :one
SELECT id,
       name,
       description,
       created_at,
       updated_at
FROM zones
WHERE id = $1::uuid
`

func (q *Queries) GetZone(ctx context.Context, id string) (Zone, error) {
	row := q.db.QueryRow(ctx, getZone, id)
	var i Zone
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const createZone = `-- name: CreateZone :one
INSERT INTO zones (name, description)
VALUES ($1, $2)
RETURNING id,
          name,
          description,
          created_at,
          updated_at
`

func (q *Queries) CreateZone(ctx context.Context, arg CreateZoneParams) (Zone, error) {
	row := q.db.QueryRow(ctx, createZone, arg.Name, arg.Description)
	var i Zone
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const updateZone = `-- name: UpdateZone :one
UPDATE zones
SET name = $2,
    description = $3,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id,
          name,
          description,
          created_at,
          updated_at
`

func (q *Queries) UpdateZone(ctx context.Context, arg UpdateZoneParams) (Zone, error) {
	row := q.db.QueryRow(ctx, updateZone, arg.ID, arg.Name, arg.Description)
	var i Zone
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const deleteZone = `-- name: DeleteZone :execrows
DELETE FROM zones
WHERE id = $1::uuid
`

func (q *Queries) DeleteZone(ctx context.Context, id string) (int64, error) {
	result, err := q.db.Exec(ctx, deleteZone, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const listZoneMembers = `-- name: ListZoneMembers :many
SELECT d.id AS device_id,
       d.display_name
FROM device_zones dz
JOIN devices d ON d.id = dz.device_id
WHERE dz.zone_id = $1::uuid
ORDER BY COALESCE(d.display_name, d.id::text) ASC,
         d.id ASC
`

func (q *Queries) ListZoneMembers(ctx context.Context, zoneID string) ([]ZoneMember, error) {
	rows, err := q.db.Query(ctx, listZoneMembers, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ZoneMember, 0)
	for rows.Next() {
		var i ZoneMember
		if err := rows.Scan(&i.DeviceID, &i.DisplayName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const addZoneMember = `-- name: AddZoneMember :execrows
INSERT INTO device_zones (device_id, zone_id)
VALUES ($2::uuid, $1::uuid)
ON CONFLICT (device_id, zone_id) DO NOTHING
`

func (q *Queries) AddZoneMember(ctx context.Context, arg AddZoneMemberParams) (int64, error) {
	result, err := q.db.Exec(ctx, addZoneMember, arg.ZoneID, arg.DeviceID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const removeZoneMember = `-- name: RemoveZoneMember :execrows
DELETE FROM device_zones
WHERE zone_id = $1::uuid
  AND device_id = $2::uuid
`

func (q *Queries) RemoveZoneMember(ctx context.Context, arg RemoveZoneMemberParams) (int64, error) {
	result, err := q.db.Exec(ctx, removeZoneMember, arg.ZoneID, arg.DeviceID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const replaceZoneMembers = `-- name: ReplaceZoneMembers :exec
WITH cleared AS (
  DELETE FROM device_zones
  WHERE zone_id = $1::uuid
)
INSERT INTO device_zones (device_id, zone_id)
SELECT device_id::uuid,
       $1::uuid
FROM unnest($2::text[]) AS desired(device_id)
ON CONFLICT (device_id, zone_id) DO NOTHING
`

func (q *Queries) ReplaceZoneMembers(ctx context.Context, arg ReplaceZoneMembersParams) error {
	_, err := q.db.Exec(ctx, replaceZoneMembers, arg.ZoneID, arg.DeviceIDs)
	return err
}

const listExistingDeviceIDs = `-- name: ListExistingDeviceIDs :many
SELECT id::text
FROM devices
WHERE id::text = ANY($1::text[])
ORDER BY id::text ASC
`

func (q *Queries) ListExistingDeviceIDs(ctx context.Context, deviceIDs []string) ([]string, error) {
	rows, err := q.db.Query(ctx, listExistingDeviceIDs, deviceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
