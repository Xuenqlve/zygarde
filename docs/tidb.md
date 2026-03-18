# TiDB

本文介绍如何在 Zygarde 中使用 TiDB 的 Compose 模板。

## TiDB Single

适用范围：

- `middleware: tidb`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: tidb-demo
version: "v1"

services:
  - name: tidb-1
    middleware: tidb
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `pd_service_name` | PD 节点的 Compose service 名称。 | `<service.name>-pd` | 任意非空字符串 |
| `tikv_service_name` | TiKV 节点的 Compose service 名称。 | `<service.name>-tikv` | 任意非空字符串 |
| `tidb_service_name` | TiDB 节点的 Compose service 名称。 | `<service.name>-tidb` | 任意非空字符串 |
| `pd_container_name` | PD 容器名称。 | 默认等于 `pd_service_name` | 任意非空字符串 |
| `tikv_container_name` | TiKV 容器名称。 | 默认等于 `tikv_service_name` | 任意非空字符串 |
| `tidb_container_name` | TiDB 容器名称。 | 默认等于 `tidb_service_name` | 任意非空字符串 |
| `pd_image` | PD 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/pd:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `tikv_image` | TiKV 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/tikv:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `tidb_image` | TiDB 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/tidb:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `pd_data_dir` | PD 数据目录，会挂载到容器内 `/data/pd`。 | `./data/<pd_service_name>` | 任意非空字符串路径 |
| `tikv_data_dir` | TiKV 数据目录，会挂载到容器内 `/data/tikv`。 | `./data/<tikv_service_name>` | 任意非空字符串路径 |
| `pd_port` | PD 对外 client 端口。 | `2379` | 正整数端口 |
| `tikv_port` | TiKV 对外端口。 | `20160` | 正整数端口 |
| `tidb_port` | TiDB SQL 端口。 | `4000` | 正整数端口 |
| `tidb_status_port` | TiDB 状态端口。 | `10080` | 正整数端口 |
| `version` | TiDB 版本选择。 | `v6.7` | `v6.7` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.7` | `pd/tikv/tidb` 默认分别使用 `pingcap/pd:v6.5.12`、`pingcap/tikv:v6.5.12`、`pingcap/tidb:v6.5.12` | 按 `docker/tidb/single_v6.7` 已验证模板收敛，镜像版本以模板事实为准 |

### 固定行为

- 当前模板固定创建 `pd + tikv + tidb` 三个服务
- PD 固定使用：
  - `--name=pd`
  - `--client-urls=http://0.0.0.0:2379`
  - `--peer-urls=http://0.0.0.0:2380`
- TiKV 固定监听容器内 `20160`
- TiDB 固定监听容器内：
  - SQL 端口 `4000`
  - status 端口 `10080`
- `doctor` 会检查：
  - TiDB `/status`
  - PD `/pd/api/v1/health`
  - TiDB SQL 端口可达性

### 推荐写法

```yaml
name: tidb-demo
version: "v1"

runtime:
  project-name: tidb-demo

services:
  - name: tidb-1
    middleware: tidb
    template: single
    values:
      version: v6.7
      pd_port: 2379
      tikv_port: 20160
      tidb_port: 4000
      tidb_status_port: 10080
      pd_data_dir: ./data/pd
      tikv_data_dir: ./data/tikv
```

### 使用建议

- 常规场景只需要关心 4 个端口和 2 个数据目录
- `pd_image`、`tikv_image`、`tidb_image` 建议仅在需要替换镜像源或 tag 时覆盖
- 3 个 service 名称和容器名没有明确需求时保持默认即可

## TiDB Cluster

适用范围：

- `middleware: tidb`
- `template: cluster`
- `environmentType: compose`

### 最小示例

