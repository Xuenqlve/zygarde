# Pkg Compose Template Guidelines

## 目标

为 `pkg/<middleware>` 下的 Compose 版实现定义统一模板，确保后续新增 `mysql`、`redis`、`mongodb` 等模块时，代码结构、默认值策略、校验方式、ComposeContext 输出和资产生成方式保持一致。

本规范以当前 [pkg/mysql/single.go](/Users/xuenqlve/go/src/github.com/xuenqlve/zygarde/pkg/mysql/single.go) 为基线抽象，不是 MySQL 私有规范。

同时必须覆盖 `compose-stack` 当前支持的 12 个中间件：

- mysql
- redis
- mongodb
- postgresql
- rabbitmq
- kafka
- tidb
- etcd
- consul
- clickhouse
- zookeeper
- elasticsearch

并且每个 middleware 的 Compose 版实现应覆盖 `compose-stack` 中已定义的场景与版本矩阵。

## 适用范围

以下场景应优先遵循本规范：

- 新增一个 middleware 的 Compose 版实现
- 为现有 middleware 新增新的 Compose template
- 为现有 middleware 新增新的版本支持
- 调整 ComposeContext 结构化输出
- 调整 `build.sh` / `check.sh` / `.env` / `README.md` 等 Compose bundle 资产输出

## 标准文件形态

推荐每个 Compose 版 middleware/template 至少包含一个主文件：

- `pkg/mysql/single.go`
- `pkg/redis/single.go`
- `pkg/mongodb/replica_set.go`

如果某个 template 明显变复杂，可以再拆分：

- `pkg/mysql/single.go`
- `pkg/mysql/single_compose_assets.go`
- `pkg/mysql/single_defaults.go`

第一阶段优先保证一个文件可读、可维护，不为拆分而拆分。

## 与 docker/ 目录的关系

`docker/<middleware>/<scenario>_<version>/` 是当前 Compose stack 已验证资产与行为的事实来源。

后续 `pkg/*` 的 Compose 版实现必须参考这些目录进行抽象，但不能简单为每个目录复制一份方法。

约束如下：

- `docker/` 中的目录用于提供已验证的行为、脚本结构和版本差异事实。
- `pkg/<middleware>/<template>.go` 应是面向 runtime 的统一实现。
- 对同一个 middleware/template，不应因为版本不同就复制多个几乎相同的方法。

以 MySQL 为例：

- `docker/mysql/single_v5.7/`
- `docker/mysql/single_v8.0/`

在 `pkg/` 中应收敛为：

- `pkg/mysql/single.go`

差异通过 `version` 参数控制，而不是写成两个平行实现。

## 版本抽象规范

同一个 middleware/template 存在多个版本时，应遵循以下规则：

1. `pkg/*` 中只保留一个主实现入口
2. 通过 `values.version` 或等价标准化字段传入版本
3. 尽可能抽取版本间共性
4. 仅将真正有差异的部分通过版本控制

不推荐：

- `pkg/mysql/single_v57.go`
- `pkg/mysql/single_v80.go`

推荐：

- `pkg/mysql/single.go`
  - `Normalize` 里补齐默认 `version`
  - `BuildRuntimeContexts` 中按版本映射镜像、脚本片段、健康检查或独立资产

## 版本差异应落在哪些位置

版本差异一般只允许出现在以下几个地方：

- 默认镜像/tag
- 启动命令参数
- 健康检查命令
- `.env` 输出值
- `build.sh / check.sh` 片段
- 独立脚本或配置文件内容

不应出现在：

- 整个 middleware 结构分叉
- 整个 `BuildRuntimeContexts` 重写一份
- render / executor 里按 middleware 版本写分支

建议做法是：

1. 先抽出共用基础配置
2. 再通过版本选择器覆盖差异字段

例如 MySQL single：

- 共用部分：
  - container name
  - data_dir
  - 端口映射
  - 基础环境变量
