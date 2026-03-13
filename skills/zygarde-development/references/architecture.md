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

项目主流程按“定义 -> 渲染 -> 部署 -> 管理”组织。

### 创建环境主链路

1. 用户通过 CLI 发起环境创建请求。
2. `internal/cli` 解析命令参数。
3. `internal/config` 加载全局配置和运行参数。
4. `internal/app` 装配依赖并调用 `internal/coordinator`。
5. `internal/coordinator` 从 `internal/store` 读取 blueprint 和 template 定义。
6. `internal/template` 解析模板并校验变量定义。
7. `internal/blueprint` 整理模板引用关系和变量绑定结果。
8. `internal/render` 生成最终 `docker-compose.yaml` 等部署产物。
9. `internal/runtime` 为当前 environment 准备工作目录、产物路径和唯一项目名。
10. `internal/deployment/compose` 执行 `docker compose up -d`。
11. `internal/environment` 持久化环境元数据和状态。
12. CLI 返回环境 ID、名称、路径和访问端点等结果。

### 管理类主链路

- `status`：读取 environment 元数据和当前状态，必要时结合 deployment 查询结果返回。
- `start`：基于已有 runtime 目录和产物重新触发 deployment。
- `stop`：调用 deployment 停止对应 environment。
- `destroy`：调用 deployment 销毁环境，并由 environment 更新最终状态与清理结果。

### 模块协作原则

- `pkg/<middleware>` 负责提供中间件特有能力，不直接驱动主流程。
- `internal/template`、`internal/blueprint`、`internal/render` 共同负责“从定义到产物”。
- `internal/environment`、`internal/deployment`、`internal/coordinator` 共同负责“从产物到运行态管理”。
- `internal/store` 负责持久化边界，不承担业务编排。

### 一期最小闭环

第一期优先打通最小可运行链路：

- 编排后端只支持 Docker Compose。
- 场景先支持单一中间件的最简单拓扑。
- 命令先支持 `create`、`status`、`destroy`。
- 持久化先采用本地文件存储。
- 目标是完成从 blueprint 定义到本地环境启动、查询、销毁的完整闭环。

## 一期目标

第一期先稳定目录骨架和模块职责边界，目标是：

- `pkg/` 能清晰承接中间件特有逻辑。
- `internal/` 能清晰承接平台核心逻辑。
- 后续新增 Docker Compose 能力时，不需要反复重构目录。
- 为第二阶段扩展到 Kubernetes 预留合理边界。
