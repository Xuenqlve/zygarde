#!/bin/bash
# docker-compose-generator 主入口脚本
# 根据中间件类型分发到专用脚本

set -euo pipefail

usage() {
    echo "docker-compose-generator - 中间件配置生成工具"
    echo ""
    echo "用法: $0 <中间件类型> [场景] [版本]"
    echo ""
    echo "支持的中间件:"
    echo "  mysql         - MySQL (single/master-slave), 版本必填: v5.7|v8.0"
    echo "  redis         - Redis (single/master-slave/cluster), 版本必填: v6.2|v7.4"
    echo "  mongodb       - MongoDB (single/replica-set/sharded), 版本必填: v6.0|v7.0"
    echo "  postgresql    - PostgreSQL (single/master-slave), 版本必填: v16|v17"
    echo "  tidb          - TiDB (single), 版本必填: v6.7"
    echo "  elasticsearch - Elasticsearch (single/cluster), 版本必填: v8.18|v8.19"
    echo "  kafka         - Kafka"
    echo "  rabbitmq      - RabbitMQ (single/cluster), 版本必填: v4.2"
    echo "  clickhouse    - ClickHouse (single/cluster), 版本必填: v24|v25"
    echo "  etcd          - etcd (single/cluster), 版本必填: v3.6"
    echo "  zookeeper     - ZooKeeper (single/cluster), 版本必填: v3.8|v3.9"
    echo "  consul        - Consul (single/cluster), 版本必填: v1.20"
    echo ""
    echo "示例:"
    echo "  $0 mysql single v8.0"
    echo "  $0 mysql master-slave v5.7"
    echo "  $0 redis single v6.2"
    echo "  $0 redis cluster v7.4"
    echo "  $0 mongodb replica-set v7.0"
    echo "  $0 postgresql master-slave v16"
    echo "  $0 rabbitmq single v4.2"
    echo "  $0 rabbitmq cluster v4.2"
    echo "  $0 tidb single v6.7"
    echo "  $0 etcd cluster v3.6"
    echo "  $0 consul cluster v1.20"
    echo "  $0 clickhouse cluster v25"
    echo "  $0 zookeeper cluster v3.9"
    echo "  $0 elasticsearch cluster v8.19"
}

if [ $# -lt 1 ]; then
    usage
    exit 1
fi

TYPE="$1"
SCENARIO="${2:-single}"
VERSION="${3:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 分发到专用脚本
case "$TYPE" in
    mysql)
        if [ -z "$VERSION" ]; then
            echo "错误: MySQL 需要指定版本 (v5.7 或 v8.0)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-mysql.sh" "$SCENARIO" "$VERSION"
        ;;
    redis)
        if [ -z "$VERSION" ]; then
            echo "错误: Redis 需要指定版本 (v6.2 或 v7.4)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-redis.sh" "$SCENARIO" "$VERSION"
        ;;
    mongodb)
        if [ -z "$VERSION" ]; then
            echo "错误: MongoDB 需要指定版本 (v6.0 或 v7.0)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-mongodb.sh" "$SCENARIO" "$VERSION"
        ;;
    postgresql)
        if [ -z "$VERSION" ]; then
            echo "错误: PostgreSQL 需要指定版本 (v16 或 v17)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-postgresql.sh" "$SCENARIO" "$VERSION"
        ;;
    rabbitmq)
        if [ -z "$VERSION" ]; then
            echo "错误: RabbitMQ 需要指定版本 (v4.2)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-rabbitmq.sh" "$SCENARIO" "$VERSION"
        ;;
    tidb)
        if [ -z "$VERSION" ]; then
            echo "错误: TiDB 需要指定版本 (v6.7)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-tidb.sh" "$SCENARIO" "$VERSION"
        ;;
    etcd)
        if [ -z "$VERSION" ]; then
            echo "错误: etcd 需要指定版本 (v3.6)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-etcd.sh" "$SCENARIO" "$VERSION"
        ;;
    consul)
        if [ -z "$VERSION" ]; then
            echo "错误: Consul 需要指定版本 (v1.20)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-consul.sh" "$SCENARIO" "$VERSION"
        ;;
    clickhouse)
        if [ -z "$VERSION" ]; then
            echo "错误: ClickHouse 需要指定版本 (v24|v25)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-clickhouse.sh" "$SCENARIO" "$VERSION"
        ;;
    zookeeper)
        if [ -z "$VERSION" ]; then
            echo "错误: ZooKeeper 需要指定版本 (v3.8|v3.9)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-zookeeper.sh" "$SCENARIO" "$VERSION"
        ;;
    elasticsearch)
        if [ -z "$VERSION" ]; then
            echo "错误: Elasticsearch 需要指定版本 (v8.18|v8.19)"
            usage
            exit 1
        fi
        exec "$SCRIPT_DIR/generate-elasticsearch.sh" "$SCENARIO" "$VERSION"
        ;;
    kafka)
        echo "该中间件脚本开发中..."
        exit 1
        ;;
    *)
        echo "不支持的中间件: $TYPE"
        exit 1
        ;;
esac
