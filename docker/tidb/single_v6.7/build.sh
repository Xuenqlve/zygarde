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

echo "[1/4] Starting TiDB single (pd+tikv+tidb)..."
"${COMPOSE_CMD[@]}" up -d

wait_running() {
  local name="$1"
  for _ in $(seq 1 60); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "running" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[2/4] Waiting pd running..."
wait_running zygarde-tidb-pd-single || { "${COMPOSE_CMD[@]}" logs pd; exit 1; }

echo "[3/4] Waiting tikv running..."
wait_running zygarde-tidb-tikv-single || { "${COMPOSE_CMD[@]}" logs tikv; exit 1; }

echo "[4/4] Waiting tidb status endpoint..."
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${TIDB_STATUS_PORT:-10080}/status" >/dev/null 2>&1; then
    echo "TiDB status endpoint is ready."
    exit 0
  fi
  sleep 2
done

echo "TiDB status endpoint not ready in time" >&2
"${COMPOSE_CMD[@]}" logs tidb || true
exit 1
