# Elasticsearch cluster v8.19

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Elasticsearch 三节点集群

## 稳定性说明

- 使用 `docker.elastic.co/elasticsearch/elasticsearch:8.19.0`。
- 默认关闭 security（便于本地联调）。
- build 以 `number_of_nodes >= 3` 作为收敛信号。
- check 覆盖集群健康、节点列表与跨节点索引读写。
