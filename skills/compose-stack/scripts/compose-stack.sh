#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

usage() {
  cat <<EOF
compose-stack - 统一中间件 compose 生成/验收/回收工具

用法:
  $0 generate <middleware> <scenario> <version>
  $0 verify <target-dir>
  $0 cleanup <target-dir>
  $0 run <middleware> <scenario> <version>

示例:
  $0 generate mysql single v8.0
  $0 verify docker/mysql/single_v8.0
  $0 cleanup docker/mysql/single_v8.0
  $0 run redis cluster v7.4
EOF
}

resolve_dir() {
  local d="$1"
  if [ -d "$d" ]; then
    (cd "$d" && pwd)
  elif [ -d "$PROJECT_ROOT/$d" ]; then
    (cd "$PROJECT_ROOT/$d" && pwd)
  else
    echo ""
  fi
}

detect_engine() {
  if command -v podman >/dev/null 2>&1; then
    ENGINE_CMD=(podman)
    if podman compose version >/dev/null 2>&1; then
      COMPOSE_CMD=(podman compose)
    elif command -v podman-compose >/dev/null 2>&1; then
      COMPOSE_CMD=(podman-compose)
    fi
  elif command -v docker >/dev/null 2>&1; then
    ENGINE_CMD=(docker)
    if docker compose version >/dev/null 2>&1; then
      COMPOSE_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
      COMPOSE_CMD=(docker-compose)
    fi
  fi

  if [ "${ENGINE_CMD+x}" != "x" ] || [ "${COMPOSE_CMD+x}" != "x" ]; then
    echo "[ERROR] 未检测到容器引擎或 compose 命令" >&2
    return 1
  fi
}

do_generate() {
  local middleware="${1:-}"
  local scenario="${2:-single}"
  local version="${3:-}"

  if [ -z "$middleware" ] || [ -z "$version" ]; then
    echo "[ERROR] generate 需要 middleware 和 version" >&2
    usage
    return 1
  fi

  case "$middleware" in
    mysql)
      "$SCRIPT_DIR/generate-mysql.sh" "$scenario" "$version"
      ;;
    redis)
      "$SCRIPT_DIR/generate-redis.sh" "$scenario" "$version"
      ;;
    mongodb)
      "$SCRIPT_DIR/generate-mongodb.sh" "$scenario" "$version"
      ;;
    postgresql)
      "$SCRIPT_DIR/generate-postgresql.sh" "$scenario" "$version"
      ;;
    rabbitmq)
      "$SCRIPT_DIR/generate-rabbitmq.sh" "$scenario" "$version"
      ;;
    kafka)
      "$SCRIPT_DIR/generate-kafka.sh" "$scenario" "$version"
      ;;
    tidb)
      "$SCRIPT_DIR/generate-tidb.sh" "$scenario" "$version"
      ;;
    *)
      echo "[ERROR] 暂不支持的中间件: $middleware" >&2
      return 1
      ;;
  esac
}

do_cleanup() {
  local target="${1:-}"
  if [ -z "$target" ]; then
    echo "[ERROR] cleanup 需要 target-dir" >&2
    return 1
  fi

  local full
  full="$(resolve_dir "$target")"
  if [ -z "$full" ]; then
    echo "[ERROR] 目录不存在: $target" >&2
    return 1
  fi

  if [ ! -f "$full/docker-compose.yml" ]; then
    echo "[ERROR] 非 compose 目录: $full" >&2
    return 1
  fi

  detect_engine
  (cd "$full" && "${COMPOSE_CMD[@]}" down -v >/dev/null 2>&1 || true)
  rm -rf "$full/data" 2>/dev/null || true
  echo "[OK] cleanup 完成: $full (compose down -v + data/)"
}

do_verify() {
  local target="${1:-}"
  if [ -z "$target" ]; then
    echo "[ERROR] verify 需要 target-dir" >&2
    return 1
  fi

  local full
  full="$(resolve_dir "$target")"
  if [ -z "$full" ]; then
    echo "[ERROR] 目录不存在: $target" >&2
    return 1
  fi

  if [ ! -f "$full/docker-compose.yml" ]; then
    echo "[ERROR] 非 compose 目录: $full" >&2
    return 1
  fi

  if [ ! -x "$full/build.sh" ] || [ ! -x "$full/check.sh" ]; then
    echo "[ERROR] 目录缺少可执行脚本 build.sh/check.sh: $full" >&2
    return 1
  fi

  local verify_rc=0
  VERIFY_TARGET="$full"
  trap 'do_cleanup "$VERIFY_TARGET" >/dev/null 2>&1 || true' EXIT

  echo "[INFO] verify 开始: $full"
  (cd "$full" && ./build.sh) || verify_rc=$?
  if [ "$verify_rc" -eq 0 ]; then
    (cd "$full" && ./check.sh) || verify_rc=$?
  fi

  if [ "$verify_rc" -eq 0 ]; then
    echo "[OK] verify 通过: $full"
  else
    echo "[FAIL] verify 失败(code=$verify_rc): $full" >&2
  fi

  return "$verify_rc"
}

do_run() {
  local middleware="${1:-}"
  local scenario="${2:-single}"
  local version="${3:-}"

  do_generate "$middleware" "$scenario" "$version"
  local out="$PROJECT_ROOT/docker/$middleware/${scenario}_${version}"
  do_verify "$out"
}

CMD="${1:-}"
shift || true

case "$CMD" in
  generate)
    do_generate "$@"
    ;;
  verify)
    do_verify "$@"
    ;;
  cleanup)
    do_cleanup "$@"
    ;;
  run)
    do_run "$@"
    ;;
  help|--help|-h|"")
    usage
    ;;
  *)
    echo "[ERROR] 未知命令: $CMD" >&2
    usage
    exit 1
    ;;
esac
