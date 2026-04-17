-- +migrate Down

ALTER TABLE device_snmp DROP COLUMN IF EXISTS os_version;
ALTER TABLE device_snmp DROP COLUMN IF EXISTS os_family;
