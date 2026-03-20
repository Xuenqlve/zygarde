# zygarde

`zygarde` 是一个面向本地开发环境的声明式中间件编排工具。用户通过一份 blueprint 文件描述要启动的中间件拓扑，`zygarde` 负责生成 Compose 运行栈、执行生命周期命令，并记录环境状态。

当前一期运行时以 Compose 为主，已经支持单中间件和多中间件组合场景。

## 核心能力

- Blueprint 管理：支持本地 blueprint 的 `list / show / validate`
- 模板管理：支持内置 middleware template 的 `list / show`
- 环境生命周期：支持 `create / up / list / status / doctor / start / stop / down`
- 中间件编排：已覆盖 12 个中间件、26 个 Compose 模板
- 诊断与回收：生成 `check.sh`，并支持失败清理、当前目录环境标记和 `down` 回收

## 支持的中间件

- MySQL：`single`、`master-slave`
- Redis：`single`、`master-slave`、`cluster`
- MongoDB：`single`、`replica-set`、`sharded`
- PostgreSQL：`single`、`master-slave`
- RabbitMQ：`single`、`cluster`
- Kafka：`single`、`cluster`
- TiDB：`single`、`cluster`
- etcd：`single`、`cluster`
- Consul：`single`、`cluster`
- ClickHouse：`single`、`cluster`
- ZooKeeper：`single`、`cluster`
- Elasticsearch：`single`、`cluster`

各中间件的参数说明见 `docs/` 下对应文档。

## 安装与运行

```bash
go build ./...
go run ./cmd --help
```

默认使用 `docker` 作为容器引擎。如果你本地使用 `podman`，可以切换：

```bash
export ZYGARDE_CONTAINER_ENGINE=podman
```

## 容器引擎差异

当前默认容器引擎是 `docker`。项目也支持 `podman`，但两者并不是完全等价实现。

当前已知处理方式：

- `docker`
  - 默认走标准 `docker compose` 生命周期
- `podman`
  - 走独立的 deployment 分支，不再简单复用 Docker 的命令路径
  - `create -> start` 不直接依赖 `compose start` 的语义一致性
  - `down` 会对部分 provider 的 `network not found` 清理噪音做幂等兼容

如果你本地主要使用 `podman`，建议：

- 保持 `ZYGARDE_CONTAINER_ENGINE=podman`
- 尽量使用项目生成的命令主链路：`create / up / status / doctor / start / stop / down`
- 避免在同一个测试目录里同时混跑多组 Compose 集成测试

## 版本兼容规则

同一个中间件的不同版本，某些命令、SQL 或检查脚本语法可能不兼容。

`zygarde` 当前的处理原则是：

- 已知版本时，直接按 `version` 选择对应命令
- 不依赖“先执行不兼容命令，再 fallback”的运行时试错流程
- 版本差异以 `docker/<middleware>/<scenario>_<version>/` 为事实来源收敛到 `pkg/*`

例如：

- MySQL `v5.7` 使用 `SHOW SLAVE STATUS`
- MySQL `v8.0` 使用 `SHOW REPLICA STATUS`

这类兼容逻辑已经内建在对应中间件实现中，但前提仍然是 blueprint 中的 `version` 配置要准确。

## Blueprint 文件

默认 blueprint 文件名是当前目录下的 `zygarde.yaml`。也可以通过 `-f/--file` 显式指定。

最小示例：

```yaml
name: demo-stack
version: "v1"
description: mysql and redis demo

runtime:
  project-name: demo-stack

services:
  - name: mysql-1
    middleware: mysql
    template: single
    values:
      version: v5.7
      port: 3306
      root_password: root123
      data_dir: ./data/mysql-1

  - name: redis-1
    middleware: redis
    template: single
    values:
      version: v6.2
      port: 6379
      data_dir: ./data/redis-1
```

## Blueprint 管理命令

### 列出本地 blueprint

