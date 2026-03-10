# MongoDB sharded v6.0

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Sharded 轻量版（6节点：3 config + 2 shard + 1 mongos）

## 稳定性说明

- build.sh 已内置稳定启动顺序：
  1) cfg/shard 就绪
  2) cfgRS 初始化并等待 PRIMARY
  3) shardRS 初始化并等待 PRIMARY
  4) mongos 就绪
  5) addShard
