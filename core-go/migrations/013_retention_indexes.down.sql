-- +migrate Down

DROP INDEX IF EXISTS discovery_run_logs_created_at_idx;

