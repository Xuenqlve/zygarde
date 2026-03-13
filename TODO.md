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
- 命令先支持 `create`、`status`、`destroy`。
- 持久化先采用本地文件存储。
- 目标是完成从 blueprint 定义到本地环境启动、查询、销毁的完整闭环。

### 主流程

1. `internal/cli` 接收用户命令和参数。
2. `internal/config` 读取并归一化运行配置。
3. `internal/app` 完成依赖装配，并将请求交给 `internal/coordinator`。
4. `internal/coordinator` 串联 `store`、`template`、`blueprint`、`render`、`runtime`、`deployment`、`environment` 完成一次完整操作。
5. `internal/store` 负责读取和保存 template、blueprint、environment 等对象。
6. `internal/template` 负责模板解析和变量校验。
7. `internal/blueprint` 负责模板引用关系和变量绑定。
8. `internal/render` 负责生成最终部署产物，例如 `docker-compose.yaml`。
9. `internal/runtime` 负责准备工作目录、产物路径和项目隔离信息。
10. `internal/deployment/compose` 负责执行 Docker Compose 部署动作。
11. `internal/environment` 负责记录环境状态、元数据和生命周期结果。

### 依赖顺序

1. 基础层：`internal/model`、`internal/config`、`internal/log`
2. 基础能力层：`internal/store`、`internal/runtime`、`internal/template`
3. 领域组装层：`internal/blueprint`、`internal/render`、`pkg/<middleware>`
4. 执行与状态层：`internal/deployment`、`internal/deployment/compose`、`internal/environment`
5. 编排与入口层：`internal/coordinator`、`internal/app`、`internal/cli`、`cmd/main.go`

### 主流程 TODO

#### P0

- [ ] 定义一期最小闭环的核心模型，落在 `internal/model`
- [ ] 定义全局配置结构，落在 `internal/config`
- [ ] 定义存储接口并实现本地文件存储，落在 `internal/store`
- [ ] 定义 runtime 目录布局与 project name 规则，落在 `internal/runtime`
- [ ] 确定一期样板中间件，优先 `mysql` 或 `redis`
- [ ] 实现 template 基础能力，支持模板读取和变量校验
- [ ] 实现 blueprint 基础能力，支持模板引用和变量绑定
- [ ] 实现 render 基础能力，生成 `docker-compose.yaml`
- [ ] 定义 deployment 接口并实现 compose 后端
- [ ] 实现 environment 生命周期和状态持久化
- [ ] 实现 coordinator 的 `create`、`status`、`destroy`
- [ ] 实现 CLI 入口并打通主链路

#### P1

- [ ] 为一期样板中间件补齐默认值、变量规范和场景定义
- [ ] 增加 `start`、`stop`、`list`
- [ ] 补充错误恢复和失败清理逻辑
- [ ] 补充基础单元测试和主链路集成测试

#### P2

- [ ] 抽象第二个中间件，验证 `pkg/*` 与 `internal/*` 边界是否稳定
- [ ] 为未来 K8s 后端预留 deployment 扩展点
- [ ] 补充模板管理、蓝图管理的完整 CRUD

### 后续追加原则

- 当前部分作为项目主流程 TODO。
- 后续可以围绕某个模块或子流程追加更细的子 TODO。
- 子 TODO 不应破坏这里定义的主流程顺序和模块边界。
