# Issues tracker

This file is the project’s **lightweight issue log**. It’s intentionally Markdown-only so it stays diff-friendly.

## How to use

- Add new items at the bottom of the **Index** table with a new stable ID: `ISS-###`.
- Keep titles short and searchable.
- Prefer objective notes: repro steps, logs, environment, and the next concrete action.

### Status values

- `open` — acknowledged, not fixed
- `investigating` — actively being debugged
- `blocked` — needs an external dependency/decision
- `fixed` — resolved (include the fix reference)
- `won't-fix` — explicitly declined (include rationale)

## Index

| ID | Title | Status | Area | Severity | Last updated |
|---|---|---|---|---|---|
| ISS-001 | Network discovery doesn’t scan outside Docker | open | core-go/discovery | high | 2025-12-15 |
| ISS-002 | Map view UX clutter; needs auto-layout | fixed | ui-node/map | medium | 2026-01-02 |
| ISS-003 | Discovery UX needs presets + automation | fixed | ui-node/discovery | high | 2026-01-02 |
| ISS-004 | Device naming is low-quality/inconsistent | open | core-go/enrichment | medium | 2026-01-02 |
| ISS-005 | Devices should be auto-classified + tagged | open | core-go/enrichment | medium | 2026-01-02 |
| ISS-006 | Devices page layout wastes space; lacks detail | open | ui-node/devices | medium | 2026-01-02 |

---

## ISS-001 — Network discovery doesn’t scan outside Docker

### Summary

Discovery works when running in Docker, but does not successfully scan host networks when `core-go` runs directly on the host.

### Impact

- Prevents local/non-container deployments from discovering devices.
- Makes development/debugging outside Docker unreliable.

### Environment

- Host OS: Linux
- Works when: `docker compose up --build` (discovery executed inside container)
- Fails when: running `core-go` directly on the host (non-Docker)

### Reproduction

1. Run `core-go` directly on the host.
2. Trigger a discovery run.
3. Observe that no/insufficient results are produced.

### Expected

Discovery should perform the same scans and produce comparable results whether running in Docker or on the host.

### Actual

Scanning does not run or does not return results when outside Docker.

### Notes / signals to capture

Add links/snippets here when available:

- `core-go` logs during a failed scan
- Relevant host capabilities/permissions (e.g., raw socket / ICMP / nmap execution)
- Effective config (env vars) for both Docker and host runs
- Host networking context (interfaces, routes, DNS)

### Hypotheses (to confirm)

One or more of the following is true:

- Host-run config differs (subnets, interface allowlist/denylist, scan method flags).
- Scan tooling is missing when running outside Docker (e.g., `nmap` not installed on host).
- Privileges differ (raw socket / ICMP capabilities; `CAP_NET_RAW` inside container vs host permissions).
- Docker mode is only scanning the internal Docker network (and “looks like it works” because containers are found).

### Next actions

- [ ] Document the exact host-run command and env vars used.
- [ ] Capture the first error line from logs for a failed run and paste it here.
- [ ] Compare Docker vs host: discovered interfaces, routes, and scan tool availability.
- [ ] Confirm whether discovery is configured to scan “all local interfaces” vs an explicit CIDR list.

### Fix reference

- _Not fixed yet._


## ISS-002 — Map view UX clutter; needs auto-layout

### Summary

The network “map” view is hard to use: too many controls for the current scope, and there’s no automatic layout/arrange to produce a readable map.

### Impact

- Increases time-to-value for new users (map looks broken or overwhelming).
- Makes it hard to validate discovery results visually.

### Observed problems

- Too many buttons/controls (some feel out of scope for MVP).
- No “auto arrange / auto layout” to make the graph readable quickly.

### Next actions

- [x] Decide the MVP control set (keep/remove candidates).
- [x] Add an “Auto layout” action (and/or auto-run on load with a safe debounce).
- [ ] Capture a screenshot + list of controls that should be removed/hidden behind “Advanced”.

### Fix reference

- `ui-node/app/(app)/map/page.tsx` removes mode bar (less clutter).
- `ui-node/app/(app)/map/MapPollingControls.tsx` adds `Auto layout` + simplifies live update controls.
- `ui-node/app/(app)/map/MapCanvas.tsx` sorts and auto-arranges on load/update (debounced).

