#!/bin/bash
set -e

POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres123}"
REPL_USER="${REPL_USER:-repl_user}"
REPL_PASSWORD="${REPL_PASSWORD:-repl_pass}"
MASTER_HOST="${POSTGRES_MASTER_HOST:-postgres-master}"
MASTER_PORT="${POSTGRES_MASTER_PORT:-5432}"

echo "Waiting for master to be ready..."
until PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$MASTER_HOST" -U "$POSTGRES_USER" -d postgres -c 'select 1' >/dev/null 2>&1; do
  sleep 1
done

echo "Creating replication user..."
PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$MASTER_HOST" -U "$POSTGRES_USER" -d postgres -c \
  "CREATE USER $REPL_USER WITH REPLICATION PASSWORD '$REPL_PASSWORD';" 2>/dev/null || echo "User may already exist"

echo "Setting up replication on slave..."
until PGPASSWORD="$POSTGRES_PASSWORD" psql -h localhost -U "$POSTGRES_USER" -d postgres -c 'select 1' >/dev/null 2>&1; do
  sleep 1
done

# Check if already set up as replica
IS_RECOVERY=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h localhost -U "$POSTGRES_USER" -d postgres -tAc "select pg_is_in_recovery();" | tr -d '[:space:]')
if [ "$IS_RECOVERY" = "t" ]; then
  echo "Already configured as replica"
  exit 0
fi

# Stop postgres to do base backup
pg_ctl stop -D /var/lib/postgresql/data || true

# Run pg_basebackup
PGPASSWORD="$REPL_PASSWORD" pg_basebackup -h "$MASTER_HOST" -p "$MASTER_PORT" -U "$REPL_USER" -D /var/lib/postgresql/data -Fp -Xs -R -P

# Configure recovery.conf for older versions or ensure it works for newer
echo "host replication $REPL_USER 0.0.0.0/0 md5" >> /var/lib/postgresql/data/pg_hba.conf

# Start postgres
pg_ctl start -D /var/lib/postgresql/data

echo "Replication setup complete"
