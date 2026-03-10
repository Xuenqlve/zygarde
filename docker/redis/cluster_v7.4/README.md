# Redis cluster v7.4

## 快速开始

```bash
# 启动并初始化集群
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
```

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| REDIS_NODE_1_PORT | 7001 | 节点1端口 |
| REDIS_NODE_2_PORT | 7002 | 节点2端口 |
| REDIS_NODE_3_PORT | 7003 | 节点3端口 |

## 场景

Redis Cluster（3主节点，无副本）

## 稳定性说明

- 集群创建后可能短暂出现 `cluster_state:fail`，check 内置重试等待收敛。
- 仅当最终 `cluster_state:ok` 才判定验收通过。

## 兼容性说明

- 集群初始化时，脚本使用容器 IP 建立集群（而非容器名），以兼容 Redis 6.2 在部分环境中的地址校验差异。
- `check.sh` 对 `cluster_state` 做强校验，最终必须为 `ok`。
