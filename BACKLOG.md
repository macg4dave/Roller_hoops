# Backlog

This document is the AI-facing execution board and developer runbook for Roller_hoops. It is optimized for coding agents and humans working through small, verifiable changes without losing the project's service boundaries or safety posture.

If this file and [docs/roadmap.md](docs/roadmap.md) disagree on active sequencing or execution detail, this file is the execution source of truth and the roadmap should be updated to match. The roadmap remains the source for higher-level product scope and milestone framing.

If this file and implementation disagree, update this file before starting new work.

Roller_hoops is a self-hosted network tracker / mapper:

- `core-go`: headless Go service for discovery, normalization, persistence, REST API, metrics, and worker behavior.
- `ui-node`: Next.js service for UI rendering, forms, workflows, authentication, signed sessions, and BFF proxying to Go.
- PostgreSQL: the only database, private to `core-go`.
- Traefik: single exposed entrypoint; keeps `core-go` and Postgres private in normal deployment.

## How Agents Must Use This File

1. Read [AGENTS.md](AGENTS.md), this file, and [docs/engineering-standards.md](docs/engineering-standards.md) before starting substantial work.
2. Choose work from `## Ready Queue` by selecting a row with `Status` = `Ready`, unless the user explicitly assigns a different task.
3. Claim exactly one task card by changing its `Status` from `Ready` to `In Progress`.
4. Do only the work described in that task card unless a blocker forces a documented expansion.
5. Keep implementation, tests, API contracts, generated types, and docs synchronized.
6. Run the listed validation commands before marking the task complete.
7. Update the task card, queue table, and affected docs before ending the session.
8. If blocked, change the task to `Blocked` and add a one-line blocker note.
9. If a user assigns a future-looking or cross-cutting issue, ground it against roadmap, feature matrix, dependencies, and open decisions before writing tasks.
10. Default non-trivial assigned issues to one parent backlog item plus explicit child tasks unless one small standalone task is clearly sufficient.
11. If a future issue is not safely startable yet, still capture it here using `Blocked`, dependencies, or an open decision.
12. When assigned work changes sequencing, user-visible scope, runtime boundaries, or delivery policy, sync the owning planning docs in the same session.

## Status Model

- `Ready`: fully specified and safe for an agent to start once listed dependencies are satisfied.
- `In Progress`: currently being worked by one agent.
- `Blocked`: cannot proceed because a dependency, decision, missing tool, or missing context prevents safe execution.
- `Review`: implementation is done but awaits human or follow-up agent review, or validation is partial.
- `Done`: validated and fully handed off.
- `Dropped`: intentionally removed from scope.

## Queue Model

- `Now`: should be worked in the current phase.
- `Next`: can be prepared now but should not start until `Now` work is stable.
- `Later`: intentionally deferred.

## Priority Model

- `P0`: urgent correctness, security, data loss, or deployability issue.
- `P1`: important current-phase work or high-value reliability/UX improvement.
- `P2`: useful cleanup, polish, or preparatory work.
- `P3`: optional or speculative until more evidence appears.

## Owner Roles

- `Tech lead`: architecture, sequencing, contracts, cross-service policy.
- `Core owner`: Go API, discovery, persistence, migrations, metrics.
- `UI owner`: Next.js UI, auth/session UI, BFF routes, operator workflows.
- `Data owner`: schema, migrations, query shape, retention.
- `Operations owner`: Docker, runbooks, health, CI, deployment.
- `Security owner`: auth, roles, secret handling, exposure boundaries.
- `Docs owner`: documentation consistency and backlog hygiene.

## Documentation Freshness Rules (Mandatory)

This file is the **known truth** for AI planning and execution. All planning docs must stay in sync with this file and with the actual codebase. Documentation drift is treated as a defect.

### Core Mandate

Every agent session that changes behavior, completes work, or updates status **must** leave the affected documentation accurate before ending. There is no "I'll update docs later." Doc updates ship in the same session as the work they describe.

### When To Update This File

- **Starting work**: change the task card status from `Ready` to `In Progress` and update the Ready Queue table.
- **Completing work**: change the task card status to `Done`, fill in the Handoff Notes, and update the Ready Queue table.
- **Blocked work**: change the task card status to `Blocked`, add a one-line blocker note, and update the Ready Queue table.
- **New work**: add a task card and a Ready Queue row before starting substantial implementation.
- **Scope changes**: update the task card scope, files-to-touch, and definition-of-done before expanding work.

### When To Update Other Docs

Every change must update the owning documentation. Use this checklist — skip only the rows that are genuinely unaffected:

