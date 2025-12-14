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
        stats = COALESCE($1, dr.stats),
        completed_at = NULL,
        last_error = NULL
FROM next
WHERE dr.id = next.id
RETURNING dr.id, dr.status, dr.scope, dr.stats, dr.started_at, dr.completed_at, dr.last_error;

-- name: UpdateDiscoveryRun :one
UPDATE discovery_runs
SET status = $2,
    stats = COALESCE($3, stats),
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
