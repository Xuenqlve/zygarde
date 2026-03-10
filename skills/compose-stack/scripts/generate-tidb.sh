#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single> <v6.7>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION_ARG="$2"

if [ "$SCENARIO" != "single" ]; then
  echo "场景错误: $SCENARIO (仅支持 single)"; usage
fi
if [ "$VERSION_ARG" != "v6.7" ]; then
  echo "版本错误: $VERSION_ARG (仅支持 v6.7)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/tidb/${SCENARIO}_${VERSION_ARG}"
mkdir -p "$OUTPUT_DIR"

# 说明：TiDB 官方镜像暂无 v6.7.x tag，当前以 v6.5.12 作为兼容落地版本。
# 若后续官方发布 v6.7.x，可通过环境变量覆盖：TIDB_IMAGE_TAG=v6.7.x
IMAGE_TAG="${TIDB_IMAGE_TAG:-v6.5.12}"
PD_IMAGE="pingcap/pd:${IMAGE_TAG}"
TIKV_IMAGE="pingcap/tikv:${IMAGE_TAG}"
TIDB_IMAGE="pingcap/tidb:${IMAGE_TAG}"

print_info "Generating TiDB $SCENARIO $VERSION_ARG"

cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  pd:
    image: ${PD_IMAGE}
    container_name: zygarde-tidb-pd-single
    restart: unless-stopped
    command: [
      "--name=pd",
      "--data-dir=/data/pd",
      "--client-urls=http://0.0.0.0:2379",
      "--peer-urls=http://0.0.0.0:2380",
      "--advertise-client-urls=http://pd:2379",
      "--advertise-peer-urls=http://pd:2380",
      "--initial-cluster=pd=http://pd:2380"
    ]
    ports:
      - "\${PD_PORT:-2379}:2379"
    volumes:
      - ./data/pd:/data/pd

  tikv:
    image: ${TIKV_IMAGE}
    container_name: zygarde-tidb-tikv-single
    restart: unless-stopped
    depends_on:
      - pd
    command: [
      "--pd=pd:2379",
      "--addr=0.0.0.0:20160",
      "--advertise-addr=tikv:20160",
      "--data-dir=/data/tikv"
    ]
    ports:
      - "\${TIKV_PORT:-20160}:20160"
    volumes:
      - ./data/tikv:/data/tikv

  tidb:
    image: ${TIDB_IMAGE}
    container_name: zygarde-tidb-single
    restart: unless-stopped
    depends_on:
      - pd
      - tikv
    command: [
      "--store=tikv",
      "--path=pd:2379",
      "--host=0.0.0.0",
      "--status=10080",
      "--advertise-address=tidb"
    ]
    ports:
      - "\${TIDB_PORT:-4000}:4000"
      - "\${TIDB_STATUS_PORT:-10080}:10080"
EOF

cat > "$OUTPUT_DIR/.env" <<EOF
TIDB_VERSION=${VERSION_ARG}
PD_PORT=2379
TIKV_PORT=20160
TIDB_PORT=4000
TIDB_STATUS_PORT=10080
EOF

cat > "$OUTPUT_DIR/build.sh" <<'BUILD_EOF'
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

echo "[1/4] Starting TiDB single (pd+tikv+tidb)..."
"${COMPOSE_CMD[@]}" up -d

wait_running() {
  local name="$1"
  for _ in $(seq 1 60); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "running" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[2/4] Waiting pd running..."
wait_running zygarde-tidb-pd-single || { "${COMPOSE_CMD[@]}" logs pd; exit 1; }

echo "[3/4] Waiting tikv running..."
wait_running zygarde-tidb-tikv-single || { "${COMPOSE_CMD[@]}" logs tikv; exit 1; }

echo "[4/4] Waiting tidb status endpoint..."
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${TIDB_STATUS_PORT:-10080}/status" >/dev/null 2>&1; then
    echo "TiDB status endpoint is ready."
    exit 0
  fi
  sleep 2
done

echo "TiDB status endpoint not ready in time" >&2
"${COMPOSE_CMD[@]}" logs tidb || true
exit 1
BUILD_EOF
chmod +x "$OUTPUT_DIR/build.sh"

cat > "$OUTPUT_DIR/check.sh" <<'CHECK_EOF'
#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-tidb-pd-single|zygarde-tidb-tikv-single|zygarde-tidb-single/'

echo "[2/4] TiDB status"
curl -fsS "http://127.0.0.1:${TIDB_STATUS_PORT:-10080}/status"
echo ""

echo "[3/4] PD health"
curl -fsS "http://127.0.0.1:${PD_PORT:-2379}/pd/api/v1/health"
echo ""

echo "[4/4] TiDB SQL port open"
if (exec 3<>/dev/tcp/127.0.0.1/${TIDB_PORT:-4000}) 2>/dev/null; then
  echo "tidb sql port ${TIDB_PORT:-4000} is reachable"
  exec 3>&-
else
  echo "tidb sql port ${TIDB_PORT:-4000} is not reachable" >&2
  exit 1
fi
CHECK_EOF
chmod +x "$OUTPUT_DIR/check.sh"

cat > "$OUTPUT_DIR/README.md" <<EOF
# TiDB ${SCENARIO} ${VERSION_ARG}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

单节点入口（TiDB）+ 单 PD + 单 TiKV，适用于本地开发联调与初始化验证。

## 稳定性说明

- 版本固定：\`pingcap/*:${IMAGE_TAG}\`（对外语义版本为 \`${VERSION_ARG}\`，当前默认映射到可用镜像 tag）。
- 启动顺序：pd -> tikv -> tidb。
- build 阶段以 TiDB status endpoint 就绪为可用信号。
- check 阶段覆盖容器状态、TiDB status、PD health、SQL 端口探活。
EOF

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "TiDB $SCENARIO $VERSION_ARG generation complete!"