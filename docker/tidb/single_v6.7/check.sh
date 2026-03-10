#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-tidb-pd-single|zygarde-tidb-tikv-single|zygarde-tidb-single/'

echo "[2/4] TiDB status"
curl -fsS "http://127.0.0.1:${TIDB_STATUS_PORT:-10080}/status"
echo ""

echo "[3/4] PD health"
curl -fsS "http://127.0.0.1:${PD_PORT:-2379}/pd/api/v1/health"
echo ""

echo "[4/4] TiDB SQL port open"
if (exec 3<>/dev/tcp/127.0.0.1/${TIDB_PORT:-4000}) 2>/dev/null; then
  echo "tidb sql port ${TIDB_PORT:-4000} is reachable"
  exec 3>&-
else
  echo "tidb sql port ${TIDB_PORT:-4000} is not reachable" >&2
  exit 1
fi
