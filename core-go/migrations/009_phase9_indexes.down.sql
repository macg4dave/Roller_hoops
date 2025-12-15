-- +migrate Down

DROP INDEX IF EXISTS discovery_run_logs_run_created_id_idx;
DROP INDEX IF EXISTS discovery_runs_started_at_id_idx;

DROP INDEX IF EXISTS interface_vlans_observed_at_idx;
DROP INDEX IF EXISTS services_observed_at_idx;
DROP INDEX IF EXISTS devices_updated_at_idx;
DROP INDEX IF EXISTS device_metadata_updated_at_idx;
DROP INDEX IF EXISTS mac_observations_observed_at_idx;
DROP INDEX IF EXISTS ip_observations_observed_at_idx;

