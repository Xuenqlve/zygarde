# Redis

本文介绍如何在 Zygarde 中使用 Redis 的 Compose 模板。

## Redis Single

适用范围：

- `middleware: redis`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: redis-demo
version: "v1"

services:
  - name: redis-1
    middleware: redis
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `redis-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Docker 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `redis-<index>` 补齐 | 任意非空字符串 |
| `image` | Redis 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.2` 时默认 `redis:6.2`；`version=v7.4` 时默认 `redis:7.4` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `port` | 宿主机暴露端口，容器内端口固定为 `6379/tcp`。 | `6379` | 正整数端口 |
| `version` | Redis 版本选择，会影响默认镜像。 | `v7.4` | `v6.2`、`v7.4` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.2` | `redis:6.2` | 使用当前默认启动参数 |
| `v7.4` | `redis:7.4` | 使用当前默认启动参数 |

### 固定行为

- 容器内端口固定为 `6379`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/data`
- 启动命令固定为：
  - `redis-server`
  - `--appendonly yes`
  - `--save "60 1000"`
- 健康检查固定使用 `redis-cli ping`
- `doctor` 会检查 `PING` 和复制角色信息，单机场景下角色应为 `master`

### 推荐写法

```yaml
name: redis-demo
version: "v1"

runtime:
  project-name: redis-demo

services:
  - name: redis-1
    middleware: redis
    template: single
    values:
      version: v6.2
      port: 6379
      data_dir: ./data/redis-1
```

### 使用建议

- 常规场景只需要关心 `port`、`version`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## Redis Master-Slave

适用范围：

- `middleware: redis`
- `template: master-slave`
- `environmentType: compose`

### 最小示例

```yaml
name: redis-master-slave-demo
version: "v1"

services:
  - name: redis-ms
    middleware: redis
    template: master-slave
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `master_service_name` | 主节点在 Compose 内部的 service 名称。 | `<service.name>-master` | 任意非空字符串 |
| `slave_service_name` | 从节点在 Compose 内部的 service 名称。 | `<service.name>-slave` | 任意非空字符串 |
| `master_container_name` | 主节点容器名称。 | 默认等于 `master_service_name` | 任意非空字符串 |
| `slave_container_name` | 从节点容器名称。 | 默认等于 `slave_service_name` | 任意非空字符串 |
| `master_image` | 主节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.2` 时默认 `redis:6.2`；`version=v7.4` 时默认 `redis:7.4` | 任意非空字符串；推荐与 `version` 对应 |
| `slave_image` | 从节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.2` 时默认 `redis:6.2`；`version=v7.4` 时默认 `redis:7.4` | 任意非空字符串；推荐与 `version` 对应 |
| `master_data_dir` | 主节点宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<master_service_name>` | 任意非空字符串路径 |
| `slave_data_dir` | 从节点宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<slave_service_name>` | 任意非空字符串路径 |
| `master_port` | 主节点宿主机暴露端口，容器内端口固定为 `6379/tcp`。 | `6379` | 正整数端口 |
| `slave_port` | 从节点宿主机暴露端口，容器内端口固定为 `6379/tcp`。 | `6380` | 正整数端口，且不能与 `master_port` 相同 |
| `version` | Redis 版本选择，会影响默认镜像。 | `v7.4` | `v6.2`、`v7.4` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.2` | `redis:6.2` | 从节点使用 `--replicaof <master_service_name> 6379` 启动 |
| `v7.4` | `redis:7.4` | 从节点使用 `--replicaof <master_service_name> 6379` 启动 |

### 固定行为

- 主从节点容器内端口都固定为 `6379`
- `restart` 固定为 `unless-stopped`
- 主从数据目录固定挂载到 `/data`
- 主节点固定启用：
  - `--appendonly yes`
  - `--save "60 1000"`
- 从节点在主节点基础上额外固定启用：
  - `--replicaof <master_service_name> 6379`
- 健康检查固定使用 `redis-cli ping`
- `doctor` 会检查主节点角色、从节点角色，以及从节点的 `master_host` / `master_link_status`

### 推荐写法

