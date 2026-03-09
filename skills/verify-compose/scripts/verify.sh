#!/bin/bash

# verify-compose 入口脚本
# 根据 docker-compose.yml 内容自动选择验证脚本

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

usage() {
    echo "用法: $0 <目录路径>"
    echo "示例: $0 docker/mysql/single_v8.0"
    echo "      $0 docker/mysql/master-slave_v5.7"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

TARGET_DIR="$1"
if [ -d "$TARGET_DIR" ]; then
    FULL_PATH="$(cd "$TARGET_DIR" && pwd)"
else
    echo "[ERROR] 目录不存在: $TARGET_DIR"
    exit 1
fi

COMPOSE_FILE="$FULL_PATH/docker-compose.yml"
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "[ERROR] 未找到 docker-compose.yml: $COMPOSE_FILE"
    exit 1
fi

# 依据 compose 内容识别中间件类型，避免仅按路径字符串误判
if rg -qi "mysql" "$COMPOSE_FILE"; then
    echo "[INFO] 检测到 MySQL，使用 verify-mysql.sh"
    exec "$SCRIPT_DIR/verify-mysql.sh" "$FULL_PATH"
elif rg -qi "redis" "$COMPOSE_FILE"; then
    echo "[INFO] Redis 验证脚本开发中"
    exit 1
elif rg -qi "postgres|postgresql" "$COMPOSE_FILE"; then
    echo "[INFO] PostgreSQL 验证脚本开发中"
    exit 1
elif rg -qi "mongo|mongodb" "$COMPOSE_FILE"; then
    echo "[INFO] MongoDB 验证脚本开发中"
    exit 1
elif rg -qi "kafka" "$COMPOSE_FILE"; then
    echo "[INFO] Kafka 验证脚本开发中"
    exit 1
else
    echo "[ERROR] 无法从 compose 内容识别中间件类型: $COMPOSE_FILE"
    exit 1
fi
