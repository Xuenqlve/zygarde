# PostgreSQL single 17

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

单节点 PostgreSQL

## 稳定性说明

- 基于官方镜像 `postgres:17`，与 bitnami 变量/路径不混用。
- 数据目录统一使用 `/var/lib/postgresql/data`，避免镜像切换导致的持久化异常。
- 验收统一走 `build.sh -> check.sh -> cleanup`。
- 首次初始化耗时取决于镜像拉取和数据目录初始化。
