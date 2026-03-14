# zygarde TODO

## Phase 1（已完成）: compose-stack 技能化闭环

目标：围绕中间件测试场景，形成统一的 **设计(generate) + 验收(verify/build/check) + 回收(cleanup/down -v)** 流程。

### ✅ 已完成能力矩阵

- MySQL（single / cluster）
- Redis（single / cluster）
- MongoDB（single / cluster）
- PostgreSQL（single / cluster）
- RabbitMQ（single / cluster）
- Kafka（single / cluster）
- TiDB（single / cluster）
- etcd（single / cluster）
- Consul（single / cluster）
- ClickHouse（single / cluster，v24/v25）
- ZooKeeper（single / cluster，v3.8/v3.9）
- Elasticsearch（single / cluster，v8.18/v8.19）

### ✅ 产出

1. `skills/compose-stack` 下完成脚本实现与统一入口接线。
2. 所有中间件均完成真实环境验收（含踩坑修复后复验）。
3. README + SKILL 文档完成经验沉淀（版本/tag、端口冲突、容器名冲突、配置兼容、验收口径）。

---

## Phase 2（进行中）: Golang 模板化编排引擎

目标：根据用户需求，自动拼装多中间件模板，输出可直接运行的 `docker-compose.yaml`，快速构建临时测试环境。

### 核心方向

- 模板标准化：定义模板元数据、变量规范、依赖关系。
- 蓝图（Blueprint）机制：支持多模板组合与变量注入。
- 环境管理：创建、状态追踪、销毁、隔离运行。
- 部署执行：统一执行 `up/down/start/stop` 并反馈状态。
- 协调器（Facade）：对外提供统一 CLI/API 接口。

### 一期最小闭环

- 编排后端先只支持 Docker Compose。
- 场景先支持单一中间件的最简单拓扑。
- 命令先支持 `create`、`status`、`destroy`，并预留 `start`、`stop` 生命周期入口。
- 持久化先采用本地文件存储。
- 目标是完成从 blueprint 定义到 pkg 产出 compose context，再到本地环境启动、查询、销毁的完整闭环。

### 主流程

1. `internal/cli` 接收用户命令和参数。
2. `internal/config` 读取运行参数与平台默认配置。
3. `internal/store` 读取 `zygarde.yaml` 或显式指定的 blueprint 文件。
4. `internal/blueprint` 整理 `services` 并补齐基础默认值。
5. `internal/app` 完成依赖装配，并将请求交给 `internal/coordinator`。
6. `internal/coordinator` 按 `middleware + template + environmentType` 从注册器中解析 pkg 实现。
7. `internal/coordinator` 对每个 service 调用 pkg 的 `Configure(...)`，按同一 middleware 实例累计多份配置。
8. 同一个 pkg 实现在一次环境创建中可以被多次 `Configure`，用于累计多个同类服务实例的配置。
9. 所有 service 配置完成后，pkg 统一通过 `BuildRuntimeContexts()` 输出 `[]EnvironmentContext`。
10. `internal/runtime` / `internal/render` 消费所有 context，生成 Compose 产物。
11. `internal/deployment/compose` 执行 Docker Compose 部署动作。
12. `internal/environment` 负责记录环境状态、元数据和生命周期结果。

### 依赖顺序

1. 基础层：`internal/model`、`internal/config`、`internal/log`
2. 基础能力层：`internal/store`、`internal/template`、`internal/runtime`
3. 领域组装层：`internal/blueprint`、`pkg/<middleware>`、`internal/render`
4. 执行与状态层：`internal/deployment`、`internal/deployment/compose`、`internal/environment`
5. 编排与入口层：`internal/coordinator`、`internal/app`、`internal/cli`、`cmd/main.go`

### 主流程 TODO

#### P0

