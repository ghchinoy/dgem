#!/usr/bin/env bash
set -euo pipefail

MODEL="${MODEL:-/mnt/gcs/dgemma}"
CANVAS="${CANVAS:-128}"
PORT="${PORT:-8080}"
ENFORCE_EAGER="${ENFORCE_EAGER:-1}"
DISABLE_MM="${DISABLE_MM:-1}"
export VLLM_WORKER_MULTIPROC_METHOD="${VLLM_WORKER_MULTIPROC_METHOD:-fork}"
EXTRA_ARGS="${EXTRA_ARGS:-}"

mkdir -p /root/.cache/flashinfer /root/.triton /root/.cache/vllm

# If JIT cache archive exists in GCS, unpack it to eliminate runtime JIT compile overhead
JIT_CACHE_ARCHIVE="/mnt/gcs/jit-cache/jit-cache.tar.gz"
if [ -f "$JIT_CACHE_ARCHIVE" ]; then
  tar -xzf "$JIT_CACHE_ARCHIVE" -C /root/ || true
fi

# On large-memory instances (e.g. RTX Pro 6000 with 80GB RAM), stage weights into /dev/shm in background
if [ "${COPY_TO_SHM:-0}" = "1" ] && [ -d "/mnt/gcs/dgemma" ] && [ -f "/mnt/gcs/dgemma/config.json" ]; then
  echo "[init] Staging weights into /dev/shm/dgemma in background (xargs -P 16)..."
  mkdir -p /dev/shm/dgemma
  find /mnt/gcs/dgemma -maxdepth 1 -type f ! -name "*.safetensors" | xargs -I {} cp -f {} /dev/shm/dgemma/
  (
    find /mnt/gcs/dgemma -maxdepth 1 -type f -name "*.safetensors" | xargs -P 16 -I {} cp -f {} /dev/shm/dgemma/
    touch /dev/shm/dgemma/.ready
    echo "[init] Background safetensors copy to /dev/shm/dgemma complete"
  ) &
  MODEL="/dev/shm/dgemma"
fi

echo "[init] Starting structured_server proxy on port $PORT (upstream: http://127.0.0.1:8000)..."
python3 /opt/dgemma/structured_server.py \
  --upstream http://127.0.0.1:8000 \
  --model dgemma \
  --tokenizer "$MODEL" \
  --canvas "$CANVAS" \
  --host 0.0.0.0 \
  --port "$PORT" &

VLLM_EXTRA_ARGS=()
if [ "$ENFORCE_EAGER" = "1" ]; then
  export TORCH_COMPILE_DISABLE="${TORCH_COMPILE_DISABLE:-1}"
  VLLM_EXTRA_ARGS+=(--enforce-eager --kernel-config '{"enable_flashinfer_autotune":false,"enable_cutedsl_warmup":false,"enable_jit_warmup":false}')
fi

if [ "$DISABLE_MM" = "1" ]; then
  VLLM_EXTRA_ARGS+=(--language-model-only --skip-mm-profiling --limit-mm-per-prompt '{"image":0,"video":0}')
fi

if [ -n "$EXTRA_ARGS" ]; then
  VLLM_EXTRA_ARGS+=($EXTRA_ARGS)
fi

echo "[init] Launching vLLM engine core on port 8000..."
exec vllm serve "$MODEL" \
  --host 127.0.0.1 \
  --port 8000 \
  --served-model-name dgemma \
  --trust-remote-code \
  --safetensors-load-strategy prefetch \
  --max-num-seqs "${MAX_SEQS:-32}" \
  --max-model-len "${MAX_MODEL_LEN:-4096}" \
  --attention-backend "${ATTN:-TRITON_ATTN}" \
  --gpu-memory-utilization "${GPU_UTIL:-0.40}" \
  --kv-cache-memory "$(( ${KV_CACHE_GB:-2} * 1073741824 ))" \
  --max-logprobs 32 \
  --enable-prefix-caching \
  --diffusion-config "{\"canvas_length\": ${CANVAS}}" \
  --override-generation-config '{"max_new_tokens": null}' \
  --async-scheduling \
  "${VLLM_EXTRA_ARGS[@]}"
