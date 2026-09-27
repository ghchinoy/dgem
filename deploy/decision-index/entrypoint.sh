#!/usr/bin/env bash
set -euo pipefail

MODEL="${MODEL:-/mnt/gcs/dgemma}"
CANVAS="${CANVAS:-128}"
ENFORCE_EAGER="${ENFORCE_EAGER:-1}"
DISABLE_MM="${DISABLE_MM:-1}"
PORT="${PORT:-8080}"
INTERNAL_PORT=8081
VLLM_PORT=8000
TEMPERATURE="${TEMPERATURE:-1.0}"
API_KEY="${API_KEY:-}"

echo "=========================================================================="
echo "  dgem Decision Index Server (Wide-Canvas & Multi-Slot Tournament Engine) "
echo "=========================================================================="
echo "  Public Port:        $PORT (/v1/systemone, /health)"
echo "  Internal Proxy:     $INTERNAL_PORT"
echo "  vLLM Engine:        $VLLM_PORT"
echo "  Temperature:        T* = $TEMPERATURE"
echo "  Auth Required:      $([ -n "$API_KEY" ] && echo "true" || echo "false (open)")"
echo "=========================================================================="

# 1. Download or verify model weights
if [ ! -d "$MODEL" ] || [ ! -f "$MODEL/model.safetensors.index.json" ]; then
  MODEL_HF="${MODEL_HF:-nvidia/diffusiongemma-26B-A4B-it-NVFP4}"
  LOCAL_WEIGHTS="/tmp/dgemma"
  if [ -d "$LOCAL_WEIGHTS" ] && [ -f "$LOCAL_WEIGHTS/model.safetensors.index.json" ]; then
    echo "==> Using pre-staged model weights at $LOCAL_WEIGHTS"
  else
    echo "==> Downloading public model weights from Hugging Face: $MODEL_HF..."
    python3 -c "
from huggingface_hub import snapshot_download
snapshot_download(repo_id='$MODEL_HF', local_dir='$LOCAL_WEIGHTS')
"
  fi
  export MODEL="$LOCAL_WEIGHTS"
fi

# 2. Launch structured_server.py on internal port 8081
export AIP_HTTP_PORT="$INTERNAL_PORT"
export PORT="$INTERNAL_PORT"
echo "==> Starting internal structured_server.py on port $INTERNAL_PORT..."
/entrypoint.sh &
SERVER_PID=$!

# Wait for internal structured server to become ready
echo -n "==> Waiting for internal structured server on port $INTERNAL_PORT..."
for i in {1..180}; do
  if curl -s "http://127.0.0.1:$INTERNAL_PORT/health" >/dev/null 2>&1; then
    echo " ready!"
    break
  fi
  sleep 2
  echo -n "."
done

# 3. Start dgem systemone serve on public port 8080
echo "==> Starting dgem systemone adapter on port $PORT -> http://127.0.0.1:$INTERNAL_PORT/v1..."
exec /usr/local/bin/dgem systemone serve \
  --port "$PORT" \
  --upstream "http://127.0.0.1:$INTERNAL_PORT/v1" \
  --temperature "$TEMPERATURE" \
  --api-key "$API_KEY"
