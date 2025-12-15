-- name: InsertAuditEvent :exec
INSERT INTO audit_events (
  actor,
  actor_role,
  action,
  target_type,
  target_id,
  details
)
VALUES ($1, $2, $3, $4, $5::uuid, COALESCE($6, '{}'::jsonb));