| What Changed | Docs To Update |
| --- | --- |
| API routes, request/response shapes, error behavior | `api/openapi.yaml`, `docs/api-contract.md`, regenerate `ui-node/lib/api-types.ts` |
| Database tables, columns, relationships, migrations | `docs/data-model.md`, `docs/migrations.md` |
| Feature added, removed, or status changed | `docs/feature-matrix.md` |
| Phase completed, milestone moved, or scope shifted | `docs/roadmap.md` |
| Service boundary, trust boundary, or runtime wiring changed | `docs/architecture.md` |
| Operator workflow, UI layout, or interaction changed | `docs/ui-ux.md` |
| Discovery behavior, scope rules, or deployment changed | `docs/discovery-capabilities.md`, `docs/discovery-deployment.md` |
| Auth, roles, session, or security boundary changed | `docs/security.md` |
| Setup, ports, env vars, or commands changed | `readme.md` |
| Bug fixed or issue resolved | `docs/issues.md` (status → `fixed` with reference) |
| Network map contract, layers, or interactions changed | `docs/network_map/interface-rules.md` |
| Runbook, monitoring, or operational procedure changed | `docs/runbooks.md` |

### Roadmap Sync

When a phase status changes (e.g., from `In progress` to `Done`) or when a phase's task list materially changes, update `docs/roadmap.md` in the same session. The roadmap phase table and phase detail sections must match reality.

### Feature Matrix Sync

When any feature is added, completed, changed in ownership, or gains/loses an endpoint or DB table, update `docs/feature-matrix.md` in the same session. No orphan features in code; no stale entries in the matrix.

### Done-Work Recording

When a task card moves to `Done`:

1. Update the Ready Queue row status to `Done`.
2. Fill in the task card's Handoff Notes with a brief summary of what was changed and any follow-up the next agent needs.
3. Update `docs/roadmap.md` if the completed work moves a phase forward.
4. Update `docs/feature-matrix.md` if feature status changed.
5. Update `docs/issues.md` if a tracked issue was resolved.

### Doc-Debt Rule

If a required doc update genuinely cannot be completed in the current session (e.g., the agent lacks information to write the update accurately), the agent must:

1. Add a note to the task card's Handoff Notes naming the specific doc and the missing update.
2. Add a `P1` child task or standalone task to the Ready Queue for the doc fix.

Silently skipping a doc update is not allowed.

## Required Reading By Work Type

Always start with:

- [AGENTS.md](AGENTS.md)
- [docs/engineering-standards.md](docs/engineering-standards.md)
- [docs/conventions.md](docs/conventions.md)
- [docs/feature-matrix.md](docs/feature-matrix.md)

Then read the relevant contract:

| Work Type | Required Docs |
| --- | --- |
| API behavior | `api/openapi.yaml`, `docs/api-contract.md` |
| Persistence or migrations | `docs/data-model.md`, `docs/migrations.md`, `core-go/migrations/` |
| Runtime/service boundaries | `docs/architecture.md`, `docker-compose.yml`, `.env.example` |
| Discovery behavior | `docs/discovery-capabilities.md`, `docs/discovery-deployment.md`, `docs/runbooks.md` |
| UI/operator workflow | `docs/ui-ux.md` |
| Map work | `docs/network_map/interface-rules.md`, `docs/network_map/network_map_ideas.md`, `docs/network_map/implementation-stack.md` |
| Auth/session/RBAC | `docs/security.md`, `docs/architecture.md`, `ui-node/lib/auth/` |
| AI-agent process | `docs/ai-coding-control.md`, `docs/vscode-ai-workflow.md`, `.github/copilot-instructions.md` |

## User-Assigned Issue Intake Workflow

- A user-assigned issue overrides the `## Ready Queue`, but it does not override the need for a grounded task card.
- Start with a short repo-grounding pass: current phase, likely owner role, affected modules, existing tasks, and open decisions.
- Default backlog shape for a non-trivial assigned issue:
  - one parent item using the next `Txx` id for the issue itself
  - one or more child tasks using `Txxa`, `Txxb`, and similar ids for implementation-ready slices
- Parent items should capture outcome, scope, sequencing context, dependencies, broad validation strategy, and handoff notes.
- Child tasks should stay small, implementation-ready, and explicit about files to touch, validation, and definition of done.
- If the issue is still speculative, record it anyway. Use `Blocked`, `Later`, explicit dependencies, or an open decision instead of waiting for perfect detail.
- After documenting an assigned issue, report the parent item, child tasks, docs updated, and any blocker or open decision before asking for the next issue.

## Task Card Templates

Use one of these shapes when adding work. Keep task cards concrete enough that another agent can execute without reading chat history.

### Standalone Or Child Task Template

```md
### T00 - Short Task Name

- Status: Ready
- Queue: Now
- Phase: P0
- Priority: P1
- Owner Role: Tech lead
- Goal: One-sentence description of the outcome.
- Scope:
  - concrete deliverable 1
  - concrete deliverable 2
- Files to Touch:
  - path/to/file
  - path/to/other-file
- Do Not Touch:
  - path/to/protected-area
- Dependencies:
  - T00a
- Validation:
  - command or manual check
- Definition of Done:
  - observable completion condition 1
  - observable completion condition 2
- Handoff Notes:
  - what the next agent should know
```

