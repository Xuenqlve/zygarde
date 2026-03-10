# Redis master-slave v7.4

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

## 稳定性说明

- build 阶段先等待 master 健康，再拉起 slave，降低首次复制抖动。
- check 阶段校验 master/slave 角色与 slave 链路状态（`master_link_status:up`）。
- 若有固定容器名冲突，compose-stack 验收前会自动清理旧容器。
