# AGENTS.md

## Purpose

This repository is intended to be built primarily with coding agents. Use the planning docs as the task system, not as passive documentation.

The project is a network tracker / mapper. Keep service boundaries, API contracts, data model docs, and operational safety ahead of clever implementation.

## Required Reading Order

Before starting substantial work, read:

1. [docs/roadmap.md](docs/roadmap.md)
2. [BACKLOG.md](BACKLOG.md)
3. [docs/issues.md](docs/issues.md)
4. [docs/feature-matrix.md](docs/feature-matrix.md)
5. [docs/engineering-standards.md](docs/engineering-standards.md)
6. [docs/conventions.md](docs/conventions.md)
7. [docs/vscode-ai-workflow.md](docs/vscode-ai-workflow.md) if using VS Code, Copilot, or Codex from the editor
8. [docs/architecture.md](docs/architecture.md) if the task affects runtime or service boundaries
9. [docs/api-contract.md](docs/api-contract.md) if the task affects API behavior
10. [docs/data-model.md](docs/data-model.md) if the task affects persisted data

For network map work, also read:

1. [docs/network_map/interface-rules.md](docs/network_map/interface-rules.md)
2. [docs/network_map/network_map_ideas.md](docs/network_map/network_map_ideas.md)
3. [docs/network_map/implementation-stack.md](docs/network_map/implementation-stack.md)

## Task Manager Usage

- Treat [BACKLOG.md](BACKLOG.md) as the execution source of truth for AI work.
- Treat [docs/issues.md](docs/issues.md) as the lightweight issue log and historical bug tracker.
- Treat [docs/roadmap.md](docs/roadmap.md) as the high-level product sequencing source.
- Treat [docs/feature-matrix.md](docs/feature-matrix.md) as the implemented/planned capability index. No orphan code.
- If the user assigns a future issue that is not implementation-ready, capture it as a blocked or parent task in [BACKLOG.md](BACKLOG.md) instead of burying it in chat.
- Do not mark issue work fixed unless validation ran or the blocker is explicitly documented.
- When behavior changes, update the owning docs in the same change.

## Agent Workflow

1. Read the assigned issue, roadmap section, or user request.
2. Check [BACKLOG.md](BACKLOG.md) for an existing task card or add one before substantial work.
3. Confirm the owning service and contract:
   - `core-go` owns discovery, persistence, normalization, and REST API truth.
   - `ui-node` owns rendering, forms, workflows, auth UI, sessions, and calls to the Go API.
   - PostgreSQL is the only database.
4. Confirm the feature exists in [docs/feature-matrix.md](docs/feature-matrix.md), or update the matrix before adding behavior.
5. For behavior changes, add or tighten a focused test before or alongside the implementation whenever practical.
6. Keep edits inside the task scope. If scope must expand, update the backlog card and owning docs first.
7. Update all affected docs (API, data-model, architecture, runbook, UX, roadmap, feature-matrix, issues) in the same session as the code change. See [BACKLOG.md](BACKLOG.md) § "Documentation Freshness Rules" for the full checklist.
8. Run the focused validation for the touched area, then the broader suite that is available locally.
9. Record any validation blocker in the final handoff and task card.

## Status Rules

Use these states in issue/task notes:

- `open`: acknowledged, not fixed
- `investigating`: actively being debugged
- `blocked`: cannot continue safely without an external dependency or decision
- `fixed`: resolved and validated, with a fix reference
- `won't-fix`: explicitly declined, with rationale

## Scope Rules

- Prefer the smallest change that satisfies the request.
- Do not combine unrelated issue work unless the user explicitly asks for it.
- Do not add speculative architecture, schema, endpoints, or UI flows.
- Avoid formatting-only churn.
- Never revert user changes unless explicitly instructed.

## Documentation Rules (Mandatory)

Documentation updates are not optional follow-ups. They ship in the same session as the code they describe. [BACKLOG.md](BACKLOG.md) § "Documentation Freshness Rules" is the authoritative checklist — read it before finishing any task.

### Summary Of Triggers

- If setup, ports, env vars, or commands change, update [readme.md](readme.md).
- If API behavior changes, update [api/openapi.yaml](api/openapi.yaml), [docs/api-contract.md](docs/api-contract.md), and regenerate `ui-node/lib/api-types.ts`.
- If data shapes, tables, or relationships change, update [docs/data-model.md](docs/data-model.md), [docs/migrations.md](docs/migrations.md), and add migrations.
- If service boundaries change, update [docs/architecture.md](docs/architecture.md).
- If operator workflow changes, update [docs/ui-ux.md](docs/ui-ux.md).
- If feature status or ownership changes, update [docs/feature-matrix.md](docs/feature-matrix.md).
- If a phase status changes or scope shifts, update [docs/roadmap.md](docs/roadmap.md).
- If a tracked issue is resolved, update [docs/issues.md](docs/issues.md) with status and fix reference.
- If discovery behavior or deployment changes, update [docs/discovery-capabilities.md](docs/discovery-capabilities.md) or [docs/discovery-deployment.md](docs/discovery-deployment.md).
- If auth, roles, or security boundaries change, update [docs/security.md](docs/security.md).

