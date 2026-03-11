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

echo "[1/2] Starting Consul cluster..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster (leader + members=3)..."
for _ in $(seq 1 120); do
  leader="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/status/leader" 2>/dev/null | tr -d '"' || true)"
  members="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/agent/members" 2>/dev/null | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))' 2>/dev/null || echo 0)"
  if [ -n "$leader" ] && [ "${members:-0}" -ge 3 ]; then
    echo "Consul cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "Consul cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
