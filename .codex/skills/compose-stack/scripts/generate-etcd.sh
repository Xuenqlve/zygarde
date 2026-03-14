#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v3.6>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v3.6" ]; then
  echo "版本错误: $VERSION (仅支持 v3.6)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/etcd/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

# etcd 3.6 语义版本，默认映射到可用 patch 版本
IMAGE="${ETCD_IMAGE:-quay.io/coreos/etcd:v3.6.0}"

print_info "Generating etcd $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  etcd:
    image: ${IMAGE}
    container_name: zygarde-etcd-single
    restart: unless-stopped
    ports:
      - "\${ETCD_CLIENT_PORT:-2379}:2379"
      - "\${ETCD_PEER_PORT:-2380}:2380"
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
      ETCD_NAME: etcd
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd:2380
      ETCD_INITIAL_CLUSTER: etcd=http://etcd:2380
      ETCD_INITIAL_CLUSTER_STATE: new
      ETCD_INITIAL_CLUSTER_TOKEN: zygarde-etcd-single
    volumes:
      - ./data/etcd:/etcd-data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ETCD_VERSION=v3.6
ETCD_CLIENT_PORT=2379
ETCD_PEER_PORT=2380
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

echo "[1/2] Starting etcd single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting etcd endpoint healthy..."
for _ in $(seq 1 90); do
  if "${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 endpoint health >/dev/null 2>&1; then
    echo "etcd single is ready."
    exit 0
  fi
  sleep 2
done

echo "etcd single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs etcd || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-etcd-single/'

echo "[2/4] Endpoint health"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 endpoint health

echo "[3/4] Member list"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 member list

echo "[4/4] KV smoke"
KEY="zygarde-smoke-$(date +%s)"
VAL="ok-$(date +%s)"
"${ENGINE_CMD[@]}" exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 put "$KEY" "$VAL" >/dev/null
OUT="$(${ENGINE_CMD[@]} exec zygarde-etcd-single etcdctl --endpoints=http://127.0.0.1:2379 get "$KEY" --print-value-only | tr -d '\r')"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# etcd ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

etcd 单节点（开发联调）

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 \`etcdctl endpoint health\` 为就绪信号。
- check 覆盖 endpoint/member/KV 读写链路。
EOF

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  etcd1:
    image: ${IMAGE}
    container_name: zygarde-etcd-1
    restart: unless-stopped
    ports:
      - "\${ETCD1_CLIENT_PORT:-2379}:2379"
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
      ETCD_NAME: etcd1
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd1:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd1:2380
      ETCD_INITIAL_CLUSTER: etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      ETCD_INITIAL_CLUSTER_STATE: new
      ETCD_INITIAL_CLUSTER_TOKEN: zygarde-etcd-cluster
    volumes:
      - ./data/etcd1:/etcd-data

  etcd2:
    image: ${IMAGE}
    container_name: zygarde-etcd-2
    restart: unless-stopped
    ports:
      - "\${ETCD2_CLIENT_PORT:-2479}:2379"
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
      ETCD_NAME: etcd2
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd2:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd2:2380
      ETCD_INITIAL_CLUSTER: etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      ETCD_INITIAL_CLUSTER_STATE: new
      ETCD_INITIAL_CLUSTER_TOKEN: zygarde-etcd-cluster
    volumes:
      - ./data/etcd2:/etcd-data

  etcd3:
    image: ${IMAGE}
    container_name: zygarde-etcd-3
    restart: unless-stopped
    ports:
      - "\${ETCD3_CLIENT_PORT:-2579}:2379"
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
      ETCD_NAME: etcd3
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd3:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd3:2380
      ETCD_INITIAL_CLUSTER: etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      ETCD_INITIAL_CLUSTER_STATE: new
      ETCD_INITIAL_CLUSTER_TOKEN: zygarde-etcd-cluster
    volumes:
      - ./data/etcd3:/etcd-data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ETCD_VERSION=v3.6
ETCD1_CLIENT_PORT=2379
ETCD2_CLIENT_PORT=2479
ETCD3_CLIENT_PORT=2579
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

echo "[1/2] Starting etcd cluster..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster endpoint health..."
for _ in $(seq 1 120); do
  if "${ENGINE_CMD[@]}" exec zygarde-etcd-1 etcdctl --endpoints=http://etcd1:2379,http://etcd2:2379,http://etcd3:2379 endpoint health >/dev/null 2>&1; then
    echo "etcd cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "etcd cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

ENDPOINTS="http://etcd1:2379,http://etcd2:2379,http://etcd3:2379"

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-etcd-[123]/'

echo "[2/4] Endpoint health"
"${ENGINE_CMD[@]}" exec zygarde-etcd-1 etcdctl --endpoints="$ENDPOINTS" endpoint health

echo "[3/4] Member list"
MEMBERS="$(${ENGINE_CMD[@]} exec zygarde-etcd-1 etcdctl --endpoints="$ENDPOINTS" member list | tee /dev/stderr | wc -l | tr -d '[:space:]')"
[ "$MEMBERS" -ge 3 ] || { echo "member count < 3" >&2; exit 1; }

echo "[4/4] KV smoke"
KEY="zygarde-cluster-smoke-$(date +%s)"
VAL="ok-$(date +%s)"
"${ENGINE_CMD[@]}" exec zygarde-etcd-2 etcdctl --endpoints="$ENDPOINTS" put "$KEY" "$VAL" >/dev/null
OUT="$(${ENGINE_CMD[@]} exec zygarde-etcd-3 etcdctl --endpoints="$ENDPOINTS" get "$KEY" --print-value-only | tr -d '\r')"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# etcd ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

etcd 3 节点集群（最小高可用）

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 3 节点 endpoint health 作为收敛信号。
- check 强校验 member 数量 + 跨节点 KV 读写链路。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "etcd $SCENARIO $VERSION generation complete!"