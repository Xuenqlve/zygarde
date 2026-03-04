# MySQL 主从复制（v8.0）

## 服务简介

双节点 MySQL 8.0 主从复制架构：
- Master（主节点）：可读写，开启 Binlog
- Slave（从节点）：只读，从 Master 同步数据

## 架构图

```
┌─────────────┐      ┌─────────────┐
│   Master    │ ────▶│   Slave     │
│  3306 (写)  │      │  3307 (读)  │
└─────────────┘      └─────────────┘
```

## 快速开始

```bash
# 进入目录
cd docker/mysql/master-slave/v8.0

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 停止服务
docker-compose down
```

## 配置说明

### 端口映射

| 节点 | 容器端口 | 主机端口 | 说明 |
|------|----------|----------|------|
| Master | 3306 | 3306 | 主节点（可读写） |
| Slave | 3306 | 3307 | 从节点（只读） |

### 数据卷

| 主机路径 | 容器路径 | 说明 |
|----------|----------|------|
| ./data/mysql-master | /var/lib/mysql | 主节点数据 |
| ./data/mysql-slave | /var/lib/mysql | 从节点数据 |

### 主从配置（MySQL 8.0）

**Master:**
- server-id: 1
- GTID 模式
- 并行复制：LOGICAL_CLOCK

**Slave:**
- server-id: 2
- read-only: 1
- super-read-only: 1
- slave_parallel_workers: 4

## 调试指南

### 检查主从状态

```bash
# 查看 Slave 状态
docker exec -it zygarde-mysql-slave mysql -uroot -proot123 -e "SHOW SLAVE STATUS\G;"
```

关键指标：
- `Slave_IO_Running`: Yes（IO 线程运行中）
- `Slave_SQL_Running`: Yes（SQL 线程运行中）
- `Seconds_Behind_Master`: 0（无延迟）

### 验证同步

```bash
# 在 Master 创建表
docker exec -it zygarde-mysql-master mysql -uroot -proot123 -e "USE test_repl; CREATE TABLE t1 (id INT);"

# 在 Slave 验证
docker exec -it zygarde-mysql-slave mysql -uroot -proot123 -e "USE test_repl; SHOW TABLES;"
```

### 常见问题

**Q: Slave 连接报错 Authentication？**
A: MySQL 8.0 使用 `caching_sha2_password`，需要创建用户时指定认证插件：
```sql
CREATE USER 'repl'@'%' IDENTIFIED WITH mysql_native_password BY 'repl123';
```

**Q: GTID 模式下如何跳过事务？**
A:
```sql
SET GTID_NEXT='uuid:transaction_id';
BEGIN;
COMMIT;
SET GTID_NEXT='AUTOMATIC';
```
