#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/6] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-tidb-pd[123]|zygarde-tidb-tikv[123]|zygarde-tidb[12]$/'

echo "[2/6] TiDB status endpoints"
curl -fsS "http://127.0.0.1:${TIDB1_STATUS_PORT:-10080}/status"
echo ""
curl -fsS "http://127.0.0.1:${TIDB2_STATUS_PORT:-10081}/status"
echo ""

echo "[3/6] PD health(3 members)"
health_json="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/health")"
echo "$health_json"
python3 - <<'PY' "$health_json"
import json,sys
h=json.loads(sys.argv[1])
if len(h)<3 or not all(x.get('health') for x in h):
  raise SystemExit('PD health check failed')
print('pd_health_ok=true')
PY

echo "[4/6] PD members + leader"
members_json="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/members")"
echo "$members_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("members=",len(d.get("members",[])),"leader=",(d.get("leader") or {}).get("name"));'

echo "[5/6] TiKV stores"
stores="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/stores" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("count",0))')"
[ "${stores:-0}" -ge 3 ] || { echo "TiKV store count < 3" >&2; exit 1; }
echo "store_count=${stores}"

echo "[6/6] TiDB SQL ports open"
for p in "${TIDB1_PORT:-4000}" "${TIDB2_PORT:-4001}"; do
  if (exec 3<>/dev/tcp/127.0.0.1/$p) 2>/dev/null; then
    echo "tidb sql port $p is reachable"
    exec 3>&-
  else
    echo "tidb sql port $p is not reachable" >&2
    exit 1
  fi
done
