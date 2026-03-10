#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
usage() {
    echo "Usage: $0 <single|master-slave|cluster> <v6.2|v7.4>"
    echo "Example: $0 cluster v7.4"
    exit 1
}

if [ $# -lt 2 ]; then usage; fi

SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "master-slave" ] && [ "$SCENARIO" != "cluster" ]; then
    echo "场景错误: $SCENARIO"
    usage
fi

if [ "$VERSION" != "v6.2" ] && [ "$VERSION" != "v7.4" ]; then
    echo "版本错误: $VERSION (v6.2 或 v7.4)"
    usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

if [ "$VERSION" = "v6.2" ]; then
    IMAGE="redis:6.2"
else
    IMAGE="redis:7.4"
fi

OUTPUT_DIR="${PROJECT_ROOT}/docker/redis/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating Redis $SCENARIO $VERSION"

# ============ Single 场景 ============
if [ "$SCENARIO" = "single" ]; then
    cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  redis:
    image: ${IMAGE}
    container_name: zygarde-redis-single
    restart: unless-stopped
    ports:
      - "\${REDIS_PORT:-6379}:6379"
    volumes:
      - ./data/redis:/data
    command:
      - redis-server
      - --appendonly
      - yes
      - --save
      - "60 1000"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 10s
EOF

    cat > "$OUTPUT_DIR/.env" <<EOF
REDIS_VERSION=${IMAGE}
REDIS_PORT=6379
EOF

    cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(podman-compose)
    fi
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(docker-compose)
    fi
else
    echo "No container engine found." >&2
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "No compose command found for current container engine." >&2
    exit 1
fi

echo "[1/2] Starting Redis single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-redis-single..."
for _ in $(seq 1 30); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-redis-single 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then
        echo "Redis is healthy."
        "${ENGINE_CMD[@]}" exec zygarde-redis-single redis-cli ping || true
        exit 0
    fi
    sleep 2
done

echo "Container zygarde-redis-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs redis || true
exit 1
BUILD_SINGLE_EOF
    chmod +x "$OUTPUT_DIR/build.sh"

    cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-redis-single redis-cli ping

echo "[3/3] Role"
"${ENGINE_CMD[@]}" exec zygarde-redis-single redis-cli info replication | grep '^role:'
CHECK_SINGLE_EOF
    chmod +x "$OUTPUT_DIR/check.sh"

# ============ Master-Slave 场景 ============
elif [ "$SCENARIO" = "master-slave" ]; then
    cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  redis-master:
    image: ${IMAGE}
    container_name: zygarde-redis-master
    restart: unless-stopped
    ports:
      - "\${REDIS_MASTER_PORT:-6379}:6379"
    volumes:
      - ./data/redis-master:/data
    command:
      - redis-server
      - --appendonly
      - yes
      - --save
      - "60 1000"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 10s

  redis-slave:
    image: ${IMAGE}
    container_name: zygarde-redis-slave
    restart: unless-stopped
    ports:
      - "\${REDIS_SLAVE_PORT:-6380}:6379"
    volumes:
      - ./data/redis-slave:/data
    command:
      - redis-server
      - --appendonly
      - yes
      - --save
      - "60 1000"
      - --replicaof
      - redis-master
      - "6379"
    depends_on:
      redis-master:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 10s
EOF

    cat > "$OUTPUT_DIR/.env" <<EOF
REDIS_VERSION=${IMAGE}
REDIS_MASTER_PORT=6379
REDIS_SLAVE_PORT=6380
EOF

    cat > "$OUTPUT_DIR/build.sh" <<'BUILD_MS_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(podman-compose)
    fi
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(docker-compose)
    fi
else
    echo "No container engine found." >&2
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "No compose command found for current container engine." >&2
    exit 1
fi

echo "[1/3] Starting Redis master/slave..."
"${COMPOSE_CMD[@]}" up -d

wait_healthy() {
    local name="$1"
    for _ in $(seq 1 30); do
        status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        sleep 2
    done
    return 1
}

echo "[2/3] Waiting for master healthy..."
wait_healthy zygarde-redis-master || { "${COMPOSE_CMD[@]}" logs redis-master; exit 1; }

