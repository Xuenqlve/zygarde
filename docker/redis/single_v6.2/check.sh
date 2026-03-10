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

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-redis-single redis-cli ping

echo "[3/3] Role"
"${ENGINE_CMD[@]}" exec zygarde-redis-single redis-cli info replication | grep '^role:'
