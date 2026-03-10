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

echo "[1/4] Starting Redis cluster nodes..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/4] Waiting for nodes..."
sleep 8

echo "[3/4] Creating cluster (3 masters, no replicas)..."
IP1="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-1)"
IP2="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-2)"
IP3="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-3)"

"${ENGINE_CMD[@]}" exec -i zygarde-redis-node-1 redis-cli --cluster create \
    "${IP1}:7001" "${IP2}:7002" "${IP3}:7003" \
    --cluster-replicas 0 --cluster-yes

echo "[4/4] Cluster info"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster info | grep cluster_state || true
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster nodes || true
