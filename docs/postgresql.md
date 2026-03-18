# PostgreSQL

本文介绍如何在 Zygarde 中使用 PostgreSQL 的 Compose 模板。

## PostgreSQL Single

适用范围：

- `middleware: postgresql`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: postgres-demo
version: "v1"

services:
  - name: postgres-1
    middleware: postgresql
    template: single
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `postgresql-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Docker 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `postgresql-<index>` 补齐 | 任意非空字符串 |
| `image` | PostgreSQL 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v16` 时默认 `postgres:16`；`version=v17` 时默认 `postgres:17` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/var/lib/postgresql/data`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `port` | 宿主机暴露端口，容器内端口固定为 `5432/tcp`。 | `5432` | 正整数端口 |
| `user` | 默认数据库用户名。 | `postgres` | 任意非空字符串 |
| `password` | 默认数据库密码。 | `postgres123` | 任意非空字符串 |
| `database` | 默认数据库名。 | `app` | 任意非空字符串 |
| `version` | PostgreSQL 版本选择，会影响默认镜像。 | `v17` | `v16`、`v17` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v16` | `postgres:16` | 使用当前默认启动参数 |
| `v17` | `postgres:17` | 使用当前默认启动参数 |

### 固定行为

- 容器内端口固定为 `5432`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/var/lib/postgresql/data`
- 容器环境变量固定使用：
  - `POSTGRES_USER`
  - `POSTGRES_PASSWORD`
  - `POSTGRES_DB`
- 健康检查固定使用 `pg_isready`
- `doctor` 会执行 `select 1` 并输出版本信息

### 推荐写法

```yaml
name: postgres-demo
version: "v1"

runtime:
  project-name: postgres-demo

services:
  - name: postgres-1
    middleware: postgresql
    template: single
    values:
      version: v16
      port: 5432
      user: postgres
      password: postgres123
      database: app
      data_dir: ./data/postgres-1
```

### 使用建议

- 常规场景只需要关心 `port`、`version`、`user`、`password`、`database`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## PostgreSQL Master-Slave

适用范围：

- `middleware: postgresql`
- `template: master-slave`
- `environmentType: compose`

### 最小示例

```yaml
name: postgres-master-slave-demo
version: "v1"

services:
  - name: postgres-ms
    middleware: postgresql
    template: master-slave
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `master_service_name` | 主库 Compose service 名称。 | `<service.name>-master` | 任意非空字符串 |
| `slave_service_name` | 从库 Compose service 名称。 | `<service.name>-slave` | 任意非空字符串 |
| `master_container_name` | 主库容器名称。 | 默认等于 `master_service_name` | 任意非空字符串 |
| `slave_container_name` | 从库容器名称。 | 默认等于 `slave_service_name` | 任意非空字符串 |
| `master_image` | 主库镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v16` 时默认 `postgres:16`；`version=v17` 时默认 `postgres:17` | 任意非空字符串；推荐与 `version` 对应 |
| `slave_image` | 从库镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v16` 时默认 `postgres:16`；`version=v17` 时默认 `postgres:17` | 任意非空字符串；推荐与 `version` 对应 |
| `master_data_dir` | 主库数据目录，会挂载到 `/var/lib/postgresql/data`。 | `./data/<master_service_name>` | 任意非空字符串路径 |
| `slave_data_dir` | 从库数据目录，会挂载到 `/var/lib/postgresql/data`。 | `./data/<slave_service_name>` | 任意非空字符串路径 |
| `master_port` | 主库宿主机暴露端口。 | `5432` | 正整数端口 |
| `slave_port` | 从库宿主机暴露端口。 | `5433` | 正整数端口，且不能与 `master_port` 相同 |
| `user` | 默认数据库用户名。 | `postgres` | 任意非空字符串 |
| `password` | 默认数据库密码。 | `postgres123` | 任意非空字符串 |
| `database` | 默认数据库名。 | `app` | 任意非空字符串 |
| `replication_user` | 复制账号名。 | `repl_user` | 任意非空字符串 |
| `replication_password` | 复制账号密码。 | `repl_pass` | 任意非空字符串 |
| `version` | PostgreSQL 版本选择，会影响默认镜像。 | `v17` | `v16`、`v17` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v16` | `postgres:16` | 固定创建一主一从复制拓扑 |
| `v17` | `postgres:17` | 固定创建一主一从复制拓扑 |

### 固定行为

- 主从容器内端口都固定为 `5432`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/var/lib/postgresql/data`
- 主库固定追加：
  - `wal_level=replica`
  - `max_wal_senders=10`
  - `max_replication_slots=10`
- `up` 期间会通过初始化脚本创建复制账号，并通过 `pg_basebackup` 初始化从库
- 从库固定以 `hot_standby=on` 启动
- `doctor` 会检查主库的 `pg_stat_replication` 和从库的 `pg_is_in_recovery()`

### 推荐写法

```yaml
name: postgres-master-slave-demo
version: "v1"

runtime:
  project-name: postgres-master-slave-demo

services:
  - name: postgres-ms
    middleware: postgresql
    template: master-slave
    values:
      version: v16
      master_port: 5432
      slave_port: 5433
      user: postgres
      password: postgres123
      database: app
      replication_user: repl_user
      replication_password: repl_pass
      master_data_dir: ./data/postgres-master
      slave_data_dir: ./data/postgres-slave
```

### 使用建议

- 常规场景只需要关心 `master_port`、`slave_port`、`user`、`password`、`database`
- `replication_user`、`replication_password` 建议在多人共享环境下显式设置
- `master_image`、`slave_image` 建议仅在需要替换镜像源或 tag 时覆盖
- `master_service_name`、`slave_service_name`、容器名没有明确需求时保持默认即可
