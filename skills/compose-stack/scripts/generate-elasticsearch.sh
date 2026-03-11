#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|cluster> <v8.18|v8.19>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "cluster" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "v8.18" ] && [ "$VERSION" != "v8.19" ]; then
  echo "版本错误: $VERSION (仅支持 v8.18|v8.19)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docker/elasticsearch/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

if [ "$VERSION" = "v8.18" ]; then
  DEFAULT_TAG="8.18.0"
else
  DEFAULT_TAG="8.19.0"
fi
IMAGE="${ELASTICSEARCH_IMAGE:-docker.elastic.co/elasticsearch/elasticsearch:${DEFAULT_TAG}}"

print_info "Generating Elasticsearch $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  es:
    image: ${IMAGE}
    container_name: zygarde-es-single
    restart: unless-stopped
    environment:
      - node.name=es1
      - discovery.type=single-node
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "\${ES_HTTP_PORT:-9200}:9200"
      - "\${ES_TRANSPORT_PORT:-9300}:9300"
    volumes:
      - ./data/es:/usr/share/elasticsearch/data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ELASTICSEARCH_VERSION=${VERSION}
ES_HTTP_PORT=9200
ES_TRANSPORT_PORT=9300
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"
[ -f ./.env ] && set -a && . ./.env && set +a

if command -v podman >/dev/null 2>&1; then
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting Elasticsearch single..."
mkdir -p ./data/es
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting Elasticsearch API ready..."
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/_cluster/health" >/dev/null 2>&1; then
    echo "Elasticsearch single is ready."
    exit 0
  fi
  sleep 2
done

echo "Elasticsearch single did not become ready" >&2
"${COMPOSE_CMD[@]}" logs es || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
[ -f ./.env ] && set -a && . ./.env && set +a
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-es-single/'

echo "[2/4] Cluster health"
HEALTH="$(curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/_cluster/health")"
echo "$HEALTH" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("status=",d.get("status"),"nodes=",d.get("number_of_nodes"));'

echo "[3/4] Version"
curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("version",{}).get("number","unknown"))'

echo "[4/4] Indexing smoke"
IDX="zygarde-smoke"
curl -fsS -X PUT "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}" >/dev/null
curl -fsS -X POST "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}/_doc/1" -H 'Content-Type: application/json' -d '{"msg":"ok"}' >/dev/null
curl -fsS "http://127.0.0.1:${ES_HTTP_PORT:-9200}/${IDX}/_doc/1" | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get("_source",{}).get("msg"); assert v=="ok", v; print("doc=ok")'
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Elasticsearch ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Elasticsearch 单节点

## 稳定性说明

- 使用 \`${IMAGE}\`。
- 默认关闭 security（便于本地联调）：\`xpack.security.enabled=false\`。
- build 以 \`_cluster/health\` 可访问作为可用信号。
- check 覆盖健康检查与索引写入读取链路。
EOF

else
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  es1:
    image: ${IMAGE}
    container_name: zygarde-es-1
    restart: unless-stopped
    environment:
      - node.name=es1
      - cluster.name=zygarde-es
      - discovery.seed_hosts=es1,es2,es3
      - cluster.initial_master_nodes=es1,es2,es3
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "\${ES1_HTTP_PORT:-9200}:9200"
    volumes:
      - ./data/es1:/usr/share/elasticsearch/data

  es2:
    image: ${IMAGE}
    container_name: zygarde-es-2
    restart: unless-stopped
    environment:
      - node.name=es2
      - cluster.name=zygarde-es
      - discovery.seed_hosts=es1,es2,es3
      - cluster.initial_master_nodes=es1,es2,es3
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "\${ES2_HTTP_PORT:-9201}:9200"
    volumes:
      - ./data/es2:/usr/share/elasticsearch/data

  es3:
    image: ${IMAGE}
    container_name: zygarde-es-3
    restart: unless-stopped
    environment:
      - node.name=es3
      - cluster.name=zygarde-es
      - discovery.seed_hosts=es1,es2,es3
      - cluster.initial_master_nodes=es1,es2,es3
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "\${ES3_HTTP_PORT:-9202}:9200"
    volumes:
      - ./data/es3:/usr/share/elasticsearch/data
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
ELASTICSEARCH_VERSION=${VERSION}
ES1_HTTP_PORT=9200
ES2_HTTP_PORT=9201
ES3_HTTP_PORT=9202
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"
[ -f ./.env ] && set -a && . ./.env && set +a

if command -v podman >/dev/null 2>&1; then
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting Elasticsearch cluster..."
mkdir -p ./data/es1 ./data/es2 ./data/es3
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting cluster ready (nodes>=3)..."
for _ in $(seq 1 180); do
  NODES="$(curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cluster/health" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("number_of_nodes",0))' 2>/dev/null || echo 0)"
  if [ "${NODES:-0}" -ge 3 ]; then
    echo "Elasticsearch cluster is ready."
    exit 0
  fi
  sleep 2
done

echo "Elasticsearch cluster did not become ready" >&2
"${COMPOSE_CMD[@]}" logs || true
exit 1
BUILD_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_CLUSTER_EOF'
#!/usr/bin/env bash
set -euo pipefail
[ -f ./.env ] && set -a && . ./.env && set +a
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-es-[123]/'

echo "[2/4] Cluster health"
HEALTH="$(curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cluster/health")"
echo "$HEALTH"
NODES="$(echo "$HEALTH" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("number_of_nodes",0))')"
[ "$NODES" -ge 3 ] || { echo "nodes<3: $NODES" >&2; exit 1; }

echo "[3/4] Node list"
curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/_cat/nodes?v"

echo "[4/4] Indexing smoke"
IDX="zygarde-cluster-smoke"
curl -fsS -X PUT "http://127.0.0.1:${ES2_HTTP_PORT:-9201}/${IDX}" >/dev/null
curl -fsS -X POST "http://127.0.0.1:${ES3_HTTP_PORT:-9202}/${IDX}/_doc/1" -H 'Content-Type: application/json' -d '{"msg":"ok"}' >/dev/null
curl -fsS "http://127.0.0.1:${ES1_HTTP_PORT:-9200}/${IDX}/_doc/1" | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get("_source",{}).get("msg"); assert v=="ok", v; print("doc=ok")'
CHECK_CLUSTER_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# Elasticsearch ${SCENARIO} ${VERSION}

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Elasticsearch 三节点集群

## 稳定性说明

- 使用 \`${IMAGE}\`。
- 默认关闭 security（便于本地联调）。
- build 以 \`number_of_nodes >= 3\` 作为收敛信号。
- check 覆盖集群健康、节点列表与跨节点索引读写。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "Elasticsearch $SCENARIO $VERSION generation complete!"