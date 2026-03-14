#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v3.8|v3.9>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v3.8" ] && [ "$VERSION" != "v3.9" ]; then
  echo "版本错误: $VERSION (仅支持 v3.8|v3.9)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/zookeeper/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

if [ "$VERSION" = "v3.8" ]; then
  DEFAULT_TAG="3.8"
else
  DEFAULT_TAG="3.9"
fi
IMAGE="${ZOOKEEPER_IMAGE:-zookeeper:${DEFAULT_TAG}}"

print_info "Generating ZooKeeper $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  zk:
    image: ${IMAGE}
    container_name: zygarde-zk-single
    restart: unless-stopped
    ports:
      - "\${ZK_CLIENT_PORT:-2181}:2181"
      - "\${ZK_FOLLOWER_PORT:-2888}:2888"
      - "\${ZK_ELECTION_PORT:-3888}:3888"
    environment:
      ZOO_MY_ID: 1
      ZOO_4LW_COMMANDS_WHITELIST: ruok,mntr,srvr,stat,conf,isro
    volumes:
      - ./data/zk:/data
      - ./datalog/zk:/datalog
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ZOOKEEPER_VERSION=${VERSION}
ZK_CLIENT_PORT=2181
ZK_FOLLOWER_PORT=2888
ZK_ELECTION_PORT=3888
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

echo "[1/2] Starting ZooKeeper single..."
mkdir -p ./data/zk ./datalog/zk
rm -rf ./data/zk/* ./datalog/zk/*
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting zookeeper ruok=imok..."
for _ in $(seq 1 90); do
  if echo ruok | "${ENGINE_CMD[@]}" exec -i zygarde-zk-single /bin/bash -lc 'cat | nc 127.0.0.1 2181' 2>/dev/null | grep -q imok; then
    echo "ZooKeeper single is ready."
    exit 0
  fi
  sleep 2
done

echo "ZooKeeper single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs zk || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-zk-single/'

echo "[2/4] ruok"
R="$({ echo ruok | "${ENGINE_CMD[@]}" exec -i zygarde-zk-single /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\r')"
[ "$R" = "imok" ] || { echo "ruok failed: $R" >&2; exit 1; }

echo "[3/4] mntr"
"${ENGINE_CMD[@]}" exec zygarde-zk-single /bin/bash -lc "echo mntr | nc 127.0.0.1 2181 | grep -E 'zk_server_state|zk_version'"

echo "[4/4] create/get znode smoke"
"${ENGINE_CMD[@]}" exec zygarde-zk-single /bin/bash -lc "zkCli.sh -server 127.0.0.1:2181 create /zygarde_smoke ok >/tmp/zk.out 2>&1 || true"
OUT="$(${ENGINE_CMD[@]} exec zygarde-zk-single /bin/bash -lc "zkCli.sh -server 127.0.0.1:2181 get /zygarde_smoke 2>/dev/null | grep -E '^ok$' | head -n1" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "znode smoke failed: $OUT" >&2; exit 1; }
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# ZooKeeper ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

ZooKeeper 单节点

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 \`ruok=imok\` 作为可用信号。
- check 覆盖 4lw 命令 + znode 创建读取链路。
EOF

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  zk1:
    image: ${IMAGE}
    container_name: zygarde-zk-1
    restart: unless-stopped
    ports:
      - "\${ZK1_CLIENT_PORT:-2181}:2181"
    environment:
      ZOO_MY_ID: 1
      ZOO_SERVERS: server.1=zk1:2888:3888;2181 server.2=zk2:2888:3888;2181 server.3=zk3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: ruok,mntr,srvr,stat,conf,isro
    volumes:
      - ./data/zk1:/data
      - ./datalog/zk1:/datalog

  zk2:
    image: ${IMAGE}
    container_name: zygarde-zk-2
    restart: unless-stopped
    ports:
      - "\${ZK2_CLIENT_PORT:-2182}:2181"
    environment:
      ZOO_MY_ID: 2
      ZOO_SERVERS: server.1=zk1:2888:3888;2181 server.2=zk2:2888:3888;2181 server.3=zk3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: ruok,mntr,srvr,stat,conf,isro
    volumes:
      - ./data/zk2:/data
      - ./datalog/zk2:/datalog

  zk3:
    image: ${IMAGE}
    container_name: zygarde-zk-3
    restart: unless-stopped
    ports:
      - "\${ZK3_CLIENT_PORT:-2183}:2181"
    environment:
      ZOO_MY_ID: 3
      ZOO_SERVERS: server.1=zk1:2888:3888;2181 server.2=zk2:2888:3888;2181 server.3=zk3:2888:3888;2181
      ZOO_4LW_COMMANDS_WHITELIST: ruok,mntr,srvr,stat,conf,isro
    volumes:
      - ./data/zk3:/data
      - ./datalog/zk3:/datalog
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ZOOKEEPER_VERSION=${VERSION}
ZK1_CLIENT_PORT=2181
ZK2_CLIENT_PORT=2182
ZK3_CLIENT_PORT=2183
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

echo "[1/2] Starting ZooKeeper cluster..."
mkdir -p ./data/zk1 ./data/zk2 ./data/zk3 ./datalog/zk1 ./datalog/zk2 ./datalog/zk3
rm -rf ./data/zk1/* ./data/zk2/* ./data/zk3/* ./datalog/zk1/* ./datalog/zk2/* ./datalog/zk3/*
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster majority healthy..."
for _ in $(seq 1 120); do
  ok=0
  for n in 1 2 3; do
    if echo ruok | "${ENGINE_CMD[@]}" exec -i "zygarde-zk-$n" /bin/bash -lc 'cat | nc 127.0.0.1 2181' 2>/dev/null | grep -q imok; then
      ok=$((ok+1))
    fi
  done
  if [ "$ok" -ge 3 ]; then
    echo "ZooKeeper cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "ZooKeeper cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-zk-[123]/'

echo "[2/4] ruok each node"
for n in 1 2 3; do
  R="$({ echo ruok | "${ENGINE_CMD[@]}" exec -i "zygarde-zk-$n" /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\r')"
  [ "$R" = "imok" ] || { echo "zk$n ruok failed: $R" >&2; exit 1; }
  echo "zk$n=$R"
done

echo "[3/4] leader/follower topology"
ROLE_COUNT="$(${ENGINE_CMD[@]} exec zygarde-zk-1 /bin/bash -lc "for h in zk1 zk2 zk3; do echo stat | nc \$h 2181 | grep Mode; done" | tee /dev/stderr | wc -l | tr -d '[:space:]')"
[ "$ROLE_COUNT" -eq 3 ] || { echo "mode lines != 3" >&2; exit 1; }

echo "[4/4] znode cross-node smoke"
"${ENGINE_CMD[@]}" exec zygarde-zk-1 /bin/bash -lc "zkCli.sh -server zk1:2181 create /zygarde_cluster_smoke ok >/tmp/zk_cluster.out 2>&1 || true"
OUT="$(${ENGINE_CMD[@]} exec zygarde-zk-3 /bin/bash -lc "zkCli.sh -server zk3:2181 get /zygarde_cluster_smoke 2>/dev/null | grep -E '^ok$' | head -n1" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "cluster znode smoke failed: $OUT" >&2; exit 1; }
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# ZooKeeper ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

ZooKeeper 三节点集群

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 3 节点 `ruok=imok` 作为收敛信号。
- check 覆盖节点健康、Mode 拓扑、跨节点 znode 读写。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "ZooKeeper $SCENARIO $VERSION generation complete!"