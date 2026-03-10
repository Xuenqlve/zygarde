#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then
  ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
  ENGINE_CMD=(docker)
else
  echo "No container engine found." >&2
  exit 1
fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mongo-rs1|zygarde-mongo-rs2|zygarde-mongo-rs3/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'

echo "[3/4] Replica-set status"
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval 'JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr})))'

echo "[4/4] Primary check"
PRIMARY="$(${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval 'rs.status().members.filter(m=>m.stateStr=="PRIMARY").length')"
if [ "$PRIMARY" -lt 1 ]; then
  echo "No PRIMARY found" >&2
  exit 1
fi
