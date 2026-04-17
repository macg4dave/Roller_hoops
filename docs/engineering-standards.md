# Engineering Standards

This document holds cross-cutting delivery rules for Roller_hoops. It is operational by design: update it when recurring engineering practice changes, and keep task-specific status in `docs/issues.md` or `docs/roadmap.md`.

If an active issue and this document disagree, treat the issue as the immediate execution source of truth and update this file when the rule should persist.

## Definition of Done

A phase, issue, or code change is complete only when all relevant conditions below are met:

- Exit criteria are validated through tests or documented manual checks.
- Go changes are formatted with `gofmt` and covered by focused tests where behavior changed.
- TypeScript/Next changes pass `npm test` and `npm run build` unless blocked by local environment.
- OpenAPI changes regenerate `ui-node/lib/api-types.ts`.
- API contract changes update `api/openapi.yaml` and `docs/api-contract.md`.
- Schema changes include migrations and updates to `docs/data-model.md`.
- Operator workflow changes update `docs/ui-ux.md`.
- Setup, environment, port, or command changes update `readme.md`.
- Feature status or ownership changes update `docs/feature-matrix.md`.

## Test Minimums

- Unit tests for pure logic, validators, parser behavior, auth decisions, and projection helpers.
- Handler or route tests for API/auth boundary behavior.
- Integration tests for database-backed Go behavior when persistence semantics change.
- UI tests for user-visible workflow state, error handling, and regression-prone interactions.
- OpenAPI drift checks after API route or schema changes.
- Docker compose smoke checks for stack-level changes.

Do not count a check as complete if it failed before exercising the relevant code. Record missing local prerequisites, such as Go or a stopped Docker daemon, as blockers.

## Contract Policy

- OpenAPI is canonical for API shape.
- `ui-node/lib/api-types.ts` is generated output and must match `api/openapi.yaml`.
- Go HTTP handlers must match the OpenAPI route list.
- Do not make silent API behavior changes. Prefer explicit versioned endpoints or documented request/response changes.
- JSON errors use the standard envelope from `docs/api-contract.md`.
- Cursor formats are opaque API details. UI code must pass them through, not parse them.

## Schema And Migration Policy

- PostgreSQL schema changes require migrations under `core-go/migrations/`.
- Keep `docs/data-model.md` aligned with table purpose, keys, and relationships.
- Prefer normalized relational state. Use JSON only for flexible evidence, stats, or upstream payloads with documented justification.
- No service except `core-go` may access the database.
- No migration should silently destroy operator-curated data.

## Responsibility Boundary Policy

- Design around responsibilities, not file length.
- Keep one layer per file: orchestration, domain logic, transport, and UI rendering should not accumulate in one module.
- Treat a new subordinate responsibility as an extraction trigger when it has its own inputs, outputs, branching, or tests.
- Keep composition roots thin. Entry points may assemble modules but should not own feature behavior.
- Keep validation separate from authoritative mutation.
- Keep data derivation separate from presentation when the derivation becomes non-trivial.
- Do not let generic helpers become catch-all buckets for unrelated logic.

## Authority Boundary Policy

- The Go API and PostgreSQL store are the source of truth for network state.
- UI state is presentation and workflow state only.
- Discovery observations are advisory until normalized and persisted by `core-go`.
- Imported inventory data must not clobber curated fields unless the behavior is explicit and documented.
- User-provided JSON and upstream inventory payloads must be validated before persistence.
- Browser sessions and roles are owned by `ui-node`; Go remains headless and unauthenticated inside the private network.

## Discovery Safety Policy

- Discovery runs must be explicitly scoped by request or configured default scope.
- Optional active scans require explicit enable flags and allowlists.
- Do not introduce scan-everything defaults.
- Network operations need timeouts, bounded concurrency, and actionable failure logs.
- Deployment-specific discovery behavior belongs in `docs/discovery-capabilities.md` and `docs/discovery-deployment.md`.

## Observability Policy

Service boundary failures should be diagnosable without reading code.

Required practices:

- Preserve `X-Request-ID` propagation.
- Emit structured logs with service, request id, method/path/status where applicable.
- Keep metrics names stable once documented.
- Avoid logging secrets, session cookies, raw credentials, or sensitive upstream payloads.
- Make discovery run logs actionable for operators.

## Documentation Ownership

| Document | Owner Role | Update Trigger |
| --- | --- | --- |
| `AGENTS.md` | Maintainer / agent lead | Agent workflow, validation, or scope policy changes |
| `docs/roadmap.md` | Product / tech lead | Phase change, milestone resequencing, or scope change |
| `docs/issues.md` | Current task owner | Issue status, repro, validation, blocker, or fix reference changes |
| `docs/feature-matrix.md` | Tech lead | Feature ownership, status, API, or DB surface changes |
| `docs/api-contract.md` | API owner | Request/response, error, route, or pagination behavior changes |
| `api/openapi.yaml` | API owner | Any API route/schema change |
| `docs/data-model.md` | Data owner | Table, relationship, migration, or persistence semantic changes |
| `docs/architecture.md` | Tech lead | Runtime boundary, trust boundary, or service responsibility changes |
| `docs/ui-ux.md` | UI owner | Operator workflow, layout, accessibility, or interaction changes |
| `docs/runbooks.md` | Operations owner | Deployment, backup, recovery, monitoring, or incident flow changes |

## Issue Intake And Documentation Sync

- Ground user-assigned future work against roadmap, feature matrix, dependencies, and open decisions before coding.
- Capture real but not-ready work as blocked or open in `docs/issues.md`.
- Prefer one parent issue plus child notes for broad work. Use a standalone issue only when the fix is small and clear.
- Each issue should name concrete validation, even if the work is documentation-only.
- Sync planning docs in the same session when work changes sequencing, runtime boundaries, or user-visible scope.
