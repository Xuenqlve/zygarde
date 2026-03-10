#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
NC='\033[0m'
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }

usage() {
  echo "Usage: $0 <single|master-slave> <v16|v17>"
  exit 1
}

[ $# -lt 2 ] && usage
SCENARIO="$1"
VERSION_ARG="$2"
VERSION="${VERSION_ARG#v}" # image tag uses 16/17

if [ "$SCENARIO" != "single" ] && [ "$SCENARIO" != "master-slave" ]; then
  echo "场景错误: $SCENARIO"; usage
fi
if [ "$VERSION" != "16" ] && [ "$VERSION" != "17" ]; then
  echo "版本错误: $VERSION_ARG (v16 或 v17)"; usage
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
IMAGE="postgres:${VERSION}"
OUTPUT_DIR="$PROJECT_ROOT/docker/postgresql/${SCENARIO}_${VERSION_ARG}"
mkdir -p "$OUTPUT_DIR"

print_info "Generating PostgreSQL $SCENARIO $VERSION"

if [ "$SCENARIO" = "single" ]; then
  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  postgres:
    image: ${IMAGE}
    container_name: zygarde-postgres-single
    restart: unless-stopped
    ports:
      - "\${POSTGRES_PORT:-5432}:5432"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres123}
      POSTGRES_DB: \${POSTGRES_DB:-app}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${POSTGRES_USER:-postgres} -d \${POSTGRES_DB:-app}"]
      interval: 5s
      timeout: 5s
      retries: 30
      start_period: 20s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
POSTGRES_VERSION=${VERSION}
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123
POSTGRES_DB=app
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
  ENGINE_CMD=(podman)
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  ENGINE_CMD=(docker)
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/2] Starting PostgreSQL single..."
"${COMPOSE_CMD[@]}" up -d

echo "[2/2] Waiting for zygarde-postgres-single..."
for _ in $(seq 1 40); do
  status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' zygarde-postgres-single 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    echo "PostgreSQL is healthy."
    "${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -c 'select 1;' >/dev/null || true
    exit 0
  fi
  sleep 2
done

echo "Container zygarde-postgres-single did not become healthy" >&2
"${COMPOSE_CMD[@]}" logs postgres || true
exit 1
BUILD_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_SINGLE_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-postgres-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -c 'select 1;'

echo "[3/3] Version"
"${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -tAc 'select version();'
CHECK_SINGLE_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# PostgreSQL $SCENARIO $VERSION

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

单节点 PostgreSQL

## 稳定性说明

