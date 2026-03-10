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
IMAGE="rabbitmq:4.2-management"
OUTPUT_DIR="$PROJECT_ROOT/docker/rabbitmq/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating RabbitMQ $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  rabbitmq:
    image: ${IMAGE}
    container_name: zygarde-rabbitmq-single
    hostname: rabbitmq
    restart: unless-stopped
    ports:
      - "\${RABBITMQ_AMQP_PORT:-5672}:5672"
      - "\${RABBITMQ_MANAGEMENT_PORT:-15672}:15672"
    environment:
      RABBITMQ_DEFAULT_USER: \${RABBITMQ_DEFAULT_USER:-admin}
      RABBITMQ_DEFAULT_PASS: \${RABBITMQ_DEFAULT_PASS:-admin123}
      RABBITMQ_ERLANG_COOKIE: \${RABBITMQ_ERLANG_COOKIE:-rabbitmq-cookie}
    volumes:
      - ./data/rabbitmq:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD-SHELL", "rabbitmq-diagnostics -q ping"]
      interval: 5s
      timeout: 5s
      retries: 40
      start_period: 20s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
RABBITMQ_VERSION=v4.2
RABBITMQ_AMQP_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_DEFAULT_USER=admin
RABBITMQ_DEFAULT_PASS=admin123
RABBITMQ_ERLANG_COOKIE=rabbitmq-cookie
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

echo "[1/2] Starting RabbitMQ single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-rabbitmq-single..."
for _ in $(seq 1 60); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-rabbitmq-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "RabbitMQ is healthy."
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-rabbitmq-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs rabbitmq || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-rabbitmq-single/'

echo "[2/3] RabbitMQ diagnostics"
"${ENGINE_CMD[@]}" exec zygarde-rabbitmq-single rabbitmq-diagnostics -q ping

echo "[3/3] Overview"
"${ENGINE_CMD[@]}" exec zygarde-rabbitmq-single rabbitmqctl status | grep -E "RabbitMQ version|Cluster name|Uptime" || true
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# RabbitMQ ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

单节点 RabbitMQ（含 Management 插件）

## 稳定性说明

