# MySQL 单节点 + Binlog（v8.0）

## 服务简介

单节点 MySQL 8.0，开启 Binlog 模式，支持数据恢复、主从复制、审计等场景。

## 快速开始

```bash
# 进入目录
cd docker/mysql/single-with-binlog/v8.0

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 停止服务
docker-compose down
```

## 配置说明

### Binlog 配置

| 参数 | 值 | 说明 |
|------|-----|------|
| server-id | 1 | 服务器唯一 ID |
| log-bin | mysql-bin | Binlog 文件前缀 |
| binlog-format | ROW | Row 格式（推荐） |
| gtid-mode | ON | GTID 模式 |
| expire_logs_days | 7 | Binlog 保留天数 |

### 端口映射

| 容器端口 | 主机端口 | 说明 |
|----------|----------|------|
| 3306 | 3306 | MySQL 服务端口 |

### 数据卷

| 主机路径 | 容器路径 | 说明 |
|----------|----------|------|
| ./data/mysql | /var/lib/mysql | MySQL 数据目录 |
| ./logs/mysql | /var/log/mysql | MySQL 日志目录 |

## 调试指南

### 查看 Binlog 状态

```bash
# 进入容器
docker exec -it zygarde-mysql-binlog mysql -uroot -proot123

# 查看 Binlog 是否开启
SHOW VARIABLES LIKE 'log_bin';
SHOW VARIABLES LIKE 'server_id';

# 查看 Binlog 文件列表
SHOW MASTER STATUS;
SHOW BINARY LOGS;
```

### 注意事项

- MySQL 8.0 使用 GTID 模式，推荐使用 `mysqlbinlog --skip-gtids` 来恢复数据
- Row 格式的 Binlog 体积较大，但更安全

### 常见问题

**Q: Binlog 占用空间太大？**
A: 调整 `expire_logs_days` 参数，或手动清理：`PURGE BINARY LOGS BEFORE '2024-01-01';`
