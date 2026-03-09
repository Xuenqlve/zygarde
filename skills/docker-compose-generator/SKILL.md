# docker-compose-generator

根据中间件类型生成 docker-compose 配置模板。

## 功能

接收中间件类型、场景（可选）、版本（可选），生成一套标准的 docker-compose 配置到 `docker/{type}/` 目录。

## 设计原则

1. **每次只生成一套集群** — 不批量生成，需要哪个版本就传对应参数
2. **最小文件原则** — 只生成必要的文件：
   - `docker-compose.yml`（必须）
   - `.env`（必须，所有变量放这里）
   - `README.md`（必须）
   - `build.sh`（必须，用于启动和初始化）
   - `check.sh`（必须，用于健康检查和功能验证）
   - `*.sql`（仅当 docker-compose 无法解决时才额外生成，如初始化 SQL）
3. **网络配置原则**：
   - 非必要不要生成 `networks` 配置
   - 仅当需要跨 compose 共享网络、固定网络名称或显式网络策略时才生成
4. **账号密码规范**：
   - 账号统一：`root`
   - 密码统一：`root123`
   - 所有变量放 `.env`

## 输入

- 中间件类型：`mysql`、`redis`、`mongodb`、`postgresql`、`tidb`、`elasticsearch`、`kafka`、`rabbitmq`、`clickhouse`、`etcd`、`zookeeper`、`consul`
- 场景（仅 mysql）：`single`、`master-slave`
- 版本（MySQL 必须指定）：`v5.7`、`v8.0`

## MySQL 版本说明

MySQL 需要指定版本，目录结构如下：

```
docker/mysql/
├── single_v5.7/       # 单节点 MySQL 5.7
├── single_v8.0/       # 单节点 MySQL 8.0
├── master-slave_v5.7/ # 主从 MySQL 5.7
└── master-slave_v8.0/ # 主从 MySQL 8.0
```

### MySQL 版本特性

| 版本 | 说明 |
|------|------|
| single_v5.7 | 单节点 + binlog 默认开启 |
| single_v8.0 | 单节点 + binlog 默认开启 |
| master-slave_v5.7 | 主从双节点 + GTID |
| master-slave_v8.0 | 主从双节点 + GTID |

## 输出

生成到 `docker/{type}/` 目录：

```
docker/{type}/
├── docker-compose.yml  # 必须
├── .env                 # 必须（变量统一放这里）
├── README.md            # 必须
├── build.sh            # 必须（启动/初始化）
├── check.sh            # 必须（检查/验证）
└── *.sql               # 仅当 docker-compose 无法解决时才生成（如 init.sql）
```

## 使用方式

```bash
# 生成一套 MySQL v8.0 single 集群
./generate.sh mysql single v8.0

# 生成一套 MySQL v8.0 master-slave 集群
./generate.sh mysql master-slave v8.0

# 生成一套 Redis 集群
./generate.sh redis

# 生成一套 Kafka 集群
./generate.sh kafka
```

## 支持的中间件

| 类型 | 默认端口 |
|------|----------|
| mysql | 3306 |
| redis | 6379 |
| mongodb | 27017 |
| postgresql | 5432 |
| tidb | 4000 |
| elasticsearch | 9200 |
| kafka | 9092 |
| rabbitmq | 5672 |
| clickhouse | 8123 |
| etcd | 2379 |
| zookeeper | 2181 |
| consul | 8500 |

## 账号密码规范

- **账号统一**：`root`（最高权限）
- **密码统一**：`root123`
- 所有变量（账号、密码、端口等）放 `.env` 文件
- 仅当 docker-compose 无法表达的配置才额外生成 `.cnf` 或 `.sql` 文件
