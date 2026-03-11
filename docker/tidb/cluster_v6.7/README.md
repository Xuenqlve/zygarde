# TiDB cluster v6.7

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

3 PD + 3 TiKV + 2 TiDB 的最小高可用集群，适用于本地多节点拓扑验证。

## 稳定性说明

- 版本固定：`pingcap/*:v6.5.12`（对外语义版本为 `v6.7`，当前默认映射到可用镜像 tag）。
- PD 使用声明式初始集群参数，避免运行时手工 join。
- build 阶段强等待：所有容器 running + 双 TiDB status endpoint + PD members 收敛到 3。
- check 阶段覆盖：容器状态、双 TiDB 状态、PD health、PD leader、TiKV store 数量、双 SQL 端口探活。
