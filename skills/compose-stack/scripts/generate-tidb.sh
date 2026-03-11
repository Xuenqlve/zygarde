#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v6.7>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION_ARG="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO (支持 single|cluster)"; usage
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

if [ "$SCENARIO" = "single" ]; then
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
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
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
CHECK_SINGLE_EOF
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

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  pd1:
    image: ${PD_IMAGE}
    container_name: zygarde-tidb-pd1
    restart: unless-stopped
    command: [
      "--name=pd1",
      "--data-dir=/data/pd",
      "--client-urls=http://0.0.0.0:2379",
      "--peer-urls=http://0.0.0.0:2380",
      "--advertise-client-urls=http://pd1:2379",
      "--advertise-peer-urls=http://pd1:2380",
      "--initial-cluster=pd1=http://pd1:2380,pd2=http://pd2:2380,pd3=http://pd3:2380",
      "--force-new-cluster"
    ]
    ports:
      - "\${PD1_PORT:-2379}:2379"
    volumes:
      - ./data/pd1:/data/pd

  pd2:
    image: ${PD_IMAGE}
    container_name: zygarde-tidb-pd2
    restart: unless-stopped
    command: [
      "--name=pd2",
      "--data-dir=/data/pd",
      "--client-urls=http://0.0.0.0:2379",
      "--peer-urls=http://0.0.0.0:2380",
      "--advertise-client-urls=http://pd2:2379",
      "--advertise-peer-urls=http://pd2:2380",
      "--join=pd1:2379"
    ]
    ports:
      - "\${PD2_PORT:-2479}:2379"
    volumes:
      - ./data/pd2:/data/pd

  pd3:
    image: ${PD_IMAGE}
    container_name: zygarde-tidb-pd3
    restart: unless-stopped
    command: [
      "--name=pd3",
      "--data-dir=/data/pd",
      "--client-urls=http://0.0.0.0:2379",
      "--peer-urls=http://0.0.0.0:2380",
      "--advertise-client-urls=http://pd3:2379",
      "--advertise-peer-urls=http://pd3:2380",
      "--join=pd1:2379"
    ]
    ports:
      - "\${PD3_PORT:-2579}:2379"
    volumes:
      - ./data/pd3:/data/pd

  tikv1:
    image: ${TIKV_IMAGE}
    container_name: zygarde-tidb-tikv1
    restart: unless-stopped
    depends_on: [pd1, pd2, pd3]
    command: [
      "--pd=pd1:2379,pd2:2379,pd3:2379",
      "--addr=0.0.0.0:20160",
      "--advertise-addr=tikv1:20160",
      "--data-dir=/data/tikv"
    ]
    volumes:
      - ./data/tikv1:/data/tikv

  tikv2:
    image: ${TIKV_IMAGE}
    container_name: zygarde-tidb-tikv2
    restart: unless-stopped
    depends_on: [pd1, pd2, pd3]
    command: [
      "--pd=pd1:2379,pd2:2379,pd3:2379",
      "--addr=0.0.0.0:20160",
      "--advertise-addr=tikv2:20160",
      "--data-dir=/data/tikv"
    ]
    volumes:
      - ./data/tikv2:/data/tikv

  tikv3:
    image: ${TIKV_IMAGE}
    container_name: zygarde-tidb-tikv3
    restart: unless-stopped
    depends_on: [pd1, pd2, pd3]
    command: [
      "--pd=pd1:2379,pd2:2379,pd3:2379",
      "--addr=0.0.0.0:20160",
      "--advertise-addr=tikv3:20160",
      "--data-dir=/data/tikv"
    ]
    volumes:
      - ./data/tikv3:/data/tikv

  tidb1:
    image: ${TIDB_IMAGE}
    container_name: zygarde-tidb1
    restart: unless-stopped
    depends_on: [tikv1, tikv2, tikv3]
    command: [
      "--store=tikv",
      "--path=pd1:2379,pd2:2379,pd3:2379",
      "--host=0.0.0.0",
      "--status=10080",
      "--advertise-address=tidb1"
    ]
    ports:
      - "\${TIDB1_PORT:-4000}:4000"
      - "\${TIDB1_STATUS_PORT:-10080}:10080"

  tidb2:
    image: ${TIDB_IMAGE}
    container_name: zygarde-tidb2
    restart: unless-stopped
    depends_on: [tikv1, tikv2, tikv3]
    command: [
      "--store=tikv",
      "--path=pd1:2379,pd2:2379,pd3:2379",
      "--host=0.0.0.0",
      "--status=10080",
      "--advertise-address=tidb2"
    ]
    ports:
      - "\${TIDB2_PORT:-4001}:4000"
      - "\${TIDB2_STATUS_PORT:-10081}:10080"
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
TIDB_VERSION=${VERSION_ARG}
PD1_PORT=2379
PD2_PORT=2479
PD3_PORT=2579
TIDB1_PORT=4000
TIDB2_PORT=4001
TIDB1_STATUS_PORT=10080
TIDB2_STATUS_PORT=10081
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

