# MySQL single v8.0

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
| MYSQL_ROOT_PASSWORD | root123 | root 密码 |
| MYSQL_PORT | 3306 | MySQL 端口 |

## 特性

- **binlog**: 已开启 (mysql-bin)
- **GTID**: 已开启
- **binlog-format**: ROW

## 账号

- root / root123

## 场景

单节点 MySQL（开启 binlog + GTID，方便后续升级为主从复制）

## 稳定性说明

- 验收统一走 `build.sh -> check.sh -> cleanup`。
- 首次启动会初始化数据目录，验收结束后由 compose-stack 统一清理 `data/`。
- 单机场景默认开启 binlog+GTID，确保后续切换主从时配置一致。
