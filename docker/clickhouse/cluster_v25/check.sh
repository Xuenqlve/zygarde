#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-clickhouse-[123]/'

echo "[2/4] Connectivity on each node"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-1 clickhouse-client -q "SELECT 1"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-2 clickhouse-client -q "SELECT 1"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-3 clickhouse-client -q "SELECT 1"

echo "[3/4] Cluster topology check"
CNT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-1 clickhouse-client -q "SELECT count() FROM system.clusters WHERE cluster='zygarde_cluster'" | tr -d '[:space:]')"
[ "${CNT:-0}" -ge 3 ] || { echo "cluster topology invalid: $CNT" >&2; exit 1; }
echo "cluster_nodes=$CNT"

echo "[4/4] Cross-node smoke via remote()"
OUT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-1 clickhouse-client -q "SELECT count() FROM remote('ch1,ch2,ch3', system.one)" | tr -d '[:space:]')"
[ "$OUT" = "3" ] || { echo "remote smoke failed: $OUT" >&2; exit 1; }
