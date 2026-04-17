# Backlog

This document is the AI-facing execution board and developer runbook for Roller_hoops. It is optimized for coding agents and humans working through small, verifiable changes without losing the project's service boundaries or safety posture.

If this file and [docs/roadmap.md](docs/roadmap.md) disagree on active sequencing or execution detail, this file is the execution source of truth and the roadmap should be updated to match. The roadmap remains the source for higher-level product scope and milestone framing.

If this file and implementation disagree, update this file before starting new work.

## Project Snapshot

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

## Ready Queue

Only rows with `Status` = `Ready` are startable without more planning unless the user explicitly assigns them.

| ID | Queue | Phase | Priority | Task | Status | Depends On | Validation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| T000 | Now | Process | P1 | AI backlog and runbook foundation | Done | None | `git diff --check` + docs consistency review |
| T009 | Now | Process | P1 | VS Code AI workspace scaffolding | Done | T000 | `git diff --check` + JSON parse check + docs consistency review |
| T001 | Now | Hardening | P0 | Auth and OpenAPI drift hardening | Review | None | `cd ui-node && npm test` + `cd ui-node && npm run build`; Go tests blocked locally until Go/Docker daemon available |
| T002 | Now | Process | P1 | Planning-doc drift reconciliation | Ready | T000 | manual roadmap/feature-matrix/API docs consistency review |
| T010 | Now | Phase 15/16 | P1 | Basic focused network diagram MVP | Ready | T002 | `cd ui-node && npm test` + `cd ui-node && npm run build`; Go/API tests if projection shape changes |
| T003 | Now | Phase 16 | P1 | Map modes contract and UI selector | Ready | T002 | `cd ui-node && npm test` + `cd ui-node && npm run build` |
| T004 | Next | Phase 16 | P1 | Security layer v1 planning slice | Ready | T002 | manual docs consistency review + OpenAPI draft review if API shape changes |
| T005 | Next | Phase 16 | P1 | Build-mode map editing parent issue | Ready | T002 | parent/child cards written + feature matrix synced |
| T006 | Next | Ops | P1 | Local validation toolchain/runbook cleanup | Ready | T000 | runbook update + one confirmed Go or Docker validation path |
| T007 | Next | Discovery | P1 | Discovery deployment smoke matrix | Ready | T006 | documented Docker bridge, host-network, and native-host smoke commands |
| T008 | Later | Docs | P2 | Markdown encoding and typography cleanup | Ready | T000 | `git diff --check` + spot-check rendered docs |

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
cd ui-node
npm ci
npm run gen:openapi
npm test
npm run build
```

Use focused Vitest runs for small changes:

```powershell
cd ui-node
npm test -- --run path/to/file.test.ts
```

### Go Commands

```powershell
cd core-go
gofmt -w <changed-go-files>
go vet ./...
go test ./...
```

If Go is not installed and Docker Desktop is running:

```powershell
docker run --rm -v "${PWD}:/src" -w /src/core-go golang:1.24-alpine go test ./...
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
3. Regenerate UI types:

   ```powershell
   cd ui-node
   npm run gen:openapi
   ```

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
  - `cd ui-node && npm test`
  - `cd ui-node && npm run build`
  - `git diff --check`
  - `cd core-go && go test ./...` when Go or Docker fallback is available
- Definition of Done:
  - UI tests and build pass
  - Go contract tests pass or local blocker is documented
  - OpenAPI generated types are committed
- Handoff Notes:
  - UI validation passed in the current working tree
  - Go validation is locally blocked when `go` is unavailable and Docker Desktop Linux daemon is stopped

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
  - `cd ui-node && npm test`
  - `cd ui-node && npm run build`
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
  - `cd ui-node && npm test`
  - `cd ui-node && npm run build`
  - `cd core-go && go test ./...` if Go/API projection code changes
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

- Status: Ready
- Queue: Next
- Phase: Ops
- Priority: P1
- Owner Role: Operations owner
- Goal: Make local validation reliable for agents on Windows/UNC workspaces and Docker/Go variants.
- Scope:
  - document UNC path caveat for npm/cmd wrappers
  - document local Go install path and Docker fallback
  - add a short troubleshooting section for Docker Desktop Linux daemon failures
  - update backlog/runbook commands if a better local command is confirmed
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
  - one confirmed UI validation command from mapped path
  - one confirmed Go validation command, or documented blocker
- Definition of Done:
  - future agents can run validations without rediscovering UNC and Docker daemon pitfalls
  - blockers are documented with exact symptoms and remedies
- Handoff Notes:
  - Current environment has working UI validation via `G:\Roller_hoops`; Go validation may be blocked by missing Go and stopped Docker Desktop Linux daemon.

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

## Global Blockers And Notes

- Go validation is unavailable on hosts without Go installed unless Docker Desktop's Linux daemon is running.
- UNC workdir validation through npm may fail because `cmd.exe` defaults to `C:\Windows`; use the mapped `G:\Roller_hoops` path.
- The backlog and issue log now coexist: use this file for execution cards and `docs/issues.md` for historical issue records or user-facing bug reports.

## Archive Policy

Do not let this file grow without bounds.

- Keep active, blocked, ready, and review task cards here.
- Move long-closed task cards to `docs/backlog-archive.md` once they no longer coordinate active work.
- Keep completed parent cards visible while dependent child tasks remain active.
- Do not delete completed cards that explain recent unmerged work.
