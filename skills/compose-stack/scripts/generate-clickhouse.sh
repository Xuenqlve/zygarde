#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v24|v25>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v24" ] && [ "$VERSION" != "v25" ]; then
  echo "版本错误: $VERSION (仅支持 v24|v25)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/clickhouse/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

if [ "$VERSION" = "v24" ]; then
  DEFAULT_TAG="24"
else
  DEFAULT_TAG="25.8"
fi
IMAGE="${CLICKHOUSE_IMAGE:-clickhouse/clickhouse-server:${DEFAULT_TAG}}"

print_info "Generating ClickHouse $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  clickhouse:
    image: ${IMAGE}
    container_name: zygarde-clickhouse-single
    restart: unless-stopped
    ports:
      - "\${CLICKHOUSE_HTTP_PORT:-8123}:8123"
      - "\${CLICKHOUSE_TCP_PORT:-9000}:9000"
    volumes:
      - ./data/clickhouse:/var/lib/clickhouse
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
CLICKHOUSE_VERSION=${VERSION}
CLICKHOUSE_HTTP_PORT=8123
CLICKHOUSE_TCP_PORT=9000
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

echo "[1/2] Starting ClickHouse single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting clickhouse ready..."
for _ in $(seq 1 120); do
  if "${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT 1" >/dev/null 2>&1; then
    echo "ClickHouse single is ready."
    exit 0
  fi
  sleep 2
done

echo "ClickHouse single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs clickhouse || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-clickhouse-single/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT 1"

echo "[3/4] Version"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "SELECT version()"

echo "[4/4] Create/Insert/Select smoke"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "CREATE TABLE IF NOT EXISTS zygarde_smoke (id UInt32, v String) ENGINE=MergeTree ORDER BY id"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-single clickhouse-client -q "INSERT INTO zygarde_smoke VALUES (1, 'ok')"
OUT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-single clickhouse-client -q "SELECT v FROM zygarde_smoke WHERE id=1 FORMAT TSVRaw" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "smoke failed: $OUT" >&2; exit 1; }
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# ClickHouse ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

