#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-etcd-single/'

echo "[2/4] Endpoint health"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 endpoint health

echo "[3/4] Member list"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 member list

echo "[4/4] KV smoke"
KEY="zygarde-smoke-$(date +%s)"
VAL="ok-$(date +%s)"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 put "$KEY" "$VAL" >/dev/null
OUT="$(${ENGINE_CMD[@]} exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 get "$KEY" --print-value-only | tr -d '\r')"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
