# ClickHouse single v25

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

ClickHouse 单节点

## 稳定性说明

- 使用 `clickhouse/clickhouse-server:25.8`。
- build 以 `clickhouse-client SELECT 1` 就绪信号判定。
- check 覆盖连接、版本、基础读写链路。
