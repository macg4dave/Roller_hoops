# Migrations

Migrations live in `core-go/migrations/` and are applied automatically by the `migrate` service when you run `docker compose up`.

## Running locally

- Apply the latest migrations: `docker compose run --rm migrate`
- Re-run from scratch (dev only): `docker compose down -v && docker compose up --build`

## Adding a migration

1. Create paired files following the existing numbering (`NNN_description.up.sql` / `NNN_description.down.sql`).
2. Keep migrations idempotent and Postgres-friendly (`IF NOT EXISTS`, defensive drops in `.down.sql`).
3. Regenerate `internal/sqlcgen` if you add queries/columns (`sqlc generate`).

The Go service checks readiness via `/readyz` which pings Postgres; it will return `503` if migrations haven’t been applied yet.
