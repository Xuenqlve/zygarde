# ZooKeeper single v3.8

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

ZooKeeper 单节点

## 稳定性说明

- 使用 `zookeeper:3.8`。
- build 以 `ruok=imok` 作为可用信号。
- check 覆盖 4lw 命令 + znode 创建读取链路。
