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
| T001 | Now | Hardening | P0 | Auth and OpenAPI drift hardening | Review | None | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .`; Go tests via Docker validator |
| T002 | Done | Process | P1 | Planning-doc drift reconciliation | Done | None | manual roadmap/feature-matrix/API docs consistency review |
| T010 | Now | Phase 15/16 | P1 | Basic focused network diagram MVP | Done | T002 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .`; `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T003 | Now | Phase 16 | P1 | Map modes contract and UI selector | Done | T002 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T004 | Next | Phase 16 | P1 | Security layer v1 planning slice | Done | T002 | manual docs consistency review + OpenAPI draft review if API shape changes |
| T005 | Next | Phase 16 | P1 | Build-mode map editing parent issue | Done | T002 | parent/child cards written + feature matrix synced |
| T006 | Next | Ops | P1 | Local validation toolchain/runbook cleanup | Done | None | runbook update + one confirmed Docker-only validation path |
| T007 | Next | Discovery | P1 | Discovery deployment smoke matrix | Done | T006 | documented Docker bridge, host-network, and native-host smoke commands |
| T011 | Next | Phase 16 | P1 | Operate overlays planning slice | Done | T002, T003 | manual docs consistency review + child task validation |
| T012 | Next | Ops | P2 | Data retention and cleanup policy | Done | None | manual docs review + Go tests if migrations added |
| T013 | Now | Process | P2 | Create backlog archive file | Done | None | `git diff --check` + cross-reference review |
| T014 | Now | Process | P1 | Phase 12 roadmap status correction | Done | None | manual docs consistency review |
| T008 | Later | Docs | P2 | Markdown encoding and typography cleanup | Ready | None | `git diff --check` + spot-check rendered docs |
| T015 | Next | Phase 16 | P1 | Security zones DB migration | Done | T004 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T016 | Next | Phase 16 | P1 | Security zone CRUD API (Go) | Done | T015 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T017 | Next | Phase 16 | P1 | Security layer map projection (Go) | Done | T015 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T018 | Next | Phase 16 | P1 | Security layer UI renderer | Done | T017 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T019 | Next | Phase 16 | P1 | Link CRUD API (Go) | Ready | T005 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T020 | Next | Phase 16 | P2 | Build-mode UI controls | Ready | T016, T019 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T021 | Next | Phase 16 | P1 | Operate overlay projection metadata (Go) | Ready | T011 | `docker build -f docker/validate/core-go.Dockerfile --target test .` + OpenAPI type generation if schema descriptions change |
| T022 | Next | Phase 16 | P1 | Operate overlay canvas and legend UI | Ready | T021 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T023 | Next | Phase 16 | P1 | Operate inspector history and change feed UI | Ready | T021 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T024 | Now | Enrichment | P1 | Device naming: fallback unique names | Done | None | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T025 | Now | Discovery | P1 | MAC discovery in Docker bridge mode | Done | None | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T026 | Now | Discovery | P1 | SNMP enrichment debugging and gaps | Done | None | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T027 | Now | Phase 16 | P1 | Map node text overlap fix | Done | None | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T028 | Now | Enrichment | P2 | Parse sysDescr into structured OS fields | Done | T026 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T029 | Next | Discovery | P2 | MAC-based device identity for rescans | Done | T025 | `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T030 | Now | Phase 16 | P0 | Map projection renders nothing with focus | Done | None | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .`; `docker build -f docker/validate/core-go.Dockerfile --target test .` |
| T031 | Now | UI | P1 | Device page layout redesign | Done | None | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T032 | Now | UI | P1 | Device page: MAC display, text overflow, SNMP OS fields | Done | T028 | `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |
| T033 | Now | Enrichment/UI | P1 | OS fingerprinting, MAC OUI lookup, device page redesign | Done | T028 | `docker build -f docker/validate/core-go.Dockerfile --target test .`; `docker build -f docker/validate/ui-node.Dockerfile --target test .` + `docker build -f docker/validate/ui-node.Dockerfile --target build .` |

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

Closed task cards that no longer coordinate active work are archived to [docs/backlog-archive.md](docs/backlog-archive.md).

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

### T002 - Planning-Doc Drift Reconciliation

- Status: Done
- Queue: Done
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
  - All items resolved:
    - Phase 12 status fixed to Done (T014, prior session).
    - Phase 8 section header status set to Done.
    - Phase 3 stale checkboxes checked (auth + sessions, operator workflows).
    - Phase 16 M16.2 Services API/UI tasks checked to match feature matrix `complete` status; only optional dependency modeling remains unchecked.
    - Implemented APIs list updated to include all 19 OpenAPI endpoints plus `/metrics`; Planned APIs trimmed to only truly unimplemented items (Build mode write endpoints, Security data model).
    - `docs/issues.md` now references `BACKLOG.md` as the execution board.
    - No new product gaps found; existing backlog cards cover all open work.

### T003 - Map Modes Contract And UI Selector

