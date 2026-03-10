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
