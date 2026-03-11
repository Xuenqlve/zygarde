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

echo "[1/5] Starting TiDB cluster (3PD + 3TiKV + 2TiDB)..."
"${COMPOSE_CMD[@]}" up -d

wait_running() {
  local name="$1"
  for _ in $(seq 1 90); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "running" ]; then return 0; fi
    sleep 2
  done
  return 1
}

for node in zygarde-tidb-pd1 zygarde-tidb-pd2 zygarde-tidb-pd3 zygarde-tidb-tikv1 zygarde-tidb-tikv2 zygarde-tidb-tikv3 zygarde-tidb1 zygarde-tidb2; do
  echo "[2/5] Waiting $node running..."
  wait_running "$node" || { "${COMPOSE_CMD[@]}" logs; exit 1; }
done

echo "[3/5] Waiting tidb1 status endpoint..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${TIDB1_STATUS_PORT:-10080}/status" >/dev/null 2>&1; then break; fi
  sleep 2
done

echo "[4/5] Waiting tidb2 status endpoint..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${TIDB2_STATUS_PORT:-10081}/status" >/dev/null 2>&1; then break; fi
  sleep 2
done

echo "[5/5] Waiting PD cluster member count == 3..."
ok=0
for _ in $(seq 1 120); do
  cnt="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/members" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d.get("members",[])))' 2>/dev/null || echo 0)"
  if [ "${cnt:-0}" -ge 3 ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "PD cluster members not ready" >&2; "${COMPOSE_CMD[@]}" logs pd1 pd2 pd3 || true; exit 1; }

echo "TiDB cluster is ready."