```yaml
name: redis-master-slave-demo
version: "v1"

runtime:
  project-name: redis-master-slave-demo

services:
  - name: redis-ms
    middleware: redis
    template: master-slave
    values:
      version: v6.2
      master_port: 6379
      slave_port: 6380
      master_data_dir: ./data/redis-master
      slave_data_dir: ./data/redis-slave
```

### 使用建议

- 常规场景只需要关心 `master_port`、`slave_port`、`version`
- `master_image`、`slave_image` 建议仅在需要替换镜像源或 tag 时覆盖
- `master_service_name`、`slave_service_name`、容器名没有明确需求时保持默认即可

## Redis Cluster

适用范围：

- `middleware: redis`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: redis-cluster-demo
version: "v1"

services:
  - name: redis-cluster
    middleware: redis
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | Redis 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.2` 时默认 `redis:6.2`；`version=v7.4` 时默认 `redis:7.4` | 任意非空字符串；推荐与 `version` 对应 |
| `node_1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-node-1` | 任意非空字符串 |
| `node_2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-node-2` | 任意非空字符串 |
| `node_3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-node-3` | 任意非空字符串 |
| `node_1_container_name` | 节点 1 容器名称。 | 默认等于 `node_1_service_name` | 任意非空字符串 |
| `node_2_container_name` | 节点 2 容器名称。 | 默认等于 `node_2_service_name` | 任意非空字符串 |
| `node_3_container_name` | 节点 3 容器名称。 | 默认等于 `node_3_service_name` | 任意非空字符串 |
| `node_1_data_dir` | 节点 1 宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<node_1_service_name>` | 任意非空字符串路径 |
| `node_2_data_dir` | 节点 2 宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<node_2_service_name>` | 任意非空字符串路径 |
| `node_3_data_dir` | 节点 3 宿主机数据目录，会挂载到容器内 `/data`。 | `./data/<node_3_service_name>` | 任意非空字符串路径 |
| `node_1_port` | 节点 1 的 Redis 服务端口。 | `7001` | 正整数端口 |
| `node_2_port` | 节点 2 的 Redis 服务端口。 | `7002` | 正整数端口 |
| `node_3_port` | 节点 3 的 Redis 服务端口。 | `7003` | 正整数端口 |
| `node_1_bus_port` | 节点 1 的集群 bus 端口。 | `17001` | 正整数端口 |
| `node_2_bus_port` | 节点 2 的集群 bus 端口。 | `17002` | 正整数端口 |
| `node_3_bus_port` | 节点 3 的集群 bus 端口。 | `17003` | 正整数端口 |
| `version` | Redis 版本选择，会影响默认镜像。 | `v7.4` | `v6.2`、`v7.4` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.2` | `redis:6.2` | 创建 3 节点、0 副本的最小集群 |
| `v7.4` | `redis:7.4` | 创建 3 节点、0 副本的最小集群 |

### 固定行为

- 当前模板固定创建 3 个 Redis 节点
- 每个节点都固定启用：
  - `--cluster-enabled yes`
  - `--cluster-config-file nodes.conf`
  - `--cluster-node-timeout 5000`
  - `--appendonly yes`
- `up` 期间会自动执行 `redis-cli --cluster create`，创建 `3 master / 0 replicas` 集群
- `doctor` 会检查 3 个节点连通性、`cluster_state`、`cluster_known_nodes`、`cluster_size`

### 推荐写法

```yaml
name: redis-cluster-demo
version: "v1"

runtime:
  project-name: redis-cluster-demo

services:
  - name: redis-cluster
    middleware: redis
    template: cluster
    values:
      version: v6.2
      node_1_port: 7001
      node_1_bus_port: 17001
      node_2_port: 7002
      node_2_bus_port: 17002
      node_3_port: 7003
      node_3_bus_port: 17003
      node_1_data_dir: ./data/redis-node-1
      node_2_data_dir: ./data/redis-node-2
      node_3_data_dir: ./data/redis-node-3
```

### 使用建议

- 常规场景只需要关心 3 个节点的服务端口、bus 端口、版本和数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
