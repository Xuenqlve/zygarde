#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v1.20>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v1.20" ]; then
  echo "版本错误: $VERSION (仅支持 v1.20)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/consul/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

IMAGE="${CONSUL_IMAGE:-hashicorp/consul:1.20}"

print_info "Generating Consul $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  consul:
    image: ${IMAGE}
    container_name: zygarde-consul-single
    restart: unless-stopped
    command: [
      "agent",
      "-server",
      "-ui",
      "-node=consul1",
      "-bootstrap-expect=1",
      "-client=0.0.0.0",
      "-bind=0.0.0.0",
      "-data-dir=/consul/data"
    ]
    ports:
      - "\${CONSUL_HTTP_PORT:-8500}:8500"
      - "\${CONSUL_DNS_PORT:-8600}:8600/udp"
      - "\${CONSUL_SERVER_PORT:-8300}:8300"
    volumes:
      - ./data/consul:/consul/data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
CONSUL_VERSION=v1.20
CONSUL_HTTP_PORT=8500
CONSUL_DNS_PORT=8600
CONSUL_SERVER_PORT=8300
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

echo "[1/2] Starting Consul single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting consul API ready..."
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/status/leader" >/dev/null 2>&1; then
    echo "Consul single is ready."
    exit 0
  fi
  sleep 2
done

echo "Consul single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs consul || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-consul-single/'

echo "[2/4] Leader"
LEADER=""
for _ in $(seq 1 30); do
  LEADER="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/status/leader" 2>/dev/null | tr -d '"' || true)"
  if [ -n "$LEADER" ]; then break; fi
  sleep 1
done
[ -n "$LEADER" ] || { echo "leader is empty" >&2; exit 1; }
echo "leader=$LEADER"

echo "[3/4] Members"
MEMBERS_JSON="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/agent/members")"
echo "$MEMBERS_JSON" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("member_count=",len(d));'

echo "[4/4] KV smoke"
KEY="zygarde/smoke/$(date +%s)"
VAL="ok-$(date +%s)"
curl -fsS -X PUT --data "$VAL" "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/kv/$KEY" >/dev/null
OUT="$(curl -fsS "http://127.0.0.1:${CONSUL_HTTP_PORT:-8500}/v1/kv/$KEY?raw")"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Consul ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Consul 单节点（server + UI）

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 leader API 可返回作为就绪信号。
- check 覆盖 leader/member/KV 读写链路。
EOF

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  consul1:
    image: ${IMAGE}
    container_name: zygarde-consul-1
    restart: unless-stopped
    command: [
      "agent",
      "-server",
      "-ui",
      "-node=consul1",
      "-bootstrap-expect=3",
      "-retry-join=consul1",
      "-retry-join=consul2",
      "-retry-join=consul3",
      "-client=0.0.0.0",
      "-bind=0.0.0.0",
      "-data-dir=/consul/data"
    ]
    ports:
      - "\${CONSUL1_HTTP_PORT:-8500}:8500"
      - "\${CONSUL1_DNS_PORT:-8600}:8600/udp"
    volumes:
      - ./data/consul1:/consul/data

  consul2:
    image: ${IMAGE}
    container_name: zygarde-consul-2
    restart: unless-stopped
    command: [
      "agent",
      "-server",
      "-node=consul2",
      "-bootstrap-expect=3",
      "-retry-join=consul1",
      "-retry-join=consul2",
      "-retry-join=consul3",
      "-client=0.0.0.0",
      "-bind=0.0.0.0",
      "-data-dir=/consul/data"
    ]
    ports:
      - "\${CONSUL2_HTTP_PORT:-9500}:8500"
    volumes:
      - ./data/consul2:/consul/data

  consul3:
    image: ${IMAGE}
    container_name: zygarde-consul-3
    restart: unless-stopped
    command: [
      "agent",
      "-server",
      "-node=consul3",
      "-bootstrap-expect=3",
      "-retry-join=consul1",
      "-retry-join=consul2",
      "-retry-join=consul3",
      "-client=0.0.0.0",
      "-bind=0.0.0.0",
      "-data-dir=/consul/data"
    ]
    ports:
      - "\${CONSUL3_HTTP_PORT:-10500}:8500"
    volumes:
      - ./data/consul3:/consul/data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
CONSUL_VERSION=v1.20
CONSUL1_HTTP_PORT=8500
CONSUL1_DNS_PORT=8600
CONSUL2_HTTP_PORT=9500
CONSUL3_HTTP_PORT=10500
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

echo "[1/2] Starting Consul cluster..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster (leader + members=3)..."
for _ in $(seq 1 120); do
  leader="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/status/leader" 2>/dev/null | tr -d '"' || true)"
  members="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/agent/members" 2>/dev/null | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))' 2>/dev/null || echo 0)"
  if [ -n "$leader" ] && [ "${members:-0}" -ge 3 ]; then
    echo "Consul cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "Consul cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-consul-[123]/'

echo "[2/4] Leader + members"
LEADER="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/status/leader" | tr -d '"')"
[ -n "$LEADER" ] || { echo "leader is empty" >&2; exit 1; }
MEMBERS="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/agent/members" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[ "$MEMBERS" -ge 3 ] || { echo "members < 3" >&2; exit 1; }
echo "leader=$LEADER members=$MEMBERS"

echo "[3/4] Raft peers"
RAFT="$(curl -fsS "http://127.0.0.1:${CONSUL1_HTTP_PORT:-8500}/v1/operator/raft/configuration")"
echo "$RAFT" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("raft_servers=",len(d.get("Servers",[])));'

echo "[4/4] KV smoke"
KEY="zygarde/cluster/smoke/$(date +%s)"
VAL="ok-$(date +%s)"
curl -fsS -X PUT --data "$VAL" "http://127.0.0.1:${CONSUL2_HTTP_PORT:-9500}/v1/kv/$KEY" >/dev/null
OUT="$(curl -fsS "http://127.0.0.1:${CONSUL3_HTTP_PORT:-10500}/v1/kv/$KEY?raw")"
[ "$OUT" = "$VAL" ] || { echo "kv smoke failed: $OUT" >&2; exit 1; }
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Consul ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Consul 三节点 server 集群（含 UI）

## 稳定性说明

- 使用 \`${IMAGE}\`。
- build 以 leader 产生 + members>=3 作为收敛信号。
- check 覆盖 leader/member/raft peers/KV 跨节点链路。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "Consul $SCENARIO $VERSION generation complete!"