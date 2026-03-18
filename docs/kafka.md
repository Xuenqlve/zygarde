# Kafka

本文介绍如何在 Zygarde 中使用 Kafka 的 Compose 模板。

## Kafka Single

适用范围：

- `middleware: kafka`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: kafka-demo
version: "v1"

services:
  - name: kafka-1
    middleware: kafka
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `kafka-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Kafka 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `kafka-<index>` 补齐 | 任意非空字符串 |
| `image` | Kafka 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `apache/kafka:4.2.0` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/var/lib/kafka/data`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `port` | 宿主机暴露的 Kafka Broker 端口。 | `9092` | 正整数端口 |
| `cluster_id` | Kafka KRaft 集群 ID。单机场景也需要显式提供，用于格式化元数据目录。 | `MkU3OEVBNTcwNTJENDM2Qk` | 任意非空字符串；推荐复用默认值或显式指定稳定值 |
| `version` | Kafka 版本选择。 | `v4.2` | `v4.2` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v4.2` | `apache/kafka:4.2.0` | 固定使用 Kafka 4.2 的单节点 KRaft 模式 |

### 固定行为

- 容器 hostname 固定为 `kafka`
- `restart` 固定为 `unless-stopped`
- 容器内 Broker 端口固定为 `9092`
- 容器内 Controller 端口固定为 `9093`
- 节点固定使用：
  - `KAFKA_NODE_ID=1`
  - `KAFKA_PROCESS_ROLES=broker,controller`
  - `KAFKA_CONTROLLER_QUORUM_VOTERS=1@kafka:9093`
- 数据卷固定挂载到 `/var/lib/kafka/data`
- 健康检查固定使用 `kafka-topics.sh --bootstrap-server localhost:9092 --list`
- `doctor` 会检查 broker API、创建主题、写入消息并校验 offset

### 推荐写法

```yaml
name: kafka-demo
version: "v1"

runtime:
  project-name: kafka-demo

services:
  - name: kafka-1
    middleware: kafka
    template: single
    values:
      version: v4.2
      port: 9092
      cluster_id: MkU3OEVBNTcwNTJENDM2Qk
      data_dir: ./data/kafka-1
```

### 使用建议

- 常规场景只需要关心 `port`、`cluster_id`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## Kafka Cluster

适用范围：

- `middleware: kafka`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: kafka-cluster-demo
version: "v1"

services:
  - name: kafka-cluster
    middleware: kafka
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | Kafka 集群节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `apache/kafka:4.2.0` | 任意非空字符串；推荐与 `version` 对应 |
| `cluster_id` | Kafka KRaft 集群 ID，3 个节点必须保持一致。 | `MkU3OEVBNTcwNTJENDM2Qk` | 任意非空字符串 |
| `kafka1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-kafka1` | 任意非空字符串 |
| `kafka2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-kafka2` | 任意非空字符串 |
| `kafka3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-kafka3` | 任意非空字符串 |
| `kafka1_container_name` | 节点 1 容器名称。 | 默认等于 `kafka1_service_name` | 任意非空字符串 |
| `kafka2_container_name` | 节点 2 容器名称。 | 默认等于 `kafka2_service_name` | 任意非空字符串 |
| `kafka3_container_name` | 节点 3 容器名称。 | 默认等于 `kafka3_service_name` | 任意非空字符串 |
| `kafka1_data_dir` | 节点 1 数据目录，会挂载到容器内 `/var/lib/kafka/data`。 | `./data/<kafka1_service_name>` | 任意非空字符串路径 |
| `kafka2_data_dir` | 节点 2 数据目录，会挂载到容器内 `/var/lib/kafka/data`。 | `./data/<kafka2_service_name>` | 任意非空字符串路径 |
| `kafka3_data_dir` | 节点 3 数据目录，会挂载到容器内 `/var/lib/kafka/data`。 | `./data/<kafka3_service_name>` | 任意非空字符串路径 |
| `kafka1_port` | 节点 1 对外 Broker 端口。 | `9092` | 正整数端口 |
| `kafka2_port` | 节点 2 对外 Broker 端口。 | `9094` | 正整数端口 |
| `kafka3_port` | 节点 3 对外 Broker 端口。 | `9096` | 正整数端口 |
| `version` | Kafka 版本选择。 | `v4.2` | `v4.2` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v4.2` | `apache/kafka:4.2.0` | 固定创建 3 节点 KRaft 集群 |

### 固定行为

- 当前模板固定创建 3 个 Kafka 节点
- 3 个节点的 hostname 固定为 `kafka1`、`kafka2`、`kafka3`
- 容器内 Broker 端口固定为 `9092`
- 容器内 Controller 端口固定为 `9093`
- `KAFKA_CONTROLLER_QUORUM_VOTERS` 固定为 `1@kafka1:9093,2@kafka2:9093,3@kafka3:9093`
- 节点 `KAFKA_ADVERTISED_LISTENERS` 固定指向自身 hostname
- 数据卷固定挂载到 `/var/lib/kafka/data`
- `doctor` 会检查 metadata quorum，并验证跨节点 produce / consume 烟测

### 推荐写法

```yaml
name: kafka-cluster-demo
version: "v1"

runtime:
  project-name: kafka-cluster-demo

services:
  - name: kafka-cluster
    middleware: kafka
    template: cluster
    values:
      version: v4.2
      cluster_id: MkU3OEVBNTcwNTJENDM2Qk
      kafka1_port: 9092
      kafka2_port: 9094
      kafka3_port: 9096
      kafka1_data_dir: ./data/kafka1
      kafka2_data_dir: ./data/kafka2
      kafka3_data_dir: ./data/kafka3
```

### 使用建议

- 常规场景只需要关心 3 个节点端口、`cluster_id` 和数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