### Parent Issue Template

```md
### T00 - Short Parent Issue Name

- Status: Ready
- Queue: Now
- Phase: P0
- Priority: P1
- Owner Role: Tech lead
- Goal: One-sentence description of the multi-step outcome.
- Scope:
  - concrete outcome 1
  - concrete outcome 2
- Files to Touch:
  - path/to/doc-or-system-area
- Do Not Touch:
  - path/to/protected-area
- Dependencies:
  - T00x
- Child Tasks:
  - T00a
  - T00b
- Validation:
  - manual doc consistency review
  - child task validation listed on each child card
- Definition of Done:
  - the parent issue is decomposed into implementation-ready child tasks
  - affected planning docs are synchronized
- Handoff Notes:
  - sequencing, blockers, or open decisions the next agent should know
```

## Ready Queue

Only rows with `Status` = `Ready` are startable without more planning unless the user explicitly assigns them.

| ID | Queue | Phase | Priority | Task | Status | Depends On | Validation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| T000 | Now | Process | P1 | AI backlog and runbook foundation | Done | None | `git diff --check` + docs consistency review |
| T009 | Now | Process | P1 | VS Code AI workspace scaffolding | Done | T000 | `git diff --check` + JSON parse check + docs consistency review |
| T001 | Now | Hardening | P0 | Auth and OpenAPI drift hardening | Review | None | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .`; Go tests via Docker validator |
| T002 | Now | Process | P1 | Planning-doc drift reconciliation | Ready | T000 | manual roadmap/feature-matrix/API docs consistency review |
| T010 | Now | Phase 15/16 | P1 | Basic focused network diagram MVP | Ready | T002 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .`; Go/API tests if projection shape changes |
| T003 | Now | Phase 16 | P1 | Map modes contract and UI selector | Ready | T002 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T004 | Next | Phase 16 | P1 | Security layer v1 planning slice | Ready | T002 | manual docs consistency review + OpenAPI draft review if API shape changes |
| T005 | Next | Phase 16 | P1 | Build-mode map editing parent issue | Ready | T002 | parent/child cards written + feature matrix synced |
| T006 | Next | Ops | P1 | Local validation toolchain/runbook cleanup | Done | T000 | runbook update + one confirmed Docker-only validation path |
| T007 | Next | Discovery | P1 | Discovery deployment smoke matrix | Ready | T006 | documented Docker bridge, host-network, and native-host smoke commands |
| T011 | Next | Phase 16 | P1 | Operate overlays planning slice | Ready | T002, T003 | manual docs consistency review + child task validation |
| T012 | Next | Ops | P2 | Data retention and cleanup policy | Ready | None | manual docs review + Go tests if migrations added |
| T013 | Now | Process | P2 | Create backlog archive file | Ready | None | `git diff --check` + cross-reference review |
| T014 | Now | Process | P1 | Phase 12 roadmap status correction | Ready | None | manual docs consistency review |
| T008 | Later | Docs | P2 | Markdown encoding and typography cleanup | Ready | T000 | `git diff --check` + spot-check rendered docs |

## Dev Runbook

### Worktree Safety

- Check `git status --short` before editing.
- Preserve user changes already present in the worktree.
- Do not run destructive Git commands such as `git reset --hard` or `git checkout --` unless the user explicitly asks.
- Do not mix unrelated backlog tasks in one change.
- Prefer `apply_patch` for manual edits.

### Local Paths

This repo may be opened through a UNC path. Some Windows tools, especially `cmd.exe` wrappers used by npm scripts, cannot use a UNC current directory.

If a command defaults to `C:\Windows` or cannot find tests, rerun from the mapped path:

```powershell
Set-Location G:\Roller_hoops
```

### VS Code AI Workflow

Shared VS Code configuration lives under `.vscode/` and prompt files live under `.github/prompts/`.

Use `Terminal: Run Task...` for common agent checks:

- `agent: show ready queue`
- `agent: show required reading`
- `validate: available local checks`
- `validate: ui`
- `go: test`
- `go: test via docker`

See `docs/vscode-ai-workflow.md` for the full task list, Copilot prompt starters, and UNC workspace notes.

### UI Commands

```powershell
docker build -f docker/validate/ui-node.Dockerfile --target deps .
# use the VS Code task `ui: gen openapi types` to copy generated types back into the workspace
docker build -f docker/validate/ui-node.Dockerfile --target test .
docker build -f docker/validate/ui-node.Dockerfile --target build .
```

Use focused Vitest runs for small changes:

```powershell
docker build -f docker/validate/ui-node.Dockerfile --target test .
```

### Go Commands

```powershell
docker build -f docker/validate/core-go.Dockerfile --target fmtcheck .
docker build -f docker/validate/core-go.Dockerfile --target vet .
docker build -f docker/validate/core-go.Dockerfile --target test .
```

### Full Stack Smoke

