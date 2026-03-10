#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-kafka-[123]/'

echo "[2/3] Broker metadata"
META="$(${ENGINE_CMD[@]} exec zygarde-kafka-1 bash -lc '/opt/kafka/bin/kafka-metadata-quorum.sh --bootstrap-server kafka1:9092 describe --status' 2>/dev/null || true)"
echo "$META"
echo "$META" | grep -q 'LeaderId' || { echo "metadata quorum not ready" >&2; exit 1; }

echo "[3/3] Produce/consume smoke test"
"${ENGINE_CMD[@]}" exec zygarde-kafka-1 bash -lc '/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka1:9092 --create --if-not-exists --topic zygarde-smoke --partitions 1 --replication-factor 3 >/dev/null'
echo 'hello-kafka' | "${ENGINE_CMD[@]}" exec -i zygarde-kafka-1 bash -lc '/opt/kafka/bin/kafka-console-producer.sh --bootstrap-server kafka1:9092 --topic zygarde-smoke >/dev/null 2>&1'
OUT="$(${ENGINE_CMD[@]} exec zygarde-kafka-2 bash -lc '/opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka2:9092 --topic zygarde-smoke --from-beginning --max-messages 1 --timeout-ms 8000 2>/dev/null' | tr -d '\r')"
[ "$OUT" = "hello-kafka" ] || { echo "smoke message mismatch: $OUT" >&2; exit 1; }
