# Zygarde Architecture

## 目录原则

- `pkg/`：存放每个中间件的特有逻辑，例如变量定义、默认值、特定场景校验、模板辅助逻辑。
- `internal/`：存放平台运行核心，用于模板、蓝图、环境、部署、编排、配置、存储等通用机制。

## pkg 一期目录

第一期按 `docker/` 下已有中间件类型建立目录：

- `pkg/clickhouse`
- `pkg/consul`
- `pkg/elasticsearch`
- `pkg/etcd`
- `pkg/kafka`
- `pkg/mongodb`
- `pkg/mysql`
- `pkg/postgresql`
- `pkg/rabbitmq`
- `pkg/redis`
- `pkg/tidb`
- `pkg/zookeeper`

## internal 一期目录设计

- `internal/app`
  - 应用装配入口，负责连接配置、存储、服务和命令入口。
- `internal/config`
  - 配置文件、环境变量、命令行参数的加载与归一化。
- `internal/template`
  - 模板元信息、模板解析、模板校验。
- `internal/blueprint`
  - blueprint 定义、模板引用关系、变量绑定。
- `internal/render`
  - 将 blueprint、template 和变量渲染为最终产物，例如 `docker-compose.yaml`。
- `internal/environment`
  - 环境实例、状态流转、元数据与生命周期管理。
- `internal/deployment`
  - 部署执行抽象，不直接绑定单一后端细节。
- `internal/deployment/compose`
  - Docker Compose 的执行实现。
- `internal/coordinator`
  - 跨模块流程编排，串联模板、蓝图、环境和部署动作。
- `internal/store`
  - 模板、蓝图、环境等对象的持久化抽象。
- `internal/model`
  - 跨模块共享的领域模型和状态枚举。
- `internal/runtime`
  - 工作目录、产物路径、项目隔离和运行时文件布局。
- `internal/log`
  - 统一日志初始化和结构化字段规范。
- `internal/cli`
  - CLI 命令定义和参数绑定。

## 边界规则

- `pkg/*` 不负责环境生命周期编排。
- `internal/*` 不承载某一种中间件的碎片化硬编码特性。
- `internal/render` 只负责产物生成，不负责真正部署。
- `internal/deployment` 只负责执行部署动作，不负责蓝图定义。
- `internal/model` 放共享对象定义，不承担流程逻辑。
- `internal/coordinator` 编排流程，但不吞并底层模块职责。

## 主流程规划

项目主流程按“定义 -> 配置累计 -> 上下文构建 -> 渲染 -> 部署 -> 管理”组织。

### 创建环境主链路

1. 用户通过 CLI 发起环境创建请求。
2. `internal/cli` 解析命令参数。
3. `internal/config` 加载全局默认配置和运行参数。
4. `internal/store` 读取 `blueprint.yaml`。
5. `internal/blueprint` 将 `services` 归一化，补齐默认 `name`、`template` 和空 `values`。
6. `internal/app` 装配依赖并调用 `internal/coordinator`。
7. `internal/coordinator` 按 `middleware + template + environmentType` 从 `internal/template` 注册器中解析 pkg 实现。
8. `internal/coordinator` 对每个 service 调用 pkg 的 `Configure(pipeline, config)`。
9. 同一个 pkg 实现按 `pipeline=service.name` 累计缓存多份服务配置。
10. 所有 service 完成配置后，`internal/coordinator` 调用 pkg 的 `BuildRuntimeContext()`，统一拿到 `[]EnvironmentContext`。
11. `internal/runtime` 和 `internal/render` 根据所有 context 生成当前 runtime 需要的产物；第一期为 `docker-compose.yaml`。
12. `internal/deployment/compose` 执行 `docker compose up -d`。
13. `internal/environment` 持久化环境元数据、产物快照和状态。
14. CLI 返回环境 ID、名称、路径和访问端点等结果。

### create 前半段链路

第一期优先打通 `main -> cli -> app -> coordinator -> store/blueprint/template` 这段链路。

建议执行顺序：

1. `cmd/main.go` 仅调用 `internal/cli` 入口，不承载业务逻辑。
2. `internal/cli` 先支持 `zygarde create -f blueprint.yaml`，默认 runtime 为 Compose。
3. `internal/cli` 将命令参数整理为标准 `CreateRequest`。
4. `internal/app` 负责装配 blueprint store、middleware registry 和 coordinator。
5. `internal/coordinator.Create` 负责读取 blueprint、归一化 services、解析 pkg 实现并逐个调用 `Configure(...)`。
6. `internal/coordinator.Create` 前半段完成后，应能返回本次已参与配置的 middleware 实例集合，供后续 `BuildRuntimeContext()` 阶段继续使用。

### create 命令建议

- 一期最小命令：`zygarde create -f blueprint.yaml`
- 一期默认 runtime：`compose`
- 非必要时，不要在第一版引入过多 flags；优先把最小主链路打通

### 管理类主链路

- `status`：读取 environment 元数据和当前状态，必要时结合 deployment 查询结果返回。
- `start`：基于已持久化的 runtime 产物重新触发 deployment。
- `stop`：调用 deployment 停止对应 environment。
- `destroy`：调用 deployment 销毁环境，并由 environment 更新最终状态与清理结果。

### 模块协作原则

- `pkg/<middleware>` 是唯一中间件扩展点，负责配置补全、配置校验和 runtime context 生产。
- `internal/template` 负责中间件注册与解析，不负责中间件细节实现。
- `internal/blueprint` 负责用户 service 输入的整理与基础归一化。
- `internal/runtime`、`internal/render` 负责消费 `[]EnvironmentContext` 并生成具体 runtime 产物。
- `internal/environment`、`internal/deployment`、`internal/coordinator` 共同负责“从产物到运行态管理”。
- `internal/store` 负责持久化边界，不承担业务编排。

### 注册与扩展规则

- pkg 注册键使用 `middleware + template + environmentType`。
- 同一个 pkg 实现可以在一次环境创建中被多次 `Configure` 调用，用于累计多个同类服务实例的配置。
- `pipeline` 统一使用 blueprint 中的 `service.name`，用于区分同一 middleware 的不同实例。
- `BuildRuntimeContext()` 在一次配置累计完成后统一输出 `[]EnvironmentContext`。
- 新增中间件或新增 runtime 时，应优先通过新增 pkg 实现接入，而不是修改主流程。

### 一期最小闭环

第一期优先打通最小可运行链路：

- 编排后端只支持 Docker Compose。
- 场景先支持单一中间件的最简单拓扑。
- 命令先支持 `create`、`status`、`destroy`。
- 持久化先采用本地文件存储。
- 目标是完成从 blueprint 定义到 pkg 产出 compose context，再到本地环境启动、查询、销毁的完整闭环。

## 一期目标

第一期先稳定目录骨架和模块职责边界，目标是：

- `pkg/` 能清晰承接中间件特有逻辑。
- `internal/` 能清晰承接平台核心逻辑。
- 后续新增 Docker Compose 能力时，不需要反复重构目录。
- 为第二阶段扩展到 Kubernetes 预留合理边界。
