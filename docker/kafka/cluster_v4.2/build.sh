#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
  ENGINE_CMD=(podman)
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  ENGINE_CMD=(docker)
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting Kafka KRaft cluster..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster readiness..."
for _ in $(seq 1 120); do
  if "${ENGINE_CMD[@]}" exec zygarde-kafka-1 bash -lc '/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka1:9092 >/dev/null 2>&1' \
    && "${ENGINE_CMD[@]}" exec zygarde-kafka-2 bash -lc '/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka2:9092 >/dev/null 2>&1' \
    && "${ENGINE_CMD[@]}" exec zygarde-kafka-3 bash -lc '/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka3:9092 >/dev/null 2>&1'; then
    echo "Kafka cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "Kafka cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
