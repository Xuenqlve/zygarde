#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-consul-[123]/'

echo "[2/4] Leader + members"
LEADER="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/status/leader" | tr -d '"')"
[ -n "$LEADER" ] || { echo "leader is empty" >&2; exit 1; }
MEMBERS="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/agent/members" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[ "$MEMBERS" -ge 3 ] || { echo "members < 3" >&2; exit 1; }
echo "leader=$LEADER members=$MEMBERS"

echo "[3/4] Raft peers"
RAFT="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/operator/raft/configuration")"
echo "$RAFT" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("raft_servers=",len(d.get("Servers",[])));'

echo "[4/4] KV smoke"
KEY="zygarde/cluster/smoke/$(date +%s)"
VAL="ok-$(date +%s)"
curl -fsS -X PUT --data "$VAL" "http://127.0.0.1:${CONSUL2_HTTP_PORT:-9500}/v1/kv/$KEY" >/dev/null
OUT="$(curl -fsS "http://127.0.0.1:${CONSUL3_HTTP_PORT:-10500}/v1/kv/$KEY?raw")"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