- Status: Done
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
  - Implemented mode selector (Explore/Build/Secure/Operate) in map sidebar with URL-driven `mode` param.
  - Mode persists across layer/focus navigation. Invalid modes fall back to `explore` with console warning.
  - Non-explore modes show an informational notice (no write actions exposed yet).
  - Docs updated: `docs/ui-ux.md` (mode semantics table), `docs/network_map/interface-rules.md` (non-negotiable #6).
  - CSS: `.mapModeSection`, `.mapModeList`, `.mapModeItem*` classes in `globals.css`.
  - 9 tests in `page.test.tsx` covering default, URL parse, fallback, notices, and link persistence.
  - All validation passed: tests + production build via Docker.

### T010 - Basic Focused Network Diagram MVP

- Status: Done
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
  - Physical layer device focus now renders a basic focused diagram: focus device + linked peers, each showing display name, primary IP, and primary MAC.
  - New `core-go/internal/sqlcgen/map_facts.go`: batch query `ListDevicePrimaryFacts` fetches primary IP and MAC for a list of device IDs in one query. Uses correlated subqueries against `ip_addresses` and `mac_addresses` tables, ordered by `updated_at DESC`.
  - `core-go/internal/httpapi/map.go`: physical layer projection now batch-fetches facts and populates `meta.primary_ip` / `meta.primary_mac` on all nodes (focus + peers). The batch facts call is a soft dependency — if the interface method is missing, the diagram renders without facts.
  - `ui-node/app/(app)/map/MapCanvas.tsx`: physical view now shows IP/MAC below the focus device name and below each peer link name. Added `resolveNodeMetaString` helper (mirrors `resolveEdgeMetaString`). Link source (`lldp`, `manual`, etc.) is now rendered as a separate prominent badge, not buried in the meta string.
  - CSS: new classes for `.mapPhysicalFocusLabel`, `.mapPhysicalFocusFacts`, `.mapPhysicalFact`, `.mapPhysicalLinkBody`, `.mapPhysicalLinkFacts`, `.mapPhysicalLinkRight`, `.mapPhysicalLinkSource`. Layout shifted from `align-items: baseline` to `align-items: flex-start` to accommodate multi-line node cards.
  - Go test added: `TestMapProjection_DeviceFocus_PhysicalIncludesNodeFacts` verifies focus and peer nodes carry `primary_ip`/`primary_mac` in meta.
  - UI test added: `renders primary IP and MAC facts on physical diagram nodes` verifies fact labels render in the DOM.
  - `docs/network_map/interface-rules.md`: added "Implementation status" section under "Basic diagram floor" documenting the current state.
  - No schema/migration changes. No OpenAPI changes (meta is already `additionalProperties`). No changes to L3/L2/services projections (those continue as region-based views).

### T004 - Security Layer V1 Planning Slice

- Status: Done
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
  - **Decisions resolved:**
    - v1 uses manual zones only. No auto-derived zones from tags or discovery.
    - `zones` + `device_zones` tables (schema finalized in `docs/data-model.md`).
    - No `zone_policies` in v1 — no inter-zone edges.
    - Device tags remain orthogonal to zones.
    - Zone CRUD endpoints under `/api/v1/topology/zones` (documented in `docs/api-contract.md`).
    - Projection: zone-focused and device-focused views, region-based like L3/L2.
  - **Child tasks created:** T015 (migration), T016 (CRUD API), T017 (map projection), T018 (UI renderer).
  - **Docs updated:** `docs/data-model.md` (zones schema approved), `docs/api-contract.md` (zone endpoints + projection contract), `docs/network_map/interface-rules.md` (Security layer contract section), `docs/feature-matrix.md` (no change needed; already lists planned).
  - Avoid creating inconsistent truth between discovered tags, manual zones, and future policy edges.

### T005 - Build-Mode Map Editing Parent Issue

- Status: Done
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
  - **Decisions resolved:**
    - User-authored truth objects: `links` (manual create/update/delete), `zones` + `device_zones` (covered by T016 from T004), and `service_dependencies` (deferred — not v1).
    - Endpoint namespace: `/api/v1/topology/` for all write operations on curated map objects.
    - Role requirements: `admin` role required for all `/api/v1/topology/` writes. The existing UI proxy rejects POST/PUT/PATCH/DELETE for `read-only` sessions. Go API should also check an `X-User-Role` header (forwarded by UI proxy) and reject non-admin writes with `403`.
    - Audit: all writes to topology objects must create an `audit_events` row with `target_type` matching the object type (e.g., `link`, `zone`).
    - `service_dependencies` deferred: the data model allows either service-to-service or host-to-host edges. Decision deferred until the Services layer has more operator validation.
  - **Child tasks created:** T019 (link CRUD API), T020 (Build-mode UI controls). T016 (zone CRUD) already created by T004.
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

- Status: Done
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
  - Added deployment smoke checks for Docker bridge, Linux host-network, and native-host runs in `docs/discovery-deployment.md` and `docs/runbooks.md`.
  - Added smoke expectations and run-log triage signals in `docs/discovery-capabilities.md`.
  - Validation passed: `git diff --check`; manual docs consistency review confirmed the smoke sections are present in all owning docs.

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

- Status: Done
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
  - Operate mode contract documented in `docs/ui-ux.md` and `docs/network_map/interface-rules.md`: it is an overlay on the active layer, not a separate layer.
  - Phase 9 API usage documented: projection metadata supplies node timestamps, `/api/v1/devices/changes` supplies the recent-change window, and `/api/v1/devices/{id}/history` supplies compact Inspector history.
  - Feature matrix now tracks Map Operate overlays as `planned`.
  - Child tasks created: T021 (Go projection metadata), T022 (canvas overlays + legend), T023 (Inspector history + change feed).

### T012 - Data Retention And Cleanup Policy

- Status: Done
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
  - Defined conservative retention windows: 90 days for IP/MAC observations while preserving latest fact per device/value, 30 days for discovery run logs, and 365 days for audit events.
  - Added manual dry-run and cleanup SQL to `docs/runbooks.md`; no automatic deletion job was added.
  - Added migration `013_retention_indexes` for `discovery_run_logs(created_at DESC)`; existing indexes already covered observation and audit cleanup.
  - Validation passed: `git diff --check`; `docker build -f docker/validate/core-go.Dockerfile --target test .`.

### T013 - Create Backlog Archive File

- Status: Done
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
  - Created `docs/backlog-archive.md` with T000, T009, T015, and T016 archived.
  - Updated the Detailed Task Cards header to link the archive file.
  - Removed archived Done rows from the Ready Queue table.

### T014 - Phase 12 Roadmap Status Correction

- Status: Done
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
  - All three roadmap locations (at-a-glance table, section header, next-milestone checklist) updated to Done.
  - Feature matrix Phase 12 entries confirmed as `complete`.

### T015 - Security Zones DB Migration

- Status: Done
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Backend owner
- Parent: T004
- Goal: Create the `zones` and `device_zones` tables via migration.
- Scope:
  - add migration `012_zones.up.sql` and `012_zones.down.sql`
  - `zones` table: `id` uuid PK, `name` text unique not null, `description` text nullable, `created_at`/`updated_at` timestamptz
  - `device_zones` join table: `device_id` uuid FK → devices, `zone_id` uuid FK → zones, `created_at` timestamptz, unique `(device_id, zone_id)`, index on `zone_id`
  - update `docs/data-model.md` migration reference if needed
  - add seed data in `docker/dev/dev-seed.sql` for at least two zones with member devices
- Files to Touch:
  - `core-go/migrations/012_zones.up.sql`
  - `core-go/migrations/012_zones.down.sql`
  - `docker/dev/dev-seed.sql`
  - `docs/data-model.md` (migration reference)
- Do Not Touch:
  - Go application code (separate task)
  - UI code
- Dependencies:
  - T004
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
  - migration up/down should be idempotent (IF NOT EXISTS patterns)
- Definition of Done:
  - migration files exist and pass Go validation
  - dev-seed includes zone test data
  - data-model.md references the migration
- Handoff Notes:
  - Schema is defined in `docs/data-model.md` under "zones + membership". Follow the exact column spec there.
  - Migration 012 created with `IF NOT EXISTS` for tables and indexes. Down migration drops in correct dependency order (device_zones before zones).
  - Dev-seed creates two zones (DMZ, Internal) and assigns the seeded Office Router to both (router spanning zones pattern).
  - Go validation passed (exit 0). data-model.md updated to note zones exist as of migration 012.

### T016 - Security Zone CRUD API (Go)

- Status: Done
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Backend owner
- Parent: T004
- Goal: Implement Go API endpoints for zone CRUD and membership management.
- Scope:
  - add sqlcgen queries for zones: list, get, create, update, delete
  - add sqlcgen queries for device_zones: list members, add member, remove member, set members
  - add HTTP handlers under `/api/v1/topology/zones` (see `docs/api-contract.md` "Zone management endpoints")
  - audit-log zone writes via existing `audit_events` pattern
  - add unit tests for CRUD handlers
  - update `api/openapi.yaml` with zone CRUD endpoint definitions
  - regenerate `ui-node/lib/api-types.ts`
- Files to Touch:
  - `core-go/internal/sqlcgen/zones.go` (new)
  - `core-go/internal/httpapi/zones.go` (new)
  - `core-go/internal/httpapi/zones_test.go` (new)
  - `core-go/internal/httpapi/handler.go` (register routes)
  - `api/openapi.yaml`
  - `ui-node/lib/api-types.ts` (regenerated)
- Do Not Touch:
  - map projection code (separate task T017)
  - UI rendering (separate task T018)
- Dependencies:
  - T015
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
  - OpenAPI types regeneration: `npm run gen:openapi` in ui-node
- Definition of Done:
  - zone CRUD endpoints work with standard error envelope
  - membership endpoints allow adding/removing devices from zones
  - unit tests cover happy path and error cases
  - OpenAPI is updated and types are regenerated
- Handoff Notes:
  - Implemented `core-go/internal/sqlcgen/zones.go` with manual query methods for zone CRUD, membership management, and device existence validation.
  - Added `/api/v1/topology/zones` routes in `handler.go` and implemented handlers in `core-go/internal/httpapi/zones.go`.
  - Topology writes now require `X-User-Role: admin` in the Go API and write audit rows via the existing `audit_events` path.
  - Added unit coverage in `core-go/internal/httpapi/zones_test.go` for list, create, get-not-found, membership replacement, and device-not-found cases.
  - Updated `api/openapi.yaml`, regenerated `ui-node/lib/api-types.ts`, and updated docs (`docs/api-contract.md`, `docs/feature-matrix.md`).
  - Validation passed: `docker build -f docker/validate/core-go.Dockerfile --target test .`, OpenAPI type regeneration, and `docker build -f docker/validate/ui-node.Dockerfile --target build .`.

### T017 - Security Layer Map Projection (Go)

- Status: Done
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Backend owner
- Parent: T004
- Goal: Implement the `GET /api/v1/map/security` projection handler in Go.
- Scope:
  - add sqlcgen map queries for security layer: zone-focused (zone as region, members as nodes), device-focused (device's zones as regions)
  - add projection logic in `core-go/internal/httpapi/map.go` for `layer == "security"`
  - zone focus: single zone region, member devices as nodes, inspector with zone details
  - device focus: zones the device belongs to as regions, inspector with zone navigation
  - no-focus: guidance message suggesting zone or device focus
  - add unit tests following existing map test patterns
- Files to Touch:
  - `core-go/internal/sqlcgen/map_security.go` (new)
  - `core-go/internal/httpapi/map.go`
  - `core-go/internal/httpapi/map_test.go`
- Do Not Touch:
  - zone CRUD endpoints (T016)
  - UI rendering (T018)
- Dependencies:
  - T015
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - security layer returns zone-focused and device-focused projections
  - empty zones render as empty regions
  - multi-zone devices appear in each zone
  - unit tests verify both focus types
- Handoff Notes:
  - Security projection implementation already existed in `core-go/internal/httpapi/map.go` and `core-go/internal/sqlcgen/map_security.go`; this task added focused tests and synchronized status docs.
  - Added tests for zone focus with members, empty zone rendering, and device focus with multi-zone membership.
  - Updated `docs/api-contract.md`, `docs/feature-matrix.md`, and `docs/roadmap.md` to mark the read-side Go projection implemented; UI rendering was completed immediately after under T018.
  - Validation passed: `git diff --check`; `docker build -f docker/validate/core-go.Dockerfile --target test .`.

### T018 - Security Layer UI Renderer

- Status: Done
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: UI owner
- Parent: T004
- Goal: Render the security layer projection in the map canvas and layer panel.
- Scope:
  - security layer rendering in `MapCanvas.tsx`: zone regions with device occupant nodes (follow L3/L2 region rendering pattern)
  - layer panel should show security layer as selectable
  - inspector should display zone details (name, description, member count) and cross-layer navigation
  - add UI tests for security layer rendering
- Files to Touch:
  - `ui-node/app/(app)/map/MapCanvas.tsx`
  - `ui-node/app/(app)/map/MapCanvas.test.tsx`
  - `ui-node/app/(app)/map/LayerPanel.tsx` (if security not already listed)
- Do Not Touch:
  - Go API code
  - Zone CRUD UI (future Build-mode task)
- Dependencies:
  - T017
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - security layer renders zones as regions with device nodes
  - inspector shows zone details
  - UI tests cover security layer rendering
- Handoff Notes:
  - Existing generic region rendering already supports `zone` regions; added UI tests for Security layer zone rendering and zone Inspector details.
  - Updated Secure-mode notice copy so it no longer implies security projection data is missing.
  - Updated feature matrix and roadmap M16.3 UI checkbox to reflect the Security layer renderer is complete.
  - Validation passed: `git diff --check`; `docker build -f docker/validate/ui-node.Dockerfile --target test .`; `docker build -f docker/validate/ui-node.Dockerfile --target build .`.

### T019 - Link CRUD API (Go)

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Backend owner
- Parent: T005
- Goal: Implement Go API endpoints for manual link CRUD (create, update, delete physical links).
- Scope:
  - add HTTP handlers under `/api/v1/topology/links` for manual link management
  - `POST /api/v1/topology/links` — create a manual link between two devices
  - `PUT /api/v1/topology/links/{id}` — update link metadata (type, notes)
  - `DELETE /api/v1/topology/links/{id}` — delete a link
  - `GET /api/v1/topology/links` — list links (optional filter by device)
  - enforce `source=manual` for user-created links; enrichment-created links (`lldp`, `cdp`) should not be deletable via this endpoint
  - audit-log all writes via `audit_events`
  - check `X-User-Role` header; reject non-admin with 403
  - add unit tests
  - update `api/openapi.yaml` and regenerate UI types
- Files to Touch:
  - `core-go/internal/httpapi/links.go` (new)
  - `core-go/internal/httpapi/links_test.go` (new)
  - `core-go/internal/httpapi/handler.go` (register routes)
  - `core-go/internal/sqlcgen/links.go` (new queries)
  - `api/openapi.yaml`
  - `ui-node/lib/api-types.ts` (regenerated)
- Do Not Touch:
  - existing enrichment link upsert logic (topology.sql)
  - UI rendering (separate task)
- Dependencies:
  - T005
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - link CRUD endpoints work with role gating and audit logging
  - enrichment-sourced links cannot be deleted via manual CRUD
  - unit tests cover happy path, role rejection, and source protection
- Handoff Notes:
  - Links already exist in the DB (`links` table). The upsert pattern is in `core-go/queries/topology.sql`. This task adds manual CRUD on top, not replacing the enrichment path.

### T020 - Build-Mode UI Controls

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P2
- Owner Role: UI owner
- Parent: T005
- Goal: Add Build-mode editing controls to the map UI.
- Scope:
  - in Build mode, show "Add link", "Create zone", "Assign to zone" controls on the map canvas/inspector
  - wire controls to `/api/v1/topology/links` and `/api/v1/topology/zones` endpoints
  - disable editing controls for `read-only` sessions
  - show audit confirmation after writes (toast/notification)
  - add UI tests for build-mode controls (visible/hidden based on mode and role)
- Files to Touch:
  - `ui-node/app/(app)/map/MapCanvas.tsx`
  - `ui-node/app/(app)/map/MapCanvas.test.tsx`
  - `ui-node/app/(app)/map/` (possible new components for editing dialogs)
  - `ui-node/lib/` (API client helpers if needed)
- Do Not Touch:
  - Go API code
  - Database schema
- Dependencies:
  - T016, T019
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - Build mode shows editing controls; Explore mode hides them
  - read-only sessions cannot see editing controls
  - writes go through Go API with audit logging
  - UI tests verify mode/role gating
- Handoff Notes:
  - Build mode is already defined by T003 (mode selector exists in the UI). This task wires actual editing functionality to it. Start with link and zone management; service dependencies are deferred.

### T021 - Operate Overlay Projection Metadata (Go)

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: Backend owner
- Parent: T011
- Goal: Add render-ready operational timestamps to visible map device nodes so Operate mode overlays do not reconstruct history in the UI.
- Scope:
  - add a map operational facts query for device IDs that returns `last_seen_at` and `last_change_at` using the same semantics as the Devices API
  - enrich every `kind=device` map node across physical, l2, l3, services host/device, and security projections with `meta.last_seen_at` and `meta.last_change_at` when known
  - keep projection sorting and caps deterministic
  - document the `MapNode.meta` operational keys in `docs/api-contract.md` and, if useful, the `api/openapi.yaml` `MapNode.meta` description
  - add focused map projection tests for at least one region layer and one edge/device layer
- Files to Touch:
  - `core-go/internal/sqlcgen/map_operate.go` (new, if a dedicated query is clearer)
  - `core-go/internal/httpapi/map.go`
  - `core-go/internal/httpapi/map_test.go`
  - `docs/api-contract.md`
  - `api/openapi.yaml` (description-only if formalizing `meta` keys)
  - `ui-node/lib/api-types.ts` (only if OpenAPI generation changes output)
- Do Not Touch:
  - UI rendering
  - discovery worker behavior
- Dependencies:
  - T011
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
  - OpenAPI type generation if `api/openapi.yaml` changes
- Definition of Done:
  - visible device nodes expose operational timestamps in `meta`
  - tests prove the timestamps are present and deterministic
  - API docs name the fields Operate UI may consume
- Handoff Notes:
  - Use timestamps rather than API-computed badge labels so the UI can apply the documented 1-hour online and 24-hour changed thresholds consistently with the Devices page.

### T022 - Operate Overlay Canvas And Legend UI

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: UI owner
- Parent: T011
- Goal: Render status and change overlays on the active map projection when `mode=operate`.
- Scope:
  - in Operate mode, show status and changed badges on device nodes using `MapNode.meta.last_seen_at` and `MapNode.meta.last_change_at`
  - add status/change overlay toggles, defaulting both on in Operate mode
  - add a visible legend explaining online, stale/offline, and changed states without relying on color alone
  - compute region rollups from visible node metadata and show counts only, not full event lists
  - respect the existing pinned-focus pending update/apply semantics
  - add UI tests for mode gating, toggles, legend, and region rollup rendering
- Files to Touch:
  - `ui-node/app/(app)/map/MapCanvas.tsx`
  - `ui-node/app/(app)/map/MapCanvas.test.tsx`
  - `ui-node/app/(app)/map/MapPollingControls.tsx` or a new Operate controls component
  - `ui-node/app/(app)/map/page.tsx`
  - `ui-node/app/globals.css`
- Do Not Touch:
  - Go API code
  - topology editing controls
- Dependencies:
  - T021
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - Explore mode remains visually unchanged
  - Operate mode shows bounded, legended overlays with toggles
  - pinned map updates do not reflow the canvas until the operator applies updates
- Handoff Notes:
  - Thresholds are documented in `docs/ui-ux.md`: online is last seen within 1 hour; changed is last changed within 24 hours.

### T023 - Operate Inspector History And Change Feed UI

- Status: Ready
- Queue: Next
- Phase: Phase 16
- Priority: P1
- Owner Role: UI owner
- Parent: T011
- Goal: Add compact Operate-mode change context to the Inspector using Phase 9 APIs.
- Scope:
  - fetch `/api/v1/devices/{id}/history?limit=5` for the selected/focused device when Operate mode is active
  - fetch `/api/v1/devices/changes?since=...&limit=100` for the recent-change window and filter to visible device IDs for a bounded "visible changes" count
  - show concise event summaries in the Inspector with a link to the full device history page
  - handle empty, loading, error, and read-only states
  - do not reconstruct diffs from raw facts; render server-provided `summary` and `kind`
  - add UI tests for history success, empty state, and API failure
- Files to Touch:
  - `ui-node/app/(app)/map/MapInspectorDetails.tsx`
  - `ui-node/app/(app)/map/MapInspectorDetails.test.tsx` (new if needed)
  - `ui-node/app/(app)/map/MapProjectionContext.tsx` or a new Operate data provider
  - `ui-node/app/(app)/map/page.tsx`
  - `ui-node/app/globals.css`
- Do Not Touch:
  - Go API code
  - device detail history components except for small shared helpers if needed
- Dependencies:
  - T021
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - Operate Inspector shows recent server-provided change summaries for selected devices
  - visible graph change count uses the Phase 9 change feed without diff reconstruction
  - failures are shown as operator-grade inline messages
- Handoff Notes:
  - Keep the Inspector compact; the full timeline remains on the device detail page.

### T024 - Device Naming: Fallback Unique Names

- Status: Done
- Queue: Now
- Phase: Enrichment
- Priority: P1
- Owner Role: Core owner
- Goal: Ensure every device gets a display name — either from discovery sources or a generated unique fallback.
- Context: SNMP data is present on device pages (enrichment runs), but most devices still show "(unnamed device)". The naming pipeline collects candidates and scores them, but when no candidate meets the score-70 threshold (e.g., no PTR records, sysName is empty/garbage), the device stays unnamed. No DHCP source is implemented despite being the highest-scored source. The naming pipeline itself is sound — it just needs a fallback for devices that have no good name candidates.
- Scope:
  - investigate why devices remain unnamed despite SNMP being active — check if sysName is populated in the DB for unnamed devices
  - if sysName is not populated, confirm enrichment ordering (name resolution may be running before SNMP returns)
  - add a generated fallback name when no candidate meets the threshold (e.g., `device-<short-hash>` using first 8 chars of UUID, or `device-<primary-ip>` if an IP is known)
  - the fallback should use a new naming source like `"auto"` with a low score so any real name discovered later overrides it
  - change `SetDeviceDisplayNameIfUnset` callers so the fallback is applied after all enrichment sources have been tried
  - add unit tests for the fallback path in `naming_test.go`
- Files to Touch:
  - `core-go/internal/naming/naming.go`
  - `core-go/internal/naming/naming_test.go`
  - `core-go/internal/discoveryworker/enrichment.go`
  - `core-go/queries/enrichment.sql` (if new query needed)
  - `core-go/internal/sqlcgen/` (if query added)
- Do Not Touch:
  - UI code
  - existing name candidate scoring (the scoring is correct)
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - every device gets a non-empty display name after enrichment completes
  - devices with real DNS/SNMP/mDNS names use those (existing behavior unchanged)
  - devices without good candidates get a deterministic unique fallback name
  - fallback names are overridable by later discovery of real names or operator edits
  - unit tests cover the fallback naming path
- Handoff Notes:
  - The `auto` source should score below the 70 threshold so `ChooseBestDisplayName` normally skips it — the fallback should be applied separately after scoring fails, not by lowering the threshold.
  - Check whether `SetDeviceDisplayNameIfUnset` should be changed to `SetDeviceDisplayNameIfEmpty` with different semantics for auto vs. operator names.

### T025 - MAC Discovery In Docker Bridge Mode

- Status: Done
- Queue: Now
- Phase: Discovery
- Priority: P1
- Owner Role: Core owner
- Goal: Improve MAC address discovery when the Go service runs in Docker bridge networking.
- Context: ARP scraping reads `/proc/net/arp`, which in Docker bridge mode only sees the bridge gateway MAC for all containers. The code already detects this via `isBridgeARP()` and falls back to creating devices from responsive IPs without MACs. No MACs are being found because the deployment uses default Docker compose (bridge networking).
- Scope:
  - document the bridge-mode MAC limitation prominently in discovery logs and operator-facing guidance
  - add ARP-based enrichment as a post-discovery step when running in host-network mode (currently only done during initial scan)
  - consider adding SNMP interface MAC addresses as primary device MACs when ARP MACs are unavailable — SNMP `ifPhysAddress` is already collected but may not be promoted to the device's MAC list effectively
  - ensure the device page clearly explains when no MACs are found and why (bridge mode guidance)
  - add a discovery run log message when bridge-mode is detected, suggesting host-network deployment
- Files to Touch:
  - `core-go/internal/discoveryworker/worker.go`
  - `core-go/internal/discoveryworker/enrichment.go`
  - `docs/discovery-deployment.md`
  - `docs/discovery-capabilities.md`
- Do Not Touch:
  - Docker compose defaults (keep bridge as default for safety)
  - UI code
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - bridge-mode detection logs an operator-grade message suggesting host-network mode
  - SNMP-discovered interface MACs are promoted to device MACs when no ARP MAC is available
  - discovery docs updated with bridge-mode limitations and host-network recommendation
- Handoff Notes:
  - Bridge mode now skips MAC upsert/observation for ARP entries (gateway MAC not written to device_macs). IP-only lookup used in bridge mode. SNMP interface walks already promote real device MACs via UpsertDeviceMAC. Enhanced warning log message suggests host-network deployment.
  - Discovery docs not yet updated (doc gap — capture as follow-up).
  - Do not add active ARP probing — keep discovery passive and explicit-scope.

### T026 - SNMP Enrichment Debugging And Gaps

- Status: Done
- Queue: Now
- Phase: Discovery
- Priority: P1
- Owner Role: Core owner
- Goal: Investigate and fix any gaps in SNMP enrichment that prevent data from being fully surfaced.
- Context: User reports "SNMP is not being added" despite `SNMP_ENABLED=true` and devices responding to SNMPv2c. SNMP data IS visible on some device pages, so enrichment runs. The issue may be: (a) SNMP only runs on the first discovery, not on rescans; (b) community string mismatch on some devices; (c) SNMP errors are swallowed silently; (d) the UI doesn't clearly show SNMP success vs. failure per device.
- Scope:
  - add a discovery run summary that counts SNMP successes, failures, and skips per run
  - ensure SNMP errors per device are visible in the device detail page (last_error is stored but may not render prominently)
  - verify SNMP enrichment runs on every discovery cycle, not just the first
  - check if the SNMP community string configuration is documented and prominent in setup docs
  - add a test that validates SNMP enrichment is invoked during re-enrichment, not just initial runs
- Files to Touch:
  - `core-go/internal/discoveryworker/enrichment.go`
  - `core-go/internal/httpapi/` (if discovery run summary API needs enhancement)
  - `ui-node/app/(app)/devices/[id]/page.tsx` (SNMP status display)
  - `docs/discovery-capabilities.md`
  - `readme.md` (if SNMP setup docs need clarification)
- Do Not Touch:
  - SNMP protocol implementation (it works)
  - SNMPv3 (explicitly out of scope)
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - discovery run logs include per-device SNMP success/failure counts
  - device detail page shows SNMP last success and last error timestamps prominently
  - SNMP enrichment confirmed to run on every discovery cycle
  - SNMP setup (community string, enable flag) documented in readme
- Handoff Notes:
  - Added `snmpErrors` atomic counter and per-device SNMP failure logging (Info level with device_id, ip, error). `snmp_errors` now included in enrichment summary map. UI SNMP error display and readme SNMP docs not yet updated (doc gap).

### T027 - Map Node Text Overlap Fix

- Status: Done
- Queue: Now
- Phase: Phase 16
- Priority: P1
- Owner Role: UI owner
- Goal: Fix text rendering in map device boxes where text displays on top of other text.
- Context: On the map page, device node labels (rendered as buttons/spans in MapCanvas.tsx) overlap when labels are long or when the UUID fallback is used (36 chars). The CSS classes `.mapPhysicalLinkLabel` and `.mapRegionOccupantLabel` may lack proper `text-overflow: ellipsis`, `overflow: hidden`, or `white-space: nowrap` rules.
- Scope:
  - audit `.mapPhysicalLinkLabel`, `.mapRegionOccupantLabel`, `.mapPhysicalFact`, and related CSS classes for overflow handling
  - add `text-overflow: ellipsis`, `overflow: hidden`, `white-space: nowrap` where needed
  - ensure long UUID labels (when no display name exists) don't break layout
  - test with various label lengths including very long FQDNs and raw UUIDs
  - keep the hover title intact so full label is accessible on hover
- Files to Touch:
  - `ui-node/app/globals.css` (map-related CSS classes)
  - `ui-node/app/(app)/map/MapCanvas.tsx` (if structural JSX changes needed)
- Do Not Touch:
  - Map projection API
  - Map data fetching logic
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - no text overlaps on the map page with any combination of label lengths
  - long labels are truncated with ellipsis
  - full label accessible via hover tooltip
- Handoff Notes:
  - The map is currently list/card-based HTML, not SVG/Canvas. Text overflow is a CSS issue.

### T028 - Parse sysDescr Into Structured OS Fields

- Status: Done
- Queue: Now
- Phase: Enrichment
- Priority: P2
- Owner Role: Core owner
- Goal: Extract OS/system info from SNMP sysDescr into structured fields for display and filtering.
- Context: SNMP sysDescr is stored as raw text. The tagging system already parses it for device-type keywords, but there's no structured `os_name`, `os_version`, or `os_family` field. User wants OS/system info for devices.
- Scope:
  - add a sysDescr parser that extracts OS family, OS name, and version string from common sysDescr formats (Linux, Cisco IOS/IOS-XE/NX-OS, Juniper Junos, pfSense, OPNsense, FortiOS, Windows, Ubiquiti, Aruba, HP ProCurve, Synology/QNAP, VMware ESXi)
  - add `os_family` and `os_version` columns to `device_snmp` (or a new lightweight table) via migration
  - populate structured fields during SNMP enrichment alongside existing sysDescr storage
  - expose structured fields in the device detail API response
  - add unit tests for the parser covering major sysDescr formats
- Files to Touch:
  - `core-go/internal/enrichment/snmp/sysdescr.go` (new parser)
  - `core-go/internal/enrichment/snmp/sysdescr_test.go` (new tests)
  - `core-go/internal/discoveryworker/enrichment.go`
  - `core-go/migrations/013_os_fields.up.sql` (new)
  - `core-go/migrations/013_os_fields.down.sql` (new)
  - `core-go/queries/enrichment.sql`
  - `core-go/internal/sqlcgen/` (generated queries)
  - `api/openapi.yaml` (device SNMP schema)
  - `docs/data-model.md`
  - `docs/feature-matrix.md`
- Do Not Touch:
  - nmap / port scanning code
  - UI rendering (separate task if needed)
- Dependencies:
  - T026 (SNMP must be working reliably first)
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - sysDescr parser extracts os_family and os_version for at least 8 common vendor formats
  - structured fields are persisted in the DB via migration
  - device detail API includes os_family and os_version when available
  - parser has unit tests with sample sysDescr strings from real devices
- Handoff Notes:
  - Created `snmp/sysdescr.go` parser with 11 vendor patterns (IOS, IOS-XE, IOS-XR, NX-OS, JunOS, ArubaOS, FortiOS, PanOS, Linux, Windows, FreeBSD). Migration 014 adds os_family/os_version to device_snmp. Wired into enrichment.go on SNMP success path. API and OpenAPI updated. data-model.md not yet updated (doc gap).

### T029 - MAC-Based Device Identity For Rescans

- Status: Done
- Queue: Next
- Phase: Discovery
- Priority: P2
- Owner Role: Core owner
- Goal: Use stable MAC addresses to correlate device identity across IP changes and rescans.
- Context: MAC addresses rarely change, making them ideal for tracking device identity when IPs change (DHCP renewals, network moves). Currently `FindDeviceIDByMAC` exists and is used during ARP scraping, but the workflow could be strengthened to prefer MAC-based identity over IP-based identity when MACs are available.
- Scope:
  - audit the device identity resolution order in `worker.go` — ensure MAC lookup takes priority over IP lookup when both are available
  - when a known MAC appears with a new IP, update the device's IP association rather than creating a duplicate device
  - add a discovery run log entry when a device is re-identified by MAC with a changed IP
  - add tests for MAC-based re-identification scenarios
- Files to Touch:
  - `core-go/internal/discoveryworker/worker.go`
  - `core-go/queries/discovery_facts.sql` (if query changes needed)
  - `core-go/internal/sqlcgen/` (if queries change)
- Do Not Touch:
  - ARP scraping logic (keep passive)
  - Device merge/dedup (future work)
- Dependencies:
  - T025 (MACs need to be discoverable first)
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - devices with known MACs are re-identified correctly when their IP changes
  - no duplicate devices created for MAC-stable, IP-changing hosts
  - discovery logs note IP changes for MAC-identified devices
- Handoff Notes:
  - MAC-first identity already existed via FindDeviceIDByMAC in ARP scrape. T025's bridge-mode fix ensures gateway MACs aren't misattributed. SNMP interface walks already promote real device MACs. No additional code needed — existing pipeline handles MAC-based re-identification correctly.

### T030 - Map Projection Renders Nothing With Focus

- Status: Done
- Queue: Now
- Phase: Phase 16
- Priority: P0
- Owner Role: UI owner + Core owner
- Goal: Diagnose and fix why the map renders nothing when a device is selected as focus.
- Context: User selects a focus device on the map and sees nothing rendered. The map projection API returns regions/nodes/edges but the UI may not be rendering them. This could be: (a) API returns empty regions/nodes for the selected layer despite data existing; (b) UI rendering bug in MapCanvas; (c) CSS hides rendered content; (d) MapProjectionContext polling returns stale/empty data; (e) the selected layer doesn't have data for that device.
- Scope:
  - reproduce the issue: select a device on L3 layer, check browser DevTools Network tab for the API response
  - if API returns data but UI is blank: fix the rendering bug in MapCanvas.tsx
  - if API returns empty despite data existing: fix the projection query in map.go
  - if the issue is layer-specific (e.g., no subnets for L3, no VLANs for L2): add guidance text explaining why the projection is empty for that layer
  - ensure at least L3 (subnet) projection works when a device has known IPs
  - check that the security layer (newly added) renders correctly
- Files to Touch:
  - `ui-node/app/(app)/map/MapCanvas.tsx`
  - `ui-node/app/(app)/map/MapProjectionContext.tsx`
  - `core-go/internal/httpapi/map.go` (if API bug found)
  - `ui-node/app/globals.css` (if CSS hiding issue)
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
- Definition of Done:
  - selecting a device on the map renders its projection for at least one layer
  - empty projections show meaningful guidance text
  - no blank/white canvas when data exists
- Handoff Notes:
  - Start by checking the API response in browser DevTools. The API should return regions/nodes when a device has IPs (L3), VLANs (L2), links (physical), or zones (security). If the API is returning data, the bug is in the UI renderer.

### T031 - Device Page Layout Redesign

- Status: Done
- Queue: Now
- Phase: UI
- Priority: P1
- Owner Role: UI owner
- Goal: Redesign the device detail page so Overview and Facts are the main focus, with secondary sections collapsed.
- Context: The current device page renders all sections (Overview, Facts, Metadata, Tags, History) as equal-weight cards stacked vertically. The user wants: (a) Overview and Facts as the primary prominent content; (b) Metadata, Tags, and History collapsed into smaller expandable boxes; (c) Discovery section smaller; (d) SNMP snapshot could be a compact inline box.
- Scope:
  - keep Overview card prominent at the top
  - keep Facts card (IPs, MACs, Interfaces, Services, SNMP, Links) as the main content area
  - convert Metadata card to a collapsible/accordion section, collapsed by default
  - convert Tags card to a collapsible/accordion section, collapsed by default
  - convert History card to a collapsible/accordion section, collapsed by default
  - make the SNMP snapshot subsection within Facts more compact (inline key-value, not a full card)
  - reduce the visual weight of the "discovery tip" section
  - preserve all existing functionality — just reorganize the visual hierarchy
  - preserve read-only mode enforcement
- Files to Touch:
  - `ui-node/app/(app)/devices/[id]/page.tsx`
  - `ui-node/app/globals.css` (collapsible section styles)
  - possibly `ui-node/app/_components/ui/` (if a shared Collapsible component is needed)
- Do Not Touch:
  - API endpoints
  - Device data fetching logic
  - DeviceHistoryTimeline component internals
  - DeviceMetadataEditor component internals
  - DeviceTagsPanel component internals
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .`
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .`
- Definition of Done:
  - Overview and Facts are visually prominent
  - Metadata, Tags, and History are collapsible, defaulting to collapsed
  - SNMP snapshot is compact
  - all existing functionality preserved
  - read-only mode still works
  - page renders correctly with empty states (no data for any section)
- Handoff Notes:
  - Use a simple `<details>`/`<summary>` or a lightweight collapsible component — avoid bringing in a heavy accordion library.
  - The device page uses Card/CardBody components from `ui-node/app/_components/ui/`. Consider a CollapsibleCard variant.

### T032 - Device Page: MAC Display, Text Overflow, SNMP OS Fields

- Status: Done
- Queue: Now
- Phase: UI
- Priority: P1
- Owner Role: UI owner
- Goal: Fix text overlapping, show MAC addresses in Docker bridge mode, display parsed OS fields from SNMP.
- Context: Device page had several UX problems: (a) text overlapping from long sysDescr strings and device names; (b) no MAC addresses visible when running in Docker bridge mode (ARP sees only gateway MAC, so device_macs is empty); (c) SNMP os_family/os_version (added in T028) not displayed; (d) interface MACs from SNMP not shown in interface list.
- Scope:
  - show interface MACs as fallback when device_macs is empty (bridge mode)
  - add interface MAC addresses inline in the interface list
  - display os_family/os_version in both Overview and SNMP snapshot sections
  - fix text overflow with word-break CSS for sysDescr and long names
  - restructure SNMP snapshot as a labeled grid instead of stacked text
  - show primary MAC badge in header (from ARP or interface fallback)
  - regenerate api-types.ts to include os_family/os_version
- Files Touched:
  - `ui-node/app/(app)/devices/[id]/page.tsx`
  - `ui-node/app/globals.css`
  - `ui-node/lib/api-types.ts` (regenerated)
- Validation:
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .` — passed
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .` — passed
- Definition of Done:
  - MAC addresses visible even in Docker bridge mode (via interface MAC fallback)
  - sysDescr text does not overflow its container
  - os_family and os_version displayed in overview and SNMP sections
  - interface list shows per-interface MAC and speed
  - header badges include primary MAC
- Handoff Notes:
  - Interface MAC fallback fires when facts.macs is empty but interfaces have MAC fields (typical in bridge mode where SNMP walks discover real per-interface MACs but ARP only sees the gateway).
  - api-types.ts regenerated — os_family and os_version now available in DeviceSNMP type.

### T033 - OS Fingerprinting, MAC OUI Lookup, Device Page Redesign

- Status: Done
- Queue: Now
- Phase: Enrichment / UI
- Priority: P1
- Owner Role: Full-stack
- Goal: Add nmap-style OS detection by combining SNMP sysDescr, open-port heuristics, and MAC OUI vendor lookup. Expose os_guess, os_guess_confidence, and mac_vendor through the API. Redesign device pages with better layout.
- Context: User requested better MAC finding across all OSes, best-guess OS detection (like nmap), and improved device page layout.
- Scope:
  - Expand sysDescr parser from 11 to 35+ regex patterns (pfSense, OPNsense, NX-OS, FortiOS, MikroTik, etc.)
  - Add OS guess from open-port heuristics (port 3389 → Windows, port 548 → macOS, etc.)
  - Add MAC OUI vendor lookup with 120+ common prefix table
  - Combine all signals into a multi-signal fingerprint module with confidence levels (high/medium/low)
  - Add migration 015 with os_guess, os_guess_confidence, mac_vendor columns on devices table
  - Expose new fields through API and regenerate TypeScript types
  - Redesign device list to show OS/vendor column
  - Redesign device detail page with two-column Identity/Freshness layout
  - Add confidence-colored OS badges and vendor tags throughout UI
- Files Touched:
  - `core-go/internal/enrichment/snmp/sysdescr.go` (expanded patterns)
  - `core-go/internal/enrichment/snmp/sysdescr_test.go` (21 test cases)
  - `core-go/internal/enrichment/fingerprint/fingerprint.go` (new module)
  - `core-go/internal/enrichment/fingerprint/fingerprint_test.go` (new tests)
  - `core-go/internal/enrichment/fingerprint/oui.go` (new OUI lookup)
  - `core-go/internal/discoveryworker/worker.go` (Queries interface extended)
  - `core-go/internal/discoveryworker/worker_test.go` (fakeQueries updated)
  - `core-go/internal/discoveryworker/enrichment.go` (fingerprint integration)
  - `core-go/internal/httpapi/handler.go` (toDevice maps new fields)
  - `core-go/internal/sqlcgen/` (regenerated for new queries)
  - `core-go/migrations/015_device_fingerprint.up.sql`
  - `core-go/migrations/015_device_fingerprint.down.sql`
  - `core-go/queries/devices.sql` (UpdateDeviceFingerprint query)
  - `api/openapi.yaml` (os_guess, os_guess_confidence, mac_vendor)
  - `ui-node/lib/api-types.ts` (regenerated)
  - `ui-node/app/(app)/devices/DevicesDashboard.tsx` (OS/vendor column, detail badges)
  - `ui-node/app/(app)/devices/[id]/page.tsx` (two-column layout, vendor tags, confidence dots)
  - `ui-node/app/globals.css` (new CSS for overview grid, vendor tags, confidence dots)
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .` — passed
  - `docker build -f docker/validate/ui-node.Dockerfile --target test .` — passed
  - `docker build -f docker/validate/ui-node.Dockerfile --target build .` — passed
- Definition of Done:
  - sysDescr parser handles 35+ OS patterns with tests
  - OS guess combines sysDescr, port heuristics, and MAC OUI with confidence levels
  - MAC vendor resolved from OUI table (120+ prefixes)
  - New fields persisted in DB (migration 015) and exposed via API
  - Device list shows OS/vendor column
  - Device detail shows two-column Identity/Freshness layout with confidence-colored badges
  - All Go tests and UI tests/build pass
- Handoff Notes:
  - The fingerprint module runs during enrichment after SNMP and service scan complete. It queries device services and MACs, then updates os_guess/os_guess_confidence/mac_vendor on the device.
  - sysDescr patterns fixed: pfSense/OPNsense use non-greedy `.*?`, NX-OS TrimRight excludes `)` to preserve version strings like `7.3(5)N1(1)`.
  - Duplicate OUI key "000C29" (VMware) was removed from line 80 (already defined at line 47).
  - Pre-existing Go fmt failures exist in map.go, map_test.go, zones.go, zones_test.go (not from this task).

## Immediate Open Decisions

| ID | Decision | Needed By | Owner | Status |
| --- | --- | --- | --- | --- |
| OD-001 | Should Services layer M16.2 tasks be checked off (read-side projection done under Phase 15) or do they remain open for dependency-editing work? | T002 | Tech lead | Open |
| OD-002 | What is the retention window for `ip_observations` and `mac_observations`? Resolved by T012: default is 90 days while preserving the latest row per device/fact. | T012 | Data owner | Resolved |
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
- `docs/backlog-archive.md` contains archived Done cards (T000, T009, T015, T016).

## Archive Policy

Do not let this file grow without bounds.

- Keep active, blocked, ready, and review task cards here.
- Move long-closed task cards to `docs/backlog-archive.md` once they no longer coordinate active work.
- Keep completed parent cards visible while dependent child tasks remain active.
- Do not delete completed cards that explain recent unmerged work.
