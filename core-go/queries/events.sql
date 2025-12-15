-- name: ListDeviceChangeEvents :many
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
LIMIT $4;

-- name: ListDeviceChangeEventsForDevice :many
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
LIMIT $4;
