#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

# 加载 .env，允许统一覆盖密码和端口变量
if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

# 检测容器引擎
if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(podman-compose)
    fi
elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD=(docker-compose)
    fi
else
    echo "No container engine found." >&2
    exit 1
fi

if [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "No compose command found for current container engine." >&2
    exit 1
fi

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

wait_healthy() {
    local name="$1"
    local retries=30
    for _ in $(seq 1 "$retries"); do
        status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        [[ "$status" == "healthy" ]] && return 0
        sleep 2
    done
    echo "Container $name did not become healthy" >&2
    return 1
}

echo "[1/4] Starting MySQL master/slave..."
${COMPOSE_CMD[@]} up -d

echo "[2/4] Waiting for zygarde-mysql-master..."
wait_healthy zygarde-mysql-master || { ${COMPOSE_CMD[@]} logs zygarde-mysql-master; exit 1; }

echo "[3/4] Waiting for zygarde-mysql-slave..."
wait_healthy zygarde-mysql-slave || { ${COMPOSE_CMD[@]} logs zygarde-mysql-slave; exit 1; }

echo "[4/4] Configuring replication..."
"${ENGINE_CMD[@]}" exec -i zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" < slave-init.sql

echo "Replication status:"
if ! "${ENGINE_CMD[@]}" exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW REPLICA STATUS\G" | \
    grep -E "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:"; then
    "${ENGINE_CMD[@]}" exec zygarde-mysql-slave mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SHOW SLAVE STATUS\G" | \
        grep -E "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:" || true
fi

echo "Done!"
