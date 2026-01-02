-- name: InsertDiscoveryRun :one
INSERT INTO discovery_runs (status, scope, stats)
VALUES ($1, $2, COALESCE($3, '{}'::jsonb))
RETURNING id, status, scope, stats, started_at, completed_at, last_error;

-- name: ClaimNextDiscoveryRun :one
WITH next AS (
    SELECT id
    FROM discovery_runs
    WHERE status = 'queued'
    ORDER BY started_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE discovery_runs dr
SET status = 'running',
        stats = COALESCE(dr.stats, '{}'::jsonb) || COALESCE($1, '{}'::jsonb),
        completed_at = NULL,
        last_error = NULL
FROM next
WHERE dr.id = next.id
RETURNING dr.id, dr.status, dr.scope, dr.stats, dr.started_at, dr.completed_at, dr.last_error;

-- name: UpdateDiscoveryRun :one
UPDATE discovery_runs
SET status = $2,
    stats = COALESCE(stats, '{}'::jsonb) || COALESCE($3, '{}'::jsonb),
    completed_at = $4,
    last_error = $5
WHERE id = $1
RETURNING id, status, scope, stats, started_at, completed_at, last_error;

-- name: GetLatestDiscoveryRun :one
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
ORDER BY started_at DESC
LIMIT 1;

-- name: GetDiscoveryRun :one
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
WHERE id = $1;

-- name: InsertDiscoveryRunLog :exec
INSERT INTO discovery_run_logs (run_id, level, message)
VALUES ($1, $2, $3);

-- name: ListDiscoveryRuns :many
SELECT id, status, scope, stats, started_at, completed_at, last_error
FROM discovery_runs
WHERE
    ($1::timestamptz IS NULL OR (started_at < $1::timestamptz OR (started_at = $1::timestamptz AND id < $2::uuid)))
ORDER BY started_at DESC, id DESC
LIMIT $3;

-- name: ListDiscoveryRunLogs :many
SELECT id, run_id, level, message, created_at
FROM discovery_run_logs
WHERE
    run_id = $1
    AND ($2::timestamptz IS NULL OR (created_at < $2::timestamptz OR (created_at = $2::timestamptz AND id < $3::bigint)))
ORDER BY created_at DESC, id DESC
LIMIT $4;
