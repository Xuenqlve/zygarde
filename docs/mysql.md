# MySQL

本文介绍如何在 Zygarde 中使用 MySQL 的 Compose 模板。

## MySQL Single

适用范围：

- `middleware: mysql`
- `template: single`
- `environmentType: compose`

### 最小示例

```yaml
name: mysql-demo
version: "v1"

services:
  - name: mysql-1
    middleware: mysql
    template: single
    values:
      root_password: root
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `service_name` | Compose 内部 service 名称，同时也是生成 `.env` 键前缀时使用的服务标识。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `mysql-<index>` 补齐 | 任意非空字符串 |
| `container_name` | Docker 容器名称。 | 默认等于当前 `service.name`；若 `service.name` 为空，则先按 `mysql-<index>` 补齐 | 任意非空字符串 |
| `image` | MySQL 镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v5.7` 时默认 `mysql:5.7`；`version=v8.0` 时默认 `mysql:8.0` | 任意非空字符串；推荐与 `version` 对应 |
| `data_dir` | 宿主机数据目录，会挂载到容器内 `/var/lib/mysql`。 | `./data/<service.name>` | 任意非空字符串路径 |
| `port` | 宿主机暴露端口，容器内端口固定为 `3306/tcp`。 | `3306` | 正整数端口 |
| `root_password` | MySQL root 用户密码，同时用于健康检查与验收脚本。 | `root` | 任意非空字符串 |
| `version` | MySQL 版本选择，会影响默认镜像、平台以及启动参数。 | `v8.0` | `v5.7`、`v8.0` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v5.7` | `mysql:5.7` | 启动命令会额外追加 `--default-authentication-plugin=mysql_native_password` |
| `v8.0` | `mysql:8.0` | 使用当前默认启动参数 |

### 固定行为

- 容器内端口固定为 `3306`
- `restart` 固定为 `unless-stopped`
- 数据卷固定挂载到 `/var/lib/mysql`
- 容器环境变量固定包含：
  - `MYSQL_ROOT_PASSWORD`
  - `MYSQL_ROOT_HOST=%`
- 健康检查固定使用 `mysqladmin ping`

### 推荐写法

```yaml
name: mysql-double-demo
version: "v1"

runtime:
  project-name: mysql-double-demo

services:
  - name: mysql-1
    middleware: mysql
    template: single
    values:
      version: v5.7
      port: 3306
      root_password: root1
      data_dir: ./data/mysql-1

  - name: mysql-2
    middleware: mysql
    template: single
    values:
      version: v8.0
      port: 3307
      root_password: root2
      data_dir: ./data/mysql-2
```

### 使用建议

- 常规场景只需要关心 `root_password`、`port`、`version`、`data_dir`
- `image` 建议仅在需要替换镜像源或 tag 时覆盖
- `service_name`、`container_name` 没有明确需求时保持默认即可

## MySQL Master-Slave

适用范围：

- `middleware: mysql`
- `template: master-slave`
- `environmentType: compose`

### 最小示例

```yaml
name: mysql-master-slave-demo
version: "v1"

services:
  - name: mysql-ms
    middleware: mysql
    template: master-slave
    values:
      root_password: root123
```

### 参数说明

| 变量名 | 变量介绍 | 默认值 | 可选值 |
| --- | --- | --- | --- |
| `master_service_name` | 主库在 Compose 内部的 service 名称。 | `<service.name>-master` | 任意非空字符串 |
| `slave_service_name` | 从库在 Compose 内部的 service 名称。 | `<service.name>-slave` | 任意非空字符串 |
| `master_container_name` | 主库容器名称。 | 默认等于 `master_service_name` | 任意非空字符串 |
| `slave_container_name` | 从库容器名称。 | 默认等于 `slave_service_name` | 任意非空字符串 |
| `master_image` | 主库镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v5.7` 时默认 `mysql:5.7`；`version=v8.0` 时默认 `mysql:8.0` | 任意非空字符串；推荐与 `version` 对应 |
| `slave_image` | 从库镜像名。通常不需要手动指定，默认会随 `version` 自动推导。 | `version=v5.7` 时默认 `mysql:5.7`；`version=v8.0` 时默认 `mysql:8.0` | 任意非空字符串；推荐与 `version` 对应 |
| `master_data_dir` | 主库宿主机数据目录，会挂载到主库容器 `/var/lib/mysql`。 | `./data/<master_service_name>` | 任意非空字符串路径 |
| `slave_data_dir` | 从库宿主机数据目录，会挂载到从库容器 `/var/lib/mysql`。 | `./data/<slave_service_name>` | 任意非空字符串路径 |
| `master_port` | 主库宿主机暴露端口，容器内端口固定为 `3306/tcp`。 | `3306` | 正整数端口 |
| `slave_port` | 从库宿主机暴露端口，容器内端口固定为 `3306/tcp`。 | `3307` | 正整数端口，且不能与 `master_port` 相同 |
| `root_password` | 主从实例共用的 root 用户密码，同时用于健康检查和复制配置。 | `root` | 任意非空字符串 |
| `replication_user` | 复制链路使用的账号名。 | `repl` | 任意非空字符串 |
| `replication_password` | 复制链路使用的账号密码。 | `repl123` | 任意非空字符串 |
| `version` | MySQL 版本选择，会影响默认镜像、平台以及复制初始化脚本。 | `v8.0` | `v5.7`、`v8.0` |

### 版本说明

| version | 默认 image | 说明 |
| --- | --- | --- |
| `v5.7` | `mysql:5.7` | 复制配置使用 `CHANGE MASTER TO` / `START SLAVE` 语法 |
| `v8.0` | `mysql:8.0` | 复制配置使用 `CHANGE REPLICATION SOURCE TO` / `START REPLICA` 语法 |

### 固定行为

- 主库和从库容器内端口都固定为 `3306`
- `restart` 固定为 `unless-stopped`
- 主从数据目录固定挂载到 `/var/lib/mysql`
- 主库会自动挂载初始化 SQL，用于创建复制账号
- `up` 期间会自动等待主从健康检查通过，并执行复制初始化脚本
- `doctor` 会检查复制状态，并验证主库新建数据库后从库可见
- 主从实例都固定启用 GTID 相关参数与 `mysql_native_password`

### 推荐写法

```yaml
name: mysql-master-slave-demo
version: "v1"

runtime:
  project-name: mysql-master-slave-demo

services:
  - name: mysql-ms
    middleware: mysql
    template: master-slave
    values:
      version: v8.0
      master_port: 3306
      slave_port: 3307
      root_password: root123
      replication_user: repl
      replication_password: repl123
      master_data_dir: ./data/mysql-master
      slave_data_dir: ./data/mysql-slave
```

### 使用建议

- 常规场景只需要关心 `root_password`、`master_port`、`slave_port`、`version`
- `replication_user`、`replication_password` 建议在多人共享环境下显式设置
- `master_image`、`slave_image` 建议仅在需要替换镜像源或 tag 时覆盖
- `master_service_name`、`slave_service_name`、容器名没有明确需求时保持默认即可
