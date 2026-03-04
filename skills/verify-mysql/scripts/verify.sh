#!/bin/bash

# MySQL docker-compose 验证脚本
# 用法: ./verify.sh <目录路径>

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
MYSQL_ROOT_PASSWORD="root123"
MYSQL_USER="app"
MYSQL_PASSWORD="app123"
MAX_WAIT_TIME=120  # 最大等待时间（秒）

# 打印带颜色的消息
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
    echo -e "${GREEN}[✓]${NC} $1"
}

print_fail() {
    echo -e "${RED}[✗]${NC} $1"
}

# 显示用法
usage() {
    echo "用法: $0 <目录路径>"
    echo "示例: $0 docker/mysql/single/v8.0"
    exit 1
}

# 检查参数
if [ $# -lt 1 ]; then
    usage
fi

TARGET_DIR="$1"
PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FULL_PATH="${PROJECT_ROOT}/${TARGET_DIR}"

# 确保目录存在
if [ ! -d "$FULL_PATH" ]; then
    print_error "目录不存在: $FULL_PATH"
    exit 1
fi

# 确保是 docker-compose 目录
if [ ! -f "$FULL_PATH/docker-compose.yml" ]; then
    print_error "不是有效的 docker-compose 目录: $FULL_PATH"
    exit 1
fi

# 判断场景类型
detect_scenario() {
    local dir="$1"
    local dirname="$(basename "$dir")"
    local parentname="$(basename "$(dirname "$dir")")"
    
    # 先检查父目录
    if [[ "$parentname" == *"master-slave"* ]] || [[ "$dirname" == *"master"* ]]; then
        echo "master-slave"
    elif [[ "$dirname" == *"single-with-binlog"* ]] || [[ "$dirname" == *"binlog"* ]]; then
        echo "binlog"
    else
        echo "single"
    fi
}

SCENARIO=$(detect_scenario "$FULL_PATH")

echo "========================================"
echo "  MySQL 配置验证工具"
echo "========================================"
echo ""
print_info "目录: $TARGET_DIR"
print_info "场景: $SCENARIO"
echo ""

# 进入目标目录
cd "$FULL_PATH"

# 1. 语法检查
print_info "步骤 1: 语法检查..."
if docker-compose config > /dev/null 2>&1; then
    print_success "语法检查通过"
else
    print_error "语法检查失败"
    docker-compose config
    exit 1
fi
echo ""

# 2. 检查 Docker 是否运行
print_info "步骤 2: 检查 Docker..."
if ! docker info > /dev/null 2>&1; then
    print_error "Docker 未运行"
    exit 1
fi
print_success "Docker 运行中"
echo ""

# 3. 检查端口占用
print_info "步骤 3: 检查端口占用..."

# 获取需要检查的端口
if [ "$SCENARIO" = "master-slave" ]; then
    MASTER_PORT=$(grep -E "^MYSQL_MASTER_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "3306")
    SLAVE_PORT=$(grep -E "^MYSQL_SLAVE_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "3307")
    
    if lsof -i:$MASTER_PORT > /dev/null 2>&1; then
        print_warning "端口 $MASTER_PORT 已被占用"
    else
        print_success "端口 $MASTER_PORT 可用"
    fi
    
    if lsof -i:$SLAVE_PORT > /dev/null 2>&1; then
        print_warning "端口 $SLAVE_PORT 已被占用"
    else
        print_success "端口 $SLAVE_PORT 可用"
    fi
else
    MYSQL_PORT=$(grep -E "^MYSQL_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "3306")
    if lsof -i:$MYSQL_PORT > /dev/null 2>&1; then
        print_warning "端口 $MYSQL_PORT 已被占用"
    else
        print_success "端口 $MYSQL_PORT 可用"
    fi
fi
echo ""

# 4. 启动服务
print_info "步骤 4: 启动服务..."
docker-compose down -v > /dev/null 2>&1 || true
docker-compose up -d

# 5. 等待容器运行
print_info "步骤 5: 等待容器启动..."

CONTAINER_NAME=""
if [ "$SCENARIO" = "master-slave" ]; then
    CONTAINER_NAME="zygarde-mysql-master"
else
    CONTAINER_NAME="zygarde-mysql-single"
    if [ "$SCENARIO" = "binlog" ]; then
        CONTAINER_NAME="zygarde-mysql-binlog"
    fi
fi

WAITED=0
while [ $WAITED -lt $MAX_WAIT_TIME ]; do
    # 检查容器是否在运行
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        # 检查容器状态是否为 running
        STATUS=$(docker inspect --format='{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
        if [ "$STATUS" = "running" ]; then
            # 额外等待一下确保 MySQL 完全就绪
            sleep 10
            break
        fi
    fi
    
    sleep 2
    WAITED=$((WAITED + 2))
    
    # 显示进度
    if [ $((WAITED % 10)) -eq 0 ]; then
        print_info "等待中... ($WAITED/$MAX_WAIT_TIME 秒)"
    fi
done

if [ $WAITED -ge $MAX_WAIT_TIME ]; then
    print_error "容器启动超时"
    docker-compose logs
    docker-compose down -v
    exit 1
fi

print_success "容器启动成功"
echo ""

# 6. 功能验证
print_info "步骤 6: 功能验证..."

if [ "$SCENARIO" = "single" ]; then
    print_info "验证: SELECT VERSION()"
    VERSION=$(docker exec "$CONTAINER_NAME" mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SELECT VERSION();" 2>/dev/null | tail -n 1)
    if [ -n "$VERSION" ]; then
        print_success "MySQL 版本: $VERSION"
    else
        print_error "无法获取版本"
        docker-compose logs
        docker-compose down -v
        exit 1
    fi

elif [ "$SCENARIO" = "binlog" ]; then
    print_info "验证: SHOW MASTER STATUS"
    MASTER_STATUS=$(docker exec "$CONTAINER_NAME" mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW MASTER STATUS\G" 2>/dev/null)
    if echo "$MASTER_STATUS" | grep -q "File:"; then
        print_success "Binlog 已开启"
        echo "$MASTER_STATUS" | grep -E "File:|Position:|Binlog_Do_DB:|Binlog_Ignore_DB:"
    else
        print_error "Binlog 未正常开启"
        docker-compose logs
        docker-compose down -v
        exit 1
    fi

elif [ "$SCENARIO" = "master-slave" ]; then
    # Master 验证
    print_info "验证 Master: SHOW MASTER STATUS"
    MASTER_STATUS=$(docker exec zygarde-mysql-master mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW MASTER STATUS\G" 2>/dev/null)
    if echo "$MASTER_STATUS" | grep -q "File:"; then
        print_success "Master 正常"
        echo "$MASTER_STATUS" | grep -E "File:|Position:"
    else
        print_error "Master 异常"
        docker-compose logs
        docker-compose down -v
        exit 1
    fi
    
    # Slave 验证
    print_info "验证 Slave: SHOW SLAVE STATUS"
    SLAVE_STATUS=$(docker exec zygarde-mysql-slave mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW SLAVE STATUS\G" 2>/dev/null)
    
    if echo "$SLAVE_STATUS" | grep -q "Slave_IO_Running: Yes"; then
        print_success "Slave IO 线程: Running"
    else
        print_error "Slave IO 线程未运行"
    fi
    
    if echo "$SLAVE_STATUS" | grep -q "Slave_SQL_Running: Yes"; then
        print_success "Slave SQL 线程: Running"
    else
        print_error "Slave SQL 线程未运行"
    fi
    
    # 检查延迟
    BEHIND=$(echo "$SLAVE_STATUS" | grep "Seconds_Behind_Master:" | awk '{print $2}')
    if [ "$BEHIND" = "NULL" ] || [ "$BEHIND" -eq 0 ]; then
        print_success "复制延迟: 无"
    else
        print_warning "复制延迟: ${BEHIND} 秒"
    fi
fi

echo ""

# 7. 清理
print_info "步骤 7: 清理环境..."
docker-compose down -v
print_success "清理完成"

echo ""
echo "========================================"
print_success "验证完成！"
echo "========================================"
