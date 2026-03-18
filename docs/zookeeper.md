# ZooKeeper

本文介绍如何在 Zygarde 中使用 ZooKeeper 的 Compose 模板。

## ZooKeeper Single

适用范围：

- `middleware: zookeeper`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: zookeeper-demo
version: "v1"

services:
  - name: zk-1
    middleware: zookeeper
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | ZooKeeper 的 Compose service 名称。 | 默认等于 `service.name` | 任意非空字符串 |
| `container_name` | 容器名称。 | 默认等于 `service_name` | 任意非空字符串 |
| `image` | 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v3.8` 时为 `zookeeper:3.8`；`version=v3.9` 时为 `zookeeper:3.9` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 数据目录，会挂载到容器内 `/data`。 | `./data/<service_name>` | 任意非空字符串路径 |
| `datalog_dir` | 事务日志目录，会挂载到容器内 `/datalog`。 | `./datalog/<service_name>` | 任意非空字符串路径 |
| `client_port` | ZooKeeper client 端口。 | `2181` | 正整数端口 |
| `follower_port` | ZooKeeper follower 端口。 | `2888` | 正整数端口 |
| `election_port` | ZooKeeper leader election 端口。 | `3888` | 正整数端口 |
| `version` | ZooKeeper 版本选择。 | `v3.9` | `v3.8`、`v3.9` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v3.8` | `zookeeper:3.8` | 按 `docker/zookeeper/single_v3.8` 已验证模板收敛 |
| `v3.9` | `zookeeper:3.9` | 按 `docker/zookeeper/single_v3.9` 已验证模板收敛 |

### 固定行为

- 容器环境变量固定包含：
  - `ZOO_MY_ID=1`
  - `ZOO_4LW_COMMANDS_WHITELIST=ruok,mntr,srvr,stat,conf,isro`
- 数据目录固定挂载到 `/data`
- 事务日志目录固定挂载到 `/datalog`
- `doctor` 会检查：
  - `ruok=imok`
  - `mntr` 中的 `zk_server_state` 和 `zk_version`
  - znode 创建和读取链路

### 推荐写法

```yaml
name: zookeeper-demo
version: "v1"

runtime:
  project-name: zookeeper-demo

services:
  - name: zk-1
    middleware: zookeeper
    template: single
    values:
      version: v3.9
      client_port: 2181
      follower_port: 2888
      election_port: 3888
      data_dir: ./data/zk
      datalog_dir: ./datalog/zk
```

### 使用建议

- 常规场景只需要关心 3 个端口和 2 个目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name` 和 `container_name` 没有明确需求时保持默认即可

## ZooKeeper Cluster

适用范围：

- `middleware: zookeeper`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: zookeeper-cluster-demo
version: "v1"

services:
  - name: zk-cluster
    middleware: zookeeper
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | 集群节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v3.8` 时为 `zookeeper:3.8`；`version=v3.9` 时为 `zookeeper:3.9` | 任意非空字符串；推荐与 `version` 对应 |
| `zk1_service_name` | ZK1 的 Compose service 名称。 | `<service.name>-zk1` | 任意非空字符串 |
| `zk2_service_name` | ZK2 的 Compose service 名称。 | `<service.name>-zk2` | 任意非空字符串 |
| `zk3_service_name` | ZK3 的 Compose service 名称。 | `<service.name>-zk3` | 任意非空字符串 |
| `zk1_container_name` | ZK1 容器名称。 | 默认等于 `zk1_service_name` | 任意非空字符串 |
| `zk2_container_name` | ZK2 容器名称。 | 默认等于 `zk2_service_name` | 任意非空字符串 |
| `zk3_container_name` | ZK3 容器名称。 | 默认等于 `zk3_service_name` | 任意非空字符串 |
| `zk1_data_dir` | ZK1 数据目录，会挂载到容器内 `/data`。 | `./data/<zk1_service_name>` | 任意非空字符串路径 |
| `zk2_data_dir` | ZK2 数据目录，会挂载到容器内 `/data`。 | `./data/<zk2_service_name>` | 任意非空字符串路径 |
| `zk3_data_dir` | ZK3 数据目录，会挂载到容器内 `/data`。 | `./data/<zk3_service_name>` | 任意非空字符串路径 |
| `zk1_datalog_dir` | ZK1 事务日志目录，会挂载到容器内 `/datalog`。 | `./datalog/<zk1_service_name>` | 任意非空字符串路径 |
| `zk2_datalog_dir` | ZK2 事务日志目录，会挂载到容器内 `/datalog`。 | `./datalog/<zk2_service_name>` | 任意非空字符串路径 |
| `zk3_datalog_dir` | ZK3 事务日志目录，会挂载到容器内 `/datalog`。 | `./datalog/<zk3_service_name>` | 任意非空字符串路径 |
| `zk1_client_port` | ZK1 client 端口。 | `2181` | 正整数端口 |
| `zk2_client_port` | ZK2 client 端口。 | `2182` | 正整数端口 |
| `zk3_client_port` | ZK3 client 端口。 | `2183` | 正整数端口 |
| `version` | ZooKeeper 版本选择。 | `v3.9` | `v3.8`、`v3.9` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v3.8` | `zookeeper:3.8` | 按 `docker/zookeeper/cluster_v3.8` 已验证模板收敛 |
| `v3.9` | `zookeeper:3.9` | 按 `docker/zookeeper/cluster_v3.9` 已验证模板收敛 |

### 固定行为

- 当前模板固定创建 `3` 个 ZooKeeper 节点
- 节点之间固定使用容器内 `2888/3888` 进行 quorum 通讯
- 模板不会对外暴露 follower/election host 端口
- 每个节点固定注入：
  - `ZOO_MY_ID`
  - `ZOO_SERVERS`
  - `ZOO_4LW_COMMANDS_WHITELIST=ruok,mntr,srvr,stat,conf,isro`
- `doctor` 会检查：
  - 3 个节点的 `ruok`
  - `stat` 返回的 leader/follower 拓扑
  - 跨节点 znode 创建和读取链路

### 推荐写法

```yaml
name: zookeeper-cluster-demo
version: "v1"

runtime:
  project-name: zookeeper-cluster-demo

services:
  - name: zk-cluster
    middleware: zookeeper
    template: cluster
    values:
      version: v3.9
      zk1_client_port: 2181
      zk2_client_port: 2182
      zk3_client_port: 2183
      zk1_data_dir: ./data/zk1
      zk2_data_dir: ./data/zk2
      zk3_data_dir: ./data/zk3
      zk1_datalog_dir: ./datalog/zk1
      zk2_datalog_dir: ./datalog/zk2
      zk3_datalog_dir: ./datalog/zk3
```

### 使用建议

- 常规场景只需要关心 3 个 client 端口和 6 个目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 3 个 service 名称和容器名没有明确需求时保持默认即可
