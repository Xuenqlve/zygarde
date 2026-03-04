# MySQL 主从复制（v5.7）

## 服务简介

双节点 MySQL 5.7 主从复制架构：
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
cd docker/mysql/master-slave/v5.7

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
| ./logs/mysql-master | /var/log/mysql | 主节点日志 |
| ./logs/mysql-slave | /var/log/mysql | 从节点日志 |

### 主从配置

**Master:**
- server-id: 1
- 开启 Binlog

**Slave:**
- server-id: 2
- read-only: 1
- super-read-only: 1

## 调试指南

### 检查主从状态

```bash
# 进入 Master 容器
docker exec -it zygarde-mysql-master mysql -uroot -proot123

# 查看 Master 状态
SHOW MASTER STATUS;

# 进入 Slave 容器
docker exec -it zygarde-mysql-slave mysql -uroot -proot123

# 查看 Slave 状态
SHOW SLAVE STATUS\G;
```

### 验证同步

```bash
# 在 Master 创建数据库
docker exec -it zygarde-mysql-master mysql -uroot -proot123 -e "CREATE DATABASE test_repl;"

# 在 Slave 验证
docker exec -it zygarde-mysql-slave mysql -uroot -proot123 -e "SHOW DATABASES;"
# 应该能看到 test_repl
```

### 常见问题

**Q: Slave IO 线程连接失败？**
A: 检查 Master 是否启动，确保网络互通，验证复制账号密码

**Q: Slave 延迟高？**
A: 检查网络带宽，调整 `slave_parallel_workers`

**Q: 如何重新同步？**
A: 
```sql
STOP SLAVE;
RESET SLAVE ALL;
-- 重新配置复制
CHANGE MASTER TO MASTER_HOST='mysql-master', ...;
START SLAVE;
```
