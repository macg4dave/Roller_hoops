package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX matches the minimal interface needed from pgxpool.Pool or pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

type Queries struct {
	db DBTX
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{db: tx}
}

const createDevice = `-- name: CreateDevice :one
INSERT INTO devices (display_name)
VALUES ($1)
RETURNING id, display_name
`

func (q *Queries) CreateDevice(ctx context.Context, displayName *string) (Device, error) {
	row := q.db.QueryRow(ctx, createDevice, displayName)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName)
	return i, err
}

const getDevice = `-- name: GetDevice :one
SELECT id, display_name
FROM devices
WHERE id = $1
`

func (q *Queries) GetDevice(ctx context.Context, id string) (Device, error) {
	row := q.db.QueryRow(ctx, getDevice, id)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName)
	return i, err
}

const listDevices = `-- name: ListDevices :many
SELECT id, display_name
FROM devices
ORDER BY created_at DESC
`

func (q *Queries) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := q.db.Query(ctx, listDevices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Device
	for rows.Next() {
		var i Device
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

const updateDevice = `-- name: UpdateDevice :one
UPDATE devices
SET display_name = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, display_name
`

type UpdateDeviceParams struct {
	ID          string
	DisplayName *string
}

func (q *Queries) UpdateDevice(ctx context.Context, arg UpdateDeviceParams) (Device, error) {
	row := q.db.QueryRow(ctx, updateDevice, arg.ID, arg.DisplayName)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName)
	return i, err
}
