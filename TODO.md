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
- [x] 实现 runtime/render 基础能力，消费 `[]EnvironmentContext` 生成完整 Compose bundle
- [x] 定义 deployment 接口并实现 compose 后端，支持 `build.sh` 驱动的真实 Compose 生命周期执行
- [x] 实现 environment 生命周期和状态持久化骨架，并补充 runtime artifact 持久化
- [x] 实现 coordinator 的 `create`、`status`、`start`、`stop`、`destroy`
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
- [x] Compose renderer 已消费真实 middleware 语义，生成完整 Compose bundle：`docker-compose.yml / .env / build.sh / check.sh / README.md`
- [x] `EnvironmentContext` 已贯穿 `Prepare / Render / Apply` 三阶段，并收敛出 `PreparePlan / RenderPlan / ApplyPlan / LifecyclePlan`
- [x] Compose runtime 私有字段已从公共 `Environment` 中逐步拆出，改由 runtime artifact 和阶段 plan 承接
- [x] `pkg/mysql/single.go` 已支持一个 blueprint 内多实例 single MySQL
- [x] `pkg/mysql/single.go` 已支持通过 `version` 选择 `v5.7 / v8.0`，并将版本差异收敛到单实现入口

#### P1

- [x] 为一期样板中间件补齐多实例配置缓存与 compose context 生成
- [x] 增加 `start`、`stop`
- [x] 将 `mysql/single` 的 Compose 渲染与版本差异（`v5.7 / v8.0`）收敛到单一 `pkg` 实现
- [x] 增加 `list`
- [x] 补充错误恢复和失败清理逻辑
- [x] 将 `start` 生命周期恢复路径补齐为稳定的真实集成测试
- [ ] 补充更多基础单元测试和主链路集成测试
- [ ] 继续收敛 `docker compose` 与 `podman compose` 的兼容性问题，并完善 Podman 独立 deployment 实现
- [ ] 补充模板管理的完整增强能力，并继续完善蓝图管理剩余 CRUD

#### P2

- [ ] 抽象第二个中间件，验证 `pkg/*` 作为唯一扩展点是否稳定
- [ ] 为未来 K8s 后端补齐 runtime context 和 pkg 实现扩展点
- [ ] 补充 `doctor` 二期能力，支持结构化诊断结果而不是仅执行 `check.sh`
- [ ] 完善项目级帮助文档与示例库，补充 blueprint / template / 多中间件组合的系统示例

当前已完成：本地 blueprint `list / show / validate`，内置 template `list / show`

### 多中间件组合测试 TODO

- [x] `mysql + redis` 组合蓝图：验证 `up -> doctor -> down`
- [x] `mysql + redis` 组合蓝图：验证 `create -> start -> doctor -> down`
- [x] `mysql + redis + rabbitmq` 组合蓝图：验证 `up -> doctor -> down`
- [x] `mysql + redis + rabbitmq` 组合蓝图：验证 `create -> start -> doctor -> down`
- [x] `postgresql + kafka` 组合蓝图：验证 `up -> doctor -> down`

### 最近完成

- [x] 定义并落地 runtime driver 统一抽象，覆盖 `Prepare / Render / Apply / Status / Start / Stop / Destroy / Cleanup`
- [x] 新增 Compose runtime driver、renderer、deployment executor 与 environment file store
- [x] 将 `CreateResult` / lifecycle result 收敛为面向 CLI 输出的用户提示文案
- [x] 修复 compose executor 的相对路径问题，统一使用绝对 `workdir` / `compose file`
- [x] 修复 `docker compose ps --format json` 在不同输出形态下的解析兼容性
- [x] 修复 stop 后 `ps` 空结果导致误判 destroyed 的问题，统一使用 `docker compose ps -a --format json`
- [x] 引入 `PreparePlan / RenderPlan / ApplyPlan / LifecyclePlan`，将主流程与生命周期阶段显式化
- [x] 引入 runtime artifact 持久化，承接 `PrimaryFile / WorkspaceDir / ProjectName` 等 runtime 私有信息
- [x] 为 `pkg/*` Compose 版实现补充规范文档，并建立 `compose-stack -> docker/目录事实 -> pkg 单实现入口` 的设计约束
- [x] `config/zygarde.yaml` 已可启动两个 MySQL single，并支持 `v5.7 + v8.0` 混合版本
- [x] 新增 `internal/tool/number_dispenser.go` 单任务级端口分发器，用于默认端口冲突规避与显式端口校验
- [x] 蓝图管理一期：本地 blueprint `list / show / validate`
- [x] 模板管理一期：内置 template `list / show`
- [x] Podman deployment 已从通用 compose executor 中拆出独立执行分支，并开始收敛 `start/down` 差异

