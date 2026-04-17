#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd "$SCRIPT_DIR/.." && pwd)
URL="http://localhost/"
USE_DEV_PROFILE=0
SHOW_LOGS=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dev)
      USE_DEV_PROFILE=1
      ;;
    --logs)
      SHOW_LOGS=1
      ;;
    *)
      echo "Unknown option '$1'. Supported options: --dev, --logs" >&2
      exit 2
      ;;
  esac
  shift
done

cd "$REPO_ROOT"

echo "Starting Roller_hoops with Docker Compose..."
if [ "$USE_DEV_PROFILE" -eq 1 ]; then
  docker compose --profile dev up --build -d
else
  docker compose up --build -d
fi

echo "Waiting for the web UI health check..."
attempt=0
while [ "$attempt" -lt 60 ]; do
  if command -v curl >/dev/null 2>&1; then
    if curl --fail --silent --show-error --max-time 3 "${URL}healthz" >/dev/null; then
      break
    fi
  else
    echo "Install curl so this script can wait for the UI health check." >&2
    exit 1
  fi

  attempt=$((attempt + 1))
  sleep 3
done

if [ "$attempt" -ge 60 ]; then
  echo "The stack started, but the UI did not become healthy within 3 minutes." >&2
  echo "Open $URL manually or run: docker compose logs -f --tail=200" >&2
  exit 1
fi

echo "Opening $URL"
open "$URL"

if [ "$SHOW_LOGS" -eq 1 ]; then
  docker compose logs -f --tail=200
fi
