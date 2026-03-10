# RabbitMQ single v4.2

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

单节点 RabbitMQ（含 Management 插件）

## 稳定性说明

- 使用 `rabbitmq:4.2-management` 镜像。
- 验收统一走 `build.sh -> check.sh -> cleanup`。
- 就绪判定采用 `rabbitmq-diagnostics -q ping` 健康检查。
