# Elasticsearch single v8.19

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

Elasticsearch 单节点

## 稳定性说明

- 使用 `docker.elastic.co/elasticsearch/elasticsearch:8.19.0`。
- 默认关闭 security（便于本地联调）：`xpack.security.enabled=false`。
- build 以 `_cluster/health` 可访问作为可用信号。
- check 覆盖健康检查与索引写入读取链路。
