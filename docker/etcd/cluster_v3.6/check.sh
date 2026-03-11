#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

ENDPOINTS="http://etcd1:2379,http://etcd2:2379,http://etcd3:2379"

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-etcd-[123]/'

echo "[2/4] Endpoint health"
"${ENGINE_CMD[@]}" exec zygarde-etcd-1 etcdctl --endpoints="$ENDPOINTS" endpoint health

echo "[3/4] Member list"
MEMBERS="$(${ENGINE_CMD[@]} exec zygarde-etcd-1 etcdctl --endpoints="$ENDPOINTS" member list | tee /dev/stderr | wc -l | tr -d '[:space:]')"
[ "$MEMBERS" -ge 3 ] || { echo "member count < 3" >&2; exit 1; }

echo "[4/4] KV smoke"
KEY="zygarde-cluster-smoke-$(date +%s)"
VAL="ok-$(date +%s)"
"${ENGINE_CMD[@]}" exec zygarde-etcd-2 etcdctl --endpoints="$ENDPOINTS" put "$KEY" "$VAL" >/dev/null
OUT="$(${ENGINE_CMD[@]} exec zygarde-etcd-3 etcdctl --endpoints="$ENDPOINTS" get "$KEY" --print-value-only | tr -d '\r')"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