### 下一阶段平台模块 TODO

#### Blueprint 管理

- [x] 增加 `blueprint create`
- [x] 增加 `blueprint copy`
- [x] 增加 `blueprint delete`
- [x] 增加 `blueprint edit`
- [x] 增加 `blueprint update`
- [x] 支持按 blueprint 名称执行，而不只按文件路径执行

#### Template 管理

- [ ] 增加 `template validate` 命令入口
- [ ] 校验模板元数据与 `internal/template` 实际注册能力一致，避免出现 catalog 与 pkg 注册漂移
- [ ] 校验默认模板标记与 `GetDefaultMiddleware(...)` 的解析结果一致
- [ ] 校验模板支持版本声明与 `pkg/<middleware>/<template>` 实际支持范围一致
- [ ] 增加模板帮助文档关联检查，确保 `docs/<middleware>.md` 存在且与模板元数据同步
- [ ] 增加模板说明字段完整性检查，避免空 description / docPath / runtimeType
- [ ] 为 `template list/show` 补充默认值/支持版本展示增强
- [ ] 评估是否增加 `template defaults/show-values` 一类只读能力，用于展示模板最小示例和值说明
- [ ] 明确 template 管理当前只支持内置只读模板，不支持外部模板的 create/update/delete；若后续不支持，应在 README/帮助文案中写清楚
- [ ] 为 `template validate` 与模板展示命令补 `test/command/` 命令测试和异常用例

#### 生命周期与诊断

- [ ] 为 `doctor` 增加结构化结果模型，区分配置检查、运行检查、脚本检查
- [ ] 评估 `stop / destroy` 的长期用户语义，明确与 `down` 的关系
- [ ] 增加更多组合场景下的异常恢复测试
- [ ] 继续完善 Podman 下 `status / doctor / down` 的兼容性边界处理

#### Runtime 扩展

- [ ] 为未来 K8s runtime 抽出最小可复用的 driver / render / deployment 扩展点
- [ ] 评估 `pkg` 模板元数据对多 runtime 的建模方式，避免只绑定 Compose

### 下一阶段重点：12 个中间件的 Compose type 实现

目标：参考 `compose-stack` 与 `docker/<middleware>/<scenario>_<version>/` 已验证资产，将 12 个中间件、26 个部署模板的 Compose 版能力统一收敛到 `pkg/*` 的单实现入口中。

约束：

- 每个中间件都应优先通过单一 `pkg/<middleware>/<template>.go` 入口承接多版本差异
- 版本差异尽量通过 `version` 控制，不复制多份近似实现
- 输出必须符合当前 `EnvironmentContext -> Prepare / Render / Apply` 主流程与 Compose bundle 规范

#### Compose type 中间件 TODO

说明：

- TODO 按 `middleware + template` 维度维护
- `version` 是同一 template 的支持参数，不单独拆成多个待办项
- 每完成一个 template，需同步交付 `pkg + docs + test/command`

当前整体进度：

- 已完成：`26 / 26`（`mysql/single`、`mysql/master-slave`、`redis/single`、`redis/master-slave`、`redis/cluster`、`mongodb/single`、`mongodb/replica-set`、`mongodb/sharded`、`postgresql/single`、`postgresql/master-slave`、`rabbitmq/single`、`rabbitmq/cluster`、`kafka/single`、`kafka/cluster`、`tidb/single`、`tidb/cluster`、`etcd/single`、`etcd/cluster`、`consul/single`、`consul/cluster`、`clickhouse/single`、`clickhouse/cluster`、`zookeeper/single`、`zookeeper/cluster`、`elasticsearch/single`、`elasticsearch/cluster`）
- 进行中：`0 / 26`
- 未开始：`0 / 26`

MySQL（支持版本：`v5.7 / v8.0`）

- [x] `single`
- [x] `master-slave`

Redis（支持版本：`v6.2 / v7.4`）

- [x] `single`
- [x] `master-slave`
- [x] `cluster`

MongoDB（支持版本：`v6.0 / v7.0`）

- [x] `single`
- [x] `replica-set`
- [x] `sharded`

