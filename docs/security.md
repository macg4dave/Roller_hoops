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