- 使用 \`rabbitmq:4.2-management\` 镜像。
- 验收统一走 \`build.sh -> check.sh -> cleanup\`。
- 就绪判定采用 \`rabbitmq-diagnostics -q ping\` 健康检查。
EOF

else
  mkdir -p "$OUTPUT_DIR/conf"

  cat > "$OUTPUT_DIR/conf/rabbitmq.conf" <<'RABBIT_CONF_EOF'
cluster_formation.peer_discovery_backend = classic_config
cluster_formation.classic_config.nodes.1 = rabbit@rabbit1
cluster_formation.classic_config.nodes.2 = rabbit@rabbit2
cluster_formation.classic_config.nodes.3 = rabbit@rabbit3
cluster_partition_handling = autoheal
queue_master_locator = min-masters
RABBIT_CONF_EOF

  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  rabbit1:
    image: ${IMAGE}
    container_name: zygarde-rabbitmq-1
    hostname: rabbit1
    restart: unless-stopped
    ports:
      - "\${RABBITMQ1_AMQP_PORT:-5672}:5672"
      - "\${RABBITMQ1_MANAGEMENT_PORT:-15672}:15672"
    environment:
      RABBITMQ_DEFAULT_USER: \${RABBITMQ_DEFAULT_USER:-admin}
      RABBITMQ_DEFAULT_PASS: \${RABBITMQ_DEFAULT_PASS:-admin123}
      RABBITMQ_ERLANG_COOKIE: \${RABBITMQ_ERLANG_COOKIE:-rabbitmq-cookie}
      RABBITMQ_NODENAME: rabbit@rabbit1
    volumes:
      - ./data/rabbit1:/var/lib/rabbitmq
      - ./conf/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
    healthcheck:
      test: ["CMD-SHELL", "rabbitmq-diagnostics -q ping"]
      interval: 5s
      timeout: 5s
      retries: 60
      start_period: 20s

  rabbit2:
    image: ${IMAGE}
    container_name: zygarde-rabbitmq-2
    hostname: rabbit2
    restart: unless-stopped
    depends_on:
      rabbit1:
        condition: service_healthy
    ports:
      - "\${RABBITMQ2_AMQP_PORT:-5673}:5672"
      - "\${RABBITMQ2_MANAGEMENT_PORT:-15673}:15672"
    environment:
      RABBITMQ_DEFAULT_USER: \${RABBITMQ_DEFAULT_USER:-admin}
      RABBITMQ_DEFAULT_PASS: \${RABBITMQ_DEFAULT_PASS:-admin123}
      RABBITMQ_ERLANG_COOKIE: \${RABBITMQ_ERLANG_COOKIE:-rabbitmq-cookie}
      RABBITMQ_NODENAME: rabbit@rabbit2
    volumes:
      - ./data/rabbit2:/var/lib/rabbitmq
      - ./conf/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
    healthcheck:
      test: ["CMD-SHELL", "rabbitmq-diagnostics -q ping"]
      interval: 5s
      timeout: 5s
      retries: 60
      start_period: 20s

  rabbit3:
    image: ${IMAGE}
    container_name: zygarde-rabbitmq-3
    hostname: rabbit3
    restart: unless-stopped
    depends_on:
      rabbit1:
        condition: service_healthy
    ports:
      - "\${RABBITMQ3_AMQP_PORT:-5674}:5672"
      - "\${RABBITMQ3_MANAGEMENT_PORT:-15674}:15672"
    environment:
      RABBITMQ_DEFAULT_USER: \${RABBITMQ_DEFAULT_USER:-admin}
      RABBITMQ_DEFAULT_PASS: \${RABBITMQ_DEFAULT_PASS:-admin123}
      RABBITMQ_ERLANG_COOKIE: \${RABBITMQ_ERLANG_COOKIE:-rabbitmq-cookie}
      RABBITMQ_NODENAME: rabbit@rabbit3
    volumes:
      - ./data/rabbit3:/var/lib/rabbitmq
      - ./conf/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
    healthcheck:
      test: ["CMD-SHELL", "rabbitmq-diagnostics -q ping"]
      interval: 5s
      timeout: 5s
      retries: 60
      start_period: 20s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
RABBITMQ_VERSION=v4.2
RABBITMQ_DEFAULT_USER=admin
RABBITMQ_DEFAULT_PASS=admin123
RABBITMQ_ERLANG_COOKIE=rabbitmq-cookie
RABBITMQ1_AMQP_PORT=5672
RABBITMQ1_MANAGEMENT_PORT=15672
RABBITMQ2_AMQP_PORT=5673
RABBITMQ2_MANAGEMENT_PORT=15673
RABBITMQ3_AMQP_PORT=5674
RABBITMQ3_MANAGEMENT_PORT=15674
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

echo "[1/2] Starting RabbitMQ cluster nodes..."
"${COMPOSE_CMD[@]}" up -d

wait_healthy() {
  local name="$1"
  for _ in $(seq 1 90); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[2/2] Waiting nodes healthy..."
wait_healthy zygarde-rabbitmq-1 || { "${COMPOSE_CMD[@]}" logs rabbit1; exit 1; }
wait_healthy zygarde-rabbitmq-2 || { "${COMPOSE_CMD[@]}" logs rabbit2; exit 1; }
wait_healthy zygarde-rabbitmq-3 || { "${COMPOSE_CMD[@]}" logs rabbit3; exit 1; }

echo "RabbitMQ cluster nodes are healthy."
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
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
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# RabbitMQ ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

3 节点 RabbitMQ 集群（classic_config 自动组网）

## 稳定性说明

- 使用 \`rabbitmq:4.2-management\` 镜像。
- 采用 \`classic_config\` 声明式集群发现，避免运行时 stop/reset/join 抖动。
- check 强校验 cluster_status 必须收敛到 rabbit1/rabbit2/rabbit3（含重试窗口）。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "RabbitMQ $SCENARIO $VERSION generation complete!"
