-- +migrate Down

DROP INDEX IF EXISTS device_zones_zone_id_idx;
DROP INDEX IF EXISTS device_zones_device_zone_uniq;
DROP TABLE IF EXISTS device_zones;
DROP TABLE IF EXISTS zones;
