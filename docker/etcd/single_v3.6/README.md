# etcd single v3.6

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

etcd 单节点（开发联调）

## 稳定性说明

- 使用 `quay.io/coreos/etcd:v3.6.0`。
- build 以 `etcdctl endpoint health` 为就绪信号。
- check 覆盖 endpoint/member/KV 读写链路。
