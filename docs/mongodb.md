# MongoDB

本文介绍如何在 Zygarde 中使用 MongoDB 的 Compose 模板。

## MongoDB Single

适用范围：

- `middleware: mongodb`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: mongodb-demo
version: "v1"

services:
  - name: mongodb-1
    middleware: mongodb
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `mongodb-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Docker 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `mongodb-<index>` 补齐 | 任意非空字符串 |
| `image` | MongoDB 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.0` 时默认 `mongo:6.0`；`version=v7.0` 时默认 `mongo:7.0` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/data/db`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `port` | 宿主机暴露端口，容器内端口固定为 `27017/tcp`。 | `27017` | 正整数端口 |
| `version` | MongoDB 版本选择，会影响默认镜像。 | `v7.0` | `v6.0`、`v7.0` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.0` | `mongo:6.0` | 使用当前默认启动参数 |
| `v7.0` | `mongo:7.0` | 使用当前默认启动参数 |

### 固定行为

- 容器内端口固定为 `27017`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/data/db`
- 启动命令固定为：
  - `mongod`
  - `--bind_ip_all`
  - `--dbpath /data/db`
- 健康检查固定使用 `mongosh --quiet --eval 'db.adminCommand({ ping: 1 }).ok'`
- `doctor` 会检查连通性并输出数据库版本

### 推荐写法

```yaml
name: mongodb-demo
version: "v1"

runtime:
  project-name: mongodb-demo

services:
  - name: mongodb-1
    middleware: mongodb
    template: single
    values:
      version: v6.0
      port: 27017
      data_dir: ./data/mongodb-1
```

### 使用建议

- 常规场景只需要关心 `port`、`version`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## MongoDB Replica-Set

适用范围：

- `middleware: mongodb`
- `template: replica-set`
- `environmentType: compose`

### 最小示例

```yaml
name: mongodb-rs-demo
version: "v1"

services:
  - name: mongodb-rs
    middleware: mongodb
    template: replica-set
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | MongoDB 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.0` 时默认 `mongo:6.0`；`version=v7.0` 时默认 `mongo:7.0` | 任意非空字符串；推荐与 `version` 对应 |
| `rs1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-rs1` | 任意非空字符串 |
| `rs2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-rs2` | 任意非空字符串 |
| `rs3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-rs3` | 任意非空字符串 |
| `rs1_container_name` | 节点 1 容器名称。 | 默认等于 `rs1_service_name` | 任意非空字符串 |
| `rs2_container_name` | 节点 2 容器名称。 | 默认等于 `rs2_service_name` | 任意非空字符串 |
| `rs3_container_name` | 节点 3 容器名称。 | 默认等于 `rs3_service_name` | 任意非空字符串 |
| `rs1_data_dir` | 节点 1 宿主机数据目录，会挂载到容器内 `/data/db`。 | `./data/<rs1_service_name>` | 任意非空字符串路径 |
| `rs2_data_dir` | 节点 2 宿主机数据目录，会挂载到容器内 `/data/db`。 | `./data/<rs2_service_name>` | 任意非空字符串路径 |
| `rs3_data_dir` | 节点 3 宿主机数据目录，会挂载到容器内 `/data/db`。 | `./data/<rs3_service_name>` | 任意非空字符串路径 |
| `rs1_port` | 节点 1 宿主机暴露端口。 | `27017` | 正整数端口 |
| `rs2_port` | 节点 2 宿主机暴露端口。 | `27018` | 正整数端口 |
| `rs3_port` | 节点 3 宿主机暴露端口。 | `27019` | 正整数端口 |
| `version` | MongoDB 版本选择，会影响默认镜像。 | `v7.0` | `v6.0`、`v7.0` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.0` | `mongo:6.0` | 固定创建 `rs0` 三节点副本集 |
| `v7.0` | `mongo:7.0` | 固定创建 `rs0` 三节点副本集 |

### 固定行为

- 当前模板固定创建 `3` 个 MongoDB 节点
- 每个节点固定启用：
  - `mongod`
  - `--replSet rs0`
  - `--bind_ip_all`
  - `--dbpath /data/db`
- `up` 期间会自动执行 `rs.initiate(...)`
- `up` 会等待 `1 PRIMARY + 2 SECONDARY` 稳定
- `doctor` 会检查副本集成员状态并确认存在 PRIMARY

### 推荐写法

```yaml
name: mongodb-rs-demo
version: "v1"

runtime:
  project-name: mongodb-rs-demo

services:
  - name: mongodb-rs
    middleware: mongodb
    template: replica-set
    values:
      version: v6.0
      rs1_port: 27017
      rs2_port: 27018
      rs3_port: 27019
      rs1_data_dir: ./data/mongo-rs1
      rs2_data_dir: ./data/mongo-rs2
      rs3_data_dir: ./data/mongo-rs3
```

