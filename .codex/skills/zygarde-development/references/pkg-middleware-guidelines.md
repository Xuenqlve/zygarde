# Pkg Middleware Guidelines

## 目标

`pkg/<middleware>` 是 Zygarde 中间件能力的唯一扩展点。

新增一个中间件能力时，优先只修改 `pkg/`，不要把中间件特有逻辑扩散到 `internal/*`。

## 适用范围

以下场景应遵循本规范：

- 新增一个 middleware
- 为现有 middleware 新增 template
- 为现有 middleware 新增 runtime 实现，例如 Compose 或 K8s
- 调整 middleware 默认值、配置校验或 context 生成逻辑

## 目录约束

建议按 middleware 和 template 组织文件：

- `pkg/mysql/single.go`
- `pkg/mysql/cluster.go`
- `pkg/redis/single.go`

如同一个 template 未来存在多个 runtime 实现，可继续按语义拆分：

- `pkg/mysql/single.go`
- `pkg/mysql/single_compose.go`
- `pkg/mysql/single_k8s.go`

## 核心职责

`pkg/<middleware>` 负责：

- 注册 middleware 实现
- 提供默认 template
- 补齐用户未填写的默认配置
- 校验 middleware 自身配置
- 为后续 runtime 阶段生产 `EnvironmentContext`

`pkg/<middleware>` 不负责：

- CLI 解析
- blueprint 文件读取
- environment 生命周期编排
- deployment 执行
- 全局状态持久化

## 注册规范

middleware 注册键使用：

- `middleware + template + environmentType`

示例：

- `mysql + single + compose`
- `mysql + single + k8s`

注册应放在 `init()` 中完成，保证 `internal/*` 不需要显式调用某个 middleware 的注册方法。

推荐模式：

1. `pkg/<middleware>` 内部 `init()` 调用 `Register(...)`
2. `pkg/register` 通过 blank import 聚合所有 middleware 包
3. `internal/app` 只依赖 `pkg/register`，不依赖具体 middleware

## Middleware 接口职责

当前 middleware 实现应至少满足：

- `Middleware() string`
- `Template() string`
- `IsDefault() bool`
- `Configure(input, index)`
- `BuildRuntimeContext(service, runtimeType)`

约束如下：

- `Middleware()` 返回固定 middleware 名称，例如 `mysql`
- `Template()` 返回固定 template 名称，例如 `single`
- `IsDefault()` 用于声明该 template 是否是某个 middleware 在某 runtime 下的默认 template
- `Configure(...)` 负责默认值补齐与配置校验
- `BuildRuntimeContext(...)` 负责生成 runtime 可消费的上下文

## Configure 规范

`Configure(...)` 是 middleware 配置阶段的主入口。

它应负责：

- 根据 middleware 默认值补齐用户配置
- 校验必填项和字段类型
- 返回标准化后的 `BlueprintService`

它不应负责：

- 执行部署
- 写文件
- 更新 environment 状态

推荐行为：

- 若 `name` 为空，使用统一默认命名规则
- 若 `values` 为空，初始化为空 map
- 所有 middleware 私有默认值都在这里补齐
- 所有 middleware 私有校验都在这里完成

## BuildRuntimeContext 规范

`BuildRuntimeContext(...)` 用于把 middleware 配置转换成 runtime 拼装所需的上下文。

它应负责：

- 接收已标准化的 service
- 输出某个 runtime 可消费的 `EnvironmentContext`
- 不泄漏 middleware 外部不需要理解的实现细节

它不应负责：

- 直接生成最终 `docker-compose.yaml`
- 直接执行 `docker compose` 或 `kubectl`

## 默认值策略

middleware 应优先做到“用户最少输入”。

推荐原则：

- 用户必填尽量只保留 `middleware`
- `template` 如可推导，应提供默认值
- `values` 尽量由 middleware 自动补齐默认值
- `name` 如未填写，应按统一规则生成

## 状态与缓存约束

如果 middleware 实现内部需要在一次 create 流程中累计多份配置：

- 必须只面向当前一次请求的生命周期生效
- 不要把某次 create 的缓存泄漏到下一次 create
- 如采用内部缓存，必须明确何时创建、何时消费、何时清理

在没有完整生命周期设计前，优先保持 middleware 实现简单、可推断。

## 新增 middleware 的最低要求

新增一个 middleware 时，至少完成：

1. 新建 `pkg/<middleware>/...` 文件
2. 实现 middleware 接口
3. 在 `init()` 中注册对应 runtime 实现
4. 加入 `pkg/register`
5. 补充默认值和基础校验
6. 若交付的是 Compose 版用户可用能力，同步补充或更新 `docs/<middleware>.md` 帮助文档
7. 补充最小编译验证或测试

## 开发原则

- 先保证 Compose 路径跑通，再扩展 K8s
- 优先共享同一个 middleware/template 的规范逻辑，避免复制粘贴
- 不要在 `internal/*` 中用 `if middleware == ...` 分支承接中间件特有逻辑
- 所有 middleware 特有行为优先收敛在 `pkg/<middleware>`
