#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"
[ -f ./.env ] && set -a && . ./.env && set +a

if command -v podman >/dev/null 2>&1; then
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting Elasticsearch single..."
mkdir -p ./data/es
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting Elasticsearch API ready..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/_cluster/health" >/dev/null 2>&1; then
    echo "Elasticsearch single is ready."
    exit 0
  fi
  sleep 2
done

echo "Elasticsearch single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs es || true
exit 1
