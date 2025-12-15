#!/usr/bin/env sh
set -euo pipefail

UI_URL="${UI_URL:-http://localhost/healthz}"
CORE_READY_URL="${CORE_READY_URL:-http://localhost:8081/readyz}"

curl -fsS "$UI_URL" >/dev/null
curl -fsS "$CORE_READY_URL" >/dev/null
echo "ok"

