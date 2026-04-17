# Backlog Archive

Archived task cards from [BACKLOG.md](../BACKLOG.md). These cards are **Done** and no longer coordinate active work. They are preserved here for historical reference.

See the [Archive Policy](../BACKLOG.md#archive-policy) in the main backlog for when cards move here.

---

## Archived Task Cards

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

### T015 - End-User Startup Scripts

- Status: Done
- Queue: Now
- Phase: Ops
- Priority: P1
- Owner Role: Operations owner
- Goal: Provide per-OS launcher scripts that start the Docker stack and open the web UI for end users.
- Scope:
  - add Windows, macOS, and Linux/WSL startup scripts
  - start the Compose stack in detached mode so the browser can open after startup
  - wait for the UI health check before opening `http://localhost/`
  - document usage in `readme.md`
  - add the capability to `docs/feature-matrix.md`
- Files to Touch:
  - `scripts/start-windows.ps1`
  - `scripts/start-windows.cmd`
  - `scripts/start-macos.command`
  - `scripts/start-linux.sh`
  - `readme.md`
  - `docs/feature-matrix.md`
  - `BACKLOG.md`
- Do Not Touch:
  - runtime service code
  - API contracts
  - database migrations
- Dependencies:
  - None
- Validation:
  - `git diff --check`
  - shell syntax/command review
  - PowerShell parse check if available
- Definition of Done:
  - each supported OS has an obvious launcher script
  - launchers start the Docker Compose stack and open the UI
  - README points end users to the scripts
- Handoff Notes:
  - User requested "same scripts that start the program for each supported OS" and that the webpage opens automatically.
  - Added Windows, macOS, and Linux/WSL launchers that run Compose detached, wait on `/healthz`, then open `http://localhost/`; documented `--dev` and `--logs` usage in the README.
  - Verified `scripts\start-windows.cmd` on Windows: Compose started successfully, UI health returned `200 {"ok":true}`, and `http://localhost/` returned the `Roller_hoops` page.

### T016 - Discovery Ping Fallback For Docker Bridge Networking

- Status: Done
- Queue: Now
- Phase: Discovery
- Priority: P0
- Owner Role: Core owner
- Goal: Fix discovery only seeing Docker-internal devices when running on Docker bridge networking (the default).
- Scope:
  - modify `pingSweep` to collect responding IPs (not just counts)
  - add ping-based device creation fallback when ARP yields no in-scope results but ping finds responders
  - detect Docker bridge ARP (all entries share one MAC) and log a warning
  - update discovery deployment and capabilities docs with Windows/bridge guidance
  - update `.env.example` to emphasize `DISCOVERY_DEFAULT_SCOPE`
- Files to Touch:
  - `core-go/internal/discoveryworker/worker.go`
  - `docs/discovery-deployment.md`
  - `docs/discovery-capabilities.md`
  - `.env.example`
  - `BACKLOG.md`
  - `docs/issues.md`
- Do Not Touch:
  - UI code
  - database migrations
  - API contracts
- Dependencies:
  - None
- Validation:
  - `docker build -f docker/validate/core-go.Dockerfile --target test .`
  - `docker build -f docker/validate/core-go.Dockerfile --target vet .`
  - `docker build -f docker/validate/core-go.Dockerfile --target fmtcheck .`
- Definition of Done:
  - discovery worker falls back to ping-based device creation when ARP is ineffective
  - bridge-mode ARP is detected and logged as a warning
  - run stats include `ping_fallback_used` flag
  - docs explain Windows/Docker Desktop limitations and recommend `DISCOVERY_DEFAULT_SCOPE`
- Handoff Notes:
  - Root cause: `/proc/net/arp` inside a Docker bridge container only shows Docker-internal entries. The ping sweep was pinging real hosts but discarding the list of responders.
  - Fix: ping sweep now collects responding IPs; when ARP finds 0 in-scope devices but ping found responders, devices are created from ping IPs (IP-only, no MAC). Bridge-mode ARP is auto-detected (all MACs identical) and warned.
  - Limitation: ping-based devices have no MAC address. For full ARP/MAC fidelity, users should use `docker-compose.hostnet.yml` (Linux) or run `core-go` natively.
  - The user must still set `DISCOVERY_DEFAULT_SCOPE` to their real subnet or provide scope per run. Without a scope, there are no targets to ping.
