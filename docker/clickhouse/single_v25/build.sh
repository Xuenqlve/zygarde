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

echo "[1/2] Starting ClickHouse single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting clickhouse ready..."
for _ in $(seq 1 120); do
  if "${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT 1" >/dev/null 2>&1; then
    echo "ClickHouse single is ready."
    exit 0
  fi
  sleep 2
done

echo "ClickHouse single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs clickhouse || true
exit 1
