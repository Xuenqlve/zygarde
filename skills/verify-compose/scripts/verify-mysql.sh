#!/bin/bash

# MySQL docker-compose 验证脚本
# 用法: ./verify-mysql.sh <目录路径>

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

MYSQL_ROOT_PASSWORD="root123"
MAX_WAIT_TIME=120

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

usage() {
    echo "用法: $0 <目录路径>"
    echo "示例: $0 docker/mysql/single_v8.0"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

TARGET_DIR="$1"
if [ -d "$TARGET_DIR" ]; then
    FULL_PATH="$(cd "$TARGET_DIR" && pwd)"
else
    print_error "目录不存在: $TARGET_DIR"
    exit 1
fi

if [ ! -f "$FULL_PATH/docker-compose.yml" ]; then
    print_error "不是有效的 docker-compose 目录: $FULL_PATH"
    exit 1
fi

cd "$FULL_PATH"

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

ENGINE_CMD=()
COMPOSE_CMD=()

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
fi

if [ "${ENGINE_CMD+x}" != "x" ]; then
    print_error "未检测到容器引擎 (docker/podman)"
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    print_error "未检测到 compose 命令"
    exit 1
fi

detect_scenario() {
    if rg -q "mysql-master|mysql-slave" docker-compose.yml; then
        echo "master-slave"
    else
        echo "single"
    fi
}

wait_for_mysql() {
    local container="$1"
    local retries=12
    local retry=0

    while [ "$retry" -lt "$retries" ]; do
        local status
        status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Status}}' "$container" 2>/dev/null || echo unknown)"
        if [ "$status" != "running" ]; then
            retry=$((retry + 1))
            print_info "容器 $container 状态: $status，等待中... ($retry/$retries)"
            sleep 10
            continue
        fi

        if "${ENGINE_CMD[@]}" exec "$container" mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT 1;" >/dev/null 2>&1; then
            return 0
        fi

        retry=$((retry + 1))
        print_info "MySQL $container 尚未就绪，等待中... ($retry/$retries)"
        sleep 10
    done

    return 1
}

cleanup() {
    print_info "步骤 7: 清理环境..."
    "${COMPOSE_CMD[@]}" down -v >/dev/null 2>&1 || true
    print_success "清理完成"
}

SCENARIO="$(detect_scenario)"
FAIL_COUNT=0

echo "========================================"
echo "  MySQL 配置验证工具"
echo "========================================"
print_info "目录: $FULL_PATH"
print_info "场景: $SCENARIO"
echo ""

print_info "步骤 1: 语法检查..."
if "${COMPOSE_CMD[@]}" config >/dev/null 2>&1; then
    print_success "语法检查通过"
else
    print_error "语法检查失败"
    "${COMPOSE_CMD[@]}" config
    exit 1
fi
echo ""

print_info "步骤 2: 检查容器引擎..."
if ! "${ENGINE_CMD[@]}" info >/dev/null 2>&1; then
    print_error "容器引擎不可用"
    exit 1
fi
print_success "容器引擎运行中"
echo ""

print_info "步骤 3: 检查端口占用..."
if [ "$SCENARIO" = "master-slave" ]; then
    MASTER_PORT="${MYSQL_MASTER_PORT:-3306}"
    SLAVE_PORT="${MYSQL_SLAVE_PORT:-3307}"

    if lsof -i:"$MASTER_PORT" >/dev/null 2>&1; then
        print_warning "端口 $MASTER_PORT 已被占用"
    else
        print_success "端口 $MASTER_PORT 可用"
    fi

    if lsof -i:"$SLAVE_PORT" >/dev/null 2>&1; then
        print_warning "端口 $SLAVE_PORT 已被占用"
    else
        print_success "端口 $SLAVE_PORT 可用"
    fi
else
    MYSQL_PORT="${MYSQL_PORT:-3306}"
    if lsof -i:"$MYSQL_PORT" >/dev/null 2>&1; then
        print_warning "端口 $MYSQL_PORT 已被占用"
    else
        print_success "端口 $MYSQL_PORT 可用"
    fi
fi
echo ""

