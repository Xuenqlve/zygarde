#!/usr/bin/env bash
set -euo pipefail
if command -v podman >/dev/null 2>&1; then ENGINE_CMD=(podman); elif command -v docker >/dev/null 2>&1; then ENGINE_CMD=(docker); else echo "No container engine found." >&2; exit 1; fi

echo "[1/4] Container status"
"${ENGINE_CMD[@]}" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /zygarde-zk-[123]/'

echo "[2/4] ruok each node"
for n in 1 2 3; do
  R="$({ echo ruok | "${ENGINE_CMD[@]}" exec -i "zygarde-zk-$n" /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\r')"
  [ "$R" = "imok" ] || { echo "zk$n ruok failed: $R" >&2; exit 1; }
  echo "zk$n=$R"
done

echo "[3/4] leader/follower topology"
ROLE_COUNT="$(${ENGINE_CMD[@]} exec zygarde-zk-1 /bin/bash -lc "for h in zk1 zk2 zk3; do echo stat | nc \$h 2181 | grep Mode; done" | tee /dev/stderr | wc -l | tr -d '[:space:]')"
[ "$ROLE_COUNT" -eq 3 ] || { echo "mode lines != 3" >&2; exit 1; }

echo "[4/4] znode cross-node smoke"
"${ENGINE_CMD[@]}" exec zygarde-zk-1 /bin/bash -lc "zkCli.sh -server zk1:2181 create /zygarde_cluster_smoke ok >/tmp/zk_cluster.out 2>&1 || true"
OUT="$(${ENGINE_CMD[@]} exec zygarde-zk-3 /bin/bash -lc "zkCli.sh -server zk3:2181 get /zygarde_cluster_smoke 2>/dev/null | grep -E '^ok$' | head -n1" | tr -d '\r')"
[ "$OUT" = "ok" ] || { echo "cluster znode smoke failed: $OUT" >&2; exit 1; }
