#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

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

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

echo "[1/2] Starting MySQL single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-mysql-single..."
for _ in $(seq 1 30); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-mysql-single 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then
        echo "MySQL is healthy."
        "${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT VERSION();" || true
        exit 0
    fi
    sleep 2
done

echo "Container zygarde-mysql-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs mysql || true
exit 1