```powershell
docker compose --profile dev up --build
```

Then verify:

- UI health: `GET http://localhost/healthz`
- Login: `admin` / `admin` from `.env.example` defaults
- Devices page renders seeded data when using the dev profile

### API Drift Checklist

When API routes or schemas change:

1. Update `api/openapi.yaml`.
2. Update `docs/api-contract.md`.
3. Regenerate UI types using the VS Code task `ui: gen openapi types`.

4. Ensure generated `ui-node/lib/api-types.ts` is committed.
5. Run UI tests/build.
6. Run Go tests or record the local blocker.

### Migration Checklist

When persistence changes:

1. Add paired migration files under `core-go/migrations/NNN_name.up.sql` and `NNN_name.down.sql`.
2. Update query files and generated sqlc output if applicable.
3. Update `docs/data-model.md`.
4. Update `docs/migrations.md` if the workflow changed.
5. Add or update integration tests.
6. Validate with a migrated database, not only unit tests.

### Discovery Safety Checklist

Before changing discovery behavior:

- Confirm explicit scope behavior.
- Confirm default scope behavior if the run omits scope.
- Confirm active scans remain gated by enable flags and allowlists.
- Confirm concurrency and timeouts are bounded.
- Confirm operator-facing run logs explain failures.
- Update `docs/discovery-capabilities.md` or `docs/discovery-deployment.md` if deployment behavior changes.

### Auth And BFF Checklist

Before changing auth, roles, or proxy behavior:

- Confirm unauthenticated users receive `401`.
- Confirm `read-only` users cannot mutate through generic or dedicated API routes.
- Confirm admin-only endpoints reject non-admin users.
- Confirm mutating proxy routes write best-effort audit events when expected.
- Confirm cookies keep the intended `HttpOnly`, `SameSite`, and `Secure` behavior.
- Add route-level regression tests for bypasses.

### UI Workflow Checklist

Before changing operator workflows:

- Keep URL state shareable when filters, focus, layer, or pagination matter.
- Preserve loading, empty, and error states.
- Avoid UI-side reconstruction of data that belongs in the Go API.
- Keep read-only controls disabled or blocked server-side.
- Update `docs/ui-ux.md` for workflow-level behavior changes.

## Detailed Task Cards

Closed task cards that no longer coordinate active work are archived to `docs/backlog-archive.md`.

### T000 - AI Backlog And Runbook Foundation

- Status: Done
- Queue: Now
- Phase: Process
- Priority: P1
- Owner Role: Docs owner
- Goal: Establish `BACKLOG.md` as the AI-facing execution board and developer runbook.
- Scope:
  - add root backlog with status model, queue model, templates, ready queue, validation runbook, and initial task cards
  - update agent/docs references so future sessions use this file
- Files to Touch:
  - `BACKLOG.md`
  - `AGENTS.md`
  - `.github/copilot-instructions.md`
  - `docs/conventions.md`
  - `docs/feature-matrix.md`
  - `readme.md`
- Do Not Touch:
  - runtime code unless needed for validation
- Dependencies:
  - None
- Validation:
  - `git diff --check`
  - manual docs consistency review
- Definition of Done:
  - agents have a single root execution board
  - task templates and validation commands are present
  - backlog is discoverable from agent instructions and docs index
- Handoff Notes:
  - created from the `text-game` backlog format and adapted to Roller_hoops service boundaries

### T001 - Auth And OpenAPI Drift Hardening

- Status: Review
- Queue: Now
- Phase: Hardening
- Priority: P0
- Owner Role: Security owner
- Attention: Has been in `Review` since creation; needs a human pass or a follow-up agent to validate and close.
- Goal: Close early-dev auth and generated-type drift found during the logic pass.
- Scope:
  - block read-only users on dedicated device tag update route
  - audit dedicated tag updates consistently with other mutating proxy routes
  - normalize unknown explicit auth roles to `read-only`
  - reject invalid admin reset roles
  - make UI DTOs alias generated OpenAPI schemas
  - tighten OpenAPI paginated envelope schemas where Go always emits arrays
- Files to Touch:
  - `ui-node/app/api/devices/[id]/tags/route.ts`
  - `ui-node/app/api/auth/admin/reset-password/route.ts`
  - `ui-node/lib/auth/users-store.ts`
  - `ui-node/lib/auth/session.ts`
  - `api/openapi.yaml`
  - `ui-node/lib/api-types.ts`
  - `ui-node/app/(app)/devices/types.ts`
  - focused tests beside changed routes/modules
- Do Not Touch:
  - Go runtime behavior unless Go contract tests reveal required alignment
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
  - `git diff --check`
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - UI tests and build pass
  - Go contract tests pass or local blocker is documented
  - OpenAPI generated types are committed
- Handoff Notes:
  - UI validation passed in the current working tree
  - Go validation now runs through the Docker validator when the daemon is available

### T009 - VS Code AI Workspace Scaffolding

