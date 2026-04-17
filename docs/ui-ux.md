# UI/UX foundations (Phase 12)

This document is the **source of truth** for the operator UI experience in Phase 12.

The UI is the product. The Go service is the truth.

## Goals

- Make day-to-day operation possible **without curl**:
  - triage devices quickly
  - run discovery and understand outcomes
  - inspect facts (IPs/MACs/interfaces/services) confidently
  - edit metadata safely
- Keep the UI **boring, fast, and resilient**:
  - SSR-first and progressively enhanced
  - predictable navigation
  - clear loading and error states
  - accessible by default

## Non-goals (explicit)

- No new auth system (auth is already UI-owned; improve UX only).
- Preserve the basic operator expectation for later map work: selecting a
  device should be able to show a readable device-to-router/switch diagram with
  IP/MAC facts.
- No implementation of broad network-map canvas work here (that’s Phase 13+).
- No UI-side reconstruction of history/diffs (Phase 9 APIs already provide this).
- No UI direct DB access.

## Operator-first UX principles

1. **Clarity beats density**
   - Prefer “obvious and scannable” over “everything on one screen”.
2. **State should be visible**
   - Every view shows:
     - last updated time
     - current filters/sort
     - pagination position (cursor state)
3. **Never trap the user**
   - Every action has:
     - a cancel path (where applicable)
     - an error message that explains what to do next
4. **Deterministic UI**
   - Stable ordering, stable IDs, stable labels.
   - Avoid UI churn during polling (no re-sorting surprises).
5. **Safe by default**
   - Read-only users see disabled actions with an explanation.
   - Destructive actions require confirmation and are rare.

## Information architecture (v1)

Primary sections (top-level routes):

- `/devices`
  - list + filters + search + paging
  - export/import entry points
- `/devices/{id}`
  - overview + facts + metadata + history
- `/discovery`
  - status + run trigger
  - run history list + run detail/logs
- `/auth/...`
  - login/account flows

Navigation rules:

- Keep a constant “app shell”: header + main content.
- Don’t hide navigation behind gestures.
- Encode view state in URL when it matters (filters, search, sort, cursor).

## Page anatomy (consistent layout)

Every page should have the same structure:

- **Page title** + short subtitle (“what is this?”)
- **Primary action** (if any) on the right
- **Secondary actions** in an overflow menu (avoid button piles)
- **Body** as cards/sections with headings

## Design system (small, internal)

We prefer a small set of internal primitives (not a full component library).

Foundation primitives (M12.1):

- `Button` (primary/secondary/danger; disabled states)
- `TextInput`, `Select`, `Textarea` (with inline validation)
- `Badge` (status chips: online/offline/changed/queued/running)
- `Card` / `Section`
- `InlineAlert` (info/warn/error)
- `EmptyState` and `Skeleton`

Workflow primitives (Phase 12.2+):

- `Table` (sortable headers, empty state, row actions)
- `Tabs` (device detail sections)
- `ConfirmDialog` (rare; for destructive or irreversible actions)

Style guidance:

- Prefer system fonts.
- Prefer high-contrast neutral palette with one accent.
- Prefer spacing consistency over clever visuals.

Accessibility requirements:

- Visible focus/selection ring.
- Keyboard navigation beyond browser defaults is out of scope (no custom keybindings/shortcuts).
- No color-only meaning (badges also carry text).
- Respect `prefers-reduced-motion`.

## Loading, polling, and perceived performance

- Use skeletons for primary tables/sections.
- Polling must be:
  - bounded (interval + backoff on errors)
  - respectful of the tab being hidden
  - stable (don’t reorder results while user is reading)
- Always show a “Last updated …” timestamp near live panels.

## Error handling (operator-grade)

- Validation errors: inline at the field, plus a summary at the top.
- API errors: show a short summary and a clear next step:
  - retry
  - check auth
  - check DB readiness
- Never show raw stack traces to operators.

## Workflow patterns

### Device triage (fast path)

- From `/devices`, operators can:
  - search
  - filter (status/changed)
  - open a device
  - see the key facts immediately

### Metadata edits (safe path)

- Inline edit with explicit save/cancel.
- Success feedback that is subtle but clear.
- On failure, keep typed input (no rage).

### Discovery operations (confidence)

- One obvious “Run discovery” button.
- Status includes:
  - queued/running/succeeded/failed
  - started/completed time
  - summary stats (devices touched, new facts)
  - link to logs for debugging

### Basic network diagram (map fast path)

- From a device, operators should be able to open a focused diagram that shows:
  - the selected device
  - nearby router/switch/peer when known
  - primary IP and MAC facts on or near each visible device
  - relationship source/confidence for each visible link
- The diagram should be ordinary and readable before it is clever.
- The focused diagram must stay bounded; expanding neighbors is an explicit
  action, not an automatic whole-network render.

## Map modes (Explore / Build / Secure / Operate)

The map supports four **modes** that control which actions and overlays are
available. Modes are orthogonal to layers — any layer can be viewed in any mode,
but available actions and visual emphasis change.

| Mode | Purpose | Write actions | Default |
| --- | --- | --- | --- |
| Explore | Read-only browsing, navigation, inspection | None | Yes |
| Build | Topology editing: create/edit links, assign zones, manage memberships | Requires Build-mode APIs (not yet implemented) | No |
| Secure | Security-focused view: highlights zones, policies, trust boundaries | Read-only until security layer APIs exist | No |
| Operate | Operational monitoring: live status, health indicators, alerts | Read-only in v1 | No |

### Mode contract

- Mode is encoded in the URL as `mode=explore|build|secure|operate`.
- Default mode is `explore`. Missing or empty mode resolves to `explore`.
- Invalid modes fall back to `explore` with a visible warning.
- Mode persists across layer switches (changing layer keeps the current mode).
- Mode persists across focus changes.
- Build mode does not expose write actions until authorized Build-mode APIs
  exist. Until then, Build mode renders the same canvas as Explore but shows
  a "Build actions coming soon" notice where the action toolbar will appear.
- Deep links include mode: `/map?layer=l3&mode=build&focusType=device&focusId=...`

### Mode semantics (v1)

- **Explore**: The default operator experience. Browse layers, inspect nodes,
  follow relationships. All existing map behavior lives here.
- **Build**: Reserved for future topology editing. In v1, selecting Build mode
  shows a disabled-action notice. No writes are possible.
- **Secure**: Reserved for security overlays (zone highlighting, policy
  visualization). In v1, selecting Secure mode shows a "security overlays
  coming soon" notice.
- **Operate**: Reserved for operational overlays (status badges, health
  indicators, alert markers). In v1, selecting Operate mode shows an
  "operational overlays coming soon" notice.## “Invented but in-scope” enhancements (Phase 12)

These are intentionally scoped so they don’t require new architecture.

- **Saved views** for `/devices` (store filter/sort presets in the URL and optionally in local storage).
- **Triage mode** toggle (compact table, quick open).
- **Explain-disabled** affordances (when a role blocks an action, show a tooltip explaining why).
- **Command palette** (optional; later Phase 12) for “Go to device”, “Run discovery”, “Export”.
- **Bulk metadata edits** (admin-only) as a follow-on once list paging is stable.

## Definition of done for Phase 12 UI

- The UI feels consistent across routes.
- Operators can complete the core workflows without confusion.
- Empty/loading/error states are present everywhere.
- Read-only role is obvious, safe, and frustration-minimizing.
