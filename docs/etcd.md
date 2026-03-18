# etcd

本文介绍如何在 Zygarde 中使用 etcd 的 Compose 模板。

## etcd Single

适用范围：

- `middleware: etcd`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: etcd-demo
version: "v1"

services:
  - name: etcd-1
    middleware: etcd
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是 etcd 节点名和对外广播地址使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `etcd-<index>` 补齐 | 任意非空字符串 |
| `container_name` | etcd 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `etcd-<index>` 补齐 | 任意非空字符串 |
| `image` | etcd 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `quay.io/coreos/etcd:v3.6.0` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/etcd-data`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `client_port` | etcd client 对外端口。 | `2379` | 正整数端口 |
| `peer_port` | etcd peer 对外端口。 | `2380` | 正整数端口 |
| `cluster_token` | 单节点集群初始化 token。 | `zygarde-etcd-single` | 任意非空字符串 |
| `version` | etcd 版本选择。 | `v3.6` | `v3.6` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v3.6` | `quay.io/coreos/etcd:v3.6.0` | 固定使用单节点 etcd 3.6 模式 |

### 固定行为

- 当前模板固定创建单节点 etcd
- `ALLOW_NONE_AUTHENTICATION` 固定为 `yes`
- `ETCD_NAME` 默认跟随 `service_name`
- 数据卷固定挂载到 `/etcd-data`
- `doctor` 会检查：
  - endpoint health
  - member list
  - put/get KV 烟测

### 推荐写法

```yaml
name: etcd-demo
version: "v1"

runtime:
  project-name: etcd-demo

services:
  - name: etcd-1
    middleware: etcd
    template: single
    values:
      version: v3.6
      client_port: 2379
      peer_port: 2380
      cluster_token: zygarde-etcd-single
      data_dir: ./data/etcd
```

### 使用建议

- 常规场景只需要关心 `client_port`、`peer_port`、`data_dir`
- `cluster_token` 建议在需要并行启动多个隔离环境时显式区分
- `image` 建议仅在需要替换镜像源或 tag 时覆盖

## etcd Cluster

适用范围：

- `middleware: etcd`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: etcd-cluster-demo
version: "v1"

services:
  - name: etcd-cluster
    middleware: etcd
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | etcd 集群镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `quay.io/coreos/etcd:v3.6.0` | 任意非空字符串；推荐与 `version` 对应 |
| `cluster_token` | 3 节点集群的初始化 token。 | `zygarde-etcd-cluster` | 任意非空字符串 |
| `etcd1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-etcd1` | 任意非空字符串 |
| `etcd2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-etcd2` | 任意非空字符串 |
| `etcd3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-etcd3` | 任意非空字符串 |
| `etcd1_container_name` | 节点 1 容器名称。 | 默认等于 `etcd1_service_name` | 任意非空字符串 |
| `etcd2_container_name` | 节点 2 容器名称。 | 默认等于 `etcd2_service_name` | 任意非空字符串 |
| `etcd3_container_name` | 节点 3 容器名称。 | 默认等于 `etcd3_service_name` | 任意非空字符串 |
| `etcd1_data_dir` | 节点 1 数据目录，会挂载到容器内 `/etcd-data`。 | `./data/<etcd1_service_name>` | 任意非空字符串路径 |
| `etcd2_data_dir` | 节点 2 数据目录，会挂载到容器内 `/etcd-data`。 | `./data/<etcd2_service_name>` | 任意非空字符串路径 |
| `etcd3_data_dir` | 节点 3 数据目录，会挂载到容器内 `/etcd-data`。 | `./data/<etcd3_service_name>` | 任意非空字符串路径 |
| `etcd1_client_port` | 节点 1 client 对外端口。 | `2379` | 正整数端口 |
| `etcd2_client_port` | 节点 2 client 对外端口。 | `2479` | 正整数端口 |
| `etcd3_client_port` | 节点 3 client 对外端口。 | `2579` | 正整数端口 |
| `version` | etcd 版本选择。 | `v3.6` | `v3.6` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v3.6` | `quay.io/coreos/etcd:v3.6.0` | 固定创建 3 节点 etcd 3.6 集群 |

### 固定行为

- 当前模板固定创建 3 个 etcd 节点
- 模板不对外暴露 peer host 端口
- 每个节点的 peer 通信固定走容器内 `2380`
- `ETCD_INITIAL_CLUSTER` 固定为 `etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380`
- `doctor` 会检查：
  - 三节点 endpoint health
  - member list 数量
  - 跨节点 put/get KV 烟测

### 推荐写法

```yaml
name: etcd-cluster-demo
version: "v1"

runtime:
  project-name: etcd-cluster-demo

services:
  - name: etcd-cluster
    middleware: etcd
    template: cluster
    values:
      version: v3.6
      etcd1_client_port: 2379
      etcd2_client_port: 2479
      etcd3_client_port: 2579
      cluster_token: zygarde-etcd-cluster
      etcd1_data_dir: ./data/etcd1
      etcd2_data_dir: ./data/etcd2
      etcd3_data_dir: ./data/etcd3
```

### 使用建议

- 常规场景只需要关心 3 个 client 端口、`cluster_token` 和 3 个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
