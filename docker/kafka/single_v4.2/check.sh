#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-kafka-single/'

echo "[2/4] Broker API versions"
"${ENGINE_CMD[@]}" exec zygarde-kafka-single bash -lc '/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 >/dev/null'

echo "[3/4] Topic produce smoke test"
TOPIC="zygarde-smoke-$(date +%s)"
MSG="hello-zygarde-$(date +%s)"
"${ENGINE_CMD[@]}" exec zygarde-kafka-single bash -lc "/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic ${TOPIC} --partitions 1 --replication-factor 1 >/dev/null"
echo "$MSG" | "${ENGINE_CMD[@]}" exec -i zygarde-kafka-single bash -lc "/opt/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic ${TOPIC} >/dev/null 2>&1"

ok=0
for _ in $(seq 1 8); do
  OFF="$(${ENGINE_CMD[@]} exec zygarde-kafka-single bash -lc "/opt/kafka/bin/kafka-get-offsets.sh --bootstrap-server localhost:9092 --topic ${TOPIC} 2>/dev/null" | awk -F: '{print $3}' | head -n1 | tr -d '[:space:]')"
  if [ -n "$OFF" ] && [ "$OFF" -ge 1 ] 2>/dev/null; then
    ok=1
    break
  fi
  sleep 1
done
[ "$ok" -eq 1 ] || { echo "smoke offset check failed: ${OFF:-empty}" >&2; exit 1; }

echo "[4/4] Metadata"
"${ENGINE_CMD[@]}" exec zygarde-kafka-single bash -lc "/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic ${TOPIC}"
