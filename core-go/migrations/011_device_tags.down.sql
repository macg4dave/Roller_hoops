-- +migrate Down

DROP INDEX IF EXISTS device_tags_tag_idx;
DROP INDEX IF EXISTS device_tags_device_id_idx;
DROP INDEX IF EXISTS device_tags_device_tag_source_uniq;
DROP TABLE IF EXISTS device_tags;