扫描目录下的标准 blueprint 文件：
- `zygarde.yaml`
- `zygarde.yml`
- `*.blueprint.yaml`
- `*.blueprint.yml`

```bash
zygarde blueprint list
zygarde blueprint list --dir ./examples
```

### 查看 blueprint 摘要

```bash
zygarde blueprint show
zygarde blueprint show -f ./examples/demo/zygarde.yaml
```

输出会展示：
- blueprint 基本信息
- runtime project name
- 归一化后的 service 列表

### 校验 blueprint

```bash
zygarde blueprint validate
zygarde blueprint validate -f ./examples/demo/zygarde.yaml
zygarde blueprint validate -f ./examples/demo/zygarde.yaml --env-type compose
```

当前 `validate` 会校验：
- YAML 可解析
- blueprint 基础结构合法
- service 默认值可归一化
- 引用的 `middleware + template + runtime` 组合已注册

## 模板管理命令

模板管理命令展示的是当前 `pkg/*` 已注册、可被 blueprint 引用的内置模板能力。模板清单由 `pkg` 侧维护，`internal` 只负责读取和展示。

### 列出可用模板

```bash
zygarde template list
zygarde template list --env-type compose
```

输出会展示：
- middleware
- template
- runtime
- 是否默认模板
- 支持版本
- 帮助文档路径
- 简要说明

### 查看单个模板详情

```bash
zygarde template show mysql/single
zygarde template show redis cluster
```

输出会展示：
- middleware
- template
- runtime
- 是否默认模板
- 支持版本
- 帮助文档路径
- 简要说明

## 环境生命周期命令

### `create`

纯创建语义，对齐 `docker compose create`：
- 生成并渲染 runtime bundle
- 创建运行时资源
- 不启动容器
- 环境状态落为 `stopped`

```bash
zygarde create
zygarde create -f ./examples/demo/zygarde.yaml
```

### `up`

创建并启动环境。

```bash
zygarde up
zygarde up -f ./examples/demo/zygarde.yaml
```

### `list`

列出本地 environment store 中的环境。

```bash
zygarde list
```

### `status`

查看环境状态。默认作用于当前目录环境；也可以显式传 `--id`。

```bash
zygarde status
zygarde status --id <environment-id>
```

### `doctor`

执行当前环境目录下生成的 `check.sh`，做运行检查。

```bash
zygarde doctor
zygarde doctor --id <environment-id>
```

### `start`

启动已经存在且处于 `stopped` 状态的环境。

```bash
zygarde start
zygarde start --id <environment-id>
```

### `stop`

停止当前运行中的环境。

```bash
zygarde stop
zygarde stop --id <environment-id>
```

### `down`

停止并销毁环境，清理本地 runtime workspace。

```bash
zygarde down
zygarde down --id <environment-id>
```

## 当前目录环境

执行 `create` 或 `up` 成功后，`zygarde` 会在当前目录写入：

```text
.zygarde/current-environment
```

它用于让 `status / doctor / start / stop / down` 默认作用于当前目录对应的环境，而不要求用户手动输入 `environment-id`。

## 生成产物

Compose 运行时会在 `.zygarde/environments/<environment-id>/` 下生成完整 bundle，至少包含：

- `docker-compose.yml`
- `.env`
- `build.sh`
- `check.sh`
- `README.md`

环境元数据和 runtime artifact 会持久化到：

- `.zygarde/environments/<environment-id>.json`
- `.zygarde/environments/<environment-id>.runtime.json`

## 测试

```bash
go test ./...
```

如需在本地用 `podman` 跑集成测试：

```bash
export ZYGARDE_CONTAINER_ENGINE=podman
go test ./test/command -count=1
```

## 当前范围与后续计划

当前已完成：

- Compose 运行时
- 12 个中间件、26 个模板
- Blueprint 管理的一期命令：`list / show / validate`
- 环境生命周期主命令

后续仍待推进：

- Blueprint / Template 的完整 CRUD
- 更结构化的 `doctor` 输出
- 更多 runtime 扩展能力，例如 K8s
