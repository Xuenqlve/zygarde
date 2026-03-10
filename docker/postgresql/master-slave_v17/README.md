# PostgreSQL master-slave 17

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

一主一从流复制 PostgreSQL

## 稳定性说明

- 基于官方镜像 `postgres:17`，主从初始化采用“主先就绪 + 从库首启克隆”模式。
- slave 首次启动会基于 `pg_basebackup -R` 自动初始化。
- check 阶段强校验主库 `pg_stat_replication` 与从库 `pg_is_in_recovery()`，并带重试窗口。
- 验收前若有残留旧容器，compose-stack 会统一清理。
