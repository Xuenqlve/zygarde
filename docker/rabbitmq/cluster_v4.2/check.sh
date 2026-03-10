#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-rabbitmq-[123]/'

echo "[2/3] Cluster convergence check"
ok=0
for _ in $(seq 1 60); do
  STATUS="$(${ENGINE_CMD[@]} exec zygarde-rabbitmq-1 rabbitmqctl cluster_status --formatter json 2>/dev/null || true)"
  if echo "$STATUS" | grep -q 'rabbit@rabbit1' \
    && echo "$STATUS" | grep -q 'rabbit@rabbit2' \
    && echo "$STATUS" | grep -q 'rabbit@rabbit3'; then
    ok=1
    echo "$STATUS"
    break
  fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "cluster did not converge to 3 nodes" >&2; exit 1; }

echo "[3/3] Diagnostics"
"${ENGINE_CMD[@]}" exec zygarde-rabbitmq-1 rabbitmq-diagnostics -q cluster_status
