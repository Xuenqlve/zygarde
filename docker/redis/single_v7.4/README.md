# Redis single v7.4

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
| REDIS_PORT | 6379 | Redis 端口 |

## 场景

单实例 Redis（appendonly 已开启）

## 稳定性说明

- 验收统一走 `build.sh -> check.sh -> cleanup`。
- 首次拉取镜像时间较长属于正常现象；二次启动会显著加快。
- 验收后由 compose-stack 自动执行 `down -v` 并清理 `data/`。
