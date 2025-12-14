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
