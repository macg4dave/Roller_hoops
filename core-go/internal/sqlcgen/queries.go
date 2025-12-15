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

const insertAuditEvent = `-- name: InsertAuditEvent :exec
INSERT INTO audit_events (
  actor,
  actor_role,
  action,
  target_type,
  target_id,
  details
)
VALUES ($1, $2, $3, $4, $5::uuid, COALESCE($6, '{}'::jsonb))
`

type InsertAuditEventParams struct {
	Actor      string
	ActorRole  *string
	Action     string
	TargetType *string
	TargetID   *string
	Details    map[string]any
}

func (q *Queries) InsertAuditEvent(ctx context.Context, arg InsertAuditEventParams) error {
	_, err := q.db.Exec(ctx, insertAuditEvent, arg.Actor, arg.ActorRole, arg.Action, arg.TargetType, arg.TargetID, arg.Details)
	return err
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

const listDevicesPage = `-- name: ListDevicesPage :many
WITH computed AS (
	SELECT
		d.id,
		d.display_name,
		m.owner,
		m.location,
		m.notes,
		d.created_at,
		d.updated_at,
		(
			SELECT MAX(ts)
			FROM (
				VALUES
					(
						(
							SELECT MAX(ia.updated_at)
							FROM ip_addresses ia
							LEFT JOIN interfaces i ON i.id = ia.interface_id
							WHERE ia.device_id = d.id OR i.device_id = d.id
						)
					),
					(
						(
							SELECT MAX(ma.updated_at)
							FROM mac_addresses ma
							LEFT JOIN interfaces i ON i.id = ma.interface_id
							WHERE ma.device_id = d.id OR i.device_id = d.id
						)
					),
					((SELECT MAX(s.observed_at) FROM services s WHERE s.device_id = d.id)),
					((SELECT MAX(ds.last_success_at) FROM device_snmp ds WHERE ds.device_id = d.id))
			) v(ts)
		) AS last_seen_at,
		(
			SELECT MAX(ts)
			FROM (
				VALUES
					(d.updated_at),
					((SELECT MAX(dm.updated_at) FROM device_metadata dm WHERE dm.device_id = d.id)),
					(
						(
							SELECT MAX(ia.created_at)
							FROM ip_addresses ia
							LEFT JOIN interfaces i ON i.id = ia.interface_id
							WHERE ia.device_id = d.id OR i.device_id = d.id
						)
					),
					(
						(
							SELECT MAX(ma.created_at)
							FROM mac_addresses ma
							LEFT JOIN interfaces i ON i.id = ma.interface_id
							WHERE ma.device_id = d.id OR i.device_id = d.id
						)
					),
					((SELECT MAX(s.created_at) FROM services s WHERE s.device_id = d.id)),
					((SELECT MAX(ds.updated_at) FROM device_snmp ds WHERE ds.device_id = d.id)),
					(
						(
							SELECT MAX(iv.observed_at)
							FROM interface_vlans iv
							JOIN interfaces i ON i.id = iv.interface_id
							WHERE i.device_id = d.id
						)
					),
					(
						(
							SELECT MAX(COALESCE(l.observed_at, l.updated_at))
							FROM links l
							WHERE l.a_device_id = d.id OR l.b_device_id = d.id
						)
					)
			) v(ts)
		) AS last_change_at
	FROM devices d
	LEFT JOIN device_metadata m ON m.device_id = d.id
)
SELECT
	q.id,
	q.display_name,
	q.owner,
	q.location,
	q.notes,
	q.created_at,
	q.updated_at,
	q.last_seen_at,
	q.last_change_at,
	q.sort_ts
FROM (
	SELECT
		c.*,
		CASE
			WHEN $3 = 'last_seen_desc' THEN COALESCE(c.last_seen_at, '1970-01-01T00:00:00Z'::timestamptz)
			WHEN $3 = 'last_change_desc' THEN COALESCE(c.last_change_at, '1970-01-01T00:00:00Z'::timestamptz)
			ELSE c.created_at
		END AS sort_ts
	FROM computed c
	WHERE
		(
			$1 IS NULL
			OR (
				c.id::text ILIKE $1
				OR COALESCE(c.display_name, '') ILIKE $1
				OR COALESCE(c.owner, '') ILIKE $1
				OR COALESCE(c.location, '') ILIKE $1
				OR COALESCE(c.notes, '') ILIKE $1
				OR EXISTS (
					SELECT 1
					FROM ip_addresses ia
					LEFT JOIN interfaces i ON i.id = ia.interface_id
					WHERE (ia.device_id = c.id OR i.device_id = c.id)
					  AND ia.ip::text ILIKE $1
				)
				OR EXISTS (
					SELECT 1
					FROM mac_addresses ma
					LEFT JOIN interfaces i ON i.id = ma.interface_id
					WHERE (ma.device_id = c.id OR i.device_id = c.id)
					  AND ma.mac::text ILIKE $1
				)
				OR EXISTS (
					SELECT 1
					FROM device_snmp ds
					WHERE ds.device_id = c.id AND (
						COALESCE(ds.sys_name, '') ILIKE $1
						OR COALESCE(ds.sys_descr, '') ILIKE $1
						OR COALESCE(ds.sys_location, '') ILIKE $1
						OR COALESCE(ds.sys_contact, '') ILIKE $1
					)
				)
			)
		)
		AND (
			$2 IS NULL
			OR $2 = ''
			OR ($2 = 'online' AND c.last_seen_at IS NOT NULL AND c.last_seen_at >= $4)
			OR ($2 = 'offline' AND (c.last_seen_at IS NULL OR c.last_seen_at < $4))
			OR ($2 = 'changed' AND c.last_change_at >= $5)
		)
) q
WHERE
	($6 IS NULL OR (q.sort_ts < $6 OR (q.sort_ts = $6 AND q.id < $7)))
ORDER BY q.sort_ts DESC, q.id DESC
LIMIT $8
`

type ListDevicesPageParams struct {
	Query        *string
	Status       *string
	Sort         string
	SeenAfter    time.Time
	ChangedAfter time.Time
	BeforeSortTs *time.Time
	BeforeID     *string
	Limit        int32
}

func (q *Queries) ListDevicesPage(ctx context.Context, arg ListDevicesPageParams) ([]DeviceListItem, error) {
	rows, err := q.db.Query(
		ctx,
		listDevicesPage,
		arg.Query,
		arg.Status,
		arg.Sort,
		arg.SeenAfter,
		arg.ChangedAfter,
		arg.BeforeSortTs,
		arg.BeforeID,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceListItem
	for rows.Next() {
		var i DeviceListItem
		if err := rows.Scan(
			&i.ID,
			&i.DisplayName,
			&i.Owner,
			&i.Location,
			&i.Notes,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.LastSeenAt,
			&i.LastChangeAt,
			&i.SortTs,
		); err != nil {
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

const upsertDeviceMetadataFillBlank = `-- name: UpsertDeviceMetadataFillBlank :one
INSERT INTO device_metadata (device_id, owner, location, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (device_id) DO UPDATE
SET owner = CASE
              WHEN device_metadata.owner IS NULL OR btrim(device_metadata.owner) = '' THEN EXCLUDED.owner
              ELSE device_metadata.owner
            END,
    location = CASE
                 WHEN device_metadata.location IS NULL OR btrim(device_metadata.location) = '' THEN EXCLUDED.location
                 ELSE device_metadata.location
               END,
    notes = CASE
              WHEN device_metadata.notes IS NULL OR btrim(device_metadata.notes) = '' THEN EXCLUDED.notes
              ELSE device_metadata.notes
            END,
    updated_at = now()
RETURNING device_id, owner, location, notes
`

func (q *Queries) UpsertDeviceMetadataFillBlank(ctx context.Context, arg UpsertDeviceMetadataParams) (DeviceMetadata, error) {
	row := q.db.QueryRow(ctx, upsertDeviceMetadataFillBlank, arg.DeviceID, arg.Owner, arg.Location, arg.Notes)
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

const listDeviceIPs = `-- name: ListDeviceIPs :many
SELECT ia.ip::text,
       ia.interface_id::text,
       i.name AS interface_name,
       ia.created_at,
       ia.updated_at
FROM ip_addresses ia
LEFT JOIN interfaces i ON i.id = ia.interface_id
WHERE ia.device_id = $1::uuid OR i.device_id = $1::uuid
ORDER BY ia.updated_at DESC, ia.ip::text ASC
`

func (q *Queries) ListDeviceIPs(ctx context.Context, deviceID string) ([]DeviceIP, error) {
	rows, err := q.db.Query(ctx, listDeviceIPs, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceIP
	for rows.Next() {
		var i DeviceIP
		if err := rows.Scan(&i.IP, &i.InterfaceID, &i.InterfaceName, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDeviceMACs = `-- name: ListDeviceMACs :many
SELECT ma.mac::text,
       ma.interface_id::text,
       i.name AS interface_name,
       ma.created_at,
       ma.updated_at
FROM mac_addresses ma
LEFT JOIN interfaces i ON i.id = ma.interface_id
WHERE ma.device_id = $1::uuid OR i.device_id = $1::uuid
ORDER BY ma.updated_at DESC, ma.mac::text ASC
`

func (q *Queries) ListDeviceMACs(ctx context.Context, deviceID string) ([]DeviceMAC, error) {
	rows, err := q.db.Query(ctx, listDeviceMACs, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceMAC
	for rows.Next() {
		var i DeviceMAC
		if err := rows.Scan(&i.MAC, &i.InterfaceID, &i.InterfaceName, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDeviceInterfaces = `-- name: ListDeviceInterfaces :many
SELECT i.id,
       i.name,
       i.ifindex,
       i.descr,
       i.alias,
       i.mac::text,
       i.admin_status,
       i.oper_status,
       i.mtu,
       i.speed_bps,
       iv.vlan_id AS pvid,
       iv.observed_at AS pvid_observed_at,
       i.created_at,
       i.updated_at
FROM interfaces i
LEFT JOIN interface_vlans iv ON iv.interface_id = i.id AND iv.role = 'pvid'
WHERE i.device_id = $1::uuid
ORDER BY i.name IS NULL, i.name ASC, i.ifindex IS NULL, i.ifindex ASC, i.id ASC
`

func (q *Queries) ListDeviceInterfaces(ctx context.Context, deviceID string) ([]DeviceInterface, error) {
	rows, err := q.db.Query(ctx, listDeviceInterfaces, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceInterface
	for rows.Next() {
		var i DeviceInterface
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Ifindex,
			&i.Descr,
			&i.Alias,
			&i.MAC,
			&i.AdminStatus,
			&i.OperStatus,
			&i.MTU,
			&i.SpeedBps,
			&i.PVID,
			&i.PVIDObservedAt,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDeviceServices = `-- name: ListDeviceServices :many
SELECT protocol,
       port,
       name,
       state,
       source,
       observed_at,
       created_at,
       updated_at
FROM services
WHERE device_id = $1::uuid
ORDER BY observed_at DESC, protocol ASC NULLS LAST, port ASC NULLS LAST, name ASC NULLS LAST
`

func (q *Queries) ListDeviceServices(ctx context.Context, deviceID string) ([]DeviceService, error) {
	rows, err := q.db.Query(ctx, listDeviceServices, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceService
	for rows.Next() {
		var i DeviceService
		if err := rows.Scan(&i.Protocol, &i.Port, &i.Name, &i.State, &i.Source, &i.ObservedAt, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDeviceSNMP = `-- name: GetDeviceSNMP :one
SELECT device_id,
       address::text,
       sys_name,
       sys_descr,
       sys_object_id,
       sys_contact,
       sys_location,
       last_success_at,
       last_error,
       updated_at
FROM device_snmp
WHERE device_id = $1::uuid
`

func (q *Queries) GetDeviceSNMP(ctx context.Context, deviceID string) (DeviceSNMP, error) {
	row := q.db.QueryRow(ctx, getDeviceSNMP, deviceID)
	var i DeviceSNMP
	err := row.Scan(
		&i.DeviceID,
		&i.Address,
		&i.SysName,
		&i.SysDescr,
		&i.SysObjectID,
		&i.SysContact,
		&i.SysLocation,
		&i.LastSuccessAt,
		&i.LastError,
		&i.UpdatedAt,
	)
	return i, err
}

const listDeviceLinks = `-- name: ListDeviceLinks :many
SELECT l.id,
       l.link_key,
       CASE WHEN l.a_device_id = $1::uuid THEN l.b_device_id::text ELSE l.a_device_id::text END AS peer_device_id,
       CASE WHEN l.a_device_id = $1::uuid THEN l.a_interface_id::text ELSE l.b_interface_id::text END AS local_interface_id,
       CASE WHEN l.a_device_id = $1::uuid THEN l.b_interface_id::text ELSE l.a_interface_id::text END AS peer_interface_id,
       l.link_type,
       l.source,
       l.observed_at,
       l.updated_at
FROM links l
WHERE l.a_device_id = $1::uuid OR l.b_device_id = $1::uuid
ORDER BY COALESCE(l.observed_at, l.updated_at) DESC, l.id DESC
`

func (q *Queries) ListDeviceLinks(ctx context.Context, deviceID string) ([]DeviceLink, error) {
	rows, err := q.db.Query(ctx, listDeviceLinks, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DeviceLink
	for rows.Next() {
		var i DeviceLink
		if err := rows.Scan(
			&i.ID,
			&i.LinkKey,
			&i.PeerDeviceID,
			&i.LocalInterfaceID,
			&i.PeerInterfaceID,
			&i.LinkType,
			&i.Source,
			&i.ObservedAt,
			&i.UpdatedAt,
		); err != nil {
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

const upsertLink = `-- name: UpsertLink :exec
INSERT INTO links (
  link_key,
  a_device_id,
  a_interface_id,
  b_device_id,
  b_interface_id,
  link_type,
  source,
  observed_at
)
VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7, $8)
ON CONFLICT (link_key) DO UPDATE
SET a_device_id = EXCLUDED.a_device_id,
    a_interface_id = EXCLUDED.a_interface_id,
    b_device_id = EXCLUDED.b_device_id,
    b_interface_id = EXCLUDED.b_interface_id,
    link_type = EXCLUDED.link_type,
    source = EXCLUDED.source,
    observed_at = EXCLUDED.observed_at,
    updated_at = now()
`

type UpsertLinkParams struct {
	LinkKey      string
	ADeviceID    string
	AInterfaceID *string
	BDeviceID    string
	BInterfaceID *string
	LinkType     *string
	Source       string
	ObservedAt   *time.Time
}

func (q *Queries) UpsertLink(ctx context.Context, arg UpsertLinkParams) error {
	_, err := q.db.Exec(ctx, upsertLink, arg.LinkKey, arg.ADeviceID, arg.AInterfaceID, arg.BDeviceID, arg.BInterfaceID, arg.LinkType, arg.Source, arg.ObservedAt)
	return err
}

const upsertInterfaceByName = `-- name: UpsertInterfaceByName :one
INSERT INTO interfaces (device_id, name)
VALUES ($1::uuid, $2)
ON CONFLICT (device_id, name) WHERE name IS NOT NULL
DO UPDATE SET updated_at = now()
RETURNING id
`

type UpsertInterfaceByNameParams struct {
	DeviceID string
	Name     string
}

func (q *Queries) UpsertInterfaceByName(ctx context.Context, arg UpsertInterfaceByNameParams) (string, error) {
	row := q.db.QueryRow(ctx, upsertInterfaceByName, arg.DeviceID, arg.Name)
	var id string
	err := row.Scan(&id)
	return id, err
}

const upsertServiceFromScan = `-- name: UpsertServiceFromScan :exec
INSERT INTO services (
  device_id,
  protocol,
  port,
  name,
  state,
  source,
  observed_at
)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
ON CONFLICT (device_id, protocol, port) WHERE protocol IS NOT NULL AND port IS NOT NULL
DO UPDATE
SET name = EXCLUDED.name,
    state = EXCLUDED.state,
    source = EXCLUDED.source,
    observed_at = EXCLUDED.observed_at,
    updated_at = now()
`

type UpsertServiceFromScanParams struct {
	DeviceID   string
	Protocol   string
	Port       int32
	Name       *string
	State      *string
	Source     *string
	ObservedAt time.Time
}

func (q *Queries) UpsertServiceFromScan(ctx context.Context, arg UpsertServiceFromScanParams) error {
	_, err := q.db.Exec(ctx, upsertServiceFromScan, arg.DeviceID, arg.Protocol, arg.Port, arg.Name, arg.State, arg.Source, arg.ObservedAt)
	return err
}

const listDeviceChangeEvents = `-- name: ListDeviceChangeEvents :many
WITH events AS (
	SELECT
		'ip_observation:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'ip_observation' AS kind,
		ip::text AS summary,
		jsonb_build_object('run_id', run_id, 'ip', ip::text) AS details
	FROM ip_observations
	UNION ALL
	SELECT
		'mac_observation:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'mac_observation' AS kind,
		mac::text AS summary,
		jsonb_build_object('run_id', run_id, 'mac', mac::text) AS details
	FROM mac_observations
	UNION ALL
	SELECT
		'interface_vlan:' || iv.id::text AS event_id,
		i.device_id,
		iv.observed_at AS event_at,
		'vlan' AS kind,
		'vlan ' || iv.vlan_id::text AS summary,
		jsonb_build_object(
			'interface_id', iv.interface_id,
			'vlan_id', iv.vlan_id,
			'role', iv.role,
			'source', iv.source
		) AS details
	FROM interface_vlans iv
	JOIN interfaces i ON i.id = iv.interface_id
	UNION ALL
	SELECT
		'link:' || l.id::text AS event_id,
		l.a_device_id AS device_id,
		COALESCE(l.observed_at, l.updated_at) AS event_at,
		'link' AS kind,
		'link to ' || l.b_device_id::text AS summary,
		jsonb_build_object(
			'a_device_id', l.a_device_id,
			'a_interface_id', l.a_interface_id,
			'b_device_id', l.b_device_id,
			'b_interface_id', l.b_interface_id,
			'link_type', l.link_type,
			'source', l.source
		) AS details
	FROM links l
	UNION ALL
	SELECT
		'link:' || l.id::text AS event_id,
		l.b_device_id AS device_id,
		COALESCE(l.observed_at, l.updated_at) AS event_at,
		'link' AS kind,
		'link to ' || l.a_device_id::text AS summary,
		jsonb_build_object(
			'a_device_id', l.a_device_id,
			'a_interface_id', l.a_interface_id,
			'b_device_id', l.b_device_id,
			'b_interface_id', l.b_interface_id,
			'link_type', l.link_type,
			'source', l.source
		) AS details
	FROM links l
	UNION ALL
	SELECT
		'metadata:' || id::text AS event_id,
		device_id,
		updated_at AS event_at,
		'metadata' AS kind,
		COALESCE(owner, location, notes, 'metadata updated') AS summary,
		jsonb_build_object('owner', owner, 'location', location, 'notes', notes) AS details
	FROM device_metadata
	UNION ALL
	SELECT
		'device_display_name:' || id::text AS event_id,
		id AS device_id,
		updated_at AS event_at,
		'display_name' AS kind,
		COALESCE(display_name, 'device updated') AS summary,
		jsonb_build_object('display_name', display_name) AS details
	FROM devices
	UNION ALL
	SELECT
		'snmp:' || device_id::text AS event_id,
		device_id,
		updated_at AS event_at,
		'snmp' AS kind,
		COALESCE(sys_name, 'snmp updated') AS summary,
		jsonb_build_object(
			'address', address::text,
			'sys_name', sys_name,
			'sys_descr', sys_descr,
			'sys_object_id', sys_object_id,
			'sys_contact', sys_contact,
			'sys_location', sys_location,
			'last_success_at', last_success_at,
			'last_error', last_error
		) AS details
	FROM device_snmp
	UNION ALL
	SELECT
		'service:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'service' AS kind,
		COALESCE(name, CONCAT(COALESCE(protocol, 'unknown'), '/', COALESCE(port::text, '0'))) AS summary,
		jsonb_build_object(
			'port', port,
			'protocol', protocol,
			'state', state,
			'source', source,
			'name', name
		) AS details
	FROM services
)
SELECT
	event_id,
	device_id,
	event_at,
	kind,
	summary,
	details
FROM events
WHERE
	($1 IS NULL OR (event_at < $1 OR (event_at = $1 AND event_id < $2)))
	AND ($3 IS NULL OR event_at >= $3)
ORDER BY event_at DESC, event_id DESC
LIMIT $4
`

func (q *Queries) ListDeviceChangeEvents(ctx context.Context, arg ListDeviceChangeEventsParams) ([]DeviceChangeEvent, error) {
	rows, err := q.db.Query(ctx, listDeviceChangeEvents, arg.BeforeEventAt, arg.BeforeEventID, arg.SinceEventAt, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DeviceChangeEvent
	for rows.Next() {
		var i DeviceChangeEvent
		if err := rows.Scan(&i.EventID, &i.DeviceID, &i.EventAt, &i.Kind, &i.Summary, &i.Details); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDeviceChangeEventsForDevice = `-- name: ListDeviceChangeEventsForDevice :many
WITH events AS (
	SELECT
		'ip_observation:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'ip_observation' AS kind,
		ip::text AS summary,
		jsonb_build_object('run_id', run_id, 'ip', ip::text) AS details
	FROM ip_observations
	UNION ALL
	SELECT
		'mac_observation:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'mac_observation' AS kind,
		mac::text AS summary,
		jsonb_build_object('run_id', run_id, 'mac', mac::text) AS details
	FROM mac_observations
	UNION ALL
	SELECT
		'interface_vlan:' || iv.id::text AS event_id,
		i.device_id,
		iv.observed_at AS event_at,
		'vlan' AS kind,
		'vlan ' || iv.vlan_id::text AS summary,
		jsonb_build_object(
			'interface_id', iv.interface_id,
			'vlan_id', iv.vlan_id,
			'role', iv.role,
			'source', iv.source
		) AS details
	FROM interface_vlans iv
	JOIN interfaces i ON i.id = iv.interface_id
	UNION ALL
	SELECT
		'link:' || l.id::text AS event_id,
		l.a_device_id AS device_id,
		COALESCE(l.observed_at, l.updated_at) AS event_at,
		'link' AS kind,
		'link to ' || l.b_device_id::text AS summary,
		jsonb_build_object(
			'a_device_id', l.a_device_id,
			'a_interface_id', l.a_interface_id,
			'b_device_id', l.b_device_id,
			'b_interface_id', l.b_interface_id,
			'link_type', l.link_type,
			'source', l.source
		) AS details
	FROM links l
	UNION ALL
	SELECT
		'link:' || l.id::text AS event_id,
		l.b_device_id AS device_id,
		COALESCE(l.observed_at, l.updated_at) AS event_at,
		'link' AS kind,
		'link to ' || l.a_device_id::text AS summary,
		jsonb_build_object(
			'a_device_id', l.a_device_id,
			'a_interface_id', l.a_interface_id,
			'b_device_id', l.b_device_id,
			'b_interface_id', l.b_interface_id,
			'link_type', l.link_type,
			'source', l.source
		) AS details
	FROM links l
	UNION ALL
	SELECT
		'metadata:' || id::text AS event_id,
		device_id,
		updated_at AS event_at,
		'metadata' AS kind,
		COALESCE(owner, location, notes, 'metadata updated') AS summary,
		jsonb_build_object('owner', owner, 'location', location, 'notes', notes) AS details
	FROM device_metadata
	UNION ALL
	SELECT
		'device_display_name:' || id::text AS event_id,
		id AS device_id,
		updated_at AS event_at,
		'display_name' AS kind,
		COALESCE(display_name, 'device updated') AS summary,
		jsonb_build_object('display_name', display_name) AS details
	FROM devices
	UNION ALL
	SELECT
		'snmp:' || device_id::text AS event_id,
		device_id,
		updated_at AS event_at,
		'snmp' AS kind,
		COALESCE(sys_name, 'snmp updated') AS summary,
		jsonb_build_object(
			'address', address::text,
			'sys_name', sys_name,
			'sys_descr', sys_descr,
			'sys_object_id', sys_object_id,
			'sys_contact', sys_contact,
			'sys_location', sys_location,
			'last_success_at', last_success_at,
			'last_error', last_error
		) AS details
	FROM device_snmp
	UNION ALL
	SELECT
		'service:' || id::text AS event_id,
		device_id,
		observed_at AS event_at,
		'service' AS kind,
		COALESCE(name, CONCAT(COALESCE(protocol, 'unknown'), '/', COALESCE(port::text, '0'))) AS summary,
		jsonb_build_object(
			'port', port,
			'protocol', protocol,
			'state', state,
			'source', source,
			'name', name
		) AS details
	FROM services
)
SELECT
	event_id,
	device_id,
	event_at,
	kind,
	summary,
	details
FROM events
WHERE
	device_id = $1
	AND ($2 IS NULL OR (event_at < $2 OR (event_at = $2 AND event_id < $3)))
ORDER BY event_at DESC, event_id DESC
LIMIT $4
`

func (q *Queries) ListDeviceChangeEventsForDevice(ctx context.Context, arg ListDeviceChangeEventsForDeviceParams) ([]DeviceChangeEvent, error) {
	rows, err := q.db.Query(ctx, listDeviceChangeEventsForDevice, arg.DeviceID, arg.BeforeEventAt, arg.BeforeEventID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DeviceChangeEvent
	for rows.Next() {
		var i DeviceChangeEvent
		if err := rows.Scan(&i.EventID, &i.DeviceID, &i.EventAt, &i.Kind, &i.Summary, &i.Details); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDiscoveryRuns = `-- name: ListDiscoveryRuns :many
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
WHERE
	($1 IS NULL OR (started_at < $1 OR (started_at = $1 AND id < $2)))
ORDER BY started_at DESC, id DESC
LIMIT $3
`

type ListDiscoveryRunsParams struct {
	BeforeStartedAt *time.Time
	BeforeID        *string
	Limit           int32
}

func (q *Queries) ListDiscoveryRuns(ctx context.Context, arg ListDiscoveryRunsParams) ([]DiscoveryRun, error) {
	rows, err := q.db.Query(ctx, listDiscoveryRuns, arg.BeforeStartedAt, arg.BeforeID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DiscoveryRun
	for rows.Next() {
		var i DiscoveryRun
		if err := rows.Scan(&i.ID, &i.Status, &i.Scope, &i.Stats, &i.StartedAt, &i.CompletedAt, &i.LastError); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listDiscoveryRunLogs = `-- name: ListDiscoveryRunLogs :many
SELECT id, run_id, level, message, created_at
FROM discovery_run_logs
WHERE
	run_id = $1
	AND ($2 IS NULL OR (created_at < $2 OR (created_at = $2 AND id < $3)))
ORDER BY created_at DESC, id DESC
LIMIT $4
`

type ListDiscoveryRunLogsParams struct {
	RunID           string
	BeforeCreatedAt *time.Time
	BeforeID        *int64
	Limit           int32
}

func (q *Queries) ListDiscoveryRunLogs(ctx context.Context, arg ListDiscoveryRunLogsParams) ([]DiscoveryRunLog, error) {
	rows, err := q.db.Query(ctx, listDiscoveryRunLogs, arg.RunID, arg.BeforeCreatedAt, arg.BeforeID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DiscoveryRunLog
	for rows.Next() {
		var i DiscoveryRunLog
		if err := rows.Scan(&i.ID, &i.RunID, &i.Level, &i.Message, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
