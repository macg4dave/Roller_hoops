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

---

## ISS-001 — Network discovery doesn’t scan outside Docker

### Summary

Discovery works when running in Docker, but does not successfully scan the host Networks.

### Impact

- Prevents local/non-container deployments from discovering devices.
- Makes development/debugging outside Docker unreliable.

### Environment

- Host OS: Linux
- When it works: 
- When it fails: `docker compose up --build`

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

### Hypotheses (to confirm)

only scanning inside Docker works because: its scanning the interal Docker network; nmap/ping is only available in the container.

### Next actions

- [ ] Document the exact host-run command and env vars used.
- [ ] Capture the first error line from logs for a failed run and paste it here.
- [ ] Compare Docker vs host: discovered interfaces, routes, and scan tool availability.

### Fix reference

- _Not fixed yet._