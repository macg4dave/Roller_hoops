-- +migrate Down

DROP TABLE IF EXISTS interface_vlans;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'interfaces_device_id_ifindex_uniq'
  ) THEN
    ALTER TABLE interfaces DROP CONSTRAINT interfaces_device_id_ifindex_uniq;
  END IF;
END $$;

ALTER TABLE interfaces
  DROP COLUMN IF EXISTS speed_bps,
  DROP COLUMN IF EXISTS mtu,
  DROP COLUMN IF EXISTS oper_status,
  DROP COLUMN IF EXISTS admin_status,
  DROP COLUMN IF EXISTS mac,
  DROP COLUMN IF EXISTS alias,
  DROP COLUMN IF EXISTS descr;

DROP TABLE IF EXISTS device_name_candidates;
DROP TABLE IF EXISTS device_snmp;