- Status: Done
- Queue: Now
- Phase: Process
- Priority: P1
- Owner Role: Docs owner
- Goal: Make VS Code a safer default workspace for Codex/Copilot agents working from this repo.
- Scope:
  - add shared VS Code extension recommendations, settings, tasks, and backlog snippets
  - add Copilot prompt files for backlog-task starts, drift review, and API changes
  - document the VS Code AI workflow and UNC workspace caveat
  - update docs indexes and feature matrix so the workflow is discoverable
- Files to Touch:
  - `.gitignore`
  - `.vscode/extensions.json`
  - `.vscode/settings.json`
  - `.vscode/tasks.json`
  - `.vscode/roller-hoops.code-snippets`
  - `.github/prompts/*.prompt.md`
  - `docs/vscode-ai-workflow.md`
  - `BACKLOG.md`
  - `AGENTS.md`
  - `.github/copilot-instructions.md`
  - `docs/conventions.md`
  - `docs/feature-matrix.md`
  - `readme.md`
- Do Not Touch:
  - product runtime code
- Dependencies:
  - T000
- Validation:
  - `git diff --check`
  - JSON parse check for `.vscode/*.json` and `.vscode/*.code-snippets`
  - manual docs consistency review
- Definition of Done:
  - VS Code exposes repeatable validation tasks for UI, Go, Docker, and agent workflow
  - Copilot prompt files point agents at backlog and contract discipline
  - `.vscode` tracks only shared safe files, not user-local state
- Handoff Notes:
  - Windows tasks use `pushd` to avoid UNC current-directory failures in npm/cmd wrappers.

### T002 - Planning-Doc Drift Reconciliation

- Status: Ready
- Queue: Now
- Phase: Process
- Priority: P1
- Owner Role: Tech lead
- Goal: Reconcile roadmap, feature matrix, API docs, and backlog so current feature status is consistent.
- Scope:
  - compare `docs/roadmap.md` current snapshot, phase statuses, and next checklist against `docs/feature-matrix.md`
  - fix Phase 12 status: at-a-glance says `Planned`, section says `In progress`, but all M12 milestones and feature matrix entries are `complete` — update to `Done`
  - fix Phase 8 blank `**Status:**` line in roadmap (should be `Done`)
  - fix Phase 3 stale unchecked cross-references to Phases 10/11 (both are `Done`)
  - reconcile Phase 16 M16.2 (Services): feature matrix says Services projection is `complete` but M16.2 tasks are unchecked — clarify what is done vs remaining
  - update stale API lists in roadmap if they lag behind `api/openapi.yaml`
  - ensure `docs/issues.md` references `BACKLOG.md` as the execution board while retaining issue history
  - add or update task cards for any real gaps found
- Files to Touch:
  - `BACKLOG.md`
  - `docs/roadmap.md`
  - `docs/feature-matrix.md`
  - `docs/issues.md`
  - `docs/api-contract.md`
- Do Not Touch:
  - runtime code
- Dependencies:
  - T000
- Validation:
  - manual roadmap/feature-matrix/API docs consistency review
  - `git diff --check`
- Definition of Done:
  - roadmap no longer claims implemented APIs are planned-only
  - feature matrix statuses match the current code and backlog
  - any open product gaps are represented by backlog cards
- Handoff Notes:
  - Feature matrix currently says some Services map work is complete while roadmap Phase 16 still has unchecked Services tasks; reconcile before implementing more map work.
  - Phase 12 is fully done (all milestones checked, all feature matrix entries `complete`) but roadmap at-a-glance still says `Planned` and section header says `In progress`. Fix all three (table, header, next-milestone checklist).
  - Phase 8 section header has a blank `**Status:**` line — set to `Done`.
  - Phase 3 still shows `[ ] Add auth + sessions (Phase 11)` and `[ ] Add richer operator workflows (Phase 10)` as unchecked — both are `Done`.

### T003 - Map Modes Contract And UI Selector

- Status: Ready
- Queue: Now
- Phase: Phase 16
- Priority: P1
- Owner Role: UI owner
- Goal: Add the Explore / Build / Secure / Operate mode contract and initial mode selector without bypassing layer separation.
- Scope:
  - define mode semantics in docs before implementation
  - add URL-driven mode state to the map page
  - render a mode selector in the map chrome
  - keep all write actions disabled unless future Build-mode APIs exist
  - add tests for URL mode parsing and safe fallbacks
- Files to Touch:
  - `docs/ui-ux.md`
  - `docs/network_map/interface-rules.md`
  - `ui-node/app/(app)/map/page.tsx`
  - `ui-node/app/(app)/map/page.test.tsx`
  - map component files as needed
- Do Not Touch:
  - Go map APIs unless the mode contract requires new query params
  - DB schema
- Dependencies:
  - T002
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - mode is deep-linkable and deterministic
  - invalid modes fall back safely
  - Build mode does not expose writes without authorized APIs
