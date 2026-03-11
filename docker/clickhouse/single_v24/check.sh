#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-clickhouse-single/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT 1"

echo "[3/4] Version"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT version()"

echo "[4/4] Create/Insert/Select smoke"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "CREATE TABLE IF NOT EXISTS zygarde_smoke (id UInt32, v String) ENGINE=MergeTree ORDER BY id"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "INSERT INTO zygarde_smoke VALUES (1, 'ok')"
OUT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-single clickhouse-client -q "SELECT v FROM zygarde_smoke WHERE id=1 FORMAT TSVRaw" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "smoke failed: $OUT" >&2; exit 1; }
