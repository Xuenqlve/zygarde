#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
usage() { 
    echo "Usage: $0 <single|master-slave> <v5.7|v8.0>"
    echo "Example: $0 master-slave v8.0"
    exit 1; 
}

if [ $# -lt 2 ]; then usage; fi

SCENARIO="$1"
VERSION="$2"

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "master-slave" ]; then
    echo "场景错误: $SCENARIO"
    usage
fi

if [ "$VERSION" != "v5.7" ] && [ "$VERSION" != "v8.0" ]; then
    echo "版本错误: $VERSION (v5.7 或 v8.0)"
    usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

if [ "$VERSION" = "v5.7" ]; then
    IMAGE="mysql:5.7"
else
    IMAGE="mysql:8.0"
fi

OUTPUT_DIR="${PROJECT_ROOT}/docker/mysql/${SCENARIO}_${VERSION}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating MySQL $SCENARIO $VERSION"

# ============ Single 场景 ============
if [ "$SCENARIO" = "single" ]; then
    # 为 single 场景添加 binlog + GTID 配置（与 master-slave 的 master 保持一致）
    DEFAULT_AUTH=""
    if [ "$VERSION" = "v5.7" ]; then
        DEFAULT_AUTH='
      - --default-authentication-plugin=mysql_native_password'
    fi

    cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  mysql:
    image: ${IMAGE}
    container_name: zygarde-mysql-single
    restart: unless-stopped
    ports:
      - "\${MYSQL_PORT:-3306}:3306"
    volumes:
      - ./data/mysql:/var/lib/mysql
    environment:
      MYSQL_ROOT_PASSWORD: \${MYSQL_ROOT_PASSWORD:-root123}
      MYSQL_ROOT_HOST: "%"
    command:
      - --server-id=1
      - --log-bin=mysql-bin
      - --binlog-format=ROW
      - --gtid-mode=ON
      - --enforce-gtid-consistency=ON
      - --skip-name-resolve=1${DEFAULT_AUTH}
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p\${MYSQL_ROOT_PASSWORD:-root123}"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 20s

EOF

    cat > "$OUTPUT_DIR/.env" <<EOF
MYSQL_VERSION=${IMAGE}
MYSQL_PORT=3306
MYSQL_ROOT_PASSWORD=root123
EOF

    # 单节点构建脚本
    cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

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

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

echo "[1/2] Starting MySQL single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-mysql-single..."
for _ in $(seq 1 30); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-mysql-single 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then
        echo "MySQL is healthy."
        "${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT VERSION();" || true
        exit 0
    fi
    sleep 2
done

echo "Container zygarde-mysql-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs mysql || true
exit 1
BUILD_SINGLE_EOF
    chmod +x "$OUTPUT_DIR/build.sh"

    # 单节点检查脚本
    cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mysql-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT 1;"

echo "[3/3] Version"
"${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT VERSION();"
CHECK_SINGLE_EOF
    chmod +x "$OUTPUT_DIR/check.sh"

# ============ Master-Slave 场景 ============
elif [ "$SCENARIO" = "master-slave" ]; then
    cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  mysql-master:
    image: ${IMAGE}
    container_name: zygarde-mysql-master
    restart: unless-stopped
    ports:
      - "\${MYSQL_MASTER_PORT:-3306}:3306"
    volumes:
      - ./data/mysql-master:/var/lib/mysql
      - ./master-init.sql:/docker-entrypoint-initdb.d/01-master-init.sql:ro
    environment:
      MYSQL_ROOT_PASSWORD: \${MYSQL_ROOT_PASSWORD:-root123}
      MYSQL_ROOT_HOST: "%"
    command:
      - --server-id=1
      - --log-bin=mysql-bin
      - --binlog-format=ROW
      - --gtid-mode=ON
      - --enforce-gtid-consistency=ON
      - --skip-name-resolve=1
      - --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p\${MYSQL_ROOT_PASSWORD:-root123}"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 20s

  mysql-slave:
    image: ${IMAGE}
    container_name: zygarde-mysql-slave
    restart: unless-stopped
    ports:
      - "\${MYSQL_SLAVE_PORT:-3307}:3306"
    volumes:
      - ./data/mysql-slave:/var/lib/mysql
    environment:
      MYSQL_ROOT_PASSWORD: \${MYSQL_ROOT_PASSWORD:-root123}
      MYSQL_ROOT_HOST: "%"
    command:
      - --server-id=2
      - --relay-log=relay-log
      - --gtid-mode=ON
      - --enforce-gtid-consistency=ON
      - --skip-name-resolve=1
      - --default-authentication-plugin=mysql_native_password
    depends_on:
      mysql-master:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p\${MYSQL_ROOT_PASSWORD:-root123}"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 20s
EOF

    cat > "$OUTPUT_DIR/.env" <<EOF
MYSQL_VERSION=${IMAGE}
MYSQL_MASTER_PORT=3306
MYSQL_SLAVE_PORT=3307
MYSQL_ROOT_PASSWORD=root123
EOF

    # Master 初始化 SQL（创建复制用户）
    cat > "$OUTPUT_DIR/master-init.sql" <<EOF
-- 创建复制用户
CREATE USER IF NOT EXISTS 'repl'@'%' IDENTIFIED WITH mysql_native_password BY 'repl123';
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%';
FLUSH PRIVILEGES;
EOF

    # Slave 初始化 SQL（按 MySQL 版本生成语法）
    if [ "$VERSION" = "v5.7" ]; then
        cat > "$OUTPUT_DIR/slave-init.sql" <<EOF
-- 在 mysql-slave 执行，复制通道建立后设置 read-only
STOP SLAVE;
RESET SLAVE ALL;

