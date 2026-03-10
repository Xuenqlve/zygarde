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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-master|zygarde-redis-slave/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli ping
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli ping

echo "[3/4] Master role"
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli info replication | grep -E '^role:|connected_slaves:'

echo "[4/4] Slave role"
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli info replication | grep -E '^role:|master_host:|master_link_status:'