echo "[3/3] Waiting for slave healthy..."
wait_healthy zygarde-redis-slave || { "${COMPOSE_CMD[@]}" logs redis-slave; exit 1; }

echo "Redis master/slave is healthy."
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli info replication | grep '^role:' || true
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli info replication | grep '^role:' || true
BUILD_MS_EOF
    chmod +x "$OUTPUT_DIR/build.sh"

    cat > "$OUTPUT_DIR/check.sh" <<'CHECK_MS_EOF'
#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-master|zygarde-redis-slave/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli ping
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli ping

echo "[3/4] Master role"
"${ENGINE_CMD[@]}" exec zygarde-redis-master redis-cli info replication | grep -E '^role:|connected_slaves:'

echo "[4/4] Slave role"
"${ENGINE_CMD[@]}" exec zygarde-redis-slave redis-cli info replication | grep -E '^role:|master_host:|master_link_status:'
CHECK_MS_EOF
    chmod +x "$OUTPUT_DIR/check.sh"

# ============ Cluster 场景 (3主) ============
elif [ "$SCENARIO" = "cluster" ]; then
    cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  redis-node-1:
    image: ${IMAGE}
    container_name: zygarde-redis-node-1
    restart: unless-stopped
    ports:
      - "\${REDIS_NODE_1_PORT:-7001}:7001"
      - "\${REDIS_NODE_1_BUS_PORT:-17001}:17001"
    volumes:
      - ./data/redis-node-1:/data
    command:
      - redis-server
      - --port
      - "7001"
      - --cluster-enabled
      - "yes"
      - --cluster-config-file
      - nodes.conf
      - --cluster-node-timeout
      - "5000"
      - --appendonly
      - "yes"

  redis-node-2:
    image: ${IMAGE}
    container_name: zygarde-redis-node-2
    restart: unless-stopped
    ports:
      - "\${REDIS_NODE_2_PORT:-7002}:7002"
      - "\${REDIS_NODE_2_BUS_PORT:-17002}:17002"
    volumes:
      - ./data/redis-node-2:/data
    command:
      - redis-server
      - --port
      - "7002"
      - --cluster-enabled
      - "yes"
      - --cluster-config-file
      - nodes.conf
      - --cluster-node-timeout
      - "5000"
      - --appendonly
      - "yes"

  redis-node-3:
    image: ${IMAGE}
    container_name: zygarde-redis-node-3
    restart: unless-stopped
    ports:
      - "\${REDIS_NODE_3_PORT:-7003}:7003"
      - "\${REDIS_NODE_3_BUS_PORT:-17003}:17003"
    volumes:
      - ./data/redis-node-3:/data
    command:
      - redis-server
      - --port
      - "7003"
      - --cluster-enabled
      - "yes"
      - --cluster-config-file
      - nodes.conf
      - --cluster-node-timeout
      - "5000"
      - --appendonly
      - "yes"
EOF

    cat > "$OUTPUT_DIR/.env" <<EOF
REDIS_VERSION=${IMAGE}
REDIS_NODE_1_PORT=7001
REDIS_NODE_1_BUS_PORT=17001
REDIS_NODE_2_PORT=7002
REDIS_NODE_2_BUS_PORT=17002
REDIS_NODE_3_PORT=7003
REDIS_NODE_3_BUS_PORT=17003
EOF

    cat > "$OUTPUT_DIR/build.sh" <<'BUILD_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(podman-compose)
    fi
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(docker-compose)
    fi
else
    echo "No container engine found." >&2
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "No compose command found for current container engine." >&2
    exit 1
fi

echo "[1/4] Starting Redis cluster nodes..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/4] Waiting for nodes..."
sleep 8

echo "[3/4] Creating cluster (3 masters, no replicas)..."
IP1="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-1)"
IP2="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-2)"
IP3="$(${ENGINE_CMD[@]} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' zygarde-redis-node-3)"

"${ENGINE_CMD[@]}" exec -i zygarde-redis-node-1 redis-cli --cluster create \
    "${IP1}:7001" "${IP2}:7002" "${IP3}:7003" \
    --cluster-replicas 0 --cluster-yes

