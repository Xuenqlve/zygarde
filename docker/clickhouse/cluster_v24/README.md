# ClickHouse cluster v24

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

ClickHouse 三节点集群（3 个 server 节点）

## 稳定性说明

- 使用 `clickhouse/clickhouse-server:24`。
- build 阶段强校验 3 节点均可执行 `SELECT 1`。
- check 阶段覆盖拓扑检测（system.clusters）与跨节点链路（remote）。