- Handoff Notes:
  - Treat mode as UI behavior first; do not blend map layers or add global graph behavior.

### T010 - Basic Focused Network Diagram MVP

- Status: Ready
- Queue: Now
- Phase: Phase 15/16
- Priority: P1
- Owner Role: UI owner
- Related Issues: ISS-007
- Goal: Make the map deliver the basic expected network diagram: a selected device connected to its likely router/switch/peer, with IP and MAC facts visible.
- Scope:
  - define the default device-focused diagram contract in `docs/network_map/interface-rules.md`
  - ensure a device focus can render a simple readable diagram such as `Device 1 -- Router 1` or `Device 1 -- Switch 1 -- Router 1`
  - show key facts on or near each node: display name, primary IP, primary MAC, and role/tag when available
  - show link confidence/source clearly (`manual`, `lldp`, `cdp`, `gateway-inferred`, `same-subnet`, or `unknown`)
  - use existing map projection data where possible before adding API shape
  - if API shape is insufficient, update `api/openapi.yaml`, `docs/api-contract.md`, generated UI types, and focused Go/UI tests
  - preserve the guardrail that the default diagram is focused and bounded, not a whole-network graph
- Files to Touch:
  - `docs/network_map/interface-rules.md`
  - `docs/network_map/network_map_ideas.md`
  - `docs/ui-ux.md`
  - `docs/api-contract.md` if projection fields or semantics change
  - `api/openapi.yaml` if projection fields or semantics change
  - `ui-node/app/(app)/map/*`
  - `core-go/internal/httpapi/map.go` and map tests only if API projection data is missing
- Do Not Touch:
  - discovery scanning breadth/defaults
  - database schema unless a separate child task proves persisted curated truth is required
  - Build-mode write behavior
- Dependencies:
  - T002
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
  - `docker build -f docker/validate/core-go.Dockerfile --target test .` if Go/API projection code changes
  - manual check: seeded/dev data can show a focused diagram with at least two connected nodes and visible IP/MAC facts
- Definition of Done:
  - opening a device-focused map gives an ordinary operator-readable network diagram, not only abstract regions
  - each visible device shows identity plus primary IP/MAC when known
  - the diagram makes relationship source/confidence visible and does not imply certainty for inferred links
  - caps and focus rules still prevent an accidental whole-network render
- Handoff Notes:
  - This is the product floor for the map. The layered explorer remains the long-term model, but users must first get the simple diagram they expect: device, router/switch/peer, IPs, MACs, and visible connections.

### T004 - Security Layer V1 Planning Slice

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Tech lead
- Goal: Turn the Security layer idea into implementation-ready child tasks.
- Scope:
  - define whether v1 security uses manual zones, derived tags, or both
  - decide minimum DB tables and API endpoints
  - define read projection shape and Build-mode write requirements
  - split implementation into migration/API/UI child tasks
- Files to Touch:
  - `BACKLOG.md`
  - `docs/data-model.md`
  - `docs/api-contract.md`
  - `docs/network_map/interface-rules.md`
  - `docs/feature-matrix.md`
- Do Not Touch:
  - runtime code until child tasks are explicit
- Dependencies:
  - T002
- Validation:
  - manual docs consistency review
  - child task validation listed on each new card
- Definition of Done:
  - Security layer has concrete child cards with files, validation, and blockers
  - open decisions are explicit instead of hidden in prose
- Handoff Notes:
  - Avoid creating inconsistent truth between discovered tags, manual zones, and future policy edges.

### T005 - Build-Mode Map Editing Parent Issue

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Tech lead
- Goal: Decompose map editing into safe, role-gated API and UI slices.
- Scope:
  - define which objects are user-authored truth (`links`, `zones`, `service_dependencies`)
  - define role requirements and audit expectations
  - decide endpoint namespace and OpenAPI shape
  - split into migration, Go API, UI, and tests
- Files to Touch:
  - `BACKLOG.md`
  - `docs/api-contract.md`
  - `docs/data-model.md`
  - `docs/security.md`
  - `docs/feature-matrix.md`
- Do Not Touch:
  - runtime code until child cards are written
- Dependencies:
  - T002
- Validation:
  - parent/child cards written
  - feature matrix synced
  - manual auth/API review
- Definition of Done:
  - Build-mode editing has implementation-ready child cards
  - no write behavior is implemented without auth and audit requirements
- Handoff Notes:
  - UI must never write directly to the DB; all curated map truth goes through Go APIs.

### T006 - Local Validation Toolchain And Runbook Cleanup

- Status: Done
- Queue: Next
- Phase: Ops
- Priority: P1
- Owner Role: Operations owner
- Goal: Make validation and stack startup reliable with Docker only on Windows/UNC workspaces.
- Scope:
  - replace local Go/Node validation assumptions with Docker-backed tasks
  - remove bind-mount-sensitive compose inputs for Traefik config, migrations, and dev seed SQL
  - document Docker-only validation and stack usage on Windows/mapped-drive workspaces
  - keep local toolchains optional rather than required
