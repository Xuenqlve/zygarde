# Elasticsearch

本文介绍如何在 Zygarde 中使用 Elasticsearch 的 Compose 模板。

## Elasticsearch Single

适用范围：

- `middleware: elasticsearch`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: elasticsearch-demo
version: "v1"

services:
  - name: es-1
    middleware: elasticsearch
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Elasticsearch 的 Compose service 名称。 | 默认等于 `service.name` | 任意非空字符串 |
| `container_name` | 容器名称。 | 默认等于 `service_name` | 任意非空字符串 |
| `image` | 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v8.18` 时为 `docker.elastic.co/elasticsearch/elasticsearch:8.18.0`；`version=v8.19` 时为 `docker.elastic.co/elasticsearch/elasticsearch:8.19.0` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 数据目录，会挂载到容器内 `/usr/share/elasticsearch/data`。 | `./data/<service_name>` | 任意非空字符串路径 |
| `http_port` | Elasticsearch HTTP 端口。 | `9210` | 正整数端口 |
| `transport_port` | Elasticsearch transport 端口。 | `9300` | 正整数端口 |
| `version` | Elasticsearch 版本选择。 | `v8.19` | `v8.18`、`v8.19` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v8.18` | `docker.elastic.co/elasticsearch/elasticsearch:8.18.0` | 按 `docker/elasticsearch/single_v8.18` 已验证模板收敛 |
| `v8.19` | `docker.elastic.co/elasticsearch/elasticsearch:8.19.0` | 按 `docker/elasticsearch/single_v8.19` 已验证模板收敛 |

### 固定行为

- 固定使用单节点模式：`discovery.type=single-node`
- 固定关闭安全：`xpack.security.enabled=false`
- 固定节点名：`node.name=es1`
- 固定 JVM 参数：`ES_JAVA_OPTS=-Xms512m -Xmx512m`
- 数据目录挂载到 `/usr/share/elasticsearch/data`
- `doctor` 会检查：
  - `_cluster/health`
  - 版本信息
  - 索引创建、写入和读取链路

### 推荐写法

```yaml
name: elasticsearch-demo
version: "v1"

runtime:
  project-name: elasticsearch-demo

services:
  - name: es-1
    middleware: elasticsearch
    template: single
    values:
      version: v8.19
      http_port: 9210
      transport_port: 9300
      data_dir: ./data/es
```

### 使用建议

- 常规场景只需要关心 `http_port`、`transport_port` 和 `data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name` 和 `container_name` 没有明确需求时保持默认即可

## Elasticsearch Cluster

适用范围：

- `middleware: elasticsearch`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: elasticsearch-cluster-demo
version: "v1"

services:
  - name: es-cluster
    middleware: elasticsearch
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | 集群节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v8.18` 时为 `docker.elastic.co/elasticsearch/elasticsearch:8.18.0`；`version=v8.19` 时为 `docker.elastic.co/elasticsearch/elasticsearch:8.19.0` | 任意非空字符串；推荐与 `version` 对应 |
| `es1_service_name` | ES1 的 Compose service 名称。 | `<service.name>-es1` | 任意非空字符串 |
| `es2_service_name` | ES2 的 Compose service 名称。 | `<service.name>-es2` | 任意非空字符串 |
| `es3_service_name` | ES3 的 Compose service 名称。 | `<service.name>-es3` | 任意非空字符串 |
| `es1_container_name` | ES1 容器名称。 | 默认等于 `es1_service_name` | 任意非空字符串 |
| `es2_container_name` | ES2 容器名称。 | 默认等于 `es2_service_name` | 任意非空字符串 |
| `es3_container_name` | ES3 容器名称。 | 默认等于 `es3_service_name` | 任意非空字符串 |
| `es1_data_dir` | ES1 数据目录，会挂载到容器内 `/usr/share/elasticsearch/data`。 | `./data/<es1_service_name>` | 任意非空字符串路径 |
| `es2_data_dir` | ES2 数据目录，会挂载到容器内 `/usr/share/elasticsearch/data`。 | `./data/<es2_service_name>` | 任意非空字符串路径 |
| `es3_data_dir` | ES3 数据目录，会挂载到容器内 `/usr/share/elasticsearch/data`。 | `./data/<es3_service_name>` | 任意非空字符串路径 |
| `es1_http_port` | ES1 HTTP 端口。 | `9220` | 正整数端口 |
| `es2_http_port` | ES2 HTTP 端口。 | `9221` | 正整数端口 |
| `es3_http_port` | ES3 HTTP 端口。 | `9222` | 正整数端口 |
| `version` | Elasticsearch 版本选择。 | `v8.19` | `v8.18`、`v8.19` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v8.18` | `docker.elastic.co/elasticsearch/elasticsearch:8.18.0` | 按 `docker/elasticsearch/cluster_v8.18` 已验证模板收敛 |
| `v8.19` | `docker.elastic.co/elasticsearch/elasticsearch:8.19.0` | 按 `docker/elasticsearch/cluster_v8.19` 已验证模板收敛 |

### 固定行为

- 当前模板固定创建 `3` 个 Elasticsearch 节点
- 集群名固定为 `zygarde-es`
- 节点发现固定使用 3 节点 `discovery.seed_hosts`
- 初始 master 节点固定为 `es1,es2,es3`
- 固定关闭安全：`xpack.security.enabled=false`
- 固定 JVM 参数：`ES_JAVA_OPTS=-Xms512m -Xmx512m`
- 模板只对外暴露 3 个节点的 HTTP 端口，不单独暴露 transport host 端口
- `doctor` 会检查：
  - `_cluster/health` 中的节点数
  - `_cat/nodes`
  - 跨节点索引写入和读取链路

### 推荐写法

```yaml
name: elasticsearch-cluster-demo
version: "v1"

runtime:
  project-name: elasticsearch-cluster-demo

services:
  - name: es-cluster
    middleware: elasticsearch
    template: cluster
    values:
      version: v8.19
      es1_http_port: 9220
      es2_http_port: 9221
      es3_http_port: 9222
      es1_data_dir: ./data/es1
      es2_data_dir: ./data/es2
      es3_data_dir: ./data/es3
```

### 使用建议

- 常规场景只需要关心 3 个 HTTP 端口和 3 个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 3 个 service 名称和容器名没有明确需求时保持默认即可
