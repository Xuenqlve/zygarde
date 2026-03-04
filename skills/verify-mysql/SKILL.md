# verify-mysql

MySQL docker-compose 配置验证工具。

## 功能

验证 MySQL docker-compose 配置是否正确，服务能否正常启动并运行。

## 使用方式

```bash
# 验证 MySQL 配置
./skills/verify-mysql/scripts/verify.sh <目录路径>

# 示例
./skills/verify-mysql/scripts/verify.sh docker/mysql/single/v8.0
./skills/verify-mysql/scripts/verify.sh docker/mysql/master-slave/v5.7
```

## 验证流程

1. 语法检查 - `docker-compose config`
2. 启动服务 - `docker-compose up -d`
3. 健康检查 - 等待容器 Running
4. 功能验证 - 执行 SQL 命令
5. 清理 - `docker-compose down -v`

## 支持的场景

| 目录结构 | 场景 | 验证命令 |
|----------|------|----------|
| single/ | 单节点 | `SELECT VERSION()` |
| single-with-binlog/ | 单节点+Binlog | `SHOW MASTER STATUS` |
| master-slave/ | 主从复制 | `SHOW MASTER STATUS` + `SHOW SLAVE STATUS` |
