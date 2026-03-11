#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-zk-single/'

echo "[2/4] ruok"
R="$({ echo ruok | "${ENGINE_CMD[@]}" exec -i zygarde-zk-single /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\r')"
[ "$R" = "imok" ] || { echo "ruok failed: $R" >&2; exit 1; }

echo "[3/4] mntr"
"${ENGINE_CMD[@]}" exec zygarde-zk-single /bin/bash -lc "echo mntr | nc 127.0.0.1 2181 | grep -E 'zk_server_state|zk_version'"

echo "[4/4] create/get znode smoke"
"${ENGINE_CMD[@]}" exec zygarde-zk-single /bin/bash -lc "zkCli.sh -server 127.0.0.1:2181 create /zygarde_smoke ok >/tmp/zk.out 2>&1 || true"
OUT="$(${ENGINE_CMD[@]} exec zygarde-zk-single /bin/bash -lc "zkCli.sh -server 127.0.0.1:2181 get /zygarde_smoke 2>/dev/null | grep -E '^ok$' | head -n1" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "znode smoke failed: $OUT" >&2; exit 1; }
