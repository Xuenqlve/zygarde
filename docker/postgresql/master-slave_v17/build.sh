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

echo "[1/3] Starting PostgreSQL master/slave..."
"${COMPOSE_CMD[@]}" up -d

wait_healthy() {
  local name="$1"
  for _ in $(seq 1 60); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[2/3] Waiting master healthy..."
wait_healthy zygarde-postgres-master || { "${COMPOSE_CMD[@]}" logs postgres-master; exit 1; }

echo "[3/3] Waiting slave healthy..."
wait_healthy zygarde-postgres-slave || { "${COMPOSE_CMD[@]}" logs postgres-slave; exit 1; }

echo "PostgreSQL master/slave is healthy."
