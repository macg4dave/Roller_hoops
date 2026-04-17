# AI Coding Control

## Purpose

This repo is expected to receive substantial AI-authored code. The control system keeps that productive without allowing architecture, contracts, or safety boundaries to drift.

Unlike an AI runtime feature, this document governs how coding agents change the repository.

## Control Layers

1. **Documentation Source Of Truth**
   - Backlog, roadmap, issues, feature matrix, architecture, API contract, data model, and UI/UX docs define expected behavior.
   - `BACKLOG.md` is the execution source of truth for AI task cards and validation handoff.
   - Agents should update the relevant document when behavior changes.

2. **Contract Gates**
   - `api/openapi.yaml` defines API shape.
   - `ui-node/lib/api-types.ts` is generated from OpenAPI.
   - Go route drift is guarded by `core-go/internal/httpapi/openapi_contract_test.go`.
   - UI build and tests catch TypeScript and workflow drift.

3. **Service Boundaries**
   - `core-go` owns truth, discovery, normalization, persistence, and REST APIs.
   - `ui-node` owns presentation, forms, sessions, and the BFF proxy.
   - PostgreSQL is private to `core-go`.

4. **Validation Evidence**
   - Every meaningful change should have a rerunnable command, test, or documented manual check.
   - Partial validation is acceptable only when the blocker is named.

5. **Operational Safety**
   - Discovery is scoped and bounded.
   - Secrets never enter the repo.
   - Remote administration is not performed unless explicitly requested by the user.

## Agent Contract

Before coding:

- Read `AGENTS.md`.
- Check `BACKLOG.md` for an existing task card or add one for substantial work.
- Identify the owning service and contract document.
- Check whether the feature exists in `docs/feature-matrix.md`.
- Check whether an existing issue or roadmap item already defines the scope.

While coding:

- Prefer small, focused edits.
- Do not add hidden behavior.
- Keep API, docs, generated types, and tests synchronized.
- Add regression tests for fixed logic errors.
- Preserve user changes already in the worktree.

Before handoff:

- Run focused validation.
- Run broader validation when practical.
- Report exact commands and blockers.
- Mention docs updated and any remaining drift risk.

## Drift Signals

Treat these as evidence that an agent should stop and realign docs/code:

- A UI type duplicates a generated OpenAPI type.
- A route exists in Go but not in `api/openapi.yaml`, or the reverse.
- A feature exists in code but not in `docs/feature-matrix.md`.
- `docs/api-contract.md` describes old behavior.
- A service reaches across its boundary, such as UI code accessing the DB or Go code depending on UI sessions.
- Optional discovery behavior can run without explicit scope, enable flag, or allowlist.
- A test only proves mocks work and never exercises the changed contract.

## Review Checklist

- Does the change fit the roadmap or issue?
- Is the owning service correct?
- Are API/schema/data-model docs synchronized?
- Are generated files intentionally generated?
- Are tests focused on the changed behavior?
- Could this scan, mutate, import, or expose more than the user intended?
- Is there any user-facing behavior that needs README or UI/UX documentation?
