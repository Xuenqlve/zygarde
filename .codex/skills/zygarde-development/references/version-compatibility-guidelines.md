# Version Compatibility Guidelines

用于处理同一中间件在不同版本之间命令、SQL、脚本参数或检查方式不兼容的问题。

## 适用范围

当你在以下位置发现版本差异时，必须参考本规范：

- `pkg/<middleware>/<template>.go`
- `build.sh` / `check.sh` 片段
- SQL 初始化脚本
- 健康检查命令
- 运行时诊断脚本

典型场景：

- MySQL `SHOW SLAVE STATUS` vs `SHOW REPLICA STATUS`
- PostgreSQL 不同版本的管理命令差异
- Kafka、MongoDB、Redis 的版本化 CLI 参数差异

## 第一原则

已知版本时，不要在运行时“先试新命令，再失败 fallback 到旧命令”。

正确做法是：

1. 先根据 `version` 明确选择命令
2. 只在版本无法静态区分时，才使用带说明的兼容 fallback

原因：

- 运行时探测会产生误导性错误日志
- 可能让 `doctor`、`build`、`status` 噪音变大
- 某些版本差异不只是“命令名不同”，还涉及字段名和语义变化

## 事实来源

版本兼容实现必须以 `docker/<middleware>/<scenario>_<version>/` 为事实来源。

要求：

- 先比对不同版本目录中的 `build.sh`、`check.sh`、配置文件和 SQL
- 明确哪些差异是语法差异，哪些差异是行为差异
- 在 `pkg/*` 中通过 `version` 收敛，而不是把差异散落到 `internal/*`

## 实现规则

- 版本差异优先收敛为显式 helper，例如：
  - `mysqlReplicaStatusCommand(version)`
  - `postgresReplicaCheck(version)`
- helper 返回的应是完整命令和对应字段，而不是只返回一半字符串
- 同一个版本差异不要在 `build` 和 `check` 两处各写一套不一致分支
- 如果某个脚本片段依赖版本，调用方必须显式传入 `version`

## 禁止事项

- 不要明知当前是 `v5.7`，仍先执行 `SHOW REPLICA STATUS`
- 不要把版本兼容分支写进 `internal/deployment/*`
- 不要在文档中声称“支持某版本”，但实现仍靠模糊 fallback 碰运气
- 不要把一个版本的失败日志当成正常兼容流程的一部分

## 测试要求

遇到版本命令兼容问题时，至少补一条单测覆盖：

- `v_old` 使用旧命令
- `v_new` 使用新命令

如果该差异会影响真实 `doctor` 或 `build` 行为，应补对应集成测试或在现有集成测试中覆盖版本矩阵。

## 文档要求

当版本兼容行为会影响用户可观察结果时，同步更新：

- `docs/<middleware>.md`

至少说明：

- 支持哪些版本
- 这些版本之间是否存在特殊行为差异
- 哪些行为由系统自动兼容处理
