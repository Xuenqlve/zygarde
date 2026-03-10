# MySQL master-slave v8.0

## 快速开始

```bash
# 启动并配置主从复制
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
```

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| MYSQL_ROOT_PASSWORD | root123 | root 密码 |
| MYSQL_MASTER_PORT | 3306 | Master 端口 |
| MYSQL_SLAVE_PORT | 3307 | Slave 端口 |

## 账号

- root / root123

## 场景

主从复制 MySQL + GTID

## 稳定性说明

- build 阶段会在 master/slave 就绪后自动执行复制初始化（含 GTID）。
- check 阶段会校验主从角色与复制线程状态（IO/SQL）。
- 若出现端口冲突或历史容器残留，compose-stack 会在验收前先执行清理。
