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
