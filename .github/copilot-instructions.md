# Copilot Instructions

This project is written primarily by AI coding agents. Follow the repository operating rules before changing code.

## Required Entry Point

Read [AGENTS.md](../AGENTS.md) first. It defines:

- Required reading order
- Backlog, task, and issue workflow
- Validation expectations
- Service boundaries
- Safety limits
- Build and test commands

Then read the contract document for the area you are changing:

- Execution board / dev runbook: [BACKLOG.md](../BACKLOG.md)
- API behavior: [api/openapi.yaml](../api/openapi.yaml) and [docs/api-contract.md](../docs/api-contract.md)
- Data model or migrations: [docs/data-model.md](../docs/data-model.md)
- Runtime/service boundaries: [docs/architecture.md](../docs/architecture.md)
- Operator UX: [docs/ui-ux.md](../docs/ui-ux.md)
- Network map work: [docs/network_map/interface-rules.md](../docs/network_map/interface-rules.md)
- Cross-cutting standards: [docs/engineering-standards.md](../docs/engineering-standards.md)
- AI-agent workflow control: [docs/ai-coding-control.md](../docs/ai-coding-control.md)
- VS Code workflow and tasks: [docs/vscode-ai-workflow.md](../docs/vscode-ai-workflow.md)

## Prime Directives

- Do not invent architecture. Follow the roadmap, feature matrix, and existing docs.
- Prefer boring, proven solutions over novel ones.
- Keep every feature traceable to documentation.
- Avoid formatting-only churn.
- Never commit secrets, credentials, private keys, tokens, PSKs, or production data.
- Preserve stable IDs and existing API contracts unless the change explicitly updates the contract.
- Add regression tests for fixed logic errors.

## Service Boundaries

`core-go` owns:

- Network discovery
- Polling and scheduling
- Normalization
- PostgreSQL access
- REST API
- Metrics and discovery run logs

`core-go` must not own:

- HTML rendering
- Browser workflows
- UI sessions
- Direct user interaction

`ui-node` owns:

- Next.js rendering
- Forms and operator workflows
- Authentication UI and signed sessions
- BFF proxying to the Go API

`ui-node` must not:

- Access PostgreSQL directly
- Scan networks
- Re-implement Go business rules

## Contract Rules

- OpenAPI is canonical: update `api/openapi.yaml` before or alongside API behavior changes.
- Regenerate `ui-node/lib/api-types.ts` after OpenAPI changes.
- Update `docs/api-contract.md` when request/response behavior changes.
- Update `docs/data-model.md` and migrations when persistence changes.
- Update `docs/feature-matrix.md` when feature status, endpoints, ownership, or DB tables change.

## Documentation Freshness

[BACKLOG.md](../BACKLOG.md) is the execution source of truth. Keep it accurate after every session.

- Documentation updates are mandatory, not optional follow-ups. Ship doc changes in the same session as the code they describe.
- After completing any task, update the BACKLOG.md task card status, handoff notes, and Ready Queue row.
- Update `docs/roadmap.md` when phase status changes or scope shifts.
- Update `docs/feature-matrix.md` when features are added, completed, or changed.
- Update `docs/issues.md` when a tracked bug is resolved, with status and fix reference.
- If a doc update cannot be completed, log the gap in the task card's handoff notes. Never skip silently.
- See [BACKLOG.md](../BACKLOG.md) § "Documentation Freshness Rules" for the full checklist and triggers.

## Validation Rules

- Go changes: run `gofmt`, `go vet ./...`, and `go test ./...` when available.
- UI changes: run `npm test` and `npm run build`.
- API changes: run `npm run gen:openapi` in `ui-node` and ensure no generated drift remains.
- Stack changes: run or document Docker compose smoke validation.
- If a command cannot run locally, state the exact blocker.

## Safety Rules

- Discovery behavior must stay explicit-scope and allowlist friendly.
- Do not add broad or implicit network scans.
- Do not perform remote shell/admin actions unless the user explicitly asks.
- Handle errors explicitly and make operational failures actionable.
