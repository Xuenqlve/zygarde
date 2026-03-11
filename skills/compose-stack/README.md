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
- 场景：`single` / `cluster(3PD+3TiKV+2TiDB)`
- 状态：✅ 已完成并验证

### etcd
- 版本：`v3.6`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

### Consul
- 版本：`v1.20`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

### ClickHouse
- 版本：`v24` / `v25`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

### ZooKeeper
- 版本：`v3.8` / `v3.9`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

### Elasticsearch
- 版本：`v8.18` / `v8.19`
- 场景：`single` / `cluster(3节点)`
- 状态：✅ 已完成并验证

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

### G. TiDB
1. 对外版本口径为 `v6.7`，但官方镜像当前无 `v6.7.x` tag，默认映射到可用 tag（当前 `v6.5.12`）。
2. single 场景验收信号建议固定为：`TiDB status endpoint` + `PD health` + `SQL 端口探活`。
3. cluster 场景 PD 在该版本下不支持 `--initial-cluster-state` 参数，已改为：
   - `pd1` 使用 `--force-new-cluster`
   - `pd2/pd3` 使用 `--join=pd1:2379`
4. cluster 强校验口径：`PD members=3`、`PD health 全部 true`、`TiKV stores>=3`、`双 TiDB SQL 端口可达`。

### H. etcd
1. 版本口径为 `v3.6`，镜像默认使用 `quay.io/coreos/etcd:v3.6.0`，可通过 `ETCD_IMAGE` 覆盖。
2. single 场景验收信号固定为：`etcdctl endpoint health` + `member list` + `KV put/get smoke`。
3. cluster 采用 3 节点静态 initial cluster，强校验口径：
   - 3 节点 endpoint health 全通过
   - member 数量 >= 3
   - 跨节点 KV 读写链路可用
4. 兼容性提示：若变更镜像源，需先重新 `generate` 再 `verify`，避免旧目录残留历史镜像配置。

### I. Consul
1. 版本口径为 `v1.20`，镜像默认使用 `hashicorp/consul:1.20`。
2. single 场景验收信号固定为：`leader` 可用 + `member` 可见 + `KV put/get smoke`。
3. cluster(3节点) 采用 server 模式 + retry-join 自动收敛，强校验口径：
   - leader 非空
   - members >= 3
   - raft peers = 3
   - 跨节点 KV 读写链路可用
4. 稳定性策略：leader 检查增加重试窗口，避免启动初期瞬时空值导致误判。

### J. ClickHouse
1. 版本口径支持 `v24/v25`，镜像 tag 需使用可用的主次版本：
   - `v24 -> clickhouse/clickhouse-server:24`
   - `v25 -> clickhouse/clickhouse-server:25.8`（`25` tag 不存在）
2. single 场景验收信号固定为：`SELECT 1` + `version()` + 基础建表写入读取 smoke。
3. cluster(3节点) 验收信号需覆盖拓扑与跨节点链路：
   - `system.clusters` 中目标集群节点数 >= 3
   - `remote('ch1,ch2,ch3', system.one)` 返回 3
4. v25 集群默认用户网络策略更严格：
   - 默认 `default` 用户仅允许本地访问，`remote()` 会认证失败
   - 已通过 `users.d/default-network.xml` 放开节点间访问并复验通过

### K. ZooKeeper
1. 版本口径支持 `v3.8/v3.9`，镜像默认使用 `zookeeper:3.8` / `zookeeper:3.9`。
2. single 场景验收信号固定为：`ruok=imok` + `mntr` + znode create/get smoke。
3. cluster(3节点) 验收需覆盖健康与角色拓扑：
   - 3 节点 `ruok` 全部 `imok`
   - `stat` 输出需包含 leader/follower 角色
   - 跨节点 znode 读写成功
4. 稳定性策略：
   - build 启动前清理 `data/datalog`，避免历史日志导致 `No snapshot found` 报错
   - `zkCli.sh` 检查改为命令模式并精确匹配数据行，避免日志尾行干扰

### L. Elasticsearch
1. 版本口径支持 `v8.18/v8.19`，镜像默认使用：
   - `v8.18 -> docker.elastic.co/elasticsearch/elasticsearch:8.18.0`
   - `v8.19 -> docker.elastic.co/elasticsearch/elasticsearch:8.19.0`
2. single 场景验收信号固定为：
   - `_cluster/health` 可访问
   - `version.number` 匹配目标版本
   - 索引写入与读取 smoke 成功
3. cluster(3节点) 验收口径：
   - `number_of_nodes >= 3`
   - `_cat/nodes` 返回 3 节点
   - 跨节点索引写入读取链路可用
4. 关键稳定性策略：
   - `build.sh/check.sh` 必须加载 `.env`，否则自定义端口不生效
   - 本地端口常冲突（9200/9201/9202），需预留可切换端口（例如 9210+ / 9220+ / 9240+）
   - single 与 cluster 使用相同容器名模式时，切换场景前需先 down 旧场景，避免容器名冲突

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
