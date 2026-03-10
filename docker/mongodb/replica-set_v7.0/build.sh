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

echo "[1/4] Starting MongoDB replica-set nodes..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/4] Waiting for nodes ready..."
for c in zygarde-mongo-rs1 zygarde-mongo-rs2 zygarde-mongo-rs3; do
  ok=0
  for _ in $(seq 1 30); do
    if "${ENGINE_CMD[@]}" exec "$c" mongosh --quiet --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ] || { echo "$c not ready" >&2; exit 1; }
done

echo "[3/4] Initiating replica-set rs0..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval '
try {
  rs.initiate({_id:"rs0", members:[
    {_id:0, host:"mongo-rs1:27017"},
    {_id:1, host:"mongo-rs2:27017"},
    {_id:2, host:"mongo-rs3:27017"}
  ]})
} catch(e) {
  if (!e.message.includes("already initialized")) throw e;
}
'

echo "[4/4] Waiting for replica-set stable (PRIMARY+SECONDARY)..."
RS_PRIMARY_WAIT_SECONDS="${RS_PRIMARY_WAIT_SECONDS:-120}"
ATTEMPTS=$((RS_PRIMARY_WAIT_SECONDS / 2))
[ "$ATTEMPTS" -lt 1 ] && ATTEMPTS=1

check_rs_stable() {
  local line
  line="$(${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval '
try {
  var s=rs.status();
  var p=s.members.filter(m=>m.stateStr=="PRIMARY").length;
  var sec=s.members.filter(m=>m.stateStr=="SECONDARY").length;
  print(p+","+sec);
} catch(e) { print("0,0"); }
' 2>/dev/null | tail -n 1 || true)"

  local p sec
  p="${line%%,*}"
  sec="${line##*,}"
  [ -z "$p" ] && p=0
  [ -z "$sec" ] && sec=0

  if [ "$p" -ge 1 ] && [ "$sec" -ge 2 ]; then
    return 0
  fi
  return 1
}

ok=0
for _ in $(seq 1 "$ATTEMPTS"); do
  if check_rs_stable; then
    ok=1
    break
  fi
  sleep 2
done

# 兜底再给一轮等待（吸收偶发选主抖动）
if [ "$ok" -ne 1 ]; then
  for _ in $(seq 1 15); do
    if check_rs_stable; then
      ok=1
      break
    fi
    sleep 2
  done
fi

if [ "$ok" -ne 1 ]; then
  echo "Replica-set not stable (PRIMARY/SECONDARY)" >&2
  ${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval 'try{print(JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr,health:m.health}))))}catch(e){print(e.message)}' >&2 || true
  exit 1
fi

echo "Replica-set PRIMARY/SECONDARY is stable."
