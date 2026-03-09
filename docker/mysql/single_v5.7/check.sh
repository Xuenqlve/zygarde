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

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123}"

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-mysql-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT 1;"

echo "[3/3] Version"
"${ENGINE_CMD[@]}" exec zygarde-mysql-single mysql -uroot "-p${MYSQL_ROOT_PASSWORD}" -e "SELECT VERSION();"