echo "[4/4] Cluster info"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster info | grep cluster_state || true
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster nodes || true
BUILD_CLUSTER_EOF
    chmod +x "$OUTPUT_DIR/build.sh"

    cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-redis-node-1|zygarde-redis-node-2|zygarde-redis-node-3/'

echo "[2/4] PING all nodes"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 ping
"${ENGINE_CMD[@]}" exec zygarde-redis-node-2 redis-cli -p 7002 ping
"${ENGINE_CMD[@]}" exec zygarde-redis-node-3 redis-cli -p 7003 ping

echo "[3/4] Cluster state"

# 强校验：cluster_state 必须为 ok（允许短暂收敛重试）
OK=0
FINAL_INFO=""
for _ in $(seq 1 10); do
  FINAL_INFO="$(${ENGINE_CMD[@]} exec zygarde-redis-node-1 redis-cli -p 7001 cluster info)"
  CSTATE="$(echo "$FINAL_INFO" | grep '^cluster_state:' | awk -F: '{print $2}' | tr -d '\r')"
  if [ "$CSTATE" = "ok" ]; then
    OK=1
    break
  fi
  sleep 2
done

# 打印最终状态（避免打印旧状态造成误导）
echo "$FINAL_INFO" | grep -E 'cluster_state|cluster_known_nodes|cluster_size'

if [ "$OK" -ne 1 ]; then
  echo "[FAIL] cluster_state 非 ok" >&2
  echo "$FINAL_INFO" >&2
  exit 1
fi

echo "[4/4] Cluster nodes"
"${ENGINE_CMD[@]}" exec zygarde-redis-node-1 redis-cli -p 7001 cluster nodes
CHECK_CLUSTER_EOF
    chmod +x "$OUTPUT_DIR/check.sh"
fi

if [ "$SCENARIO" = "single" ]; then
    cat > "$OUTPUT_DIR/README.md" <<EOF
# Redis $SCENARIO $VERSION

## 快速开始

\`\`\`bash
# 启动
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
\`\`\`

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| REDIS_PORT | 6379 | Redis 端口 |

## 场景

单实例 Redis（appendonly 已开启）

## 稳定性说明

- 验收统一走 \`build.sh -> check.sh -> cleanup\`。
- 首次拉取镜像时间较长属于正常现象；二次启动会显著加快。
- 验收后由 compose-stack 自动执行 \`down -v\` 并清理 \`data/\`。
EOF
elif [ "$SCENARIO" = "master-slave" ]; then
    cat > "$OUTPUT_DIR/README.md" <<EOF
# Redis $SCENARIO $VERSION

## 快速开始

\`\`\`bash
# 启动
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
\`\`\`

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| REDIS_MASTER_PORT | 6379 | Master 端口 |
| REDIS_SLAVE_PORT | 6380 | Slave 端口 |

## 场景

主从复制（1主1从）

## 稳定性说明

- build 阶段先等待 master 健康，再拉起 slave，降低首次复制抖动。
- check 阶段校验 master/slave 角色与 slave 链路状态（\`master_link_status:up\`）。
- 若有固定容器名冲突，compose-stack 验收前会自动清理旧容器。
EOF
else
    cat > "$OUTPUT_DIR/README.md" <<EOF
# Redis $SCENARIO $VERSION

## 快速开始

\`\`\`bash
# 启动并初始化集群
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
\`\`\`

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| REDIS_NODE_1_PORT | 7001 | 节点1端口 |
| REDIS_NODE_2_PORT | 7002 | 节点2端口 |
| REDIS_NODE_3_PORT | 7003 | 节点3端口 |

## 场景

Redis Cluster（3主节点，无副本）

## 稳定性说明

- 集群创建后可能短暂出现 \`cluster_state:fail\`，check 内置重试等待收敛。
- 仅当最终 \`cluster_state:ok\` 才判定验收通过。

## 兼容性说明

- 集群初始化时，脚本使用容器 IP 建立集群（而非容器名），以兼容 Redis 6.2 在部分环境中的地址校验差异。
- \`check.sh\` 对 \`cluster_state\` 做强校验，最终必须为 \`ok\`。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "Redis $SCENARIO $VERSION generation complete!"