- [x] 设计并打通 `zygarde create -f zygarde.yaml` 的前半段链路：`main -> cli -> app -> coordinator -> store/blueprint/template`
- [x] 定义一期最小闭环的核心模型，落在 `internal/model`
- [x] 定义平台默认配置与运行参数结构，落在 `internal/config`
- [x] 定义存储接口并实现本地文件存储，落在 `internal/store`
- [x] 定义 middleware 注册键和注册器能力，支持 `middleware + template + environmentType`
- [x] 实现 blueprint 基础能力，支持 service 默认补齐与唯一性校验
- [x] 实现 pkg 的 `Configure` / `BuildRuntimeContexts` 主流程，先以 `mysql + single + compose` 与 `mock + echo + compose` 跑通
- [x] 定义 runtime 目录布局与 project name 规则，落在 `internal/runtime`
- [x] 实现 runtime/render 基础能力，消费 `[]EnvironmentContext` 生成 `docker-compose.yaml`
- [x] 定义 deployment 接口并实现 compose 后端骨架
- [x] 实现 environment 生命周期和状态持久化骨架
- [x] 实现 coordinator 的 `create`、`status`、`destroy`
- [x] 实现 CLI 入口并打通主链路

### create 前半段 TODO

- [x] `cmd/main.go` 只保留 CLI 启动入口
- [x] `internal/cli` 增加 `create` 命令与 `-f/--file` 参数
- [x] `internal/cli` 定义 `CreateRequest`
- [x] `internal/app` 装配 create 所需依赖
- [x] `internal/store` 提供 `LoadBlueprint(path)` 能力
- [x] `internal/blueprint` 实现 service 默认补齐与唯一性校验
- [x] `internal/coordinator.Create` 完成 blueprint 读取、service 遍历和 pkg `Configure(...)` 调用
- [x] `internal/coordinator.Create` 输出后续 `BuildRuntimeContext()` 所需的中间结果

### 当前进度

- [x] 单文件 `zygarde.yaml` 模型已确定，并已支持基础 YAML 读取和默认文件发现
- [x] `mysql + single + compose` 的样板 middleware 已接入注册器
- [x] `mock + echo + compose` 已接入注册器，可用于前半段链路调试
- [x] `BuildRuntimeContexts()` 的多实例累计与统一输出已接入 `coordinator.Create`
- [x] `create -> runtime prepare -> render -> compose apply -> environment save` 已形成可执行主链路
- [x] `status`、`start`、`stop`、`destroy` CLI 已接入 `coordinator -> runtime driver -> compose executor`
- [x] compose executor 已具备真实 `docker compose up/down/start/stop/ps -a` 执行能力
- [x] compose executor 已补 fake runner 单测与基于 `docker/mysql/single_v5.7` 的真实 MySQL 集成测试
- [ ] Compose renderer 当前仍为占位实现，尚未消费真实 middleware 语义生成可运行服务定义
- [ ] `create` 主链路虽已完整接通，但对真实中间件模板的渲染仍需继续完善

#### P1

- [x] 为一期样板中间件补齐多实例配置缓存与 compose context 生成
- [x] 增加 `start`、`stop`
- [ ] 增加 `list`
- [ ] 补充错误恢复和失败清理逻辑
- [ ] 将 `start` 生命周期恢复路径补齐为稳定的真实集成测试
- [ ] 补充更多基础单元测试和主链路集成测试

#### P2

- [ ] 抽象第二个中间件，验证 `pkg/*` 作为唯一扩展点是否稳定
- [ ] 为未来 K8s 后端补齐 runtime context 和 pkg 实现扩展点
- [ ] 补充模板管理、蓝图管理的完整 CRUD

### 最近完成

- [x] 定义并落地 runtime driver 统一抽象，覆盖 `Prepare / Render / Apply / Status / Start / Stop / Destroy / Cleanup`
- [x] 新增 Compose runtime driver、renderer、deployment executor 与 environment file store
- [x] 将 `CreateResult` / lifecycle result 收敛为面向 CLI 输出的用户提示文案
- [x] 修复 compose executor 的相对路径问题，统一使用绝对 `workdir` / `compose file`
- [x] 修复 `docker compose ps --format json` 在不同输出形态下的解析兼容性
- [x] 修复 stop 后 `ps` 空结果导致误判 destroyed 的问题，统一使用 `docker compose ps -a --format json`

### 后续追加原则

- 当前部分作为项目主流程 TODO。
- 后续可以围绕某个模块或子流程追加更细的子 TODO。
- 子 TODO 不应破坏这里定义的主流程顺序和模块边界。
