# Consul

本文介绍如何在 Zygarde 中使用 Consul 的 Compose 模板。

## Consul Single

适用范围：

- `middleware: consul`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: consul-demo
version: "v1"

services:
  - name: consul-1
    middleware: consul
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `consul-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Consul 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `consul-<index>` 补齐 | 任意非空字符串 |
| `image` | Consul 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `hashicorp/consul:1.20` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/consul/data`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `http_port` | Consul HTTP API 对外端口。 | `8500` | 正整数端口 |
| `dns_port` | Consul DNS 对外端口。 | `8600` | 正整数端口 |
| `server_port` | Consul server RPC 对外端口。 | `8300` | 正整数端口 |
| `version` | Consul 版本选择。 | `v1.20` | `v1.20` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v1.20` | `hashicorp/consul:1.20` | 固定使用单节点 Consul server 模式 |

### 固定行为

- 当前模板固定创建单节点 Consul server
- 启动参数固定包含：
  - `agent`
  - `-server`
  - `-ui`
  - `-node=consul1`
  - `-bootstrap-expect=1`
- 数据卷固定挂载到 `/consul/data`
- `doctor` 会检查：
  - leader
  - members
  - KV put/get 烟测

### 推荐写法

```yaml
name: consul-demo
version: "v1"

runtime:
  project-name: consul-demo

services:
  - name: consul-1
    middleware: consul
    template: single
    values:
      version: v1.20
      http_port: 8500
      dns_port: 8600
      server_port: 8300
      data_dir: ./data/consul
```

### 使用建议

- 常规场景只需要关心 `http_port`、`dns_port`、`server_port`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖

## Consul Cluster

适用范围：

- `middleware: consul`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: consul-cluster-demo
version: "v1"

services:
  - name: consul-cluster
    middleware: consul
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | Consul 集群镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `hashicorp/consul:1.20` | 任意非空字符串；推荐与 `version` 对应 |
| `consul1_service_name` | 节点 1 的 Compose service 名称。 | `<service.name>-consul1` | 任意非空字符串 |
| `consul2_service_name` | 节点 2 的 Compose service 名称。 | `<service.name>-consul2` | 任意非空字符串 |
| `consul3_service_name` | 节点 3 的 Compose service 名称。 | `<service.name>-consul3` | 任意非空字符串 |
| `consul1_container_name` | 节点 1 容器名称。 | 默认等于 `consul1_service_name` | 任意非空字符串 |
| `consul2_container_name` | 节点 2 容器名称。 | 默认等于 `consul2_service_name` | 任意非空字符串 |
| `consul3_container_name` | 节点 3 容器名称。 | 默认等于 `consul3_service_name` | 任意非空字符串 |
| `consul1_data_dir` | 节点 1 数据目录，会挂载到容器内 `/consul/data`。 | `./data/<consul1_service_name>` | 任意非空字符串路径 |
| `consul2_data_dir` | 节点 2 数据目录，会挂载到容器内 `/consul/data`。 | `./data/<consul2_service_name>` | 任意非空字符串路径 |
| `consul3_data_dir` | 节点 3 数据目录，会挂载到容器内 `/consul/data`。 | `./data/<consul3_service_name>` | 任意非空字符串路径 |
| `consul1_http_port` | 节点 1 HTTP API 对外端口。 | `8500` | 正整数端口 |
| `consul1_dns_port` | 节点 1 DNS 对外端口。 | `8600` | 正整数端口 |
| `consul2_http_port` | 节点 2 HTTP API 对外端口。 | `9500` | 正整数端口 |
| `consul3_http_port` | 节点 3 HTTP API 对外端口。 | `10500` | 正整数端口 |
| `version` | Consul 版本选择。 | `v1.20` | `v1.20` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v1.20` | `hashicorp/consul:1.20` | 固定创建 3 节点 Consul server 集群 |

### 固定行为

- 当前模板固定创建 3 个 Consul server 节点
- 只有 `consul1` 对外暴露 DNS 端口
- 3 个节点固定使用：
  - `-bootstrap-expect=3`
  - `-retry-join=consul1`
  - `-retry-join=consul2`
  - `-retry-join=consul3`
- `doctor` 会检查：
  - leader
  - members 数量
  - raft server 数量
  - 跨节点 KV 烟测

### 推荐写法

```yaml
name: consul-cluster-demo
version: "v1"

runtime:
  project-name: consul-cluster-demo

services:
  - name: consul-cluster
    middleware: consul
    template: cluster
    values:
      version: v1.20
      consul1_http_port: 8500
      consul1_dns_port: 8600
      consul2_http_port: 9500
      consul3_http_port: 10500
      consul1_data_dir: ./data/consul1
      consul2_data_dir: ./data/consul2
      consul3_data_dir: ./data/consul3
```

### 使用建议

- 常规场景只需要关心 4 个对外端口和 3 个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
