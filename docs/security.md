# Security

## Golden rule

**No secrets in this repository.**

This includes (non-exhaustive):

- passwords, API keys, tokens
- private keys / SSH keys
- Wi‑Fi PSKs
- exported device configs containing secrets


## Recommended protections

- Keep this repo private if it contains internal hostnames or private IP addressing.
- Enable GitHub secret scanning (if hosted on GitHub).
- Consider adding a pre-commit hook / CI step for secret scanning (e.g., gitleaks).

## Local development secrets

Docker Compose loads environment variables from a local `.env` file by default.

- `.env` is **gitignored** in this repo.
- Use it for local-only values like `POSTGRES_PASSWORD`.
- Do not copy production secrets into `.env`.

## Production secret injection

For production deployments:

- Inject secrets via your platform’s secret manager (preferred) or a secure CI/CD mechanism.
- Avoid printing secrets in logs.
- Rotate secrets periodically (and always after an incident).


## UI authentication variables

- `AUTH_SESSION_SECRET` (required in production) is used to HMAC the session token stored in the `roller_session` cookie. Rotate this secret if it leaks to invalidate sessions.
- `AUTH_USERS` is a comma-separated list of `username:password:role` entries that the Next.js UI accepts for login. Example: `admin:admin:admin,viewer:readonly:read-only`, which ships as the default quickstart configuration so the first-time login is `admin` / `admin`. The first entry acts as a fallback for legacy tooling that assumes a single admin user.
- `read-only` users can still view devices, discovery status, and export snapshots, but the UI disables mutating controls (device creation, metadata updates, display name choices, discovery triggers, imports) and the `/api/[...path]` proxy rejects `POST/PUT/PATCH/DELETE` calls issued with a `read-only` session role.

Example `.env` snippet:

```env
AUTH_USERS=admin:admin:admin,viewer:readonly:read-only
AUTH_SESSION_SECRET=some-production-secret
```
