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

echo "[1/2] Starting Elasticsearch cluster..."
mkdir -p ./data/es1 ./data/es2 ./data/es3
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster ready (nodes>=3)..."
for _ in $(seq 1 180); do
  NODES="$(curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cluster/health" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("number_of_nodes",0))' 2>/dev/null || echo 0)"
  if [ "${NODES:-0}" -ge 3 ]; then
    echo "Elasticsearch cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "Elasticsearch cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