- 差异部分：
  - `image`
  - 可能的命令参数
  - 可能的 healthcheck 细节
  - README / 检查脚本里的版本说明

## 推荐的版本实现方式

建议每个 middleware/template 内部形成“基础配置 + 版本覆盖”的模式。

例如：

1. `Normalize`
   - 统一补默认 `version`
   - 校验 version 是否在支持矩阵中
2. `BuildRuntimeContexts`
   - 先生成一份版本无关基础 spec
   - 再按 version 覆盖差异字段
3. `Assets`
   - 先生成共用脚本片段
   - 再按 version 补差异内容

不要把版本处理散落到多个函数中相互覆盖，优先集中在可枚举的版本分支中。

## 支持矩阵约束

每个 middleware Compose 版实现都应显式维护自身支持矩阵，并与 `compose-stack` 当前支持版本保持一致。

例如：

- mysql
  - `single`: `v5.7 / v8.0`
  - `master-slave`: `v5.7 / v8.0`
- redis
  - `single`: `v6.2 / v7.4`
  - `master-slave`: `v6.2 / v7.4`
  - `cluster`: `v6.2 / v7.4`

要求：

- 未支持的版本在 `Normalize` 或 `Validate` 阶段直接报错
- 不允许静默回退到其他版本
- 版本矩阵更新时，应同步更新文档与最低验证用例

## Compose 版模块必须承担的职责

一个 `pkg/<middleware>/<template>.go` 的 Compose 版实现，必须明确完成以下工作：

1. 注册 Compose runtime middleware
2. 补齐 middleware 默认值
3. 校验 middleware 自身配置
4. 将标准化后的 service 转换为 `runtime.ComposeContext`
5. 产出 Compose service 定义
6. 产出 Compose bundle 资产片段

它不应负责：

- 直接生成 `docker-compose.yml`
- 直接执行 `docker compose`
- 持久化 environment 或 runtime artifact
- 理解 CLI / blueprint 文件路径等平台级逻辑

## 推荐结构

推荐模块内部至少保持以下结构顺序：

1. runtime 注册
2. `New...Spec()`
3. `type ...Spec struct`
4. `Middleware / Template / IsDefault`
5. `Normalize`
6. `Configure`
7. `BuildRuntimeContexts`
8. `Validate`
9. 默认值与辅助函数

这样后续每个模块结构都一致，便于 review 和复制模板。

## Normalize 规范

`Normalize` 应负责：

- 生成默认 `name`
- 合并默认值与用户输入
- 补齐 Compose runtime 需要的通用值键
- 补齐默认 `version`
- 完成字段类型归一化

以 Compose 版常见字段为例，优先通过 `internal/runtime/compose` 中定义的常量访问：

- `service_name`
- `container_name`
- `image`
- `data_dir`
- `port`
- `version`

约束：

- 不要在代码中散落裸字符串键名
- 能归一化的类型在这里归一化，不要拖到 Render 或 Apply
- 默认值补齐要尽可能做到“用户最少输入”
- 版本合法性应在这里或 `Validate` 中尽早拦截

## Validate 规范

`Validate` 应只校验 middleware 自身语义，不承担 runtime 执行职责。

必须至少校验：

- middleware/template 是否匹配
- 必填字段是否存在
- 字段类型是否正确
- 端口、目录、镜像名等基础合法性
- version 是否在支持矩阵内

建议：

- 报错信息统一带 `middleware + template + field` 语义
- 校验逻辑在 `Configure` 和 `BuildRuntimeContexts` 中都可复用

## BuildRuntimeContexts 规范

`BuildRuntimeContexts` 是 Compose 版实现的核心。

它的职责是把标准化后的 `BlueprintService` 转换为：

- `runtime.ComposeContext`

当前建议输出至少包含三部分：

1. `ServiceName / Middleware / Template`
2. `runtime.ServiceSpec`
3. `[]runtime.AssetSpec`

