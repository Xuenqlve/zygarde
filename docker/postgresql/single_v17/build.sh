#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
  ENGINE_CMD=(podman)
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  ENGINE_CMD=(docker)
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting PostgreSQL single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-postgres-single..."
for _ in $(seq 1 40); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-postgres-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "PostgreSQL is healthy."
    "${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -c 'select 1;' >/dev/null || true
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-postgres-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs postgres || true
exit 1
