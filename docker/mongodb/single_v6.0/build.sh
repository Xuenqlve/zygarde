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

echo "[1/2] Starting MongoDB single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-mongodb-single..."
for _ in $(seq 1 30); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-mongodb-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "MongoDB is healthy."
    "${ENGINE_CMD[@]}" exec zygarde-mongodb-single mongosh --quiet --eval 'db.adminCommand({ ping: 1 })' || true
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-mongodb-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs mongodb || true
exit 1
