---
name: zygarde-development
description: Zygarde 项目的 AI 开发规范。用于本项目的功能开发、修复、重构、目录调整与架构演进，覆盖需求理解、方案设计、TODO 管理、目录职责边界、实现约束、验证与提交流程；当工作涉及 pkg 中间件能力或 internal 核心模块时必须使用。
---

# zygarde-development

Zygarde 项目的统一开发 skill。

## 适用范围

以下场景必须使用本 skill：
- 新功能开发
- 缺陷修复
- 架构调整或目录调整
- `pkg/` 下中间件能力建设
- `internal/` 下平台核心模块建设

## 工作原则

- 先理解需求和上下文，再设计方案，再执行实现。
- 始终维护可追踪的 TODO，不跳步、不一口气做完所有修改。
- 优先最小可行改动，避免无关重构和无边界扩散。
- 严格遵守 `pkg/` 与 `internal/` 的职责边界。
- 平台级抽象优先稳定、清晰、可扩展，不为短期实现破坏长期结构。

## 标准工作流

1. 阅读相关代码、README、配置和目录结构。
2. 明确目标、边界、输入输出、风险和约束。
3. 产出简洁的开发方案；如果存在明显分歧，列出可选方案与权衡。
4. 按 [todo-template.md](references/todo-template.md) 维护 TODO。
5. 实现时按 TODO 分步推进，每一步都确认落点是否正确。
6. 完成后执行必要验证，至少覆盖编译、测试或目录结果检查。
7. 总结改动并提交。

## 目录落点规则

目录职责说明见 [architecture.md](references/architecture.md)。

必须遵守以下规则：
- 某个中间件独有的模板变量、默认值、校验和辅助逻辑，进入 `pkg/<middleware>`。
- 模板管理、蓝图编排、环境状态、部署流程、存储抽象等平台核心逻辑，进入 `internal/*`。
- 不要把环境生命周期编排逻辑放进 `pkg/*`。
- 不要把某一种中间件的硬编码分支散落在多个 `internal/*` 模块中。
- 渲染逻辑与部署执行逻辑分离。
- 领域模型、存储接口、流程编排三类职责保持解耦。

## 主流程约束

实现功能时，优先围绕项目主链路组织代码，而不是按临时需求分散落点。

一期标准主流程如下：

1. `internal/cli` 接收用户命令和参数。
2. `internal/config` 读取运行参数与平台默认配置。
3. `internal/store` 读取 `zygarde.yaml` 或显式指定的 blueprint 文件。
4. `internal/blueprint` 整理 `services` 并补齐基础默认值。
5. `internal/app` 完成依赖装配，并将请求交给 `internal/coordinator`。
6. `internal/coordinator` 通过 `internal/template` 按 `middleware + template + environmentType` 解析 pkg 实现。
7. `internal/coordinator` 按 `pipeline=service.name` 调用 pkg 的 `Configure(...)` 累计配置。
8. pkg 在配置累计完成后统一返回 `[]EnvironmentContext`。
9. `internal/runtime` 根据 runtime driver 规则执行 `Prepare / Render / Apply`。
10. `internal/render` 根据 context 生成 runtime 产物；第一期为 Compose 产物。
11. `internal/deployment/compose` 负责执行 Docker Compose 生命周期动作。
12. `internal/environment` 负责记录环境状态、元数据和生命周期结果。

在没有充分理由时，不要绕过 `coordinator` 直接从 CLI 调用 pkg，也不要让 `internal/*` 承担某个 middleware 的补默认值和校验细节。

实现 `zygarde create` 时，第一步优先打通前半段链路：
- `cmd/main.go -> internal/cli -> internal/app -> internal/coordinator`
- `internal/coordinator` 再串联 `store -> blueprint -> template/pkg`
- 在 render / deployment 尚未完成前，也要先让 `Create` 具备“读取 blueprint、归一化 services、调用 pkg Configure”的能力

当前阶段已经进一步打通到：
- `coordinator -> runtime driver -> render -> deployment -> environment`
- `status/start/stop/destroy` 也应沿 `coordinator -> runtime driver -> environment` 组织，而不是绕过主流程直接执行 runtime 命令

## 实现约束

- 新增代码前，先确定所属模块，再决定文件路径。
- 若一个能力只服务于某一个中间件或某个 runtime 下的中间件实现，应放在对应 `pkg/<middleware>`。
- 若一个能力负责中间件注册、blueprint 归一化、runtime 产物拼装、部署执行等通用流程，应放在 `internal/`。
- 新目录或新模块建立时，命名要与现有语义一致，避免临时名称。
- 没有明确收益时，不提前引入复杂框架或过度抽象。

## 交付检查

提交前至少检查：
- 目录落点是否符合 [architecture.md](references/architecture.md)
- TODO 是否已更新为最新状态
- 是否完成最低限度验证
- 是否只包含与当前任务相关的改动

## 参考资料

- 目录与架构边界： [architecture.md](references/architecture.md)
- pkg 实现规范： [pkg-middleware-guidelines.md](references/pkg-middleware-guidelines.md)
- runtime driver 规则： [runtime-driver-guidelines.md](references/runtime-driver-guidelines.md)
- TODO 模板： [todo-template.md](references/todo-template.md)