- 基于官方镜像 \`postgres:${VERSION}\`，与 bitnami 变量/路径不混用。
- 数据目录统一使用 \`/var/lib/postgresql/data\`，避免镜像切换导致的持久化异常。
- 验收统一走 \`build.sh -> check.sh -> cleanup\`。
- 首次初始化耗时取决于镜像拉取和数据目录初始化。
EOF

else
  mkdir -p "$OUTPUT_DIR/scripts"

  cat > "$OUTPUT_DIR/scripts/01-master-init.sh" <<'INIT_MASTER_EOF'
#!/usr/bin/env bash
set -euo pipefail

REPL_USER="${REPL_USER:-repl_user}"
REPL_PASSWORD="${REPL_PASSWORD:-repl_pass}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
DO
\$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${REPL_USER}') THEN
    EXECUTE format('CREATE ROLE %I WITH REPLICATION LOGIN PASSWORD %L', '${REPL_USER}', '${REPL_PASSWORD}');
  END IF;
END
\$\$;
SQL

# allow replication connections (idempotent append)
if ! grep -q "host replication ${REPL_USER}" "$PGDATA/pg_hba.conf"; then
  echo "host replication ${REPL_USER} 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
fi
INIT_MASTER_EOF
  chmod +x "$OUTPUT_DIR/scripts/01-master-init.sh"

  cat > "$OUTPUT_DIR/scripts/start-slave.sh" <<'START_SLAVE_EOF'
#!/usr/bin/env bash
set -euo pipefail

PGDATA="${PGDATA:-/var/lib/postgresql/data}"
REPL_USER="${REPL_USER:-repl_user}"
REPL_PASSWORD="${REPL_PASSWORD:-repl_pass}"
MASTER_HOST="${MASTER_HOST:-postgres-master}"
MASTER_PORT="${MASTER_PORT:-5432}"

if [ ! -s "$PGDATA/PG_VERSION" ]; then
  echo "[slave] empty data dir, waiting master ready..."
  until pg_isready -h "$MASTER_HOST" -p "$MASTER_PORT" -U "${POSTGRES_USER:-postgres}" >/dev/null 2>&1; do
    sleep 2
  done

  rm -rf "$PGDATA"/*
  export PGPASSWORD="$REPL_PASSWORD"
  pg_basebackup -h "$MASTER_HOST" -p "$MASTER_PORT" -U "$REPL_USER" -D "$PGDATA" -Fp -Xs -R -P
  unset PGPASSWORD
  chmod 700 "$PGDATA"
fi

exec postgres -c hot_standby=on
START_SLAVE_EOF
  chmod +x "$OUTPUT_DIR/scripts/start-slave.sh"

  cat > "$OUTPUT_DIR/docker-compose.yml" <<EOF
services:
  postgres-master:
    image: ${IMAGE}
    container_name: zygarde-postgres-master
    restart: unless-stopped
    ports:
      - "\${POSTGRES_MASTER_PORT:-5432}:5432"
    volumes:
      - ./data/postgres-master:/var/lib/postgresql/data
      - ./scripts/01-master-init.sh:/docker-entrypoint-initdb.d/01-master-init.sh:ro
    environment:
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres123}
      POSTGRES_DB: \${POSTGRES_DB:-app}
      REPL_USER: \${REPL_USER:-repl_user}
      REPL_PASSWORD: \${REPL_PASSWORD:-repl_pass}
    command: ["postgres", "-c", "wal_level=replica", "-c", "max_wal_senders=10", "-c", "max_replication_slots=10"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${POSTGRES_USER:-postgres} -d \${POSTGRES_DB:-app}"]
      interval: 5s
      timeout: 5s
      retries: 40
      start_period: 20s

  postgres-slave:
    image: ${IMAGE}
    container_name: zygarde-postgres-slave
    user: postgres
    restart: unless-stopped
    depends_on:
      postgres-master:
        condition: service_healthy
    ports:
      - "\${POSTGRES_SLAVE_PORT:-5433}:5432"
    volumes:
      - ./data/postgres-slave:/var/lib/postgresql/data
      - ./scripts/start-slave.sh:/scripts/start-slave.sh:ro
    environment:
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres123}
      REPL_USER: \${REPL_USER:-repl_user}
      REPL_PASSWORD: \${REPL_PASSWORD:-repl_pass}
      MASTER_HOST: postgres-master
      MASTER_PORT: 5432
    command: ["bash", "/scripts/start-slave.sh"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${POSTGRES_USER:-postgres}"]
      interval: 5s
      timeout: 5s
      retries: 40
      start_period: 25s
EOF

  cat > "$OUTPUT_DIR/.env" <<EOF
POSTGRES_VERSION=${VERSION}
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123
POSTGRES_DB=app
POSTGRES_MASTER_PORT=5432
POSTGRES_SLAVE_PORT=5433
REPL_USER=repl_user
REPL_PASSWORD=repl_pass
EOF

  cat > "$OUTPUT_DIR/build.sh" <<'BUILD_MS_EOF'
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"; cd "$ROOT_DIR"

if command -v podman >/dev/null 2>&1; then
  ENGINE_CMD=(podman)
  if podman compose version >/dev/null 2>&1; then COMPOSE_CMD=(podman compose); else COMPOSE_CMD=(podman-compose); fi
elif command -v docker >/dev/null 2>&1; then
  ENGINE_CMD=(docker)
  if docker compose version >/dev/null 2>&1; then COMPOSE_CMD=(docker compose); else COMPOSE_CMD=(docker-compose); fi
else
  echo "No container engine found." >&2; exit 1
fi

echo "[1/3] Starting PostgreSQL master/slave..."
"${COMPOSE_CMD[@]}" up -d

wait_healthy() {
  local name="$1"
  for _ in $(seq 1 60); do
    status="$(${ENGINE_CMD[@]} inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[2/3] Waiting master healthy..."
wait_healthy zygarde-postgres-master || { "${COMPOSE_CMD[@]}" logs postgres-master; exit 1; }

echo "[3/3] Waiting slave healthy..."
wait_healthy zygarde-postgres-slave || { "${COMPOSE_CMD[@]}" logs postgres-slave; exit 1; }

echo "PostgreSQL master/slave is healthy."
BUILD_MS_EOF
  chmod +x "$OUTPUT_DIR/build.sh"

  cat > "$OUTPUT_DIR/check.sh" <<'CHECK_MS_EOF'
#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-postgres-master|zygarde-postgres-slave/'

echo "[2/4] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-postgres-master psql -U "${POSTGRES_USER:-postgres}" -d postgres -c 'select 1;'
"${ENGINE_CMD[@]}" exec zygarde-postgres-slave psql -U "${POSTGRES_USER:-postgres}" -d postgres -c 'select 1;'

echo "[3/4] Replication on master"
ok=0
for _ in $(seq 1 60); do
  CNT="$(${ENGINE_CMD[@]} exec zygarde-postgres-master psql -U "${POSTGRES_USER:-postgres}" -d postgres -tAc "select count(*) from pg_stat_replication;" | tr -d '[:space:]')"
  if [ "${CNT:-0}" -ge 1 ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "No replica found on master" >&2; exit 1; }
echo "replica_count=${CNT}"

echo "[4/4] Slave recovery mode"
ok=0
for _ in $(seq 1 60); do
  REC="$(${ENGINE_CMD[@]} exec zygarde-postgres-slave psql -U "${POSTGRES_USER:-postgres}" -d postgres -tAc "select pg_is_in_recovery();" | tr -d '[:space:]')"
  if [ "$REC" = "t" ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "Slave is not in recovery mode" >&2; exit 1; }
echo "recovery_mode=${REC}"
CHECK_MS_EOF
  chmod +x "$OUTPUT_DIR/check.sh"

  cat > "$OUTPUT_DIR/README.md" <<EOF
# PostgreSQL $SCENARIO $VERSION

## 快速开始

\`\`\`bash
./build.sh
./check.sh
docker compose down -v
\`\`\`

## 场景

一主一从流复制 PostgreSQL

## 稳定性说明

- 基于官方镜像 \`postgres:${VERSION}\`，主从初始化采用“主先就绪 + 从库首启克隆”模式。
- slave 首次启动会基于 \`pg_basebackup -R\` 自动初始化。
- check 阶段强校验主库 \`pg_stat_replication\` 与从库 \`pg_is_in_recovery()\`，并带重试窗口。
- 验收前若有残留旧容器，compose-stack 会统一清理。
EOF
fi

print_success "Done: $OUTPUT_DIR"
echo ""
print_success "PostgreSQL $SCENARIO $VERSION generation complete!"
