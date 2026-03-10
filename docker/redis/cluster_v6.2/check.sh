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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-node-1|zygarde-redis-node-2|zygarde-redis-node-3/'

echo "[2/4] PING all nodes"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 ping
"${ENGINE_CMD[@]}" exec zygarde-redis-node-2 redis-cli -p 7002 ping
"${ENGINE_CMD[@]}" exec zygarde-redis-node-3 redis-cli -p 7003 ping

echo "[3/4] Cluster state"

# 强校验：cluster_state 必须为 ok（允许短暂收敛重试）
OK=0
FINAL_INFO=""
for _ in $(seq 1 10); do
  FINAL_INFO="$(${ENGINE_CMD[@]} exec zygarde-redis-node-1 redis-cli -p 7001 cluster info)"
  CSTATE="$(echo "$FINAL_INFO" | grep '^cluster_state:' | awk -F: '{print $2}' | tr -d '\r')"
  if [ "$CSTATE" = "ok" ]; then
    OK=1
    break
  fi
  sleep 2
done

# 打印最终状态（避免打印旧状态造成误导）
echo "$FINAL_INFO" | grep -E 'cluster_state|cluster_known_nodes|cluster_size'

if [ "$OK" -ne 1 ]; then
  echo "[FAIL] cluster_state 非 ok" >&2
  echo "$FINAL_INFO" >&2
  exit 1
fi

echo "[4/4] Cluster nodes"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster nodes
