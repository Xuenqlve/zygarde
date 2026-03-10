#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(podman-compose)
    fi
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(docker-compose)
    fi
else
    echo "No container engine found." >&2
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "No compose command found for current container engine." >&2
    exit 1
fi

echo "[1/3] Starting Redis master/slave..."
"${COMPOSE_CMD[@]}" up -d

wait_healthy() {
    local name="$1"
    for _ in $(seq 1 30); do
        status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        sleep 2
    done
    return 1
}

echo "[2/3] Waiting for master healthy..."
wait_healthy zygarde-redis-master || { "${COMPOSE_CMD[@]}" logs redis-master; exit 1; }

echo "[3/3] Waiting for slave healthy..."
wait_healthy zygarde-redis-slave || { "${COMPOSE_CMD[@]}" logs redis-slave; exit 1; }

echo "Redis master/slave is healthy."
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli info replication | grep '^role:' || true
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli info replication | grep '^role:' || true
