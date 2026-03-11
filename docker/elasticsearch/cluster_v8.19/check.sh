#!/usr/bin/env bash
set -euo pipefail
[ -f ./.env ] && set -a && . ./.env && set +a
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-es-[123]/'

echo "[2/4] Cluster health"
HEALTH="$(curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cluster/health")"
echo "$HEALTH"
NODES="$(echo "$HEALTH" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("number_of_nodes",0))')"
[ "$NODES" -ge 3 ] || { echo "nodes<3: $NODES" >&2; exit 1; }

echo "[3/4] Node list"
curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cat/nodes?v"

echo "[4/4] Indexing smoke"
IDX="zygarde-cluster-smoke"
curl -fsS -X PUT "http://127.0.0.1:${ES2_HTTP_PORT:-9201}/${IDX}" >/dev/null
curl -fsS -X POST "http://127.0.0.1:${ES3_HTTP_PORT:-9202}/${IDX}/_doc/1" -H 'Content-Type: application/json' -d '{"msg":"ok"}' >/dev/null
curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/${IDX}/_doc/1" | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get("_source",{}).get("msg"); assert v=="ok", v; print("doc=ok")'