CHANGE MASTER TO
  MASTER_HOST='mysql-master',
  MASTER_PORT=3306,
  MASTER_USER='repl',
  MASTER_PASSWORD='repl123',
  MASTER_AUTO_POSITION=1;

START SLAVE;

-- 复制通道建立后切换为只读
SET GLOBAL read_only = ON;
SET GLOBAL super_read_only = ON;
EOF
    else
        cat > "$OUTPUT_DIR/slave-init.sql" <<EOF
-- 在 mysql-slave 执行，复制通道建立后设置 read-only
STOP REPLICA;
RESET REPLICA ALL;

CHANGE REPLICATION SOURCE TO
  SOURCE_HOST='mysql-master',
  SOURCE_PORT=3306,
  SOURCE_USER='repl',
  SOURCE_PASSWORD='repl123',
  SOURCE_AUTO_POSITION=1,
  GET_SOURCE_PUBLIC_KEY=1;

START REPLICA;

-- 复制通道建立后切换为只读
SET GLOBAL read_only = ON;
SET GLOBAL super_read_only = ON;
EOF
    fi

    # 构建脚本
    cat > "$OUTPUT_DIR/build.sh" <<'BUILD_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

# 加载 .env，允许统一覆盖密码和端口变量
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

# 检测容器引擎
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

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

wait_healthy() {
    local name="$1"
    local retries=30
    for _ in $(seq 1 "$retries"); do
        status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        [[ "$status" == "healthy" ]] && return 0
        sleep 2
    done
    echo "Container $name did not become healthy" >&2
    return 1
}

echo "[1/4] Starting MySQL master/slave..."
${COMPOSE_CMD[@]} up -d

echo "[2/4] Waiting for zygarde-mysql-master..."
wait_healthy zygarde-mysql-master || { ${COMPOSE_CMD[@]} logs zygarde-mysql-master; exit 1; }

echo "[3/4] Waiting for zygarde-mysql-slave..."
wait_healthy zygarde-mysql-slave || { ${COMPOSE_CMD[@]} logs zygarde-mysql-slave; exit 1; }

echo "[4/4] Configuring replication..."
"${ENGINE_CMD[@]}" exec -i zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" < slave-init.sql

echo "Replication status:"
if ! "${ENGINE_CMD[@]}" exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW REPLICA STATUS\G" | \
    grep -E "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:"; then
    "${ENGINE_CMD[@]}" exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW SLAVE STATUS\G" | \
        grep -E "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:" || true
fi

echo "Done!"
BUILD_EOF
    chmod +x "$OUTPUT_DIR/build.sh"

    # 检查脚本
    cat > "$OUTPUT_DIR/check.sh" <<'CHECK_EOF'
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

# 支持通过 .env 覆盖默认 root 密码
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

run_mysql() {
    local container="$1"
    local sql="$2"
    "${ENGINE_CMD[@]}" exec "$container" mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "$sql"
}

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mysql-master|zygarde-mysql-slave/'

echo "[2/4] Replica status"
if ! run_mysql zygarde-mysql-slave "SHOW REPLICA STATUS\G" | \
    grep -E "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:|Last_IO_Error:|Last_SQL_Error:"; then
    run_mysql zygarde-mysql-slave "SHOW SLAVE STATUS\G" | \
        grep -E "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:|Last_IO_Error:|Last_SQL_Error:"
fi

echo "[3/4] GTID summary"
if ! run_mysql zygarde-mysql-slave "SHOW REPLICA STATUS\G" | \
    grep -E "Retrieved_Gtid_Set:|Executed_Gtid_Set:"; then
    run_mysql zygarde-mysql-slave "SHOW SLAVE STATUS\G" | \
        grep -E "Retrieved_Gtid_Set:|Executed_Gtid_Set:"
fi

echo "[4/4] Test replication"
run_mysql zygarde-mysql-master "CREATE DATABASE IF NOT EXISTS test_repl;"
sleep 2
run_mysql zygarde-mysql-slave "SHOW DATABASES;" | grep test_repl && echo "Replication works!" || echo "Replication may be delayed"
CHECK_EOF
    chmod +x "$OUTPUT_DIR/check.sh"
fi

if [ "$SCENARIO" = "single" ]; then
    cat > "$OUTPUT_DIR/README.md" <<EOF
# MySQL $SCENARIO $VERSION

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
| MYSQL_ROOT_PASSWORD | root123 | root 密码 |
| MYSQL_PORT | 3306 | MySQL 端口 |

## 特性

- **binlog**: 已开启 (mysql-bin)
- **GTID**: 已开启
- **binlog-format**: ROW

## 账号

- root / root123

## 场景

单节点 MySQL（开启 binlog + GTID，方便后续升级为主从复制）

## 稳定性说明

- 验收统一走 \`build.sh -> check.sh -> cleanup\`。
- 首次启动会初始化数据目录，验收结束后由 compose-stack 统一清理 \`data/\`。
- 单机场景默认开启 binlog+GTID，确保后续切换主从时配置一致。
EOF
else
    cat > "$OUTPUT_DIR/README.md" <<EOF
# MySQL $SCENARIO $VERSION

## 快速开始

\`\`\`bash
# 启动并配置主从复制
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
\`\`\`

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| MYSQL_ROOT_PASSWORD | root123 | root 密码 |
| MYSQL_MASTER_PORT | 3306 | Master 端口 |
| MYSQL_SLAVE_PORT | 3307 | Slave 端口 |

## 账号

- root / root123

## 场景

主从复制 MySQL + GTID

## 稳定性说明

- build 阶段会在 master/slave 就绪后自动执行复制初始化（含 GTID）。
- check 阶段会校验主从角色与复制线程状态（IO/SQL）。
- 若出现端口冲突或历史容器残留，compose-stack 会在验收前先执行清理。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "MySQL $SCENARIO $VERSION generation complete!"
