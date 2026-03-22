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
6. 若交付涉及 `pkg/<middleware>` 的用户可配置能力，同步更新 `docs/` 下对应帮助文档。
7. 若交付涉及用户命令主链路或 Compose 版中间件可运行能力，同步评估并补充 `test/command/` 下对应功能测试。
8. 新增测试时，优先将单元测试统一落在 `test/<domain>/` 下，避免继续分散在 `internal/*` 或 `pkg/*`；历史存量测试可逐步迁移。
9. 完成后执行必要验证，至少覆盖编译、测试或目录结果检查。
10. 总结改动并提交。

## 目录落点规则

目录职责说明见 [architecture.md](references/architecture.md)。

必须遵守以下规则：
- 某个中间件独有的模板变量、默认值、校验和辅助逻辑，进入 `pkg/<middleware>`。
- 模板可用性清单、支持版本、默认 template、帮助文档路径等“模板元数据”，优先由 `pkg/*` 暴露，不要把这类事实来源重新定义在 `internal/*`。
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

当前 runtime 主流程约束进一步明确为：
- `pkg/*` 在 `BuildRuntimeContexts()` 中返回 `[]EnvironmentContext`，而不是返回某个 runtime 私有 executor 参数。
- `EnvironmentContext` 是串联 `Prepare / Render / Apply` 三阶段的统一材料接口。
- `PrepareInput / RenderInput / ApplyInput` 允许按开发阶段渐进补充字段，不要求一开始设计成大而全对象。
- `Prepare` 负责环境级目录、命名、路径规划，不负责生成 runtime 产物内容。
- `Render` 负责根据 `[]EnvironmentContext` 生成完整 runtime bundle；对 Compose 来说不只生成 `docker-compose.yml`，还要生成脚本与附属资产。
- `Apply` 负责执行已生成 bundle；对 Compose 来说默认执行 `build.sh`，而不是直接在 executor 中硬编码 `docker compose up -d`。
- 阶段产物应优先建模为 plan/result，而不是把 runtime 私有字段塞入全局公共 model。

当前 `pkg/*` 的 Compose 版建设目标还必须满足：
- 最终要覆盖 `compose-stack` 当前支持的 12 个中间件能力，而不是只覆盖单个 demo middleware。
- 对同一个 middleware/template 的多个版本目录，应在 `pkg/*` 中收敛为单一实现入口，通过 `version` 控制差异，而不是按版本复制多个方法。
- `docker/<middleware>/<scenario>_<version>/` 应被视为已验证行为与版本差异的事实来源；`pkg/*` 负责对这些差异做抽象收敛。
- 如果需要对外暴露“系统当前支持哪些 middleware/template/runtime/version”，应优先由 `pkg` 层维护模板元数据注册表，再由 `internal/app -> coordinator -> cli` 做消费和展示。

关于模板管理，当前建议遵守：
- `template list / template show` 这类能力属于平台命令，应落在 `internal/*` 主链路中实现。
- 但模板可用性元数据本身应放在 `pkg`，例如 `pkg/catalog` 或其他 `pkg/*` 聚合入口。
- `internal/*` 不应重新硬编码一份“支持哪些中间件、哪些 template、哪些版本”的平行清单。
- 模板管理默认展示的内容应至少包含：`middleware`、`template`、`runtime`、支持版本、默认模板标记、帮助文档路径、简要说明。
- 新增或变更 `pkg/<middleware>/<template>` 能力时，如果影响模板可发现性，必须同步更新模板元数据。

关于 Blueprint 管理，当前建议遵守：
- Blueprint 管理的目标，是把 blueprint 从“手工维护的 YAML 输入文件”提升为“平台内可发现、可校验、可派生、可维护的本地资源”。
- Blueprint 管理能力属于平台命令，应落在 `internal/*` 主链路中实现，不要把文件写入、删除、复制、名称解析逻辑散落在 CLI 层临时拼接。
- Blueprint 的事实来源仍然是本地 YAML 文件；当前阶段不引入数据库、远程注册中心或额外索引服务。
- Blueprint 管理应优先支持“路径 + 名称”双引用方式：先按文件路径解析，找不到时再按 blueprint `name` 在约定目录中搜索。
- Blueprint 管理当前推荐能力顺序为：`create -> copy -> list/show/validate -> update -> delete -> edit`。
- `create` 产物应是标准 blueprint YAML 骨架，而不是 runtime bundle；runtime 产物仍由 `zygarde create/up` 主链路生成。
- `copy` 优先作为 blueprint 复用能力，适合从现有环境派生新环境；其优先级高于交互式 `edit`。
- `update` 优先做结构化更新，而不是编辑器式自由编辑。当前优先支持：
  - blueprint 级字段：`name`、`description`、`runtime.project-name`
  - service 级字段：新增 service、删除 service、更新指定 service 的 `middleware` / `template` / `values`
