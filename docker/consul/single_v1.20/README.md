# Consul single v1.20

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Consul 单节点（server + UI）

## 稳定性说明

- 使用 `hashicorp/consul:1.20`。
- build 以 leader API 可返回作为就绪信号。
- check 覆盖 leader/member/KV 读写链路。