- Files to Touch:
  - `BACKLOG.md`
  - `readme.md`
  - `docs/runbooks.md`
  - optional `.github/workflows/ci.yml` only if CI commands need alignment
- Do Not Touch:
  - product runtime code
- Dependencies:
  - T000
- Validation:
  - one confirmed Docker-backed UI validation command from mapped path
  - one confirmed Docker-backed Go validation command from mapped path
- Definition of Done:
  - future agents can run validations with Docker only, without reinstalling local Go/Node toolchains
  - bind-mount-sensitive validation paths are replaced or documented away
- Handoff Notes:
  - Shared validation tasks now build Docker validation targets instead of invoking local Go/Node CLIs.
  - Compose services for Traefik config, migrations, and dev seed no longer depend on host bind mounts, which improves Windows mapped-drive support.

### T007 - Discovery Deployment Smoke Matrix

- Status: Ready
- Queue: Next
- Phase: Discovery
- Priority: P1
- Owner Role: Operations owner
- Goal: Define repeatable discovery smoke checks for Docker bridge, Linux host-network, and native-host deployments.
- Scope:
  - specify safe test scopes and expected capability differences
  - document required env vars and capabilities
  - document how to identify ICMP, ARP, SNMP, and nmap failures from run logs
  - add backlog child cards if automation is needed
- Files to Touch:
  - `docs/discovery-capabilities.md`
  - `docs/discovery-deployment.md`
  - `docs/runbooks.md`
  - `BACKLOG.md`
- Do Not Touch:
  - discovery implementation unless a documented smoke reveals a bug
- Dependencies:
  - T006
- Validation:
  - documented commands for each deployment mode
  - at least one local smoke if environment supports it
- Definition of Done:
  - operators and agents know which discovery behaviors should pass in each deployment mode
  - failure modes are actionable from logs
- Handoff Notes:
  - Keep all examples narrow and explicitly scoped; do not recommend broad network scans.

### T008 - Markdown Encoding And Typography Cleanup

- Status: Ready
- Queue: Later
- Phase: Docs
- Priority: P2
- Owner Role: Docs owner
- Goal: Make Markdown render cleanly across Windows terminals, GitHub, and editor previews.
- Scope:
  - audit docs for mojibake or console-encoding artifacts
  - standardize on UTF-8 files
  - replace accidental corrupted punctuation, not intentional technical symbols
  - avoid churn in large roadmap sections unless corruption is real in the file
- Files to Touch:
  - `readme.md`
  - `docs/*.md`
  - `docs/network_map/*.md`
- Do Not Touch:
  - runtime code
- Dependencies:
  - T000
- Validation:
  - `git diff --check`
  - spot-check GitHub or editor Markdown rendering
- Definition of Done:
  - docs render cleanly without visible mojibake
  - changes are mechanical and scoped
- Handoff Notes:
  - PowerShell `Get-Content` may display UTF-8 punctuation incorrectly depending on console encoding; verify file bytes/rendering before editing.

### T011 - Operate Overlays Planning Slice

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Tech lead
- Goal: Turn M16.4 (Operate overlays — history-aware status on the map) into implementation-ready child tasks.
- Scope:
  - define which Phase 9 APIs power "last seen" and "changed" overlays on map nodes
  - define overlay rendering rules (badges, color, legend) that keep the map readable without becoming a monitoring dashboard
  - decide toggle/dismiss UX for overlays
  - split into API-extension (if needed), UI, and test child tasks
- Files to Touch:
  - `BACKLOG.md`
  - `docs/ui-ux.md`
  - `docs/network_map/interface-rules.md`
  - `docs/feature-matrix.md`
- Do Not Touch:
  - runtime code until child tasks are explicit
- Dependencies:
  - T002
  - T003 (modes contract must exist before Operate mode is safe)
- Validation:
  - manual docs consistency review
  - child task validation listed on each new card
- Definition of Done:
  - Operate overlays have concrete child cards with files, validation, and blockers
  - Phase 9 API dependencies are confirmed as sufficient or gaps are documented
- Handoff Notes:
  - Operate mode is an overlay on any layer, not a separate layer. Keep it distinct from the Explore default.

### T012 - Data Retention And Cleanup Policy

- Status: Ready
- Queue: Next
- Phase: Ops
- Priority: P2
- Owner Role: Data owner
- Goal: Implement and document a data retention policy for append-only tables so the database stays performant over time.
- Scope:
  - define retention windows for `ip_observations`, `mac_observations`, `discovery_run_logs`, and `audit_events`
  - document retention strategy in `docs/data-model.md` and `docs/runbooks.md`
  - add a migration or cron-friendly SQL for pruning stale observations beyond the retention window
  - add an index review for time-range queries that retention depends on
