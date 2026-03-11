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

echo "[1/2] Starting Consul single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting consul API ready..."
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/status/leader" >/dev/null 2>&1; then
    echo "Consul single is ready."
    exit 0
  fi
  sleep 2
done

echo "Consul single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs consul || true
exit 1
