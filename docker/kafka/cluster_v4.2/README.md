# Kafka cluster v4.2

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Kafka KRaft 3 节点集群（broker+controller 合并）

## 稳定性说明

- 使用 `apache/kafka:4.2.0` 镜像。
- 采用 KRaft 模式，不依赖 ZooKeeper。
- check 包含 metadata quorum 状态 + 跨节点生产消费烟测。
