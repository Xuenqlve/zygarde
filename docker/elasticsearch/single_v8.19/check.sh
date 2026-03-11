#!/usr/bin/env bash
set -euo pipefail
[ -f ./.env ] && set -a && . ./.env && set +a
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-es-single/'

echo "[2/4] Cluster health"
HEALTH="$(curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/_cluster/health")"
echo "$HEALTH" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("status=",d.get("status"),"nodes=",d.get("number_of_nodes"));'

echo "[3/4] Version"
curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("version",{}).get("number","unknown"))'

echo "[4/4] Indexing smoke"
IDX="zygarde-smoke"
curl -fsS -X PUT "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}" >/dev/null
curl -fsS -X POST "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}/_doc/1" -H 'Content-Type: application/json' -d '{"msg":"ok"}' >/dev/null
curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}/_doc/1" | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get("_source",{}).get("msg"); assert v=="ok", v; print("doc=ok")'
