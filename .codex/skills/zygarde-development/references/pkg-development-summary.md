# Pkg Development Summary

`pkg/*` 开发中间件时，优先阅读本文件，再按需展开细则。

本文件用于收敛以下规范的核心约束，避免只读其中一份时遗漏关键规则：

- `pkg-middleware-guidelines.md`
- `pkg-compose-template-guidelines.md`

## 目标

`pkg/<middleware>` 是 Zygarde 中间件能力的唯一扩展点。

AI 开发中间件时，应优先把中间件私有逻辑收敛到 `pkg/*`，不要把中间件默认值、版本差异、校验逻辑、Compose 资产片段散落到 `internal/*`。

## 第一原则

`docker/<middleware>/<scenario>_<version>/` 是 Compose 版中间件行为的事实来源。

这意味着：

- AI 不应凭经验重新发明一套 Compose 配置
- AI 必须先阅读对应 `docker/` 目录，再设计 `pkg/*` 实现
- `pkg/*` 的职责是把这些已验证行为抽象收敛，而不是替代它们

例如：

- `docker/mysql/single_v5.7/`
- `docker/mysql/single_v8.0/`

应收敛为：

- `pkg/mysql/single.go`

通过 `version` 承接版本差异，而不是复制多个平行实现。

## 开发顺序

开发一个 Compose 版中间件时，推荐按以下顺序执行：

1. 阅读对应 `docker/<middleware>/<scenario>_<version>/` 目录
2. 提炼场景共性与版本差异
3. 确定 `pkg/<middleware>/<template>.go` 的单入口落点
4. 在 `Normalize / Configure / Validate / BuildRuntimeContexts` 中完成实现
5. 同步补充 `docs/<middleware>.md`
6. 同步评估并补充 `test/command/<middleware>_test.go`

## 职责边界

`pkg/*` 负责：

- 注册 middleware/template/runtime 实现
- 提供默认 template
- 暴露模板元数据，例如支持版本、默认模板标记、帮助文档路径、模板说明
- 补齐用户配置默认值
- 校验 middleware 自身参数
- 生成 `EnvironmentContext`
- 输出 Compose service 与 bundle 资产片段

`pkg/*` 不负责：

- CLI 参数解析
- blueprint 文件读取
- environment 生命周期编排
- runtime/deployment 执行
- 全局状态持久化

需要额外强调：

- 如果平台要对外提供 `template list / template show` 这类模板管理能力，模板事实来源仍然必须在 `pkg`
- `internal/*` 只负责读取和展示，不应再定义一份独立的模板白名单或版本矩阵
- 可以使用集中式的 `pkg/catalog` 一类聚合注册表，但本质上仍属于 `pkg` 侧能力

## 目录与实现规则

- 一个 `middleware + template` 在 `pkg/*` 中优先使用单一入口文件
- 版本差异通过 `version` 控制，不按版本复制多个近似实现
- 中间件独有辅助逻辑应继续收敛在对应 `pkg/<middleware>` 下
- 没有明确收益时，不要提前拆过多文件

## 默认值与校验规则

- 默认值、类型归一化和参数校验应尽量在 `Normalize / Configure` 阶段完成
- 用户显式配置的值应优先校验，不要静默改写
- 用户未配置的值，才允许通过默认值策略或工具自动补齐
- 对未支持的版本，应直接报错，不允许静默回退
- 已知版本的命令、SQL 或检查差异，应直接按 `version` 选择，不要依赖运行时试错 fallback；细则见 `version-compatibility-guidelines.md`

## Compose 交付三件套

每交付一个 Compose 版中间件用户可运行能力，至少同步交付三部分：

1. `pkg/<middleware>/<template>.go`
2. `docs/<middleware>.md`
3. `test/command/<middleware>_test.go`

只改代码、不补文档或测试，交付不完整。

如果该交付会影响平台侧可发现性，还应同步更新 `pkg` 中的模板元数据注册内容，使 `template list / template show` 能反映真实能力。

## 文档规则

帮助文档应面向用户，而不是面向 AI 或研发过程。

要求：

- 文档按 template 维度清晰区分
- 参数说明以当前真实实现为准
- 说明默认值、可选值、版本说明、固定行为和可执行示例
- 不写“当前已实现 / 后续预留 / 内部实现来源”这类研发流水账

## Compose 集成测试规则

中间件 Compose 集成测试应优先验证用户主链路，而不是重复验证底层 executor 细节。

推荐口径：

- `up -> doctor -> down`

要求：

- 默认按当前目录环境工作
- `doctor` 验证检查脚本可通过
- `down` 验证运行资源与工作目录可回收
- 测试失败时必须具备兜底清理

通用测试骨架应优先沉淀到：

- `test/command/base.go`

单个中间件测试文件只保留：

- blueprint 生成
- 场景参数
- 中间件私有断言

## 测试目录规则

- 新增测试统一优先放在 `test/` 目录下管理
- 单元测试按主题拆分到 `test/<domain>/`
- 用户命令主链路与 Compose 生命周期功能测试统一放在 `test/command/`
- 不要继续在 `internal/*` 或 `pkg/*` 下新增零散测试文件，除非有明确理由且已说明

## 当前项目特有约束

- 当前项目按“一次执行完成即退出”的单任务 CLI 工具来设计
- 端口分发、编号分配等一次性辅助能力可以放在 `internal/tool/*` 中实现为单任务级全局工具
- Compose 命令执行当前已支持通过配置切换容器引擎，例如 `docker` / `podman`
- 当前目录环境标记用于支撑 `status / doctor / down` 的用户体验，不要求用户显式记忆 `EnvironmentID`
- 模板管理命令是平台入口，但模板元数据维护责任在 `pkg`，不是在 `internal`

## 禁止事项

- 不要脱离 `docker/<middleware>/<scenario>_<version>/` 凭经验重写 Compose 实现
- 不要把中间件特有逻辑散落到 `internal/*`
- 不要把模板可用性清单、版本矩阵、默认模板映射重新硬编码到 `internal/*`
- 不要为不同版本复制多个近似 `pkg` 实现文件
- 不要只补代码，不补文档和测试