```yaml
name: tidb-cluster-demo
version: "v1"

services:
  - name: tidb-cluster
    middleware: tidb
    template: cluster
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `pd_image` | PD 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/pd:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `tikv_image` | TiKV 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/tikv:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `tidb_image` | TiDB 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `pingcap/tidb:v6.5.12` | 任意非空字符串；推荐与 `version` 对应 |
| `pd1_service_name` | PD1 的 Compose service 名称。 | `<service.name>-pd1` | 任意非空字符串 |
| `pd2_service_name` | PD2 的 Compose service 名称。 | `<service.name>-pd2` | 任意非空字符串 |
| `pd3_service_name` | PD3 的 Compose service 名称。 | `<service.name>-pd3` | 任意非空字符串 |
| `tikv1_service_name` | TiKV1 的 Compose service 名称。 | `<service.name>-tikv1` | 任意非空字符串 |
| `tikv2_service_name` | TiKV2 的 Compose service 名称。 | `<service.name>-tikv2` | 任意非空字符串 |
| `tikv3_service_name` | TiKV3 的 Compose service 名称。 | `<service.name>-tikv3` | 任意非空字符串 |
| `tidb1_service_name` | TiDB1 的 Compose service 名称。 | `<service.name>-tidb1` | 任意非空字符串 |
| `tidb2_service_name` | TiDB2 的 Compose service 名称。 | `<service.name>-tidb2` | 任意非空字符串 |
| `pd1_container_name` | PD1 容器名称。 | 默认等于 `pd1_service_name` | 任意非空字符串 |
| `pd2_container_name` | PD2 容器名称。 | 默认等于 `pd2_service_name` | 任意非空字符串 |
| `pd3_container_name` | PD3 容器名称。 | 默认等于 `pd3_service_name` | 任意非空字符串 |
| `tikv1_container_name` | TiKV1 容器名称。 | 默认等于 `tikv1_service_name` | 任意非空字符串 |
| `tikv2_container_name` | TiKV2 容器名称。 | 默认等于 `tikv2_service_name` | 任意非空字符串 |
| `tikv3_container_name` | TiKV3 容器名称。 | 默认等于 `tikv3_service_name` | 任意非空字符串 |
| `tidb1_container_name` | TiDB1 容器名称。 | 默认等于 `tidb1_service_name` | 任意非空字符串 |
| `tidb2_container_name` | TiDB2 容器名称。 | 默认等于 `tidb2_service_name` | 任意非空字符串 |
| `pd1_data_dir` | PD1 数据目录，会挂载到容器内 `/data/pd`。 | `./data/<pd1_service_name>` | 任意非空字符串路径 |
| `pd2_data_dir` | PD2 数据目录，会挂载到容器内 `/data/pd`。 | `./data/<pd2_service_name>` | 任意非空字符串路径 |
| `pd3_data_dir` | PD3 数据目录，会挂载到容器内 `/data/pd`。 | `./data/<pd3_service_name>` | 任意非空字符串路径 |
| `tikv1_data_dir` | TiKV1 数据目录，会挂载到容器内 `/data/tikv`。 | `./data/<tikv1_service_name>` | 任意非空字符串路径 |
| `tikv2_data_dir` | TiKV2 数据目录，会挂载到容器内 `/data/tikv`。 | `./data/<tikv2_service_name>` | 任意非空字符串路径 |
| `tikv3_data_dir` | TiKV3 数据目录，会挂载到容器内 `/data/tikv`。 | `./data/<tikv3_service_name>` | 任意非空字符串路径 |
| `pd1_port` | PD1 对外 client 端口。 | `2379` | 正整数端口 |
| `pd2_port` | PD2 对外 client 端口。 | `2479` | 正整数端口 |
| `pd3_port` | PD3 对外 client 端口。 | `2579` | 正整数端口 |
| `tidb1_port` | TiDB1 SQL 端口。 | `4000` | 正整数端口 |
| `tidb2_port` | TiDB2 SQL 端口。 | `4001` | 正整数端口 |
| `tidb1_status_port` | TiDB1 状态端口。 | `10080` | 正整数端口 |
| `tidb2_status_port` | TiDB2 状态端口。 | `10081` | 正整数端口 |
| `version` | TiDB 版本选择。 | `v6.7` | `v6.7` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v6.7` | `pd/tikv/tidb` 默认分别使用 `pingcap/pd:v6.5.12`、`pingcap/tikv:v6.5.12`、`pingcap/tidb:v6.5.12` | 按 `docker/tidb/cluster_v6.7` 已验证模板收敛，镜像版本以模板事实为准 |

### 固定行为

- 当前模板固定创建 `3 PD + 3 TiKV + 2 TiDB`
- 模板不会对外暴露 TiKV host 端口
- PD1 固定使用 `--force-new-cluster` 初始化 3 成员集群
- PD2、PD3 固定通过 `--join=<pd1>:2379` 加入
- TiKV 固定通过 3 个 PD endpoint 启动
- `doctor` 会检查：
  - 两个 TiDB `/status`
  - PD health 是否 3 成员健康
  - PD member 数量和 leader
  - store 数量是否至少为 3
  - 两个 TiDB SQL 端口可达性

### 推荐写法

```yaml
name: tidb-cluster-demo
version: "v1"

runtime:
  project-name: tidb-cluster-demo

services:
  - name: tidb-cluster
    middleware: tidb
    template: cluster
    values:
      version: v6.7
      pd1_port: 2379
      pd2_port: 2479
      pd3_port: 2579
      tidb1_port: 4000
      tidb2_port: 4001
      tidb1_status_port: 10080
      tidb2_status_port: 10081
      pd1_data_dir: ./data/pd1
      pd2_data_dir: ./data/pd2
      pd3_data_dir: ./data/pd3
      tikv1_data_dir: ./data/tikv1
      tikv2_data_dir: ./data/tikv2
      tikv3_data_dir: ./data/tikv3
```

### 使用建议

- 常规场景只需要关心 7 个对外端口和 6 个数据目录
- `pd_image`、`tikv_image`、`tidb_image` 建议仅在需要替换镜像源或 tag 时覆盖
- 节点 service 名称和容器名没有明确需求时保持默认即可
