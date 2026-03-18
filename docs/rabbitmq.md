# RabbitMQ

本文介绍如何在 Zygarde 中使用 RabbitMQ 的 Compose 模板。

## RabbitMQ Single

适用范围：

- `middleware: rabbitmq`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: rabbitmq-demo
version: "v1"

services:
  - name: rabbitmq-1
    middleware: rabbitmq
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `rabbitmq-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Docker 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `rabbitmq-<index>` 补齐 | 任意非空字符串 |
| `image` | RabbitMQ 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `rabbitmq:4.2-management` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/var/lib/rabbitmq`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `amqp_port` | 宿主机暴露的 AMQP 端口。 | `5672` | 正整数端口 |
| `management_port` | 宿主机暴露的管理端口。 | `15672` | 正整数端口 |
| `default_user` | 默认管理员用户名。 | `admin` | 任意非空字符串 |
| `default_pass` | 默认管理员密码。 | `admin123` | 任意非空字符串 |
| `erlang_cookie` | Erlang cookie，用于节点标识和后续集群扩展。 | `rabbitmq-cookie` | 任意非空字符串 |
| `version` | RabbitMQ 版本选择。 | `v4.2` | `v4.2` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v4.2` | `rabbitmq:4.2-management` | 固定启用管理插件镜像 |

### 固定行为

- 容器内 AMQP 端口固定为 `5672`
- 容器内管理端口固定为 `15672`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/var/lib/rabbitmq`
- 健康检查固定使用 `rabbitmq-diagnostics -q ping`
- `doctor` 会执行 `rabbitmq-diagnostics -q ping` 和 `rabbitmqctl status`

### 推荐写法

```yaml
name: rabbitmq-demo
version: "v1"

runtime:
  project-name: rabbitmq-demo

services:
  - name: rabbitmq-1
    middleware: rabbitmq
    template: single
    values:
      version: v4.2
      amqp_port: 5672
      management_port: 15672
      default_user: admin
      default_pass: admin123
      erlang_cookie: rabbitmq-cookie
      data_dir: ./data/rabbitmq-1
```

### 使用建议

- 常规场景只需要关心 `amqp_port`、`management_port`、`default_user`、`default_pass`
- `image` 建议仅在需要替换镜像源时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## RabbitMQ Cluster

适用范围：

- `middleware: rabbitmq`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: rabbitmq-cluster-demo
version: "v1"

services:
  - name: rabbitmq-cluster
    middleware: rabbitmq
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | RabbitMQ 集群节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `rabbitmq:4.2-management` | 任意非空字符串；推荐与 `version` 对应 |
| `rabbit1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-rabbit1` | 任意非空字符串 |
| `rabbit2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-rabbit2` | 任意非空字符串 |
| `rabbit3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-rabbit3` | 任意非空字符串 |
| `rabbit1_container_name` | 节点 1 容器名称。 | 默认等于 `rabbit1_service_name` | 任意非空字符串 |
| `rabbit2_container_name` | 节点 2 容器名称。 | 默认等于 `rabbit2_service_name` | 任意非空字符串 |
| `rabbit3_container_name` | 节点 3 容器名称。 | 默认等于 `rabbit3_service_name` | 任意非空字符串 |
| `rabbit1_data_dir` | 节点 1 数据目录，会挂载到容器内 `/var/lib/rabbitmq`。 | `./data/<rabbit1_service_name>` | 任意非空字符串路径 |
| `rabbit2_data_dir` | 节点 2 数据目录，会挂载到容器内 `/var/lib/rabbitmq`。 | `./data/<rabbit2_service_name>` | 任意非空字符串路径 |
| `rabbit3_data_dir` | 节点 3 数据目录，会挂载到容器内 `/var/lib/rabbitmq`。 | `./data/<rabbit3_service_name>` | 任意非空字符串路径 |
| `rabbit1_amqp_port` | 节点 1 AMQP 端口。 | `5672` | 正整数端口 |
| `rabbit2_amqp_port` | 节点 2 AMQP 端口。 | `5673` | 正整数端口 |
| `rabbit3_amqp_port` | 节点 3 AMQP 端口。 | `5674` | 正整数端口 |
| `rabbit1_management_port` | 节点 1 管理端口。 | `15672` | 正整数端口 |
| `rabbit2_management_port` | 节点 2 管理端口。 | `15673` | 正整数端口 |
| `rabbit3_management_port` | 节点 3 管理端口。 | `15674` | 正整数端口 |
| `default_user` | 集群默认管理员用户名。 | `admin` | 任意非空字符串 |
| `default_pass` | 集群默认管理员密码。 | `admin123` | 任意非空字符串 |
| `erlang_cookie` | 集群 Erlang cookie，3 个节点必须保持一致。 | `rabbitmq-cookie` | 任意非空字符串 |
| `version` | RabbitMQ 版本选择。 | `v4.2` | `v4.2` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v4.2` | `rabbitmq:4.2-management` | 固定创建 3 节点经典自动发现集群 |

### 固定行为

- 当前模板固定创建 3 个 RabbitMQ 节点
- 3 个节点的 hostname 固定为 `rabbit1`、`rabbit2`、`rabbit3`
- 每个节点都会固定挂载一个只读集群配置文件到 `/etc/rabbitmq/rabbitmq.conf`
- 集群发现方式固定为 `classic_config`
- `cluster_partition_handling` 固定为 `autoheal`
- `queue_master_locator` 固定为 `min-masters`
- 健康检查固定使用 `rabbitmq-diagnostics -q ping`
- `doctor` 会检查 `rabbitmqctl cluster_status --formatter json`，并验证 3 个节点都已加入集群

### 推荐写法

```yaml
name: rabbitmq-cluster-demo
version: "v1"

runtime:
  project-name: rabbitmq-cluster-demo

services:
  - name: rabbitmq-cluster
    middleware: rabbitmq
    template: cluster
    values:
      version: v4.2
      rabbit1_amqp_port: 5672
      rabbit2_amqp_port: 5673
      rabbit3_amqp_port: 5674
      rabbit1_management_port: 15672
      rabbit2_management_port: 15673
      rabbit3_management_port: 15674
      default_user: admin
      default_pass: admin123
      erlang_cookie: rabbitmq-cookie
      rabbit1_data_dir: ./data/rabbit1
      rabbit2_data_dir: ./data/rabbit2
      rabbit3_data_dir: ./data/rabbit3
```

### 使用建议

- 常规场景只需要关心 3 个节点的 AMQP 端口、管理端口、数据目录和默认账号
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
