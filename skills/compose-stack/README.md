# compose-stack README

统一的中间件 docker-compose 生成 + 验收 + 回收方案沉淀。

## 一、统一规范（最终版）

### 1) 目录产物规范
每个生成目录必须包含：
- `docker-compose.yml`
- `build.sh`
- `check.sh`
- `data/`
- `.env`（建议）
- `README.md`（建议）

### 2) 验收主流程（固定）
1. `./build.sh`
2. `./check.sh`
3. `docker compose down -v`
4. `rm -rf data/`

> 已由 `compose-stack verify` 统一执行，不再为每个中间件写独立 verify 流程。

---

## 二、已完成支持矩阵

### MySQL
- 版本：`v5.7` / `v8.0`
- 场景：`single` / `master-slave`
- 状态：✅ 已完成并验证

### Redis
- 版本：`v6.2` / `v7.4`
- 场景：`single` / `master-slave` / `cluster`
- 状态：✅ 已完成并验证

### MongoDB
- 版本：`v6.0` / `v7.0`
- 场景：`single` / `replica-set` / `sharded(轻量6节点)`
- 状态：✅ 已完成并验证

### PostgreSQL
- 版本：`v16` / `v17`
- 场景：`single` / `master-slave`
- 状态：✅ 已完成并验证

### RabbitMQ
- 版本：`v4.2`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

### Kafka
- 版本：`v4.2`
- 场景：`single(KRaft)` / `cluster(KRaft 3节点)`
- 状态：✅ 已完成并验证

### TiDB
- 版本：`v6.7`
- 场景：`single`
- 状态：🚧 已实现，待完整验收

---

## 三、关键经验总结

### A. MySQL
1. single 也应默认开启 `binlog + GTID`，避免后续切主从时配置不一致。
2. master-slave 验收必须检查复制线程状态（IO/SQL）。
3. 验收前清理旧容器和 data 目录，可明显降低历史脏数据导致的误判。

### B. Redis
1. `v6.2` cluster 对主机名地址更敏感，可能出现：
   - `ERR Invalid node address specified`
2. 统一改为**容器 IP:port** 建集群，可兼容 `v6.2/v7.4`。
3. cluster 校验必须强判 `cluster_state:ok`（带重试），避免“短暂 fail 被误判为通过”。

### C. MongoDB
1. replica-set 验收不要只看 `myState=1`，应改为：
   - `PRIMARY>=1` 且 `SECONDARY>=2`
2. replica-set 选主有抖动，等待窗口建议默认 `120s` 并加兜底重试。
3. sharded 启动顺序必须严格：
   - cfg/shard 就绪 → cfgRS init+primary → shardRS init+primary → mongos 就绪 → addShard
4. 端口探活必须按角色区分：
   - cfg: `27019`
   - shard: `27018`
   - mongos: `27017`

### D. PostgreSQL
1. 官方镜像（`postgres:16/17`）与 bitnami 环境变量不兼容，不能混用配置。
2. single/master-slave 数据目录统一使用官方路径：`/var/lib/postgresql/data`。
3. master-slave 建链应采用“主先就绪 + 从库首启 `pg_basebackup -R`”模式，避免运行中改库导致不稳定。
4. 主从校验建议强判并带重试窗口：
   - master: `pg_stat_replication` 至少 1 条
   - slave: `pg_is_in_recovery() = true`

### E. RabbitMQ
1. RabbitMQ 4.2 下，运行时手工 `stop/reset/join_cluster` 容易抖动（Khepri 元数据路径相关错误）。
2. 集群场景建议使用声明式 `classic_config` 自动组网，预定义 rabbit1/rabbit2/rabbit3 节点。
3. 验收建议分两段：
   - 节点健康检查：`rabbitmq-diagnostics -q ping`
   - 集群收敛检查：`cluster_status` 必须出现 3 个目标节点（含重试窗口）

### F. Kafka
1. Kafka v4.2 场景建议统一采用 KRaft（无需 ZooKeeper），减少组件复杂度。
2. single 场景烟测采用“生产消息 + topic offset 校验（>=1）”更稳定，避免短窗口消费抖动误判。
3. cluster 场景验收需要双校验：
   - 元数据仲裁：`kafka-metadata-quorum describe --status`（LeaderId + CurrentVoters）
   - 数据链路：跨节点生产消费烟测

---

## 四、推荐执行顺序（新增中间件时）

1. 先实现 `single`
2. 再实现复制型场景（master-slave / replica-set）
3. 最后实现集群型场景（cluster / sharded）
4. 每个场景先跑一轮 `run`，再跑一次 `verify` 复验稳定性

---

## 五、常用命令

```bash
# 生成
./skills/compose-stack/scripts/compose-stack.sh generate <middleware> <scenario> <version>

# 验收
./skills/compose-stack/scripts/compose-stack.sh verify <target-dir>

# 回收
./skills/compose-stack/scripts/compose-stack.sh cleanup <target-dir>

# 一键（生成 + 验收）
./skills/compose-stack/scripts/compose-stack.sh run <middleware> <scenario> <version>
```
