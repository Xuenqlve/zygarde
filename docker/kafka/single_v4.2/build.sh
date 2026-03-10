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

echo "[1/2] Starting Kafka single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-kafka-single..."
for _ in $(seq 1 90); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-kafka-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "Kafka is healthy."
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-kafka-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs kafka || true
exit 1