### Backlog Is The Source Of Truth

- [BACKLOG.md](BACKLOG.md) is the execution source of truth for all AI work. Read it before starting, update it before finishing.
- Every completed task must update the BACKLOG.md task card (status, handoff notes) and the Ready Queue table.
- If a doc update cannot be completed, log the gap as a task card or handoff note. Never silently skip it.

## Validation Rules

- Never claim a task is done without running the relevant validation or explaining why it could not be run.
- Treat compile, type-check, OpenAPI drift, or contract-test failures as blockers.
- Add regression tests for newly fixed bugs, especially auth, API, discovery, map projection, and import/export behavior.
- If validation is partial, say exactly what passed and what remains blocked.

## Project Constraints

- Keep `core-go` headless. It must not render HTML, own UI sessions, or depend on UI state.
- Keep `ui-node` out of the database and out of network scanning.
- OpenAPI is canonical for API shape. Generated TypeScript types must not drift.
- Migrations are mandatory for schema changes.
- Discovery must stay explicit-scope and allowlist friendly. Do not add broad network scans by default.
- No secrets, private keys, tokens, PSKs, or production credentials in the repo.
- Prefer deterministic behavior and small, verifiable edits.

## Safety Limits

- Do not attempt outbound remote shell access or remote administration from this workspace unless the user explicitly asks for it.
- This includes `ssh`, `scp`, `sftp`, remote `rsync`, `plink`, `pscp`, and Windows remoting commands such as `Enter-PSSession`, `New-PSSession`, or `Invoke-Command -ComputerName`.
- Do not widen discovery behavior beyond documented scopes, allowlists, and deployment guidance.
- Treat destructive local commands as approval-gated by default. Prefer narrow repo-level actions over broad filesystem commands.

## Build And Test

Use the shared Docker-backed validation workflow by default. Local Go/Node toolchains remain optional for developers who prefer native runs. A `Makefile` wraps all Docker commands for convenience (`make help` lists targets).

- Go formatting check: `docker build -f docker/validate/core-go.Dockerfile --target fmtcheck .` (`make go-fmt`)
- Go vet: `docker build -f docker/validate/core-go.Dockerfile --target vet .` (`make go-vet`)
- Go tests: `docker build -f docker/validate/core-go.Dockerfile --target test .` (`make go-test`)
- All Go checks: `make go-validate`
- UI install/dependency check: `docker build -f docker/validate/ui-node.Dockerfile --target deps .` (`make ui-deps`)
- OpenAPI type generation: use the VS Code task `ui: gen openapi types` or `make gen-types`
- UI tests: `docker build -f docker/validate/ui-node.Dockerfile --target test .` (`make ui-test`)
- UI build/type check: `docker build -f docker/validate/ui-node.Dockerfile --target build .` (`make ui-build`)
- All UI checks: `make ui-validate`
- All validation: `make validate`
- Full stack smoke: `docker compose --profile dev up --build` (`make dev`)
- Dev tools shell (Go + Node + psql): `make devtools`

## Module Boundary Rules

- Organize code by owning domain first. Prefer existing module folders over new ad hoc top-level files.
- Keep composition roots thin:
  - `core-go/cmd/core-go` wires the Go service.
  - `core-go/internal/httpapi` parses and shapes HTTP.
  - `ui-node/app` composes Next routes and UI.
- Move reusable discovery, enrichment, map, tagging, auth, or validation logic into owning modules instead of growing route handlers.
- Keep data shaping separate from presentation.
- Keep validation separate from authoritative mutation when both concerns appear in the same workflow.
- Do not let helper files become buckets for unrelated logic.
- Place new tests beside the owning module when practical.

## Responsibility Heuristics

- If a route parses requests and decides domain behavior, move the behavior into a domain module.
- If a UI component controls page flow and renders a distinct panel or dialog, extract the subview when it becomes independently testable.
- If a function becomes easy to change only because it has access to everything, extract the narrower responsibility first.
- Use the sentence test during review: if a file's responsibility cannot be described in one sentence without the word `and`, consider splitting it.
