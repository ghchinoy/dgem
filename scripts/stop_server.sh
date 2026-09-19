#!/usr/bin/env bash
set -euo pipefail

PID_FILE="diffgemma.pid"

if [[ -f "$PID_FILE" ]]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    echo "==> Stopping diffgemma server (PID $PID)..."
    kill "$PID" || true
    sleep 1
    if kill -0 "$PID" 2>/dev/null; then
      echo "==> Force stopping PID $PID..."
      kill -9 "$PID" || true
    fi
  fi
  rm -f "$PID_FILE"
fi

if pgrep -f "diffgemma serve" >/dev/null 2>&1; then
  echo "==> Killing remaining diffgemma serve instances..."
  pkill -f "diffgemma serve" || true
fi

echo "==> diffgemma server stopped."
