-- +migrate Up

-- Parsed fields extracted from sysDescr for structured OS identification.
ALTER TABLE device_snmp ADD COLUMN IF NOT EXISTS os_family text NULL;
ALTER TABLE device_snmp ADD COLUMN IF NOT EXISTS os_version text NULL;