- Files to Touch:
  - `docs/data-model.md`
  - `docs/runbooks.md`
  - `core-go/migrations/` (if adding a cleanup function or index)
  - `BACKLOG.md`
- Do Not Touch:
  - discovery or enrichment runtime behavior
  - UI code
- Dependencies:
  - None (Phase 9 observation tables already exist)
- Validation:
  - manual docs review
  - `docker build -f docker/validate/core-go.Dockerfile --target test .` if migrations are added
- Definition of Done:
  - retention policy is documented with specific windows and cleanup approach
  - operators know how to prune old data or have an automated path
- Handoff Notes:
  - Phase 7 blockers and Phase 9 shared tasks both mention this need but no implementation exists yet.
  - Start with documentation and manual SQL; automate later if volume justifies it.

### T013 - Create Backlog Archive File

- Status: Ready
- Queue: Now
- Phase: Process
- Priority: P2
- Owner Role: Docs owner
- Goal: Create `docs/backlog-archive.md` so Done task cards can be archived per the Archive Policy.
- Scope:
  - create `docs/backlog-archive.md` with a header explaining its purpose and linking back to `BACKLOG.md`
  - move T000 and T009 (Done, no active dependents) into the archive
  - keep T006 in the main backlog until T007 completes (T007 depends on T006)
- Files to Touch:
  - `docs/backlog-archive.md` (new)
  - `BACKLOG.md`
- Do Not Touch:
  - runtime code
- Dependencies:
  - None
- Validation:
  - `git diff --check`
  - confirm cross-references are correct
- Definition of Done:
  - archive file exists and contains T000 and T009
  - main backlog no longer carries cards that are Done with no active dependents
  - archive file is linked from the BACKLOG.md Archive Policy section
- Handoff Notes:
  - The BACKLOG.md Detailed Task Cards header already references this file. Just create it.

### T014 - Phase 12 Roadmap Status Correction

- Status: Ready
- Queue: Now
- Phase: Process
- Priority: P1
- Owner Role: Docs owner
- Goal: Mark Phase 12 as Done in all roadmap locations since every milestone is complete.
- Scope:
  - update the at-a-glance table row for Phase 12 from `Planned` to `Done`
  - update the Phase 12 section status from `In progress` to `Done`
  - check the Phase 12 item in the next-milestone checklist
  - verify feature matrix rows for Phase 12 all say `complete` (they do; confirm only)
- Files to Touch:
  - `docs/roadmap.md`
- Do Not Touch:
  - runtime code
- Dependencies:
  - None
- Validation:
  - manual docs consistency review
- Definition of Done:
  - Phase 12 status is `Done` everywhere in the roadmap
  - feature matrix is consistent
- Handoff Notes:
  - This is a documentation-only fix. All Phase 12 milestones (M12.1–M12.5) have every task checked off and all feature matrix entries are `complete`.

## Immediate Open Decisions

| ID | Decision | Needed By | Owner | Status |
| --- | --- | --- | --- | --- |
| OD-001 | Should Services layer M16.2 tasks be checked off (read-side projection done under Phase 15) or do they remain open for dependency-editing work? | T002 | Tech lead | Open |
| OD-002 | What is the retention window for `ip_observations` and `mac_observations`? (30 days? 90 days? configurable?) | T012 | Data owner | Open |
| OD-003 | T001 has been in `Review` since creation — does it need a human review pass, or can a follow-up agent validate and close it? | T001 | Security owner | Open |

## Agent Execution Rules

- Prefer the smallest task that unblocks the phase and removes the most operator friction.
- Do not silently expand scope across multiple tasks.
- Do not mark a task `Done` without running its listed validation.
- If validation cannot be run, leave the task at `Review` and record exactly what is unverified.
- Update related docs in the same session when the task changes setup, scope, or behavior.
- If a task card is missing fields, repair the card before writing code.
- If the user request conflicts with the queue, follow the user request and then update this file to reflect reality.

## Global Blockers And Notes

- Go validation is unavailable on hosts without Go installed unless Docker Desktop's Linux daemon is running.
- UNC workdir validation through npm may fail because `cmd.exe` defaults to `C:\Windows`; use the mapped `G:\Roller_hoops` path.
- The backlog and issue log now coexist: use this file for execution cards and `docs/issues.md` for historical issue records or user-facing bug reports.
- T001 (Auth hardening) has been in `Review` since creation. See OD-003 in Open Decisions.
- Phase 12 is done but the roadmap still says `Planned`/`In progress`. T014 fixes this; T002 covers the broader drift.
- `docs/backlog-archive.md` does not yet exist. T013 creates it. T000 and T009 are archive-eligible.

## Archive Policy

Do not let this file grow without bounds.

- Keep active, blocked, ready, and review task cards here.
- Move long-closed task cards to `docs/backlog-archive.md` once they no longer coordinate active work.
- Keep completed parent cards visible while dependent child tasks remain active.
- Do not delete completed cards that explain recent unmerged work.
