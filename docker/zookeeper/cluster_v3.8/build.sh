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

echo "[1/2] Starting ZooKeeper cluster..."
mkdir -p ./data/zk1 ./data/zk2 ./data/zk3 ./datalog/zk1 ./datalog/zk2 ./datalog/zk3
rm -rf ./data/zk1/* ./data/zk2/* ./data/zk3/* ./datalog/zk1/* ./datalog/zk2/* ./datalog/zk3/*
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster majority healthy..."
for _ in $(seq 1 120); do
  ok=0
  for n in 1 2 3; do
    if echo ruok | "${ENGINE_CMD[@]}" exec -i "zygarde-zk-$n" /bin/bash -lc 'cat | nc 127.0.0.1 2181' 2>/dev/null | grep -q imok; then
      ok=$((ok+1))
    fi
  done
  if [ "$ok" -ge 3 ]; then
    echo "ZooKeeper cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "ZooKeeper cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
