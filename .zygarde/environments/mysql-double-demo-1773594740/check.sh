#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

docker compose -p "mysql-double-demo" -f "docker-compose.yml" ps

docker exec mysql-1 mysql -uroot "-p${MYSQL_MYSQL_1_ROOT_PASSWORD}" -e "SELECT 1;"

docker exec mysql-2 mysql -uroot "-p${MYSQL_MYSQL_2_ROOT_PASSWORD}" -e "SELECT 1;"