echo "[1/5] Starting TiDB cluster (3PD + 3TiKV + 2TiDB)..."
"${COMPOSE_CMD[@]}" up -d

wait_running() {
  local name="$1"
  for _ in $(seq 1 90); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "running" ]; then return 0; fi
    sleep 2
  done
  return 1
}

for node in zygarde-tidb-pd1 zygarde-tidb-pd2 zygarde-tidb-pd3 zygarde-tidb-tikv1 zygarde-tidb-tikv2 zygarde-tidb-tikv3 zygarde-tidb1 zygarde-tidb2; do
  echo "[2/5] Waiting $node running..."
  wait_running "$node" || { "${COMPOSE_CMD[@]}" logs; exit 1; }
done

echo "[3/5] Waiting tidb1 status endpoint..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${TIDB1_STATUS_PORT:-10080}/status" >/dev/null 2>&1; then break; fi
  sleep 2
done

echo "[4/5] Waiting tidb2 status endpoint..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${TIDB2_STATUS_PORT:-10081}/status" >/dev/null 2>&1; then break; fi
  sleep 2
done

echo "[5/5] Waiting PD cluster member count == 3..."
ok=0
for _ in $(seq 1 120); do
  cnt="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/members" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d.get("members",[])))' 2>/dev/null || echo 0)"
  if [ "${cnt:-0}" -ge 3 ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "PD cluster members not ready" >&2; "${COMPOSE_CMD[@]}" logs pd1 pd2 pd3 || true; exit 1; }

echo "TiDB cluster is ready."
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail

if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/6] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-tidb-pd[123]|zygarde-tidb-tikv[123]|zygarde-tidb[12]$/'

echo "[2/6] TiDB status endpoints"
curl -fsS "http://127.0.0.1:${TIDB1_STATUS_PORT:-10080}/status"
echo ""
curl -fsS "http://127.0.0.1:${TIDB2_STATUS_PORT:-10081}/status"
echo ""

echo "[3/6] PD health(3 members)"
health_json="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/health")"
echo "$health_json"
python3 - <<'PY' "$health_json"
import json,sys
h=json.loads(sys.argv[1])
if len(h)<3 or not all(x.get('health') for x in h):
  raise SystemExit('PD health check failed')
print('pd_health_ok=true')
PY

echo "[4/6] PD members + leader"
members_json="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/members")"
echo "$members_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("members=",len(d.get("members",[])),"leader=",(d.get("leader") or {}).get("name"));'

echo "[5/6] TiKV stores"
stores="$(curl -fsS "http://127.0.0.1:${PD1_PORT:-2379}/pd/api/v1/stores" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("count",0))')"
[ "${stores:-0}" -ge 3 ] || { echo "TiKV store count < 3" >&2; exit 1; }
echo "store_count=${stores}"

echo "[6/6] TiDB SQL ports open"
for p in "${TIDB1_PORT:-4000}" "${TIDB2_PORT:-4001}"; do
  if (exec 3<>/dev/tcp/127.0.0.1/$p) 2>/dev/null; then
    echo "tidb sql port $p is reachable"
    exec 3>&-
  else
    echo "tidb sql port $p is not reachable" >&2
    exit 1
  fi
done
CHECK_CLUSTER_EOF
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

3 PD + 3 TiKV + 2 TiDB 的最小高可用集群，适用于本地多节点拓扑验证。

## 稳定性说明

- 版本固定：\`pingcap/*:${IMAGE_TAG}\`（对外语义版本为 \`${VERSION_ARG}\`，当前默认映射到可用镜像 tag）。
- PD 使用声明式初始集群参数，避免运行时手工 join。
- build 阶段强等待：所有容器 running + 双 TiDB status endpoint + PD members 收敛到 3。
- check 阶段覆盖：容器状态、双 TiDB 状态、PD health、PD leader、TiKV store 数量、双 SQL 端口探活。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "TiDB $SCENARIO $VERSION_ARG generation complete!"