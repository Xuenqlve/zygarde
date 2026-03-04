# MySQL 单节点（v8.0）

## 服务简介

单节点 MySQL 8.0 实例，适用于开发测试环境。

## 快速开始

```bash
# 进入目录
cd docker/mysql/single/v8.0

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f mysql

# 停止服务
docker-compose down
```

## 配置说明

### 端口映射

| 容器端口 | 主机端口 | 说明 |
|----------|----------|------|
| 3306 | 3306 | MySQL 服务端口 |

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| MYSQL_ROOT_PASSWORD | root123 | Root 用户密码 |
| MYSQL_DATABASE | app_db | 初始化创建的数据库 |
| MYSQL_USER | app | 普通用户 |
| MYSQL_PASSWORD | app123 | 普通用户密码 |

### 数据卷

| 主机路径 | 容器路径 | 说明 |
|----------|----------|------|
| ./data/mysql | /var/lib/mysql | MySQL 数据目录 |

## 调试指南

### 连接测试

```bash
# 使用 MySQL 客户端连接
docker exec -it zygarde-mysql-single mysql -uroot -proot123

# 或从主机连接
mysql -h 127.0.0.1 -P 3306 -uroot -proot123
```

### 查看日志

```bash
# 查看所有日志
docker-compose logs mysql

# 实时查看
docker-compose logs -f mysql
```

### 注意事项

- MySQL 8.0 默认使用 `caching_sha2_password` 认证插件
- 如果客户端不支持，可通过 `--default-authentication-plugin=mysql_native_password` 切换

### 常见问题

**Q: 容器启动失败？**
A: 检查端口是否被占用：`lsof -i :3306`
