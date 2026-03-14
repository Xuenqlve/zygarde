#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
usage() {
  echo "Usage: $0 <single|replica-set|sharded> <v6.0|v7.0>"
  echo "Example: $0 replica-set v7.0"
  exit 1
}

if [ $# -lt 2 ]; then usage; fi

SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "replica-set" ] && [ "$SCENARIO" != "sharded" ]; then
  echo "场景错误: $SCENARIO"
  usage
fi

if [ "$VERSION" != "v6.0" ] && [ "$VERSION" != "v7.0" ]; then
  echo "版本错误: $VERSION (v6.0 或 v7.0)"
  usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

if [ "$VERSION" = "v6.0" ]; then
  IMAGE="mongo:6.0"
else
  IMAGE="mongo:7.0"
fi

OUTPUT_DIR="${PROJECT_ROOT}/docker/mongodb/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating MongoDB $SCENARIO $VERSION"

# ============ Single 场景 ============
if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  mongodb:
    image: ${IMAGE}
    container_name: zygarde-mongodb-single
    restart: unless-stopped
    ports:
      - "\${MONGO_PORT:-27017}:27017"
    volumes:
      - ./data/mongodb:/data/db
    command:
      - mongod
      - --bind_ip_all
      - --dbpath
      - /data/db
    healthcheck:
      test: ["CMD", "mongosh", "--quiet", "--eval", "db.adminCommand({ ping: 1 }).ok"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 20s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
MONGO_VERSION=${IMAGE}
MONGO_PORT=27017
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

echo "[1/2] Starting MongoDB single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-mongodb-single..."
for _ in $(seq 1 30); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-mongodb-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "MongoDB is healthy."
    "${ENGINE_CMD[@]}" exec zygarde-mongodb-single mongosh --quiet --eval 'db.adminCommand({ ping: 1 })' || true
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-mongodb-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs mongodb || true
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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mongodb-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-mongodb-single mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'

echo "[3/3] Version"
"${ENGINE_CMD[@]}" exec zygarde-mongodb-single mongosh --quiet --eval 'db.version()'
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

# ============ Replica Set 场景 (1主2从) ============
elif [ "$SCENARIO" = "replica-set" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  mongo-rs1:
    image: ${IMAGE}
    container_name: zygarde-mongo-rs1
    restart: unless-stopped
    ports:
      - "\${MONGO_RS1_PORT:-27017}:27017"
    volumes:
      - ./data/mongo-rs1:/data/db
    command: ["mongod", "--replSet", "rs0", "--bind_ip_all", "--dbpath", "/data/db"]

  mongo-rs2:
    image: ${IMAGE}
    container_name: zygarde-mongo-rs2
    restart: unless-stopped
    ports:
      - "\${MONGO_RS2_PORT:-27018}:27017"
    volumes:
      - ./data/mongo-rs2:/data/db
    command: ["mongod", "--replSet", "rs0", "--bind_ip_all", "--dbpath", "/data/db"]

  mongo-rs3:
    image: ${IMAGE}
    container_name: zygarde-mongo-rs3
    restart: unless-stopped
    ports:
      - "\${MONGO_RS3_PORT:-27019}:27017"
    volumes:
      - ./data/mongo-rs3:/data/db
    command: ["mongod", "--replSet", "rs0", "--bind_ip_all", "--dbpath", "/data/db"]
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
MONGO_VERSION=${IMAGE}
MONGO_RS1_PORT=27017
MONGO_RS2_PORT=27018
MONGO_RS3_PORT=27019
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_RS_EOF'
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

echo "[1/4] Starting MongoDB replica-set nodes..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/4] Waiting for nodes ready..."
for c in zygarde-mongo-rs1 zygarde-mongo-rs2 zygarde-mongo-rs3; do
  ok=0
  for _ in $(seq 1 30); do
    if "${ENGINE_CMD[@]}" exec "$c" mongosh --quiet --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ] || { echo "$c not ready" >&2; exit 1; }
done

echo "[3/4] Initiating replica-set rs0..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval '
try {
  rs.initiate({_id:"rs0", members:[
    {_id:0, host:"mongo-rs1:27017"},
    {_id:1, host:"mongo-rs2:27017"},
    {_id:2, host:"mongo-rs3:27017"}
  ]})
} catch(e) {
  if (!e.message.includes("already initialized")) throw e;
}
'

echo "[4/4] Waiting for replica-set stable (PRIMARY+SECONDARY)..."
RS_PRIMARY_WAIT_SECONDS="${RS_PRIMARY_WAIT_SECONDS:-120}"
ATTEMPTS=$((RS_PRIMARY_WAIT_SECONDS / 2))
[ "$ATTEMPTS" -lt 1 ] && ATTEMPTS=1

check_rs_stable() {
  local line
  line="$(${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval '
try {
  var s=rs.status();
  var p=s.members.filter(m=>m.stateStr=="PRIMARY").length;
  var sec=s.members.filter(m=>m.stateStr=="SECONDARY").length;
  print(p+","+sec);
} catch(e) { print("0,0"); }
' 2>/dev/null | tail -n 1 || true)"

  local p sec
  p="${line%%,*}"
  sec="${line##*,}"
  [ -z "$p" ] && p=0
  [ -z "$sec" ] && sec=0

  if [ "$p" -ge 1 ] && [ "$sec" -ge 2 ]; then
    return 0
  fi
  return 1
}

ok=0
for _ in $(seq 1 "$ATTEMPTS"); do
  if check_rs_stable; then
    ok=1
    break
  fi
  sleep 2
done

# 兜底再给一轮等待（吸收偶发选主抖动）
if [ "$ok" -ne 1 ]; then
  for _ in $(seq 1 15); do
    if check_rs_stable; then
      ok=1
      break
    fi
    sleep 2
  done
fi

if [ "$ok" -ne 1 ]; then
  echo "Replica-set not stable (PRIMARY/SECONDARY)" >&2
  ${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval 'try{print(JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr,health:m.health}))))}catch(e){print(e.message)}' >&2 || true
  exit 1
fi

echo "Replica-set PRIMARY/SECONDARY is stable."
BUILD_RS_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_RS_EOF'
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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mongo-rs1|zygarde-mongo-rs2|zygarde-mongo-rs3/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'

echo "[3/4] Replica-set status"
"${ENGINE_CMD[@]}" exec zygarde-mongo-rs1 mongosh --quiet --eval 'JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr})))'

echo "[4/4] Primary check"
PRIMARY="$(${ENGINE_CMD[@]} exec zygarde-mongo-rs1 mongosh --quiet --eval 'rs.status().members.filter(m=>m.stateStr=="PRIMARY").length')"
if [ "$PRIMARY" -lt 1 ]; then
  echo "No PRIMARY found" >&2
  exit 1
fi
CHECK_RS_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

# ============ Sharded 轻量场景 (6节点) ============
elif [ "$SCENARIO" = "sharded" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  cfg1:
    image: ${IMAGE}
    container_name: zygarde-mongo-cfg1
    restart: unless-stopped
    volumes:
      - ./data/cfg1:/data/db
    command: ["mongod", "--configsvr", "--replSet", "cfgRS", "--bind_ip_all", "--port", "27019"]

  cfg2:
    image: ${IMAGE}
    container_name: zygarde-mongo-cfg2
    restart: unless-stopped
    volumes:
      - ./data/cfg2:/data/db
    command: ["mongod", "--configsvr", "--replSet", "cfgRS", "--bind_ip_all", "--port", "27019"]

  cfg3:
    image: ${IMAGE}
    container_name: zygarde-mongo-cfg3
    restart: unless-stopped
    volumes:
      - ./data/cfg3:/data/db
    command: ["mongod", "--configsvr", "--replSet", "cfgRS", "--bind_ip_all", "--port", "27019"]

  shard1:
    image: ${IMAGE}
    container_name: zygarde-mongo-shard1
    restart: unless-stopped
    volumes:
      - ./data/shard1:/data/db
    command: ["mongod", "--shardsvr", "--replSet", "shardRS", "--bind_ip_all", "--port", "27018"]

  shard2:
    image: ${IMAGE}
    container_name: zygarde-mongo-shard2
    restart: unless-stopped
    volumes:
      - ./data/shard2:/data/db
    command: ["mongod", "--shardsvr", "--replSet", "shardRS", "--bind_ip_all", "--port", "27018"]

  mongos:
    image: ${IMAGE}
    container_name: zygarde-mongos
    restart: unless-stopped
    depends_on:
      - cfg1
      - cfg2
      - cfg3
      - shard1
      - shard2
    ports:
      - "\${MONGOS_PORT:-27017}:27017"
    command: ["mongos", "--configdb", "cfgRS/cfg1:27019,cfg2:27019,cfg3:27019", "--bind_ip_all", "--port", "27017"]
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
MONGO_VERSION=${IMAGE}
MONGOS_PORT=27017
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SHARD_EOF'
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

wait_ping() {
  local c="$1"; local port="$2"; local retries="${3:-40}"
  local ok=0
  for _ in $(seq 1 "$retries"); do
    if "${ENGINE_CMD[@]}" exec "$c" mongosh --quiet --port "$port" --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1; break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ]
}

wait_rs_primary() {
  local c="$1"; local port="$2"; local retries="${3:-60}"
  for _ in $(seq 1 "$retries"); do
    state="$(${ENGINE_CMD[@]} exec "$c" mongosh --quiet --port "$port" --eval 'try{rs.status().myState}catch(e){0}' 2>/dev/null | tail -n 1 || true)"
    if [ "$state" = "1" ]; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "[1/6] Starting sharded topology..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/6] Waiting config/shard services ready..."
wait_ping zygarde-mongo-cfg1 27019 50 || { echo "cfg1 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-cfg2 27019 50 || { echo "cfg2 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-cfg3 27019 50 || { echo "cfg3 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-shard1 27018 50 || { echo "shard1 not ready" >&2; exit 1; }
wait_ping zygarde-mongo-shard2 27018 50 || { echo "shard2 not ready" >&2; exit 1; }

echo "[3/6] Initiating config replica-set..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-cfg1 mongosh --quiet --port 27019 --eval '
try {
  rs.initiate({_id:"cfgRS", configsvr:true, members:[
    {_id:0, host:"cfg1:27019"},
    {_id:1, host:"cfg2:27019"},
    {_id:2, host:"cfg3:27019"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary zygarde-mongo-cfg1 27019 70 || { echo "cfgRS primary not ready" >&2; exit 1; }

echo "[4/6] Initiating shard replica-set..."
"${ENGINE_CMD[@]}" exec zygarde-mongo-shard1 mongosh --quiet --port 27018 --eval '
try {
  rs.initiate({_id:"shardRS", members:[
    {_id:0, host:"shard1:27018"},
    {_id:1, host:"shard2:27018"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary zygarde-mongo-shard1 27018 70 || { echo "shardRS primary not ready" >&2; exit 1; }

echo "[5/6] Waiting mongos ready..."
wait_ping zygarde-mongos 27017 60 || { echo "mongos not ready" >&2; ${ENGINE_CMD[@]} logs zygarde-mongos >&2 || true; exit 1; }

echo "[6/6] Adding shard to mongos..."
"${ENGINE_CMD[@]}" exec zygarde-mongos mongosh --quiet --port 27017 --eval '
try {
  sh.addShard("shardRS/shard1:27018,shard2:27018")
} catch(e) {
  if (!(e.message.includes("already") || e.message.includes("exists"))) throw e;
}
sh.status()
'
BUILD_SHARD_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SHARD_EOF'
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
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mongo-cfg|zygarde-mongo-shard|zygarde-mongos/'

echo "[2/4] mongos ping"
"${ENGINE_CMD[@]}" exec zygarde-mongos mongosh --quiet --port 27017 --eval 'db.adminCommand({ ping: 1 })'

echo "[3/4] listShards"
SHARDS="$(${ENGINE_CMD[@]} exec zygarde-mongos mongosh --quiet --port 27017 --eval 'JSON.stringify(db.adminCommand({ listShards: 1 }))')"
echo "$SHARDS"
echo "$SHARDS" | grep -q '"ok":1' || { echo "listShards not ok" >&2; exit 1; }

echo "[4/4] shard count"
COUNT="$(${ENGINE_CMD[@]} exec zygarde-mongos mongosh --quiet --port 27017 --eval 'db.adminCommand({ listShards: 1 }).shards.length')"
if [ "$COUNT" -lt 1 ]; then
  echo "No shard found" >&2
  exit 1
fi
CHECK_SHARD_EOF
  chmod +x "$OUTPUT_DIR/check.sh"
fi

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/README.md" <<EOF
# MongoDB $SCENARIO $VERSION

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

单实例 MongoDB
EOF
elif [ "$SCENARIO" = "replica-set" ]; then
  cat > "$OUTPUT_DIR/README.md" <<EOF
# MongoDB $SCENARIO $VERSION

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Replica Set（1主2从）
EOF
else
  cat > "$OUTPUT_DIR/README.md" <<EOF
# MongoDB $SCENARIO $VERSION

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

Sharded 轻量版（6节点：3 config + 2 shard + 1 mongos）

## 稳定性说明

- build.sh 已内置稳定启动顺序：
  1) cfg/shard 就绪
  2) cfgRS 初始化并等待 PRIMARY
  3) shardRS 初始化并等待 PRIMARY
  4) mongos 就绪
  5) addShard
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "MongoDB $SCENARIO $VERSION generation complete!"
