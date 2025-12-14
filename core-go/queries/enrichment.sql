-- name: InsertDeviceNameCandidate :exec
INSERT INTO device_name_candidates (device_id, name, source, address)
VALUES ($1::uuid, $2, $3, $4::inet)
ON CONFLICT (device_id, source, name, address)
DO NOTHING;

-- name: ListDeviceNameCandidates :many
SELECT device_id,
       name,
       source,
       address::text,
       observed_at
FROM device_name_candidates
WHERE device_id = $1::uuid
ORDER BY observed_at DESC, source ASC, name ASC;

-- name: SetDeviceDisplayNameIfUnset :execrows
UPDATE devices
SET display_name = $2,
    updated_at = now()
WHERE id = $1::uuid
  AND (display_name IS NULL OR btrim(display_name) = '');

-- name: UpsertDeviceSNMP :exec
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
    updated_at = now();

-- name: UpsertInterfaceFromSNMP :one
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
RETURNING id;

-- name: UpsertInterfaceMAC :exec
INSERT INTO mac_addresses (device_id, interface_id, mac)
VALUES ($1::uuid, $2::uuid, $3::macaddr)
ON CONFLICT (interface_id, mac) WHERE interface_id IS NOT NULL
DO UPDATE SET device_id = EXCLUDED.device_id,
              updated_at = now();

-- name: LinkDeviceMACToInterface :execrows
UPDATE mac_addresses
SET interface_id = $3::uuid,
    updated_at = now()
WHERE device_id = $1::uuid
  AND mac = $2::macaddr
  AND interface_id IS NULL;

-- name: UpsertInterfaceVLAN :exec
INSERT INTO interface_vlans (interface_id, vlan_id, role, source)
VALUES ($1::uuid, $2, $3, $4)
ON CONFLICT (interface_id, role) DO UPDATE
SET vlan_id = EXCLUDED.vlan_id,
    source = EXCLUDED.source,
    observed_at = now();

