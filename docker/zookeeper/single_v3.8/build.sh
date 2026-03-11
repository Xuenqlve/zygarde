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

echo "[1/2] Starting ZooKeeper single..."
mkdir -p ./data/zk ./datalog/zk
rm -rf ./data/zk/* ./datalog/zk/*
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting zookeeper ruok=imok..."
for _ in $(seq 1 90); do
  if echo ruok | "${ENGINE_CMD[@]}" exec -i zygarde-zk-single /bin/bash -lc 'cat | nc 127.0.0.1 2181' 2>/dev/null | grep -q imok; then
    echo "ZooKeeper single is ready."
    exit 0
  fi
  sleep 2
done

echo "ZooKeeper single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs zk || true
exit 1
