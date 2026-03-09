# verify-compose

验证 docker-compose 配置是否正确，服务能否正常启动运行。

## 功能

根据目标目录中的 `docker-compose.yml` 内容自动选择验证脚本：
- MySQL → `verify-mysql.sh`
- 其他中间件 → 开发中

## 使用方式

```bash
# 验证配置（自动选择脚本）
./skills/verify-compose/scripts/verify.sh <目录路径>

# 也可以直接调用 MySQL 验证脚本
./skills/verify-compose/scripts/verify-mysql.sh <目录路径>
```

示例：

```bash
./skills/verify-compose/scripts/verify.sh docker/mysql/single_v8.0
./skills/verify-compose/scripts/verify.sh docker/mysql/master-slave_v5.7
```

## 验证脚本选择逻辑

1. 检查目标目录是否存在 `docker-compose.yml`
2. 从 `docker-compose.yml` 内容识别中间件类型（不是仅按路径名）
3. MySQL 场景转发到 `verify-mysql.sh`

## 验证流程

1. 语法检查 - `compose config`
2. 容器引擎检查 - `docker` 或 `podman`
3. 端口占用检查
4. 启动服务 - `compose up -d`
5. 等待就绪 - 等待容器 running + SQL 探活
6. 功能验证 - 单机或主从验证
7. 清理 - `compose down -v`

## 等待逻辑（重要）

- 单容器最多等待约 120 秒（12 次 × 10 秒）
- 每轮先检查容器状态，再执行 `SELECT 1` 探活
- 主从场景会分别等待 `master` 和 `slave`

## MySQL 验证点

- 单机：`SELECT VERSION()`
- 主从：
  - `SHOW MASTER STATUS`
  - 优先 `SHOW REPLICA STATUS`，回退 `SHOW SLAVE STATUS`
  - 校验 IO/SQL 线程是否为 `Yes`

## 兼容性

- 容器引擎：`docker`、`podman`
- compose 命令：`docker compose`、`docker-compose`、`podman compose`、`podman-compose`

## 待开发

- `verify-redis.sh`
- `verify-postgresql.sh`
- `verify-mongodb.sh`
- `verify-kafka.sh`