ClickHouse 单节点

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 \`clickhouse-client SELECT 1\` 就绪信号判定。
- check 覆盖连接、版本、基础读写链路。
EOF

else
  mkdir -p "$OUTPUT_DIR/config/ch1/config.d" "$OUTPUT_DIR/config/ch2/config.d" "$OUTPUT_DIR/config/ch3/config.d" \
           "$OUTPUT_DIR/config/ch1/users.d" "$OUTPUT_DIR/config/ch2/users.d" "$OUTPUT_DIR/config/ch3/users.d"

  cat > "$OUTPUT_DIR/config/ch1/config.d/cluster.xml" <<'CLUSTER_XML'
<clickhouse>
  <remote_servers>
    <zygarde_cluster>
      <shard>
        <replica><host>ch1</host><port>9000</port></replica>
        <replica><host>ch2</host><port>9000</port></replica>
        <replica><host>ch3</host><port>9000</port></replica>
      </shard>
    </zygarde_cluster>
  </remote_servers>
</clickhouse>
CLUSTER_XML
  cp "$OUTPUT_DIR/config/ch1/config.d/cluster.xml" "$OUTPUT_DIR/config/ch2/config.d/cluster.xml"
  cp "$OUTPUT_DIR/config/ch1/config.d/cluster.xml" "$OUTPUT_DIR/config/ch3/config.d/cluster.xml"

  cat > "$OUTPUT_DIR/config/ch1/users.d/default-network.xml" <<'USERS_XML'
<clickhouse>
  <users>
    <default>
      <networks>
        <ip>0.0.0.0/0</ip>
        <ip>::/0</ip>
      </networks>
    </default>
  </users>
</clickhouse>
USERS_XML
  cp "$OUTPUT_DIR/config/ch1/users.d/default-network.xml" "$OUTPUT_DIR/config/ch2/users.d/default-network.xml"
  cp "$OUTPUT_DIR/config/ch1/users.d/default-network.xml" "$OUTPUT_DIR/config/ch3/users.d/default-network.xml"

  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  ch1:
    image: ${IMAGE}
    container_name: zygarde-clickhouse-1
    restart: unless-stopped
    ports:
      - "\${CH1_HTTP_PORT:-8123}:8123"
      - "\${CH1_TCP_PORT:-9000}:9000"
    volumes:
      - ./data/ch1:/var/lib/clickhouse
      - ./config/ch1/config.d/cluster.xml:/etc/clickhouse-server/config.d/cluster.xml:ro
      - ./config/ch1/users.d/default-network.xml:/etc/clickhouse-server/users.d/default-network.xml:ro

  ch2:
    image: ${IMAGE}
    container_name: zygarde-clickhouse-2
    restart: unless-stopped
    ports:
      - "\${CH2_HTTP_PORT:-8124}:8123"
      - "\${CH2_TCP_PORT:-9001}:9000"
    volumes:
      - ./data/ch2:/var/lib/clickhouse
      - ./config/ch2/config.d/cluster.xml:/etc/clickhouse-server/config.d/cluster.xml:ro
      - ./config/ch2/users.d/default-network.xml:/etc/clickhouse-server/users.d/default-network.xml:ro

  ch3:
    image: ${IMAGE}
    container_name: zygarde-clickhouse-3
    restart: unless-stopped
    ports:
      - "\${CH3_HTTP_PORT:-8125}:8123"
      - "\${CH3_TCP_PORT:-9002}:9000"
    volumes:
      - ./data/ch3:/var/lib/clickhouse
      - ./config/ch3/config.d/cluster.xml:/etc/clickhouse-server/config.d/cluster.xml:ro
      - ./config/ch3/users.d/default-network.xml:/etc/clickhouse-server/users.d/default-network.xml:ro
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
CLICKHOUSE_VERSION=${VERSION}
CH1_HTTP_PORT=8123
CH2_HTTP_PORT=8124
CH3_HTTP_PORT=8125
CH1_TCP_PORT=9000
CH2_TCP_PORT=9001
CH3_TCP_PORT=9002
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

echo "[1/2] Starting ClickHouse cluster..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting all nodes ready..."
for _ in $(seq 1 120); do
  if "${ENGINE_CMD[@]}" exec zygarde-clickhouse-1 clickhouse-client -q "SELECT 1" >/dev/null 2>&1 \
    && "${ENGINE_CMD[@]}" exec zygarde-clickhouse-2 clickhouse-client -q "SELECT 1" >/dev/null 2>&1 \
    && "${ENGINE_CMD[@]}" exec zygarde-clickhouse-3 clickhouse-client -q "SELECT 1" >/dev/null 2>&1; then
    echo "ClickHouse cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "ClickHouse cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-clickhouse-[123]/'

echo "[2/4] Connectivity on each node"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-1 clickhouse-client -q "SELECT 1"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-2 clickhouse-client -q "SELECT 1"
"${ENGINE_CMD[@]}" exec zygarde-clickhouse-3 clickhouse-client -q "SELECT 1"

echo "[3/4] Cluster topology check"
CNT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-1 clickhouse-client -q "SELECT count() FROM system.clusters WHERE cluster='zygarde_cluster'" | tr -d '[:space:]')"
[ "${CNT:-0}" -ge 3 ] || { echo "cluster topology invalid: $CNT" >&2; exit 1; }
echo "cluster_nodes=$CNT"

echo "[4/4] Cross-node smoke via remote()"
OUT="$(${ENGINE_CMD[@]} exec zygarde-clickhouse-1 clickhouse-client -q "SELECT count() FROM remote('ch1,ch2,ch3', system.one)" | tr -d '[:space:]')"
[ "$OUT" = "3" ] || { echo "remote smoke failed: $OUT" >&2; exit 1; }
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# ClickHouse ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

ClickHouse 三节点集群（3 个 server 节点）

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 阶段强校验 3 节点均可执行 \`SELECT 1\`。
- check 阶段覆盖拓扑检测（system.clusters）与跨节点链路（remote）。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "ClickHouse $SCENARIO $VERSION generation complete!"