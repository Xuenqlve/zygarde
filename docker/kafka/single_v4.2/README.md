# Kafka single v4.2

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Kafka KRaft 单节点（broker+controller 合并）

## 稳定性说明

- 使用 `apache/kafka:4.2.0` 镜像。
- 验收统一走 `build.sh -> check.sh -> cleanup`。
- check 包含 topic 创建 + 生产消费烟测。
