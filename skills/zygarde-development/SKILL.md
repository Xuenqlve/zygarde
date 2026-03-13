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

在没有充分理由时，不要绕过 `coordinator` 直接从 CLI 调用底层模块，也不要把部署状态写回逻辑散落到多个包中。

## 实现约束

- 新增代码前，先确定所属模块，再决定文件路径。
- 若一个能力未来会支持多个中间件或多个编排后端，应优先放在 `internal/` 的通用抽象中。
- 若一个能力只服务于某一个中间件的运行细节，应放在对应 `pkg/<middleware>`。
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
- TODO 模板： [todo-template.md](references/todo-template.md)
