# Redis master-slave v6.2

## 快速开始

```bash
# 启动
./build.sh

# 检查状态
./check.sh

# 停止
docker compose down -v
```

## 配置说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| REDIS_MASTER_PORT | 6379 | Master 端口 |
| REDIS_SLAVE_PORT | 6380 | Slave 端口 |

## 场景

主从复制（1主1从）
