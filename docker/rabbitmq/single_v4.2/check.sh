#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-rabbitmq-single/'

echo "[2/3] RabbitMQ diagnostics"
"${ENGINE_CMD[@]}" exec zygarde-rabbitmq-single rabbitmq-diagnostics -q ping

echo "[3/3] Overview"
"${ENGINE_CMD[@]}" exec zygarde-rabbitmq-single rabbitmqctl status | grep -E "RabbitMQ version|Cluster name|Uptime" || true
