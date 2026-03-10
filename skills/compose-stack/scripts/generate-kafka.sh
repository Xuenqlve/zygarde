#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v4.2>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v4.2" ]; then
  echo "版本错误: $VERSION (仅支持 v4.2)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
IMAGE="apache/kafka:4.2.0"
OUTPUT_DIR="$PROJECT_ROOT/docker/kafka/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating Kafka $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  kafka:
    image: ${IMAGE}
    container_name: zygarde-kafka-single
    hostname: kafka
    restart: unless-stopped
    ports:
      - "\${KAFKA_PORT:-9092}:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      CLUSTER_ID: \${KAFKA_CLUSTER_ID:-MkU3OEVBNTcwNTJENDM2Qk}
    volumes:
      - ./data/kafka:/var/lib/kafka/data
    healthcheck:
      test: ["CMD-SHELL", "bash -lc '/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1'"]
      interval: 5s
      timeout: 5s
      retries: 60
      start_period: 30s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
KAFKA_VERSION=v4.2
KAFKA_PORT=9092
KAFKA_CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SINGLE_EOF'
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

echo "[1/2] Starting Kafka single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-kafka-single..."
for _ in $(seq 1 90); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-kafka-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "Kafka is healthy."
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-kafka-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs kafka || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
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
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Kafka ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Kafka KRaft 单节点（broker+controller 合并）

## 稳定性说明

- 使用 \`apache/kafka:4.2.0\` 镜像。
- 验收统一走 \`build.sh -> check.sh -> cleanup\`。
- check 包含 topic 创建 + 生产消费烟测。
EOF

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  kafka1:
    image: ${IMAGE}
    container_name: zygarde-kafka-1
    hostname: kafka1
    restart: unless-stopped
    ports:
      - "\${KAFKA1_PORT:-9092}:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka1:9093,2@kafka2:9093,3@kafka3:9093
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka1:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      CLUSTER_ID: \${KAFKA_CLUSTER_ID:-MkU3OEVBNTcwNTJENDM2Qk}
    volumes:
      - ./data/kafka1:/var/lib/kafka/data

  kafka2:
    image: ${IMAGE}
    container_name: zygarde-kafka-2
    hostname: kafka2
    restart: unless-stopped
    ports:
      - "\${KAFKA2_PORT:-9094}:9092"
    environment:
      KAFKA_NODE_ID: 2
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka1:9093,2@kafka2:9093,3@kafka3:9093
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka2:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      CLUSTER_ID: \${KAFKA_CLUSTER_ID:-MkU3OEVBNTcwNTJENDM2Qk}
    volumes:
      - ./data/kafka2:/var/lib/kafka/data

  kafka3:
    image: ${IMAGE}
    container_name: zygarde-kafka-3
    hostname: kafka3
    restart: unless-stopped
    ports:
      - "\${KAFKA3_PORT:-9096}:9092"
    environment:
      KAFKA_NODE_ID: 3
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka1:9093,2@kafka2:9093,3@kafka3:9093
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka3:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      CLUSTER_ID: \${KAFKA_CLUSTER_ID:-MkU3OEVBNTcwNTJENDM2Qk}
    volumes:
      - ./data/kafka3:/var/lib/kafka/data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
KAFKA_VERSION=v4.2
KAFKA_CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk
KAFKA1_PORT=9092
KAFKA2_PORT=9094
KAFKA3_PORT=9096
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_CLUSTER_EOF'
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
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
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
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Kafka ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Kafka KRaft 3 节点集群（broker+controller 合并）

## 稳定性说明

- 使用 \`apache/kafka:4.2.0\` 镜像。
- 采用 KRaft 模式，不依赖 ZooKeeper。
- check 包含 metadata quorum 状态 + 跨节点生产消费烟测。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "Kafka $SCENARIO $VERSION generation complete!"
