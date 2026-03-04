# MySQL 单节点 + Binlog（v5.7）

## 服务简介

单节点 MySQL 5.7，开启 Binlog 模式，支持数据恢复、主从复制、审计等场景。

## 快速开始

```bash
# 进入目录
cd docker/mysql/single-with-binlog/v5.7

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

### 数据恢复示例

```bash
# 查看 Binlog 事件
mysqlbinlog --no-dateors mysql-bin.000001

# 基于时间恢复（需要停止写入）
mysqlbinlog --stop-datetime="2024-01-01 12:00:00" mysql-bin.000001 | mysql -uroot -p

# 基于位置恢复
mysqlbinlog --start-position=123 --stop-position=456 mysql-bin.000001 | mysql -uroot -p
```

### 常见问题

**Q: Binlog 占用空间太大？**
A: 调整 `expire_logs_days` 参数，或手动清理：`PURGE BINARY LOGS BEFORE '2024-01-01';`

**Q: 如何确认 Binlog 正常写入？**
A: 执行 `SHOW MASTER STATUS;` 查看当前 Binlog 文件和位置
