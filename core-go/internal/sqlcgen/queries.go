package sqlcgen

import (
	"context"
	"time"

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
LEFT JOIN device_metadata m ON m.device_id = i.id
`

func (q *Queries) CreateDevice(ctx context.Context, displayName *string) (Device, error) {
	row := q.db.QueryRow(ctx, createDevice, displayName)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName, &i.Owner, &i.Location, &i.Notes)
	return i, err
}

const getDevice = `-- name: GetDevice :one
SELECT d.id,
       d.display_name,
       m.owner,
       m.location,
       m.notes
FROM devices d
LEFT JOIN device_metadata m ON m.device_id = d.id
WHERE d.id = $1
`

func (q *Queries) GetDevice(ctx context.Context, id string) (Device, error) {
	row := q.db.QueryRow(ctx, getDevice, id)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName, &i.Owner, &i.Location, &i.Notes)
	return i, err
}

const listDevices = `-- name: ListDevices :many
SELECT d.id,
       d.display_name,
       m.owner,
       m.location,
       m.notes
FROM devices d
LEFT JOIN device_metadata m ON m.device_id = d.id
ORDER BY d.created_at DESC
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
		if err := rows.Scan(&i.ID, &i.DisplayName, &i.Owner, &i.Location, &i.Notes); err != nil {
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
LEFT JOIN device_metadata m ON m.device_id = u.id
`

type UpdateDeviceParams struct {
	ID          string
	DisplayName *string
}

func (q *Queries) UpdateDevice(ctx context.Context, arg UpdateDeviceParams) (Device, error) {
	row := q.db.QueryRow(ctx, updateDevice, arg.ID, arg.DisplayName)
	var i Device
	err := row.Scan(&i.ID, &i.DisplayName, &i.Owner, &i.Location, &i.Notes)
	return i, err
}

const upsertDeviceMetadata = `-- name: UpsertDeviceMetadata :one
INSERT INTO device_metadata (device_id, owner, location, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (device_id) DO UPDATE
SET owner = EXCLUDED.owner,
    location = EXCLUDED.location,
    notes = EXCLUDED.notes,
    updated_at = now()
RETURNING device_id, owner, location, notes
`

type UpsertDeviceMetadataParams struct {
	DeviceID string
	Owner    *string
	Location *string
	Notes    *string
}

func (q *Queries) UpsertDeviceMetadata(ctx context.Context, arg UpsertDeviceMetadataParams) (DeviceMetadata, error) {
	row := q.db.QueryRow(ctx, upsertDeviceMetadata, arg.DeviceID, arg.Owner, arg.Location, arg.Notes)
	var i DeviceMetadata
	err := row.Scan(&i.DeviceID, &i.Owner, &i.Location, &i.Notes)
	return i, err
}

const insertDiscoveryRun = `-- name: InsertDiscoveryRun :one
INSERT INTO discovery_runs (status, scope, stats)
VALUES ($1, $2, COALESCE($3, '{}'::jsonb))
RETURNING id, status, scope, stats, started_at, completed_at, last_error
`

type InsertDiscoveryRunParams struct {
	Status string
	Scope  *string
	Stats  map[string]any
}

func (q *Queries) InsertDiscoveryRun(ctx context.Context, arg InsertDiscoveryRunParams) (DiscoveryRun, error) {
	row := q.db.QueryRow(ctx, insertDiscoveryRun, arg.Status, arg.Scope, arg.Stats)
	var i DiscoveryRun
	err := row.Scan(
		&i.ID,
		&i.Status,
		&i.Scope,
		&i.Stats,
		&i.StartedAt,
		&i.CompletedAt,
		&i.LastError,
	)
	return i, err
}

const updateDiscoveryRun = `-- name: UpdateDiscoveryRun :one
UPDATE discovery_runs
SET status = $2,
    stats = COALESCE($3, stats),
    completed_at = $4,
    last_error = $5
WHERE id = $1
RETURNING id, status, scope, stats, started_at, completed_at, last_error
`

type UpdateDiscoveryRunParams struct {
	ID          string
	Status      string
	Stats       map[string]any
	CompletedAt *time.Time
	LastError   *string
}

func (q *Queries) UpdateDiscoveryRun(ctx context.Context, arg UpdateDiscoveryRunParams) (DiscoveryRun, error) {
	row := q.db.QueryRow(ctx, updateDiscoveryRun, arg.ID, arg.Status, arg.Stats, arg.CompletedAt, arg.LastError)
	var i DiscoveryRun
	err := row.Scan(
		&i.ID,
		&i.Status,
		&i.Scope,
		&i.Stats,
		&i.StartedAt,
		&i.CompletedAt,
		&i.LastError,
	)
	return i, err
}

const getLatestDiscoveryRun = `-- name: GetLatestDiscoveryRun :one
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
ORDER BY started_at DESC
LIMIT 1
`

func (q *Queries) GetLatestDiscoveryRun(ctx context.Context) (DiscoveryRun, error) {
	row := q.db.QueryRow(ctx, getLatestDiscoveryRun)
	var i DiscoveryRun
	err := row.Scan(
		&i.ID,
		&i.Status,
		&i.Scope,
		&i.Stats,
		&i.StartedAt,
		&i.CompletedAt,
		&i.LastError,
	)
	return i, err
}

const getDiscoveryRun = `-- name: GetDiscoveryRun :one
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
WHERE id = $1
`

func (q *Queries) GetDiscoveryRun(ctx context.Context, id string) (DiscoveryRun, error) {
	row := q.db.QueryRow(ctx, getDiscoveryRun, id)
	var i DiscoveryRun
	err := row.Scan(
		&i.ID,
		&i.Status,
		&i.Scope,
		&i.Stats,
		&i.StartedAt,
		&i.CompletedAt,
		&i.LastError,
	)
	return i, err
}

const insertDiscoveryRunLog = `-- name: InsertDiscoveryRunLog :exec
INSERT INTO discovery_run_logs (run_id, level, message)
VALUES ($1, $2, $3)
`

type InsertDiscoveryRunLogParams struct {
	RunID   string
	Level   string
	Message string
}

func (q *Queries) InsertDiscoveryRunLog(ctx context.Context, arg InsertDiscoveryRunLogParams) error {
	_, err := q.db.Exec(ctx, insertDiscoveryRunLog, arg.RunID, arg.Level, arg.Message)
	return err
}