PostgreSQL（支持版本：`v16 / v17`）

- [x] `single`
- [x] `master-slave`

RabbitMQ（支持版本：`v4.2`）

- [x] `single`
- [x] `cluster`

Kafka（支持版本：`v4.2`）

- [x] `single`
- [x] `cluster`

TiDB（支持版本：`v6.7`）

- [x] `single`
- [x] `cluster`

etcd（支持版本：`v3.6`）

- [x] `single`
- [x] `cluster`

Consul（支持版本：`v1.20`）

- [x] `single`
- [x] `cluster`

ClickHouse（支持版本：`v24 / v25`）

- [x] `single`
- [x] `cluster`

ZooKeeper（支持版本：`v3.8 / v3.9`）

- [x] `single`
- [x] `cluster`

Elasticsearch（支持版本：`v8.18 / v8.19`）

- [x] `single`
- [x] `cluster`

### 后续追加原则

- 当前部分作为项目主流程 TODO。
- 后续可以围绕某个模块或子流程追加更细的子 TODO。
- 子 TODO 不应破坏这里定义的主流程顺序和模块边界。

---

## 子 TODO：mysql/single 全流程测试与 lifecycle/doctor 梳理

### 背景

- 当前 `mysql/single` 已基本跑通主链路，但缺少覆盖完整 create 生命周期的专项测试。
- `Create` 现已调整为纯创建语义，对齐 `docker compose create`，负责创建运行时资源但不启动环境。
- 缺少一个统一的健康检查入口来判断生成配置和 Docker 实际运行状态是否正确。
- 测试完成后需要明确资源回收路径，避免残留环境和容器。

### 目标

- 为 `mysql/single` 补齐完整的 create 生命周期测试。
- 明确 `Create / Start / Stop / Destroy` 的职责边界。
- 设计并落地 `doctor` 检查链路。
- 保证测试过程中的环境和 Docker 资源可回收。

### 范围

- 会改：
  - `test/command/mysql_test.go`
  - `internal/app`
  - `internal/cli`
  - `internal/coordinator`
  - 可能涉及 `internal/runtime` / `internal/deployment/compose`
- 本轮先不做：
  - 其他中间件测试
  - K8s 相关能力
  - 大范围重构 create 主流程

### TODO

#### P0

- [x] 梳理并调整 `Create` 语义：纯创建运行时资源但不启动环境
- [x] 为 `mysql/single` 设计 `test/command/mysql_test.go` 的完整测试方案
- [x] 明确测试回收策略，确保失败场景也能执行 `destroy/down`
- [x] 补齐 `internal/app/app.go` 的测试使用路径，确保 `Create / Stop / Start / Destroy` 能串成完整生命周期
- [x] 设计并实现 `doctor` 命令，一期通过执行环境目录下 `check.sh` 完成检查
- [x] 设计并实现 `down` 命令，语义对齐 `docker compose down`
- [x] 明确 `down` 与现有 `destroy` 的关系，优先复用当前 `destroy + cleanup` 链路
- [x] 调整 `mysql_test.go` 测试口径为 `up -> doctor -> down`

#### P1

- [x] 落地 `test/command/mysql_test.go`，覆盖 `create -> verify -> start -> doctor -> down`
- [x] 落地 `test/command/mysql_test.go`，覆盖 `up -> doctor -> down`
- [x] 在测试中校验生成的环境元数据、runtime artifact、Compose bundle 和 MySQL 实际可访问性
- [x] 评估并补充 `start` 的真实恢复验证，确认 stop 后可以重新启动成功
- [x] 调整 `Create` 与 `Start` 的职责边界：`Create` 纯创建，`Start` 负责启动
- [x] 在 `app / coordinator / runtime / deployment/compose` 打通 `doctor`
- [x] 在 `app / coordinator / runtime / deployment/compose` 打通 `down`
- [x] 为 `mysql/single` 落地全流程测试，覆盖 `up -> doctor -> down`
- [ ] 评估 `stop / destroy` 是否继续保留为用户命令，还是仅保留内部能力

#### P2

- [ ] 设计 `doctor` 命令的职责边界和输出格式
- [ ] 在 `app/coordinator/runtime` 上补 `doctor` 主链路
- [ ] 让 `doctor` 同时检查配置正确性和 Docker 运行态正确性
- [ ] 为 `doctor` 补基础测试和 CLI 入口
