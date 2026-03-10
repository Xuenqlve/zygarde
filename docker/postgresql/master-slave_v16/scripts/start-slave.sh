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
