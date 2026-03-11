# ZooKeeper cluster v3.9

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

ZooKeeper 三节点集群

## 稳定性说明

- 使用 `zookeeper:3.9`。
- build 以 3 节点  作为收敛信号。
- check 覆盖节点健康、Mode 拓扑、跨节点 znode 读写。
