#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
else
    echo "No container engine found." >&2
    exit 1
fi

# 支持通过 .env 覆盖默认 root 密码
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

run_mysql() {
    local container="$1"
    local sql="$2"
    "${ENGINE_CMD[@]}" exec "$container" mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "$sql"
}

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mysql-master|zygarde-mysql-slave/'

echo "[2/4] Replica status"
if ! run_mysql zygarde-mysql-slave "SHOW REPLICA STATUS\G" | \
    grep -E "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:|Last_IO_Error:|Last_SQL_Error:"; then
    run_mysql zygarde-mysql-slave "SHOW SLAVE STATUS\G" | \
        grep -E "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:|Last_IO_Error:|Last_SQL_Error:"
fi

echo "[3/4] GTID summary"
if ! run_mysql zygarde-mysql-slave "SHOW REPLICA STATUS\G" | \
    grep -E "Retrieved_Gtid_Set:|Executed_Gtid_Set:"; then
    run_mysql zygarde-mysql-slave "SHOW SLAVE STATUS\G" | \
        grep -E "Retrieved_Gtid_Set:|Executed_Gtid_Set:"
fi

echo "[4/4] Test replication"
run_mysql zygarde-mysql-master "CREATE DATABASE IF NOT EXISTS test_repl;"
sleep 2
run_mysql zygarde-mysql-slave "SHOW DATABASES;" | grep test_repl && echo "Replication works!" || echo "Replication may be delayed"
