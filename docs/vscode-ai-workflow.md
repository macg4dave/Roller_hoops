# VS Code AI Workflow

This guide exists for Codex, Copilot, and human reviewers using VS Code on this repository.

## Start Here

1. Open the repository root, not a subfolder.
2. Read `AGENTS.md`.
3. Read `BACKLOG.md`.
4. Run the VS Code task `agent: show ready queue`.
5. Claim one backlog task before substantial work.

## Recommended Extensions

VS Code should prompt for the shared recommendations in `.vscode/extensions.json`:

- GitHub Copilot and Copilot Chat
- Go
- Docker
- YAML
- GitHub Actions
- Markdown tools

These are recommendations only; the project must still build and test from the CLI.

All shared validation tasks are Docker-backed so the workspace can be used without installing local Go or Node.js toolchains.

## Shared Tasks

Use `Terminal: Run Task...` and prefer these labels:

| Task | Purpose |
| --- | --- |
| `agent: show ready queue` | Print startable backlog rows |
| `agent: show required reading` | Print the top of `AGENTS.md` |
| `git: diff check` | Catch whitespace and conflict-marker problems |
| `ui: npm ci` | Validate UI dependency install inside Docker |
| `ui: gen openapi types` | Regenerate `ui-node/lib/api-types.ts` using Docker and copy it back into the workspace |
| `ui: test` | Run Vitest inside Docker |
| `ui: build` | Run Next build/type check inside Docker |
| `go: fmt check` | Check Go formatting inside Docker |
| `go: vet` | Run Go vet inside Docker |
| `go: test` | Run Go tests inside Docker |
| `go: test via docker` | Alias for the Docker-backed Go test path |
| `validate: ui` | OpenAPI typegen, UI tests, UI build |
| `validate: available local checks` | Whitespace check plus Docker-backed UI validation |
| `docker: dev stack up` | Build and start the dev stack |
| `docker: stack down` | Stop and remove dev-profile stack volumes |

## UNC Workspace Note

If the repo is opened through a UNC path, some direct `cmd.exe` workflows may fail because Windows launches from `C:\Windows`.

The shared VS Code tasks avoid local Node/Go execution, which removes most UNC friction. If a manual command fails with `No test files found` from `C:/Windows`, rerun from a mapped path:

```powershell
Set-Location G:\Roller_hoops
```

## AI Coding Loop

For AI agents:

1. Use `agent: show ready queue`.
2. Read the selected task card in `BACKLOG.md`.
3. Read the required docs named by that task.
4. Make the smallest scoped change.
5. Prefer tests beside the changed module.
6. Run the focused validation task.
7. Run `git: diff check`.
8. Update `BACKLOG.md` handoff notes if validation is partial or blocked.

## Copilot Prompt Starters

Good prompts for Copilot Chat:

- `Read AGENTS.md and BACKLOG.md. Summarize the next Ready task and the validation commands before editing.`
- `Review this diff against docs/engineering-standards.md and docs/ai-coding-control.md. Name concrete drift risks.`
- `For this API change, check api/openapi.yaml, docs/api-contract.md, and ui-node/lib/api-types.ts for drift.`
- `For this UI change, check read-only role behavior, URL state, loading/error states, and docs/ui-ux.md.`

## Do Not Use AI Approval As Validation

Copilot/Codex agreement is not validation. Only rerunnable checks count:

- tests
- builds
- OpenAPI generation and diff checks
- Go contract tests
- Docker smoke checks
- documented manual checks when automation is not available
