#!/bin/bash
# OpenOcta 宿主机启停脚本（非容器）
# 用法与 deploy.sh 类似：默认 restart；代码变更后用 rebuild 重新编译并重启。

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="${OPENOCTA_BIN:-$SCRIPT_DIR/openocta}"
PORT="${OPENOCTA_PORT:-18900}"
DATA_DIR="${OPENOCTA_STATE_DIR:-$HOME/.openocta}"
PID_FILE="$DATA_DIR/openocta.pid"
LOG_FILE="$DATA_DIR/host-service.log"
RUN_MODE="${OPENOCTA_RUN_MODE:-service}"

is_running() {
  if [ ! -f "$PID_FILE" ]; then
    return 1
  fi
  local pid
  pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [ -z "$pid" ]; then
    return 1
  fi
  if kill -0 "$pid" 2>/dev/null; then
    return 0
  fi
  return 1
}

cleanup_stale_pid() {
  if [ -f "$PID_FILE" ] && ! is_running; then
    rm -f "$PID_FILE"
  fi
}

ensure_binary() {
  if [ ! -x "$BINARY" ]; then
    echo "ERROR: 未找到可执行文件: $BINARY"
    echo "       请先执行: $SCRIPT_DIR/build.sh build"
    echo "       或设置环境变量 OPENOCTA_BIN 指向 openocta 二进制路径。"
    exit 1
  fi
}

do_start() {
  cleanup_stale_pid
  if is_running; then
    local pid
    pid="$(cat "$PID_FILE")"
    echo "==> OpenOcta 已在运行 (pid=$pid)"
    echo "==> http://127.0.0.1:$PORT"
    return 0
  fi

  ensure_binary
  mkdir -p "$DATA_DIR"

  echo "==========================================="
  echo "==> 启动 OpenOcta（宿主机）..."
  echo "==========================================="
  echo "==> 二进制: $BINARY"
  echo "==> 状态目录: $DATA_DIR"
  echo "==> 端口: $PORT"

  export OPENOCTA_STATE_DIR="$DATA_DIR"
  export OPENOCTA_RUN_MODE="$RUN_MODE"

  nohup "$BINARY" gateway run --port "$PORT" >>"$LOG_FILE" 2>&1 &
  local pid=$!
  echo "$pid" >"$PID_FILE"

  # 等待进程就绪；若启动失败则清理 pid 文件
  sleep 1
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "ERROR: 启动失败，请查看日志: $LOG_FILE"
    exit 1
  fi

  echo "==> 已启动 (pid=$pid)"
  echo "==> 日志: $LOG_FILE"
  echo "==> OpenOcta 运行于 http://127.0.0.1:$PORT"
}

do_stop() {
  cleanup_stale_pid
  if ! is_running; then
    echo "==> OpenOcta 未在运行"
    rm -f "$PID_FILE"
    return 0
  fi

  local pid
  pid="$(cat "$PID_FILE")"
  echo "==========================================="
  echo "==> 停止 OpenOcta（pid=$pid）..."
  echo "==========================================="

  kill "$pid" 2>/dev/null || true

  local i=0
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$i" -ge 10 ]; then
      echo "==> 进程未响应 SIGTERM，发送 SIGKILL..."
      kill -9 "$pid" 2>/dev/null || true
      break
    fi
    sleep 1
    i=$((i + 1))
  done

  rm -f "$PID_FILE"
  echo "==> 已停止"
}

do_status() {
  cleanup_stale_pid
  if is_running; then
    local pid
    pid="$(cat "$PID_FILE")"
    echo "==> 运行中 (pid=$pid)"
    echo "==> http://127.0.0.1:$PORT"
    echo "==> 日志: $LOG_FILE"
    return 0
  fi
  echo "==> 未运行"
  return 1
}

do_restart() {
  do_stop
  do_start
}

do_rebuild() {
  echo "==========================================="
  echo "==> Step 1: 重新编译二进制..."
  echo "==========================================="
  "$SCRIPT_DIR/build.sh" build

  echo "==========================================="
  echo "==> Step 2: 重启服务..."
  echo "==========================================="
  do_restart
  echo "==> 重新编译并部署完成！"
}

ACTION="${1:-restart}"

case "$ACTION" in
  start)
    do_start
    ;;
  stop)
    do_stop
    ;;
  restart)
    do_restart
    ;;
  rebuild|build)
    do_rebuild
    ;;
  status)
    do_status
    ;;
  *)
    echo "用法: $0 [start | stop | restart | rebuild | status]"
    echo "  start   : 启动宿主机上的 OpenOcta 进程"
    echo "  stop    : 停止进程"
    echo "  restart : 重启进程（默认）"
    echo "  rebuild : 重新编译并重启（前后端代码变更时使用）"
    echo "  status  : 查看运行状态"
    echo ""
    echo "环境变量（可选）:"
    echo "  OPENOCTA_BIN       - 二进制路径（默认: $SCRIPT_DIR/openocta）"
    echo "  OPENOCTA_PORT      - 监听端口（默认: 18900）"
    echo "  OPENOCTA_STATE_DIR - 状态目录（默认: ~/.openocta）"
    echo "  OPENOCTA_RUN_MODE  - 运行模式（默认: service）"
    exit 1
    ;;
esac
