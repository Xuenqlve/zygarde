#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-consul-single/'

echo "[2/4] Leader"
LEADER=""
for _ in $(seq 1 30); do
  LEADER="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/status/leader" 2>/dev/null | tr -d '"' || true)"
  if [ -n "$LEADER" ]; then break; fi
  sleep 1
done
[ -n "$LEADER" ] || { echo "leader is empty" >&2; exit 1; }
echo "leader=$LEADER"

echo "[3/4] Members"
MEMBERS_JSON="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/agent/members")"
echo "$MEMBERS_JSON" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("member_count=",len(d));'

echo "[4/4] KV smoke"
KEY="zygarde/smoke/$(date +%s)"
VAL="ok-$(date +%s)"
curl -fsS -X PUT --data "$VAL" "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/kv/$KEY" >/dev/null
OUT="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/kv/$KEY?raw")"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