- `edit` 如果实现，应只是“解析 blueprint 后调用 `$EDITOR` 打开文件”的便捷入口，不承担结构化修改逻辑；机器可控的修改仍应走 `update`。
- Blueprint 文件读写、删除、复制、名称解析等存储动作，应统一收敛到 `internal/store` 抽象，不要让 `coordinator` 或 `cli` 直接操作文件系统细节。
- Blueprint 管理新增命令时，应同步补 `test/command/` 下的命令级测试，至少覆盖：
  - 路径解析
  - 名称解析
  - 文件创建/覆盖/删除结果
  - 关键 YAML 字段是否按预期落盘
- 若 blueprint 管理能力影响 README 中的命令集合、默认文件命名或引用方式，必须同步更新 `README.md` 与 `TODO.md`。

关于一次性工具与任务级全局状态，当前建议遵守：
- 当前项目可按“一次执行完成即退出”的单任务 CLI 工具来设计，不必默认按长生命周期服务建模。
- 对端口分发、编号分配等一次性辅助能力，可以放在 `internal/tool/*` 中实现为“单任务级全局工具”。
- 这类工具应在任务开始时初始化，在本次任务内复用，不要求为跨任务状态持久化设计。
- 这类工具只用于减少默认值冲突和启动时报错，不承担长期状态管理职责。
- 若用户显式配置了值，应以校验为主，不要静默改写用户输入；若用户未配置，工具才负责补默认值。

关于 `EnvironmentContext` 的增量设计，当前建议遵守：
- 先定义最小接口，再用主流程反推字段，不提前设计大而全公共结构。
- 如果 `Prepare` 缺字段，只补 `PrepareInput`。
- 如果 `Render` 缺字段，只补 `RenderInput`。
- 如果 `Apply` 缺字段，只补 `ApplyInput`。
- 避免为了某一个 runtime 临时需求污染其他阶段输入。

关于 Compose 产物生成，当前建议遵守：
- 统一参考 `compose-stack` 的目录规范与脚本入口。
- Render 阶段生成完整 Compose bundle，至少包含：
  - `docker-compose.yml`
  - `build.sh`
  - `check.sh`
  - `.env`
  - `README.md`
  - `data/`
- 多个 `EnvironmentContext` 可共同贡献同名资产；Render 必须通过“资产池 + 合并策略”统一归并，而不是简单覆盖。
- `docker-compose.yml` 通过结构合并生成，不走文本拼接。
- `.env` 按键值归并；相同 key 不同值应视为冲突。
- `build.sh` 与 `check.sh` 按脚本片段归并，由 Render 统一生成完整脚本壳。
- SQL、配置文件等独立资产应允许按唯一文件名直接落盘，不强行参与同名合并。

## 实现约束

- 新增代码前，先确定所属模块，再决定文件路径。
- 若一个能力只服务于某一个中间件或某个 runtime 下的中间件实现，应放在对应 `pkg/<middleware>`。
- 若一个能力负责中间件注册、blueprint 归一化、runtime 产物拼装、部署执行等通用流程，应放在 `internal/`。
- 新目录或新模块建立时，命名要与现有语义一致，避免临时名称。
- 没有明确收益时，不提前引入复杂框架或过度抽象。
- 不要把 Compose 专属字段继续塞进“伪公共 environment 模型”中；runtime 私有产物应放到对应阶段 plan/result 中。
- 不要在 `internal/*` 中推断某个 middleware 的镜像、端口、环境变量默认值；这些必须在 `pkg/*` 中先转换好。
- 不要在 `internal/*` 中维护独立的模板能力白名单、版本矩阵或帮助文档映射；这类信息应从 `pkg` 元数据读取。
- 新增单元测试统一放在 `test/` 目录下，按主题拆分，例如 `test/command/`、`test/app/`、`test/coordinator/`；不要继续在 `internal/*` 下新增零散单元测试文件。

## 交付检查

提交前至少检查：
- 目录落点是否符合 [architecture.md](references/architecture.md)
- TODO 是否已更新为最新状态
- 是否完成最低限度验证
- 是否只包含与当前任务相关的改动

## 参考资料

- 目录与架构边界： [architecture.md](references/architecture.md)
- pkg 开发总览： [pkg-development-summary.md](references/pkg-development-summary.md)
- pkg 实现规范： [pkg-middleware-guidelines.md](references/pkg-middleware-guidelines.md)
- pkg Compose 模板规范： [pkg-compose-template-guidelines.md](references/pkg-compose-template-guidelines.md)
- 版本命令兼容规范： [version-compatibility-guidelines.md](references/version-compatibility-guidelines.md)
- runtime driver 规则： [runtime-driver-guidelines.md](references/runtime-driver-guidelines.md)
- TODO 模板： [todo-template.md](references/todo-template.md)
