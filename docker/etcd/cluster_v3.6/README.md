# etcd cluster v3.6

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

etcd 3 节点集群（最小高可用）

## 稳定性说明

- 使用 `quay.io/coreos/etcd:v3.6.0`。
- build 以 3 节点 endpoint health 作为收敛信号。
- check 强校验 member 数量 + 跨节点 KV 读写链路。
