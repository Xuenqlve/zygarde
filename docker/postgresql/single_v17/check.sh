#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/3] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-postgres-single/'

echo "[2/3] Connectivity"
"${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -c 'select 1;'

echo "[3/3] Version"
"${ENGINE_CMD[@]}" exec zygarde-postgres-single psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-app}" -tAc 'select version();'
