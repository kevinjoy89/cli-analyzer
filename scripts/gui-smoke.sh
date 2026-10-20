#!/bin/bash
# 启动 GUI 并捕获 stdout/stderr，用于在没有截图权限时确认渲染层无报错。
# 运行 8 秒后关闭，输出日志。用法：bash scripts/gui-smoke.sh
set -u
cd "$(dirname "$0")/.." || exit 1

APP="build/bin/CLI Analyzer.app/Contents/MacOS/cli-analyzer"
LOG="work/gui-smoke.log"
mkdir -p work

if [ ! -x "$APP" ]; then
  echo "missing $APP — run: wails build"
  exit 1
fi

"$APP" > "$LOG" 2>&1 &
PID=$!
echo "started pid=$PID"

# 轮询判活，不空等：进程若在 8 秒内退出，说明启动失败
for i in $(seq 1 8); do
  kill -0 "$PID" 2>/dev/null || { echo "process exited early (${i}s)"; break; }
  sleep 1
done

ALIVE=no
kill -0 "$PID" 2>/dev/null && ALIVE=yes

kill "$PID" 2>/dev/null
sleep 1
kill -9 "$PID" 2>/dev/null

echo "alive_after_8s=$ALIVE"
echo "--- log ($(wc -l < "$LOG" | tr -d ' ') lines) ---"
cat "$LOG"