print_info "步骤 4: 启动服务..."
"${COMPOSE_CMD[@]}" down -v >/dev/null 2>&1 || true
"${COMPOSE_CMD[@]}" up -d
echo ""

print_info "步骤 5: 等待容器与 MySQL 就绪..."
if [ "$SCENARIO" = "master-slave" ]; then
    if ! wait_for_mysql "zygarde-mysql-master"; then
        print_error "Master MySQL 未就绪"
        "${COMPOSE_CMD[@]}" logs
        cleanup
        exit 1
    fi
    if ! wait_for_mysql "zygarde-mysql-slave"; then
        print_error "Slave MySQL 未就绪"
        "${COMPOSE_CMD[@]}" logs
        cleanup
        exit 1
    fi
else
    if ! wait_for_mysql "zygarde-mysql-single"; then
        print_error "Single MySQL 未就绪"
        "${COMPOSE_CMD[@]}" logs
        cleanup
        exit 1
    fi
fi
print_success "MySQL 服务就绪"
echo ""

print_info "步骤 6: 功能验证..."

if [ "$SCENARIO" = "single" ]; then
    VERSION="$(${ENGINE_CMD[@]} exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT VERSION();" 2>/dev/null | tail -n 1 || true)"
    if [ -n "$VERSION" ]; then
        print_success "MySQL 版本: $VERSION"
    else
        print_error "无法获取 MySQL 版本"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
else
    MASTER_STATUS="$(${ENGINE_CMD[@]} exec zygarde-mysql-master mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW MASTER STATUS\\G" 2>/dev/null || true)"
    if echo "$MASTER_STATUS" | rg -q "File:"; then
        print_success "Master binlog 正常"
    else
        print_error "Master 状态异常"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    if [ -f "$FULL_PATH/slave-init.sql" ]; then
        print_info "执行 slave-init.sql 配置复制..."
        if ! "${ENGINE_CMD[@]}" exec -i zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" < "$FULL_PATH/slave-init.sql"; then
            print_error "slave-init.sql 执行失败"
            FAIL_COUNT=$((FAIL_COUNT + 1))
        fi
        sleep 3
    fi

    REPLICA_STATUS="$(${ENGINE_CMD[@]} exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW REPLICA STATUS\\G" 2>/dev/null || true)"
    if [ -z "$REPLICA_STATUS" ] || ! echo "$REPLICA_STATUS" | rg -q "Running:"; then
        REPLICA_STATUS="$(${ENGINE_CMD[@]} exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW SLAVE STATUS\\G" 2>/dev/null || true)"
    fi

    IO_RUNNING="$(echo "$REPLICA_STATUS" | rg -o "(Replica_IO_Running|Slave_IO_Running):\\s*\\w+" | tail -n 1 | awk -F': ' '{print $2}')"
    SQL_RUNNING="$(echo "$REPLICA_STATUS" | rg -o "(Replica_SQL_Running|Slave_SQL_Running):\\s*\\w+" | tail -n 1 | awk -F': ' '{print $2}')"

    if [ "$IO_RUNNING" = "Yes" ]; then
        print_success "Replica IO 线程运行正常"
    else
        print_error "Replica IO 线程未运行"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    if [ "$SQL_RUNNING" = "Yes" ]; then
        print_success "Replica SQL 线程运行正常"
    else
        print_error "Replica SQL 线程未运行"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    BEHIND="$(echo "$REPLICA_STATUS" | rg -o "(Seconds_Behind_Source|Seconds_Behind_Master):\\s*\\w+" | tail -n 1 | awk -F': ' '{print $2}')"
    if [ -z "$BEHIND" ] || [ "$BEHIND" = "NULL" ] || [ "$BEHIND" = "0" ]; then
        print_success "复制延迟: 无"
    else
        print_warning "复制延迟: ${BEHIND} 秒"
    fi
fi

echo ""
cleanup
echo ""
echo "========================================"
if [ "$FAIL_COUNT" -gt 0 ]; then
    print_fail "验证失败，发现 ${FAIL_COUNT} 个问题"
    echo "========================================"
    exit 1
fi
print_success "验证通过"
echo "========================================"
