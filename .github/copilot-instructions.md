# Copilot instructions for this repo

## Prime directives

1. **Human-readable first**
   - Prefer Markdown + YAML for documentation and data.
   - Keep per-device and per-service files small and diff-friendly.
2. **No secrets**
   - Never add passwords, PSKs, tokens, or private keys.
   - Store only `credential_refs` pointing to an external secret manager.
3. **Stable IDs**
   - IDs are cross-referenced across files; avoid renames.
   - When adding something, update all required references.
4. **Minimize churn**
   - Avoid formatting-only changes.
   - Preserve existing structure and key names.

## Development guidance

- **Do the right tests**
  - Execute existing unit/integration checks for touched areas.
  - Add regression tests for newly exposed behavior and document commands to run in PR notes.
- **Document alongside**
  - Refresh relevant Markdown docs (roadmap, network guides, README sections) when features or topology shift.
  - Keep changelog/history notes concise and focused on impact.
- **Use automation**
  - Prefer existing scripts (e.g., `scripts/validate_inventory.py`) or CI workflows; add helpers only when broadly useful.

### Go best practices

- **Formatting & linting**
  - Run `gofmt` or `goimports` on modified Go files.
  - Resolve outstanding `golangci-lint` issues or document why they remain.
- **Dependency care**
  - Manage dependencies via `go.mod`/`go.sum`; tidy only when dependencies change and vendor only when directed.
  - Skip unused imports or variables so compiler/linter errors stay minimal.
- **Testing**
  - Cover exported functions with table-driven tests and deterministic inputs.
  - Stub external services/interfaces and use `t.Helper()` for shared setups.
- **Error handling**
  - Wrap errors with context (`fmt.Errorf("...: %w", err)` or `errors.Join`) and check them explicitly.
  - Favor early returns over deep nesting.
- **Code style**
  - Keep functions focused and extract helpers with clear names for complex logic.
  - Prefer immutable structs and inject dependencies explicitly.

### JS/Node best practices

- **Styling & linting**
  - Respect configured ESLint/Prettier rules; document any disabled rules with TODO comments explaining why.
  - Stick to repo formatting conventions and avoid mixing tabs and spaces.
- **Dependency hygiene**
  - Update `package.json` and `package-lock.json` together; run `npm install` or `npm ci` before committing lock updates.
  - Audit new packages for maintenance quality and licensing.
- **Asynchronous patterns**
  - Use `async/await` with error handling (`try/catch` or `.catch()`); do not swallow promise rejections.
  - Employ `Promise.allSettled`/`Promise.all` with timeout wrappers for parallel workloads to avoid unbounded concurrency.
- **Testing**
  - Run existing Jest/Mocha tests after touching JS; add tests for exposed behavior and mock external calls cleanly.
  - Keep test data minimal, deterministic, and reset modules between tests if necessary.
- **Runtime safety**
  - Validate inputs (e.g., `typeof` checks or schema validation) before usage.
  - Log contextual details safely, avoiding sensitive payload exposure in public-facing services.

---

## Copilot Instructions — Network Tracker / Mapper Project

This project is written 100% by AI. Follow these rules strictly. Consistency, safety, and architectural discipline matter more than cleverness.

### Core principles

- Do not invent architecture. Follow the roadmap and existing docs.
- Prefer boring, proven solutions over novel ones.
- Never break API contracts without updating docs first.
- Clarity beats conciseness in service boundaries.
- Every feature must be traceable to documentation.

If unsure, stop and read `/docs` before writing code.

### Mandatory documentation contract

The `/docs` folder is the source of truth. You must read it before coding and update it when behaviour changes.

Required documents:

- `/docs/roadmap.md`
- `/docs/feature-matrix.md`
- `/docs/data-model.md`
- `/docs/api-contract.md`
- `/docs/architecture.md`

Network map work (UI, projection APIs, map data model):

- Read `/docs/network_map/interface-rules.md` first and treat it as the interaction contract (non-negotiables + v1 must-haves).
- Cross-check against `/docs/network_map/network_map_ideas.md` (product concept + mocks) and `/docs/network_map/implementation-stack.md` (tooling choices).

Rules:

- Never implement a feature not listed in feature-matrix without permission.
- Never change the DB schema without updating data-model without permission.
- Never change API behaviour without updating api-contract without permission.
- Docs first, code second.

### Feature matrix discipline

Every feature must include:

- Name
- Description
- Owning service
- API endpoints
- DB tables
- Status: planned | in-progress | complete | deprecated

No orphan code. No hidden behaviour.

### Service responsibilities

#### Go (Core Service)

Owns:

- Network discovery
- Polling and scheduling
- Normalisation
- Database access
- REST API
- WebSockets

Forbidden:

- HTML rendering
- UI logic
- Auth UI
- Direct user interaction
- Business rules tied to UI

Go owns truth and state.

#### Node.js (UI Service)

Owns:

- UI rendering
- API calls
- HTML
- Forms
- User workflows
- Auth and sessions

Calling Go APIs

Forbidden:

- Network scanning
- Direct DB access
- Re-implementing business logic

Node owns presentation and interaction.

### API rules

- REST over HTTP
- JSON only
- Versioned endpoints if behaviour changes
- Explicit error responses

The API is a contract, not an implementation detail.

### Database rules

- PostgreSQL
- Migrations are mandatory
- No silent schema changes
- No shared DB access between services

### Data model discipline

- Tables must match `/docs/data-model.md`
- Foreign keys preferred
- Avoid unstructured JSON unless justified in docs

### Docker rules

- One process per container
- Env vars only
- No local state outside volumes
- All config via environment variables
- No hardcoded paths

### Safety & defensive coding

#### Go

- Always handle errors explicitly
- No panics in request paths
- Timeouts on all network operations
- Context propagation everywhere

#### Node.js

- Validate all input
- Never trust API responses implicitly
- Avoid blocking the event loop
- Handle partial failures gracefully

### Logging & observability

- Structured logs only
- No printf debugging
- Log at service boundaries
- Errors must be actionable

### AI workflow

Before coding:

- Read `/docs`
- Confirm feature exists
- Confirm owning service
- Confirm API and data model

When writing code:

- Small commits
- No speculative features
- Match existing style
- Update docs if behaviour changed
- Re-check feature matrix
