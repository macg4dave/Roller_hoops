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

const insertDeviceNameCandidate = `-- name: InsertDeviceNameCandidate :exec
INSERT INTO device_name_candidates (device_id, name, source, address)
VALUES ($1::uuid, $2, $3, $4::inet)
ON CONFLICT (device_id, source, name, address)
DO NOTHING
`

type InsertDeviceNameCandidateParams struct {
	DeviceID string
	Name     string
	Source   string
	Address  *string
}

func (q *Queries) InsertDeviceNameCandidate(ctx context.Context, arg InsertDeviceNameCandidateParams) error {
	_, err := q.db.Exec(ctx, insertDeviceNameCandidate, arg.DeviceID, arg.Name, arg.Source, arg.Address)
	return err
}

const listDeviceNameCandidates = `-- name: ListDeviceNameCandidates :many
SELECT device_id,
       name,
       source,
       address::text,
       observed_at
FROM device_name_candidates
WHERE device_id = $1::uuid
ORDER BY observed_at DESC, source ASC, name ASC
`

func (q *Queries) ListDeviceNameCandidates(ctx context.Context, deviceID string) ([]DeviceNameCandidate, error) {
	rows, err := q.db.Query(ctx, listDeviceNameCandidates, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceNameCandidate
	for rows.Next() {
		var i DeviceNameCandidate
		if err := rows.Scan(&i.DeviceID, &i.Name, &i.Source, &i.Address, &i.ObservedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setDeviceDisplayNameIfUnset = `-- name: SetDeviceDisplayNameIfUnset :execrows
UPDATE devices
SET display_name = $2,
    updated_at = now()
WHERE id = $1::uuid
  AND (display_name IS NULL OR btrim(display_name) = '')
`

type SetDeviceDisplayNameIfUnsetParams struct {
	ID          string
	DisplayName string
}

func (q *Queries) SetDeviceDisplayNameIfUnset(ctx context.Context, arg SetDeviceDisplayNameIfUnsetParams) (int64, error) {
	tag, err := q.db.Exec(ctx, setDeviceDisplayNameIfUnset, arg.ID, arg.DisplayName)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const upsertDeviceSNMP = `-- name: UpsertDeviceSNMP :exec
INSERT INTO device_snmp (
  device_id,
  address,
  sys_name,
  sys_descr,
  sys_object_id,
  sys_contact,
  sys_location,
  last_success_at,
  last_error,
  updated_at
)
VALUES ($1::uuid, $2::inet, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (device_id) DO UPDATE
SET address = EXCLUDED.address,
    sys_name = EXCLUDED.sys_name,
    sys_descr = EXCLUDED.sys_descr,
    sys_object_id = EXCLUDED.sys_object_id,
    sys_contact = EXCLUDED.sys_contact,
    sys_location = EXCLUDED.sys_location,
    last_success_at = EXCLUDED.last_success_at,
    last_error = EXCLUDED.last_error,
    updated_at = now()
`

type UpsertDeviceSNMPParams struct {
	DeviceID      string
	Address       *string
	SysName       *string
	SysDescr      *string
	SysObjectID   *string
	SysContact    *string
	SysLocation   *string
	LastSuccessAt *time.Time
	LastError     *string
}

func (q *Queries) UpsertDeviceSNMP(ctx context.Context, arg UpsertDeviceSNMPParams) error {
	_, err := q.db.Exec(
		ctx,
		upsertDeviceSNMP,
		arg.DeviceID,
		arg.Address,
		arg.SysName,
		arg.SysDescr,
		arg.SysObjectID,
		arg.SysContact,
		arg.SysLocation,
		arg.LastSuccessAt,
		arg.LastError,
	)
	return err
}

const upsertInterfaceFromSNMP = `-- name: UpsertInterfaceFromSNMP :one
INSERT INTO interfaces (
  device_id,
  ifindex,
  name,
  descr,
  alias,
  mac,
  admin_status,
  oper_status,
  mtu,
  speed_bps
)
VALUES ($1::uuid, $2, $3, $4, $5, $6::macaddr, $7, $8, $9, $10)
ON CONFLICT (device_id, ifindex) DO UPDATE
SET name = EXCLUDED.name,
    descr = EXCLUDED.descr,
    alias = EXCLUDED.alias,
    mac = EXCLUDED.mac,
    admin_status = EXCLUDED.admin_status,
    oper_status = EXCLUDED.oper_status,
    mtu = EXCLUDED.mtu,
    speed_bps = EXCLUDED.speed_bps,
    updated_at = now()
RETURNING id
`

type UpsertInterfaceFromSNMPParams struct {
	DeviceID    string
	Ifindex     int32
	Name        *string
	Descr       *string
	Alias       *string
	MAC         *string
	AdminStatus *int32
	OperStatus  *int32
	MTU         *int32
	SpeedBps    *int64
}

func (q *Queries) UpsertInterfaceFromSNMP(ctx context.Context, arg UpsertInterfaceFromSNMPParams) (string, error) {
	row := q.db.QueryRow(
		ctx,
		upsertInterfaceFromSNMP,
		arg.DeviceID,
		arg.Ifindex,
		arg.Name,
		arg.Descr,
		arg.Alias,
		arg.MAC,
		arg.AdminStatus,
		arg.OperStatus,
		arg.MTU,
		arg.SpeedBps,
	)
	var id string
	err := row.Scan(&id)
	return id, err
}

const upsertInterfaceMAC = `-- name: UpsertInterfaceMAC :exec
INSERT INTO mac_addresses (device_id, interface_id, mac)
VALUES ($1::uuid, $2::uuid, $3::macaddr)
ON CONFLICT (interface_id, mac) WHERE interface_id IS NOT NULL
DO UPDATE SET device_id = EXCLUDED.device_id,
              updated_at = now()
`

type UpsertInterfaceMACParams struct {
	DeviceID    string
	InterfaceID string
	MAC         string
}

func (q *Queries) UpsertInterfaceMAC(ctx context.Context, arg UpsertInterfaceMACParams) error {
	_, err := q.db.Exec(ctx, upsertInterfaceMAC, arg.DeviceID, arg.InterfaceID, arg.MAC)
	return err
}

const linkDeviceMACToInterface = `-- name: LinkDeviceMACToInterface :execrows
UPDATE mac_addresses
SET interface_id = $3::uuid,
    updated_at = now()
WHERE device_id = $1::uuid
  AND mac = $2::macaddr
  AND interface_id IS NULL
`

type LinkDeviceMACToInterfaceParams struct {
	DeviceID    string
	MAC         string
	InterfaceID string
}

func (q *Queries) LinkDeviceMACToInterface(ctx context.Context, arg LinkDeviceMACToInterfaceParams) (int64, error) {
	tag, err := q.db.Exec(ctx, linkDeviceMACToInterface, arg.DeviceID, arg.MAC, arg.InterfaceID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const upsertInterfaceVLAN = `-- name: UpsertInterfaceVLAN :exec
INSERT INTO interface_vlans (interface_id, vlan_id, role, source)
VALUES ($1::uuid, $2, $3, $4)
ON CONFLICT (interface_id, role) DO UPDATE
SET vlan_id = EXCLUDED.vlan_id,
    source = EXCLUDED.source,
    observed_at = now()
`

type UpsertInterfaceVLANParams struct {
	InterfaceID string
	VlanID      int32
	Role        string
	Source      string
}

func (q *Queries) UpsertInterfaceVLAN(ctx context.Context, arg UpsertInterfaceVLANParams) error {
	_, err := q.db.Exec(ctx, upsertInterfaceVLAN, arg.InterfaceID, arg.VlanID, arg.Role, arg.Source)
	return err
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

const claimNextDiscoveryRun = `-- name: ClaimNextDiscoveryRun :one
WITH next AS (
  SELECT id
  FROM discovery_runs
  WHERE status = 'queued'
  ORDER BY started_at ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE discovery_runs dr
SET status = 'running',
    stats = COALESCE($1, dr.stats),
    completed_at = NULL,
    last_error = NULL
FROM next
WHERE dr.id = next.id
RETURNING dr.id, dr.status, dr.scope, dr.stats, dr.started_at, dr.completed_at, dr.last_error
`

func (q *Queries) ClaimNextDiscoveryRun(ctx context.Context, stats map[string]any) (DiscoveryRun, error) {
	row := q.db.QueryRow(ctx, claimNextDiscoveryRun, stats)
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

const findDeviceIDByMAC = `-- name: FindDeviceIDByMAC :one
SELECT device_id
FROM mac_addresses
WHERE mac = $1::macaddr
  AND device_id IS NOT NULL
ORDER BY created_at ASC
LIMIT 1
`

func (q *Queries) FindDeviceIDByMAC(ctx context.Context, mac string) (string, error) {
	row := q.db.QueryRow(ctx, findDeviceIDByMAC, mac)
	var deviceID string
	err := row.Scan(&deviceID)
	return deviceID, err
}

const findDeviceIDByIP = `-- name: FindDeviceIDByIP :one
SELECT device_id
FROM ip_addresses
WHERE ip = $1::inet
  AND device_id IS NOT NULL
ORDER BY created_at ASC
LIMIT 1
`

func (q *Queries) FindDeviceIDByIP(ctx context.Context, ip string) (string, error) {
	row := q.db.QueryRow(ctx, findDeviceIDByIP, ip)
	var deviceID string
	err := row.Scan(&deviceID)
	return deviceID, err
}

const upsertDeviceIP = `-- name: UpsertDeviceIP :exec
INSERT INTO ip_addresses (device_id, ip)
VALUES ($1::uuid, $2::inet)
ON CONFLICT (device_id, ip) WHERE device_id IS NOT NULL
DO UPDATE SET updated_at = now()
`

type UpsertDeviceIPParams struct {
	DeviceID string
	IP       string
}

func (q *Queries) UpsertDeviceIP(ctx context.Context, arg UpsertDeviceIPParams) error {
	_, err := q.db.Exec(ctx, upsertDeviceIP, arg.DeviceID, arg.IP)
	return err
}

const upsertDeviceMAC = `-- name: UpsertDeviceMAC :exec
INSERT INTO mac_addresses (device_id, mac)
VALUES ($1::uuid, $2::macaddr)
ON CONFLICT (device_id, mac) WHERE device_id IS NOT NULL
DO UPDATE SET updated_at = now()
`

type UpsertDeviceMACParams struct {
	DeviceID string
	MAC      string
}

func (q *Queries) UpsertDeviceMAC(ctx context.Context, arg UpsertDeviceMACParams) error {
	_, err := q.db.Exec(ctx, upsertDeviceMAC, arg.DeviceID, arg.MAC)
	return err
}

const insertIPObservation = `-- name: InsertIPObservation :exec
INSERT INTO ip_observations (run_id, device_id, ip)
VALUES ($1::uuid, $2::uuid, $3::inet)
ON CONFLICT (run_id, device_id, ip) DO NOTHING
`

type InsertIPObservationParams struct {
	RunID    string
	DeviceID string
	IP       string
}

func (q *Queries) InsertIPObservation(ctx context.Context, arg InsertIPObservationParams) error {
	_, err := q.db.Exec(ctx, insertIPObservation, arg.RunID, arg.DeviceID, arg.IP)
	return err
}

const insertMACObservation = `-- name: InsertMACObservation :exec
INSERT INTO mac_observations (run_id, device_id, mac)
VALUES ($1::uuid, $2::uuid, $3::macaddr)
ON CONFLICT (run_id, device_id, mac) DO NOTHING
`

type InsertMACObservationParams struct {
	RunID    string
	DeviceID string
	MAC      string
}

func (q *Queries) InsertMACObservation(ctx context.Context, arg InsertMACObservationParams) error {
	_, err := q.db.Exec(ctx, insertMACObservation, arg.RunID, arg.DeviceID, arg.MAC)
	return err
}
