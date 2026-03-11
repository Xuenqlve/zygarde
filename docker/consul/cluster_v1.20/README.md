# Consul cluster v1.20

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Consul 三节点 server 集群（含 UI）

## 稳定性说明

- 使用 `hashicorp/consul:1.20`。
- build 以 leader 产生 + members>=3 作为收敛信号。
- check 覆盖 leader/member/raft peers/KV 跨节点链路。
