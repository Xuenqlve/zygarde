# PostgreSQL single 16

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

单节点 PostgreSQL

## 稳定性说明

- 验收统一走 `build.sh -> check.sh -> cleanup`。
- 首次初始化耗时取决于镜像拉取和数据目录初始化。
