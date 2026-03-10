# compose-stack

统一的中间件 docker-compose 生成 + 验收 + 回收技能。

> 经验沉淀与稳定性总结见：`skills/compose-stack/README.md`

## 目标

一个技能同时负责：
- 生成配置（generate）
- 验收（verify）
- 回收（cleanup）
- 一键执行（run = generate + verify）

## 强制规范

所有生成目录必须包含：
- `docker-compose.yml`
- `build.sh`（启动/初始化）
- `check.sh`（功能检查）
- `data/`（运行数据目录）
- `.env`、`README.md`（建议）

验收固定主流程：
1. `./build.sh`
2. `./check.sh`
3. `docker compose down -v`
4. `rm -rf data/`

## 命令

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

## 当前支持

- mysql: single/master-slave (v5.7/v8.0)
- redis: single/master-slave/cluster (v6.2/v7.4)
- mongodb: single/replica-set/sharded (v6.0/v7.0)
- postgresql: single/master-slave (v16/v17)
- rabbitmq: single/cluster(3节点) (v4.2)
- kafka: single/cluster(KRaft 3节点) (v4.2)
- tidb: single (v6.7)

## Redis Cluster 兼容策略（重要）

- 对于 Redis `v6.2`，`redis-cli --cluster create` 在部分环境下对主机名地址更敏感，可能出现：
  - `ERR Invalid node address specified`
- 为保证 `v6.2/v7.4` 一致稳定，cluster 初始化统一采用**容器 IP:port** 建集群（而非容器名）。
- check 阶段采用强校验：`cluster_state` 必须最终为 `ok`（含重试收敛）。

## 文档约定（统一）

所有中间件 README 建议包含：
- 场景说明
- 快速开始
- 稳定性说明（启动顺序、就绪判定、关键重试参数）
- 已知兼容策略（如版本差异、地址解析差异）

## MongoDB 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v6.0` ✅ / `v7.0` ✅
- replica-set: `v6.0` ✅ / `v7.0` ✅
- sharded: `v6.0` ✅ / `v7.0` ✅

### 关键经验

1. **Replica Set v7.0 可能出现首轮选主抖动**
   - 仅判断 `myState=1` 不够稳定。
   - 已改为多条件稳定判定：`PRIMARY>=1 且 SECONDARY>=2`。
   - 默认等待窗口提升到 `120s`，并增加一轮兜底重试。

2. **Sharded 启动顺序必须严格控制**
   - 先等待 cfg/shard 节点可 ping；
   - 再初始化并等待 `cfgRS` PRIMARY；
   - 再初始化并等待 `shardRS` PRIMARY；
   - 最后等待 mongos 可用，再执行 `sh.addShard`。
   - 否则容易出现 mongos 早启导致的 `FailedToSatisfyReadPreference`。

3. **端口探活需按角色区分**
   - cfg 节点：27019
   - shard 节点：27018
   - mongos：27017

## PostgreSQL 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v16` ✅ / `v17` ✅
- master-slave: `v16` ✅ / `v17` ✅

### 关键经验

1. **官方镜像与 bitnami 配置不可混用**
   - 已统一使用 `postgres:16/17` 与官方变量/目录。
   - 数据目录统一为 `/var/lib/postgresql/data`。

2. **主从初始化采用首启克隆模式**
   - master 先就绪并完成 replication 用户与 pg_hba 放行；
   - slave 首次启动执行 `pg_basebackup -R` 初始化；
   - 避免运行中“改从库”导致的不确定性。

3. **主从校验使用强校验 + 重试窗口**
   - master: `pg_stat_replication >= 1`；
   - slave: `pg_is_in_recovery() = true`；
   - 增加重试窗口，避免刚建链路时的瞬时抖动误判。

## RabbitMQ 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v4.2` ✅
- cluster(3节点): `v4.2` ✅

### 关键经验

1. **RabbitMQ 4.2 不建议沿用旧版手工 join 流程**
   - `stop_app -> reset -> join_cluster` 在 4.2 + Khepri 下可能出现 `meta.dets enoent` / timeout 抖动。

2. **集群组网改为声明式 classic_config 更稳定**
   - 通过 `rabbitmq.conf` 预定义节点列表（rabbit1/rabbit2/rabbit3）。
   - 启动后自动收敛，不再依赖运行中重置节点。

3. **验收应分离“节点健康”和“集群收敛”**
   - build 阶段只负责容器健康。
   - check 阶段强校验 `cluster_status` 包含 3 个运行节点，并带重试窗口。

## Kafka 验收经验与稳定性策略（新增）

### 已验证矩阵

- single(KRaft): `v4.2` ✅
- cluster(KRaft 3节点): `v4.2` ✅

### 关键经验

1. **优先采用 KRaft 模式，避免 ZooKeeper 依赖**
   - single 场景采用 broker+controller 合并节点。
   - cluster 场景采用 3 节点 controller quorum（1/2/3）。

2. **single 烟测应避免“直接消费比对”抖动**
   - 生产成功后用 `kafka-get-offsets` 校验目标 topic offset >= 1 更稳定。
   - 作为链路可用性验收信号，比短窗口消费更可靠。

3. **cluster 校验需覆盖 quorum 与数据链路**
   - 先校验 `kafka-metadata-quorum --status`（LeaderId/CurrentVoters）。
   - 再做跨节点生产消费烟测，确保 broker 间复制链路可用。
