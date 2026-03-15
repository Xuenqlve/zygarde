#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

docker compose -p "mysql-double-demo" -f "docker-compose.yml" up -d

echo "MySQL mysql-1 (v5.7) compose stack started"

echo "MySQL mysql-2 (v8.0) compose stack started"

