# RabbitMQ cluster v4.2

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

3 节点 RabbitMQ 集群（classic_config 自动组网）

## 稳定性说明

- 使用 `rabbitmq:4.2-management` 镜像。
- 采用 `classic_config` 声明式集群发现，避免运行时 stop/reset/join 抖动。
- check 强校验 cluster_status 必须收敛到 rabbit1/rabbit2/rabbit3（含重试窗口）。
