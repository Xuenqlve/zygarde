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
- tidb: single/cluster(3PD+3TiKV+2TiDB) (v6.7)
- etcd: single/cluster(3节点) (v3.6)
- consul: single/cluster(3节点) (v1.20)
- clickhouse: single/cluster(3节点) (v24/v25)
- zookeeper: single/cluster(3节点) (v3.8/v3.9)
- elasticsearch: single/cluster(3节点) (v8.18/v8.19)

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

## etcd 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v3.6` ✅
- cluster(3节点): `v3.6` ✅

### 关键经验

1. **镜像策略需先确认 tag 可用性**
   - `bitnami/etcd:3.6.0` 不可用时，已切换到 `quay.io/coreos/etcd:v3.6.0`。
   - 若更换镜像源，建议先重新 generate，避免历史目录残留旧镜像配置。

2. **single 验收建议固定 3 类信号**
   - `endpoint health`（可用性）
   - `member list`（成员状态）
   - `KV put/get`（功能链路）

3. **cluster 验收建议强校验成员与链路**
   - 3 节点 `endpoint health` 全部通过。
   - `member list` 数量至少 3。
   - 跨节点 `put/get` 成功，确保复制链路可用。

## Consul 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v1.20` ✅
- cluster(3节点): `v1.20` ✅

### 关键经验

1. **single 需要等待 leader 产生再判通过**
   - API ready 不代表立即有 leader。
   - `check.sh` 已增加 leader 重试窗口，避免瞬时空值误判。

2. **cluster 建议使用 retry-join 自动收敛**
   - 3 节点 server 模式，`retry-join=consul1/2/3`。
   - 验收信号应同时包含 leader + members 收敛。

3. **cluster 强校验口径**
   - `leader` 非空
   - `members >= 3`
   - `raft peers = 3`
   - 跨节点 `KV put/get` 成功

## ClickHouse 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v24` ✅ / `v25` ✅
- cluster(3节点): `v24` ✅ / `v25` ✅

### 关键经验

1. **镜像 tag 需使用可用的主次版本**
   - `clickhouse/clickhouse-server:25` 不存在。
   - 已固定 `v25 -> 25.8`，避免拉取失败。

2. **cluster 验收要同时看拓扑与跨节点链路**
   - `system.clusters` 检查集群节点数。
   - `remote('ch1,ch2,ch3', system.one)` 作为跨节点链路 smoke。

3. **v25 默认用户网络限制会影响 remote()**
   - 默认 `default` 用户仅本地可访问，跨节点会报认证失败。
   - 已通过 `users.d/default-network.xml` 放开节点网络并复验通过。

## ZooKeeper 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v3.8` ✅ / `v3.9` ✅
- cluster(3节点): `v3.8` ✅ / `v3.9` ✅

### 关键经验

1. **single 验收建议固定 3 类信号**
   - `ruok=imok`（基础可用性）
   - `mntr/stat`（状态可观测）
   - znode create/get（功能链路）

2. **cluster 验收需覆盖角色拓扑**
   - 3 节点 `ruok` 全部通过。
   - `stat` 输出必须能看到 leader/follower。
   - 跨节点 znode 读写成功。

3. **脏数据与日志干扰是两大稳定性风险**
   - 启动前清理 `data/datalog`，避免 `No snapshot found`。
   - `zkCli.sh` 结果解析需精确匹配数据行，避免日志尾行误判。

## Elasticsearch 验收经验与稳定性策略（新增）

### 已验证矩阵

- single: `v8.18` ✅ / `v8.19` ✅
- cluster(3节点): `v8.18` ✅ / `v8.19` ✅

### 关键经验

1. **脚本必须加载 .env 才能正确支持端口覆盖**
   - 若 build/check 未加载 `.env`，会回退默认 920x 端口，导致误判或冲突。

2. **本地端口冲突是高频问题**
   - 9200/9201/9202 常被占用，验收时建议明确改用 9210+/9220+/9240+ 端口段。

3. **容器命名冲突需要显式清理旧场景**
   - single 与 cluster 使用固定容器名，切换场景前应先 `docker compose down -v`。

4. **cluster 验收必须覆盖节点数量与数据链路**
   - `number_of_nodes >= 3`
   - `_cat/nodes` 正常
   - 跨节点索引写入/读取 smoke 成功