---

## ISS-003 — Discovery UX needs presets + automation

### Summary

Discovery is not working well in practice: it needs more “auto-search” behaviors and operator-friendly presets so scans are effective without lots of manual tuning.

### Impact

- Users run scans that miss devices, then don’t know how to improve coverage.
- Too many knobs without guidance increases misconfiguration.

### Desired behaviors

- Provide scan “presets” (fast/normal/deep) and sensible defaults.
- Support selectable “nmap tags” (named groups of options) to tune scanning without exposing raw flags.
- Improve automation (auto-discover likely subnets/interfaces, safer retries, clearer progress and results).

### Next actions

- [x] Define initial presets and map each to concrete scan settings (UI preset + `preset` request field; stored in run stats).
- [x] Decide where “tags” live (config file vs DB vs UI-only constants).
- [x] Add basic guidance text in the UI for what each preset/tag changes.

### Fix reference

- `ui-node/app/(app)/devices/DiscoveryPanel.tsx` adds scan tags + scope suggestions + clearer guidance.
- `ui-node/app/(app)/discovery/tags.ts` defines initial tag set + formatting.
- `core-go/internal/httpapi/handler.go` accepts `tags` and serves `/api/v1/discovery/scope-suggestions`.
- `core-go/internal/discoveryworker/tags.go` applies tags to per-run behavior.

---

## ISS-004 — Device naming is low-quality/inconsistent

### Summary

Automatically generated device names are often unhelpful or inconsistent, making the device list harder to understand and search.

### Impact

- Reduces the usefulness of discovery output (hard to tell what is what).
- Increases manual renaming workload.

### Notes / ideas

- Prefer stable, high-signal sources in order (e.g., DHCP hostname → reverse DNS → mDNS/NetBIOS → SNMP sysName → vendor+MAC fallback).
- Track “name candidates” with a confidence score and source.

### Next actions

- [ ] Document current naming behavior and where it’s computed.
- [ ] Add candidate ordering rules + a “best name” selection strategy.

### Fix reference

- _Not fixed yet._

---

## ISS-005 — Devices should be auto-classified + tagged

### Summary

Discovered devices should be auto-tagged/classified on a best-guess basis (router, switch, AP, PC, printer, VM, etc.) to make the UI more useful immediately.

### Impact

- Improves usability of the device list, filters, and map without manual tagging.
- Enables better defaults (icon, grouping, and map styling).

### Signals to use (examples)

- OUIs/vendor, open ports, SNMP `sysDescr`, reverse DNS patterns, gateway roles, and DHCP metadata.

### Next actions

- [ ] Define an initial taxonomy (small, practical set).
- [ ] Implement “best guess” tagging with transparent signals (explainable heuristics).
- [ ] Add a manual override flow in the UI (user-applied tags should win).

### Fix reference

- _Not fixed yet._

---

## ISS-006 — Devices page layout wastes space; lacks detail

### Summary

The Devices page has poor information density: most of the page is unused space, and key device details aren’t surfaced.

### Impact

- Slower operations (too much clicking; too little at-a-glance context).
- Devices feel “empty” even when discovery gathered facts.

### Desired improvements

- Better layout (table density, responsive columns, less whitespace).
- Surface key details: primary IP/MAC, vendor, last seen, tags/type, open ports (if available), interfaces/VLAN (if available), notes/owner/location.

### Next actions

- [ ] Capture screenshots and list the top 10 fields operators want visible.
- [ ] Define the “compact” table view (defaults + optional columns).

### Fix reference

- _Not fixed yet._

---

## Gaps / openings to exploit

These are “cheap wins” or missing definitions that can unblock multiple issues at once.

- Define a discovery capabilities matrix (Docker vs host vs “scanner on target network”), including required privileges and tools: [docs/discovery-capabilities.md](discovery-capabilities.md).
- Create a small, canonical preset list for scanning (fast/normal/deep) and translate it to both UI wording and `core-go` config.
- Define a minimal device taxonomy + icon set that the UI can reuse across list/map/detail views.
- Add a “definition of done” checklist for UX issues (screenshot before/after, acceptance criteria, and performance notes).
