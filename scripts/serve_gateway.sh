#!/usr/bin/env bash
set -euo pipefail

GATEWAY_PORT="${DGEM_GATEWAY_PORT:-8090}"
LOCAL_URL="${DGEM_LOCAL_URL:-http://127.0.0.1:8080/v1}"
TEMPLATES_DIR="${DGEM_TEMPLATES_DIR:-./templates}"
PID_FILE="dgem-gateway.pid"
LOG_FILE="gateway.log"

BIN="./bin/dgem"
if [[ ! -x "$BIN" ]]; then
  echo "==> Building dgem binary..."
  go build -o "$BIN" .
fi

# Check if already running on that port
if lsof -i ":$GATEWAY_PORT" >/dev/null 2>&1; then
  echo "==> Gateway is already running or port $GATEWAY_PORT is in use."
  curl -s "http://127.0.0.1:$GATEWAY_PORT/health" || true
  exit 0
fi

echo "==> Launching dgem serve (Gateway & Decision Studio) in background..."
echo "    Local URL:    $LOCAL_URL"
echo "    Port:         $GATEWAY_PORT"
echo "    Templates:    $TEMPLATES_DIR"
echo "    Logs:         $LOG_FILE"
echo "    PID file:     $PID_FILE"

nohup "$BIN" serve --local --local-url "$LOCAL_URL" --port "$GATEWAY_PORT" --templates-dir "$TEMPLATES_DIR" > "$LOG_FILE" 2>&1 &
GATEWAY_PID=$!
echo "$GATEWAY_PID" > "$PID_FILE"
echo "==> Gateway process started with PID $GATEWAY_PID"

echo -n "==> Waiting for gateway to become healthy..."
for i in {1..30}; do
  if curl -s "http://127.0.0.1:$GATEWAY_PORT/health" >/dev/null 2>&1; then
    echo " ready!"
    echo "==> Decision Studio UI: http://localhost:$GATEWAY_PORT"
    echo "==> REST API:           http://localhost:$GATEWAY_PORT/api/decide"
    echo "==> MCP Streamable:     http://localhost:$GATEWAY_PORT/mcp"
    exit 0
  fi
  sleep 0.5
  echo -n "."
done

echo ""
echo "Warning: Gateway started with PID $GATEWAY_PID but did not respond within 15 seconds."
echo "Check $LOG_FILE for details."
exit 1
