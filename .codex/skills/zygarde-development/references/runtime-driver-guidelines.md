# Runtime Driver Guidelines

## 目标

为 Zygarde 定义统一的 runtime driver 规则，使 Docker Compose 与未来的 Kubernetes runtime 都能复用同一套主流程和生命周期抽象。

runtime 的职责不是承载 middleware 细节，而是消费标准化后的 `[]EnvironmentContext`，完成运行时产物生成与环境生命周期操作。

## 适用范围

以下场景应遵循本规范：

- 设计或实现 `internal/runtime` 抽象
- 新增 `compose` runtime
- 新增 `k8s` runtime
- 调整 create / status / start / stop / destroy / cleanup 主流程
- 调整 runtime 与 render / deployment / environment 的边界

## 核心原则

- runtime 只消费标准 `[]EnvironmentContext`，不理解某个 middleware 的私有分支。
- middleware 特有默认值、校验和 context 构造逻辑必须留在 `pkg/*`。
- runtime 必须覆盖整个环境生命周期，而不只是 create 阶段。
- runtime 必须提供稳定的工作目录、项目标识和元数据边界。
- 生命周期操作必须基于已持久化的 environment 元数据执行，而不是每次重新解析 blueprint。
- Compose 与 K8s 必须复用同一套 driver 接口语义，只允许具体实现不同。

## 分层职责

建议按以下分层理解主流程：

1. `pkg/*`
   - 负责 middleware 特有逻辑
   - 输出标准 `EnvironmentContext`
2. `internal/runtime`
   - 定义统一 driver 接口
   - 管理 runtime layout、生命周期动作和运行时标识
3. `internal/render`
   - 把 `[]EnvironmentContext` 渲染为某个 runtime 的产物
   - Compose 输出 `docker-compose.yaml`
   - K8s 输出 manifests
4. `internal/environment`
   - 持久化环境元数据、状态、产物路径和访问端点
5. `internal/coordinator`
   - 编排 blueprint、middleware、runtime、environment 全流程

## Driver 能力面

统一 runtime driver 应覆盖以下能力：

- `Prepare`
  - 初始化 runtime 工作目录、环境标识、项目名等基础信息
- `Render`
  - 根据 `[]EnvironmentContext` 生成 runtime 产物
- `Apply`
  - 执行环境创建或更新
- `Status`
  - 查询环境当前运行状态
- `Start`
  - 启动已存在但处于停止态的环境
- `Stop`
  - 停止环境但保留元数据和产物
- `Destroy`
  - 下线并销毁环境运行资源
- `Cleanup`
  - 清理本地产物、临时文件或残留工作目录

## 推荐接口

推荐先围绕统一 driver 接口设计：

```go
type Driver interface {
    Type() EnvironmentType

    Prepare(ctx context.Context, req PrepareRequest) (*PreparedRuntime, error)
    Render(ctx context.Context, req RenderRequest) (*RenderResult, error)

    Apply(ctx context.Context, env model.Environment) (*OperationResult, error)
    Status(ctx context.Context, env model.Environment) (*StatusResult, error)
    Start(ctx context.Context, env model.Environment) (*OperationResult, error)
    Stop(ctx context.Context, env model.Environment) (*OperationResult, error)
    Destroy(ctx context.Context, env model.Environment) (*OperationResult, error)
    Cleanup(ctx context.Context, env model.Environment) (*OperationResult, error)
}
```

约束如下：

- `Prepare` 与 `Render` 属于产物阶段。
- `Apply`、`Status`、`Start`、`Stop`、`Destroy`、`Cleanup` 属于生命周期阶段。
- 所有 runtime 都必须遵守同一组方法语义。

## 推荐领域对象

### RuntimeLayout

用于描述一个环境的运行时目录布局。

建议至少包含：

- `RootDir`
- `RenderDir`
- `DataDir`
- `LogsDir`
- `MetadataFile`

### PreparedRuntime

用于描述 runtime 初始化后的基础信息。

建议至少包含：

- `EnvironmentID`
- `Name`
- `Type`
- `ProjectName`
- `Layout`

### RenderResult

用于描述 runtime 产物生成结果。

建议至少包含：

- `Artifacts`
- `PrimaryFile`

### OperationResult

用于描述一次生命周期操作的结果。

建议至少包含：

- `Message`
- `Changed`
- `Endpoints`

### StatusResult

用于描述环境状态查询结果。

建议至少包含：

- `Status`
- `Message`
- `Endpoints`

## 主流程约束

完整 create 主流程建议按以下顺序组织：

1. `store` 读取 blueprint
2. `blueprint` 归一化 services
3. `coordinator` 调用 middleware `Configure(...)`
4. middleware 统一输出 `[]EnvironmentContext`
5. `runtime.Prepare`
6. `runtime.Render`
7. `runtime.Apply`
8. `environment` 持久化环境元数据、状态和端点
9. CLI 输出创建结果

管理类流程建议如下：

- `status`
  - 读取 environment 元数据
  - 定位 runtime driver
  - 调用 `Status`
  - 更新或返回当前状态
- `start`
  - 基于 environment 元数据调用 `Start`
- `stop`
  - 基于 environment 元数据调用 `Stop`
- `destroy`
  - 基于 environment 元数据调用 `Destroy`
  - 视需要再调用 `Cleanup`

## Compose 与 K8s 映射要求

Docker Compose 与 Kubernetes runtime 必须遵守同一套抽象语义。

示例映射：

- Compose
  - `Prepare`: 初始化工作目录与 compose project name
  - `Render`: 生成 `docker-compose.yaml`
  - `Apply`: 执行 `docker compose up -d`
  - `Status`: 查询 compose project / container 状态
  - `Start`: 执行 `docker compose start`
  - `Stop`: 执行 `docker compose stop`
  - `Destroy`: 执行 `docker compose down`
  - `Cleanup`: 删除本地工作目录和临时产物
- K8s
  - `Prepare`: 初始化 manifest 目录、namespace 或应用标识
  - `Render`: 生成 YAML manifests
  - `Apply`: 执行 `kubectl apply -f`
  - `Status`: 查询 namespace / workload / pod / service 状态
  - `Start`: 恢复 workload 到运行态
  - `Stop`: 缩容或停止 workload
  - `Destroy`: 删除 runtime 资源
  - `Cleanup`: 删除本地 manifest 产物

## 边界规则

- 不要让 runtime 直接理解 blueprint 结构。
- 不要让 runtime 直接承接 middleware 私有默认值或校验逻辑。
- 不要让 `coordinator` 同时面对两套平行抽象，例如既编排 runtime 又单独编排 deployment。
- 不要让 `status/start/stop/destroy` 重新依赖 blueprint 作为运行事实来源。
- 运行事实应以 `internal/environment` 持久化结果为准。

## 当前阶段建议

当前优先级建议如下：

1. 先稳定 runtime driver 接口与 layout 规则
2. 先实现 Compose runtime 的 `Prepare + Render + Apply`
3. 再补 `Status + Destroy`
4. 再补 `Start + Stop + Cleanup`
5. 最后按相同接口扩展 K8s runtime

在第一期闭环尚未完成前，不要一次性实现全部 runtime 能力；但接口和职责边界应一次设计清楚。
