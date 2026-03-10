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

wait_ping() {
  local c="$1"; local port="$2"; local retries="${3:-40}"
  local ok=0
  for _ in $(seq 1 "$retries"); do
    if "${ENGINE_CMD[@]}" exec "$c" mongosh --quiet --port "$port" --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1; break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ]
}

wait_rs_primary() {
  local c="$1"; local port="$2"; local retries="${3:-60}"
  for _ in $(seq 1 "$retries"); do
    state="$(${ENGINE_CMD[@]} exec "$c" mongosh --quiet --port "$port" --eval 'try{rs.status().myState}catch(e){0}' 2>/dev/null | tail -n 1 || true)"
    if [ "$state" = "1" ]; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "[1/6] Starting sharded topology..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/6] Waiting config/shard services ready..."
wait_ping zygarde-mongo-cfg1 27019 50 || { echo "cfg1 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-cfg2 27019 50 || { echo "cfg2 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-cfg3 27019 50 || { echo "cfg3 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-shard1 27018 50 || { echo "shard1 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-shard2 27018 50 || { echo "shard2 not ready" >&2; exit 1; }

echo "[3/6] Initiating config replica-set..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-cfg1 mongosh --quiet --port 27019 --eval '
try {
  rs.initiate({_id:"cfgRS", configsvr:true, members:[
    {_id:0, host:"cfg1:27019"},
    {_id:1, host:"cfg2:27019"},
    {_id:2, host:"cfg3:27019"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary zygarde-mongo-cfg1 27019 70 || { echo "cfgRS primary not ready" >&2; exit 1; }

echo "[4/6] Initiating shard replica-set..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-shard1 mongosh --quiet --port 27018 --eval '
try {
  rs.initiate({_id:"shardRS", members:[
    {_id:0, host:"shard1:27018"},
    {_id:1, host:"shard2:27018"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary zygarde-mongo-shard1 27018 70 || { echo "shardRS primary not ready" >&2; exit 1; }

echo "[5/6] Waiting mongos ready..."
wait_ping zygarde-mongos 27017 60 || { echo "mongos not ready" >&2; ${ENGINE_CMD[@]} logs zygarde-mongos >&2 || true; exit 1; }

echo "[6/6] Adding shard to mongos..."
"${ENGINE_CMD[@]}" exec zygarde-mongos mongosh --quiet --port 27017 --eval '
try {
  sh.addShard("shardRS/shard1:27018,shard2:27018")
} catch(e) {
  if (!(e.message.includes("already") || e.message.includes("exists"))) throw e;
}
sh.status()
'