## ServiceSpec 规范

`runtime.ServiceSpec` 应表达最终 Compose service 需要的结构化字段，例如：

- `Image`
- `ContainerName`
- `Restart`
- `Environment`
- `Ports`
- `Volumes`
- `Command`
- `HealthCheck`

要求：

- `pkg/*` 负责把 middleware 语义转换成 `ServiceSpec`
- `internal/render/compose` 不应再猜 middleware 特有逻辑

## AssetSpec 规范

每个 ComposeContext 应根据场景输出自己的 bundle 资产片段。

当前优先支持：

- `.env`
- `build.sh`
- `check.sh`
- `README.md`
- 独立脚本 / SQL / 配置文件

建议的资产归类：

### `.env`

使用 `runtime.AssetMergeEnv`。

要求：

- 多实例场景下不要输出全局冲突 key
- 应基于 `service name` 生成实例级 env key

示例：

- `MYSQL_MYSQL_1_PORT=3306`
- `MYSQL_MYSQL_1_ROOT_PASSWORD=root1`

### `build.sh`

使用 `runtime.AssetMergeScript`。

要求：

- 每个 context 只输出自己的脚本片段
- 不要输出完整脚本壳
- 公共 shebang / `set -euo pipefail` / `docker compose ...` 主入口由 Render 统一生成

### `check.sh`

使用 `runtime.AssetMergeScript`。

要求：

- 每个 context 只负责自己的验收片段
- 多实例时不能依赖全局共享变量
- 优先引用实例级 env key，或直接写入已标准化值

### `README.md`

使用 `runtime.AssetMergeReadme`。

要求：

- 每个 context 提供可拼接 section
- 不要求在 `pkg/*` 输出完整 README

### 独立文件

例如：

- `init.sql`
- `master-init.sql`
- `slave-init.sql`
- `custom.cnf`

应使用 `runtime.AssetMergeUnique`，并提供唯一 `FileName`。

## 多实例约束

Compose 版模块必须优先考虑多实例场景。

至少要处理以下冲突：

1. 端口冲突
2. `data_dir` 冲突
3. `.env` key 冲突
4. `check.sh` 中共享变量冲突
5. `container_name` 冲突

建议：

- `data_dir` 默认值优先按 `service name` 推导，例如 `./data/<service-name>`
- `.env` 变量优先按 `middleware + service name` 生成实例级 key
- `container_name` 默认跟随 `service name`

## 常量与键名规范

Compose 版通用值键、字段键、资产路径键，优先在 `internal/runtime/compose` 中集中声明，再由 `pkg/*` 引用。

当前建议集中管理的内容包括：

- value keys
  - `service_name`
  - `container_name`
  - `image`
  - `data_dir`
  - `port`
  - `root_password`
  - `version`
- 后续可继续补：
  - 常见资产 key
  - 常见 env key 规则
  - 常见 volume / network / healthcheck 键

不要在多个 middleware 文件中重复声明相同字符串常量。

## 推荐实现步骤

新增一个 Compose 版 middleware/template 时，建议按以下顺序实现：

1. 写 `Middleware / Template / IsDefault`
2. 明确该 template 的支持版本矩阵
3. 写 `Normalize`
4. 写 `Validate`
5. 写 `Configure`
6. 写 `BuildRuntimeContexts`
7. 先补最小 `ServiceSpec`
8. 再补 `.env / build.sh / check.sh / README` 资产
9. 最后用真实 `zygarde create` 或 Compose 集成测试验证

## 最低验收标准

一个新的 Compose 版 `pkg` 模块，至少应满足：

1. `go build ./...` 通过
2. 能通过一个最小 blueprint 生成有效 bundle
3. `build.sh` 能执行成功
4. `check.sh` 能验证服务可用
5. 多实例场景下不会发生默认 key / path / port 冲突
6. 支持矩阵中的每个版本都能被正确选择，不会误落到其他版本配置
