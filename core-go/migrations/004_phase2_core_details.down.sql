-- +migrate Down

-- services
ALTER TABLE services
  DROP CONSTRAINT IF EXISTS services_port_range_chk,
  DROP CONSTRAINT IF EXISTS services_protocol_chk;

DROP INDEX IF EXISTS services_port_idx;
DROP INDEX IF EXISTS services_device_id_idx;

ALTER TABLE services
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS port,
  DROP COLUMN IF EXISTS protocol;

-- mac_addresses
ALTER TABLE mac_addresses
  DROP CONSTRAINT IF EXISTS mac_addresses_device_or_interface_chk;

DROP INDEX IF EXISTS mac_addresses_interface_id_mac_uniq;
DROP INDEX IF EXISTS mac_addresses_device_id_mac_uniq;
DROP INDEX IF EXISTS mac_addresses_interface_id_idx;
DROP INDEX IF EXISTS mac_addresses_device_id_idx;
DROP INDEX IF EXISTS mac_addresses_mac_idx;

-- ip_addresses
ALTER TABLE ip_addresses
  DROP CONSTRAINT IF EXISTS ip_addresses_device_or_interface_chk;

DROP INDEX IF EXISTS ip_addresses_interface_id_ip_uniq;
DROP INDEX IF EXISTS ip_addresses_device_id_ip_uniq;
DROP INDEX IF EXISTS ip_addresses_interface_id_idx;
DROP INDEX IF EXISTS ip_addresses_device_id_idx;
DROP INDEX IF EXISTS ip_addresses_ip_idx;

-- interfaces
DROP INDEX IF EXISTS interfaces_device_id_name_uniq;
DROP INDEX IF EXISTS interfaces_device_id_idx;

ALTER TABLE interfaces
  DROP COLUMN IF EXISTS ifindex,
  DROP COLUMN IF EXISTS name;

