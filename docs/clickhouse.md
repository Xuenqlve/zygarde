# ClickHouse

本文介绍如何在 Zygarde 中使用 ClickHouse 的 Compose 模板。

## ClickHouse Single

适用范围：

- `middleware: clickhouse`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: clickhouse-demo
version: "v1"

services:
  - name: clickhouse-1
    middleware: clickhouse
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | ClickHouse 的 Compose service 名称。 | 默认等于 `service.name` | 任意非空字符串 |
| `container_name` | 容器名称。 | 默认等于 `service_name` | 任意非空字符串 |
| `image` | 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v24` 时为 `clickhouse/clickhouse-server:24`；`version=v25` 时为 `clickhouse/clickhouse-server:25.8` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 数据目录，会挂载到容器内 `/var/lib/clickhouse`。 | `./data/<service_name>` | 任意非空字符串路径 |
| `http_port` | ClickHouse HTTP 端口。 | `8123` | 正整数端口 |
| `tcp_port` | ClickHouse Native TCP 端口。 | `9000` | 正整数端口 |
| `version` | ClickHouse 版本选择。 | `v25` | `v24`、`v25` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v24` | `clickhouse/clickhouse-server:24` | 按 `docker/clickhouse/single_v24` 已验证模板收敛 |
| `v25` | `clickhouse/clickhouse-server:25.8` | 按 `docker/clickhouse/single_v25` 已验证模板收敛 |

### 固定行为

- 容器内 HTTP 端口固定为 `8123`
- 容器内 Native TCP 端口固定为 `9000`
- 数据目录固定挂载到 `/var/lib/clickhouse`
- 健康检查固定使用 `clickhouse-client -q "SELECT 1"`
- `doctor` 会检查：
  - `SELECT 1`
  - `SELECT version()`
  - 建表、插入、查询、删表的基础读写链路

### 推荐写法

```yaml
name: clickhouse-demo
version: "v1"

runtime:
  project-name: clickhouse-demo

services:
  - name: clickhouse-1
    middleware: clickhouse
    template: single
    values:
      version: v25
      http_port: 8123
      tcp_port: 9000
      data_dir: ./data/clickhouse
```

### 使用建议

- 常规场景只需要关心 `http_port`、`tcp_port` 和 `data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name` 和 `container_name` 没有明确需求时保持默认即可

## ClickHouse Cluster

适用范围：

- `middleware: clickhouse`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: clickhouse-cluster-demo
version: "v1"

services:
  - name: clickhouse-cluster
    middleware: clickhouse
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `image` | 集群节点镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v24` 时为 `clickhouse/clickhouse-server:24`；`version=v25` 时为 `clickhouse/clickhouse-server:25.8` | 任意非空字符串；推荐与 `version` 对应 |
| `ch1_service_name` | CH1 的 Compose service 名称。 | `<service.name>-ch1` | 任意非空字符串 |
| `ch2_service_name` | CH2 的 Compose service 名称。 | `<service.name>-ch2` | 任意非空字符串 |
| `ch3_service_name` | CH3 的 Compose service 名称。 | `<service.name>-ch3` | 任意非空字符串 |
| `ch1_container_name` | CH1 容器名称。 | 默认等于 `ch1_service_name` | 任意非空字符串 |
| `ch2_container_name` | CH2 容器名称。 | 默认等于 `ch2_service_name` | 任意非空字符串 |
| `ch3_container_name` | CH3 容器名称。 | 默认等于 `ch3_service_name` | 任意非空字符串 |
| `ch1_data_dir` | CH1 数据目录，会挂载到容器内 `/var/lib/clickhouse`。 | `./data/<ch1_service_name>` | 任意非空字符串路径 |
| `ch2_data_dir` | CH2 数据目录，会挂载到容器内 `/var/lib/clickhouse`。 | `./data/<ch2_service_name>` | 任意非空字符串路径 |
| `ch3_data_dir` | CH3 数据目录，会挂载到容器内 `/var/lib/clickhouse`。 | `./data/<ch3_service_name>` | 任意非空字符串路径 |
| `ch1_http_port` | CH1 HTTP 端口。 | `8123` | 正整数端口 |
| `ch2_http_port` | CH2 HTTP 端口。 | `8124` | 正整数端口 |
| `ch3_http_port` | CH3 HTTP 端口。 | `8125` | 正整数端口 |
| `ch1_tcp_port` | CH1 Native TCP 端口。 | `9000` | 正整数端口 |
| `ch2_tcp_port` | CH2 Native TCP 端口。 | `9001` | 正整数端口 |
| `ch3_tcp_port` | CH3 Native TCP 端口。 | `9002` | 正整数端口 |
| `version` | ClickHouse 版本选择。 | `v25` | `v24`、`v25` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v24` | `clickhouse/clickhouse-server:24` | 按 `docker/clickhouse/cluster_v24` 已验证模板收敛 |
| `v25` | `clickhouse/clickhouse-server:25.8` | 按 `docker/clickhouse/cluster_v25` 已验证模板收敛，并额外生成 `users.d/default-network.xml` |

### 固定行为

- 当前模板固定创建 `3` 个 ClickHouse 节点
- 每个节点固定挂载一份 `config.d/cluster.xml`
- `cluster.xml` 会按当前 service 名称动态生成 `zygarde_cluster`
- `v25` 会额外生成 `users.d/default-network.xml` 允许默认用户跨节点访问
- 容器内 HTTP 端口固定为 `8123`
- 容器内 Native TCP 端口固定为 `9000`
- `doctor` 会检查：
  - 3 个节点的 `SELECT 1`
  - `system.clusters` 中 `zygarde_cluster` 节点数量
  - `remote()` 跨节点查询链路

### 推荐写法

```yaml
name: clickhouse-cluster-demo
version: "v1"

runtime:
  project-name: clickhouse-cluster-demo

services:
  - name: clickhouse-cluster
    middleware: clickhouse
    template: cluster
    values:
      version: v25
      ch1_http_port: 8123
      ch2_http_port: 8124
      ch3_http_port: 8125
      ch1_tcp_port: 9000
      ch2_tcp_port: 9001
      ch3_tcp_port: 9002
      ch1_data_dir: ./data/ch1
      ch2_data_dir: ./data/ch2
      ch3_data_dir: ./data/ch3
```

### 使用建议

- 常规场景只需要关心 6 个端口和 3 个数据目录
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- 3 个 service 名称和容器名没有明确需求时保持默认即可
