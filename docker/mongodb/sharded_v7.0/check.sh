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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mongo-cfg|zygarde-mongo-shard|zygarde-mongos/'

echo "[2/4] mongos ping"
"${ENGINE_CMD[@]}" exec zygarde-mongos mongosh --quiet --port 27017 --eval 'db.adminCommand({ ping: 1 })'

echo "[3/4] listShards"
SHARDS="$(${ENGINE_CMD[@]} exec zygarde-mongos mongosh --quiet --port 27017 --eval 'JSON.stringify(db.adminCommand({ listShards: 1 }))')"
echo "$SHARDS"
echo "$SHARDS" | grep -q '"ok":1' || { echo "listShards not ok" >&2; exit 1; }

echo "[4/4] shard count"
COUNT="$(${ENGINE_CMD[@]} exec zygarde-mongos mongosh --quiet --port 27017 --eval 'db.adminCommand({ listShards: 1 }).shards.length')"
if [ "$COUNT" -lt 1 ]; then
  echo "No shard found" >&2
  exit 1
fi