### 使用建议

- 常规场景只需要关心 `version`、三个节点端口和三个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可

## MongoDB Sharded

适用范围：

- `middleware: mongodb`
- `template: sharded`
- `environmentType: compose`

### 最小示例

```yaml
name: mongodb-sharded-demo
version: "v1"

services:
  - name: mongodb-sharded
    middleware: mongodb
    template: sharded
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | MongoDB 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v6.0` 时默认 `mongo:6.0`；`version=v7.0` 时默认 `mongo:7.0` | 任意非空字符串；推荐与 `version` 对应 |
| `cfg1_service_name` | 配置副本集节点 1 的 service 名称。 | `<service.name>-cfg1` | 任意非空字符串 |
| `cfg2_service_name` | 配置副本集节点 2 的 service 名称。 | `<service.name>-cfg2` | 任意非空字符串 |
| `cfg3_service_name` | 配置副本集节点 3 的 service 名称。 | `<service.name>-cfg3` | 任意非空字符串 |
| `cfg1_container_name` | 配置副本集节点 1 容器名称。 | 默认等于 `cfg1_service_name` | 任意非空字符串 |
| `cfg2_container_name` | 配置副本集节点 2 容器名称。 | 默认等于 `cfg2_service_name` | 任意非空字符串 |
| `cfg3_container_name` | 配置副本集节点 3 容器名称。 | 默认等于 `cfg3_service_name` | 任意非空字符串 |
| `cfg1_data_dir` | 配置副本集节点 1 数据目录。 | `./data/<cfg1_service_name>` | 任意非空字符串路径 |
| `cfg2_data_dir` | 配置副本集节点 2 数据目录。 | `./data/<cfg2_service_name>` | 任意非空字符串路径 |
| `cfg3_data_dir` | 配置副本集节点 3 数据目录。 | `./data/<cfg3_service_name>` | 任意非空字符串路径 |
| `shard1_service_name` | 分片副本集节点 1 的 service 名称。 | `<service.name>-shard1` | 任意非空字符串 |
| `shard2_service_name` | 分片副本集节点 2 的 service 名称。 | `<service.name>-shard2` | 任意非空字符串 |
| `shard1_container_name` | 分片副本集节点 1 容器名称。 | 默认等于 `shard1_service_name` | 任意非空字符串 |
| `shard2_container_name` | 分片副本集节点 2 容器名称。 | 默认等于 `shard2_service_name` | 任意非空字符串 |
| `shard1_data_dir` | 分片副本集节点 1 数据目录。 | `./data/<shard1_service_name>` | 任意非空字符串路径 |
| `shard2_data_dir` | 分片副本集节点 2 数据目录。 | `./data/<shard2_service_name>` | 任意非空字符串路径 |
| `mongos_service_name` | `mongos` 路由器的 service 名称。 | `<service.name>-mongos` | 任意非空字符串 |
| `mongos_container_name` | `mongos` 路由器容器名称。 | 默认等于 `mongos_service_name` | 任意非空字符串 |
| `mongos_port` | 对外暴露的 `mongos` 端口。 | `27017` | 正整数端口 |
| `version` | MongoDB 版本选择，会影响默认镜像。 | `v7.0` | `v6.0`、`v7.0` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.0` | `mongo:6.0` | 固定创建 `cfgRS + shardRS + mongos` 的最小分片拓扑 |
| `v7.0` | `mongo:7.0` | 固定创建 `cfgRS + shardRS + mongos` 的最小分片拓扑 |

### 固定行为

- 当前模板固定拓扑为：
  - `3` 个 config server
  - `2` 个 shard 成员
  - `1` 个 mongos
- config server 固定运行：
  - `mongod --configsvr --replSet cfgRS --bind_ip_all --port 27019`
- shard server 固定运行：
  - `mongod --shardsvr --replSet shardRS --bind_ip_all --port 27018`
- `mongos` 固定通过 `cfgRS/...` 连接配置副本集
- `up` 期间会自动初始化 `cfgRS`、`shardRS`，并执行 `sh.addShard(...)`
- `doctor` 会检查 `mongos ping`、`listShards` 和分片数量

### 推荐写法

```yaml
name: mongodb-sharded-demo
version: "v1"

runtime:
  project-name: mongodb-sharded-demo

services:
  - name: mongodb-sharded
    middleware: mongodb
    template: sharded
    values:
      version: v6.0
      mongos_port: 27017
      cfg1_data_dir: ./data/cfg1
      cfg2_data_dir: ./data/cfg2
      cfg3_data_dir: ./data/cfg3
      shard1_data_dir: ./data/shard1
      shard2_data_dir: ./data/shard2
```

### 使用建议

- 常规场景只需要关心 `version`、`mongos_port` 和 5 个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 各节点 service 名称和容器名没有明确需求时保持默认即可
