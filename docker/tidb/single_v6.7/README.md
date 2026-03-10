# TiDB single v6.7

## 快速开始

```bash
./build.sh
./check.sh
docker compose down -v
```

## 场景

单节点入口（TiDB）+ 单 PD + 单 TiKV，适用于本地开发联调与初始化验证。

## 稳定性说明

- 版本固定：`pingcap/*:v6.5.12`（对外语义版本为 `v6.7`，当前默认映射到可用镜像 tag）。
- 启动顺序：pd -> tikv -> tidb。
- build 阶段以 TiDB status endpoint 就绪为可用信号。
- check 阶段覆盖容器状态、TiDB status、PD health、SQL 端口探活。
