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

# Record container boot and warmup stage telemetry in /tmp/dgemma/warmup_state.json
mkdir -p /tmp/dgemma
BOOT_TS="$(date +%s.%N)"
printf '{"phase":"mounting_gcs","boot_ts":%s}\n' "$BOOT_TS" > /tmp/dgemma/warmup_state.json

# On large-memory instances (e.g. RTX Pro 6000 with 80GB RAM), stage weights into /tmp/dgemma (in-memory rootfs tmpfs)
if [ "${COPY_TO_SHM:-0}" = "1" ] && [ -d "/mnt/gcs/dgemma" ] && [ -f "/mnt/gcs/dgemma/config.json" ]; then
  echo "[init] Staging weights into /tmp/dgemma RAM disk (16-stream parallel range copy)..."
  STAGE_START_TS="$(date +%s.%N)"
  printf '{"phase":"staging_tmpfs","boot_ts":%s,"stage_start_ts":%s}\n' "$BOOT_TS" "$STAGE_START_TS" > /tmp/dgemma/warmup_state.json
  find /mnt/gcs/dgemma -maxdepth 1 -type f ! -name "*.safetensors" | xargs -I {} cp -f {} /tmp/dgemma/
  (
    set -e
    python3 -c '
import glob, os, time
from concurrent.futures import ThreadPoolExecutor
src_files = sorted(glob.glob("/mnt/gcs/dgemma/*.safetensors"))
chunk = 32 * 1024 * 1024
tasks = []
for sf in src_files:
    df = os.path.join("/tmp/dgemma", os.path.basename(sf))
    sz = os.path.getsize(sf)
    with open(df, "wb") as f:
        f.truncate(sz)
    for off in range(0, sz, chunk):
        tasks.append((sf, df, off, min(chunk, sz - off)))
def copy_range(t):
    sf, df, off, length = t
    fd_in = os.open(sf, os.O_RDONLY)
    fd_out = os.open(df, os.O_WRONLY)
    try:
        data = os.pread(fd_in, length, off)
        os.pwrite(fd_out, data, off)
    finally:
        os.close(fd_in)
        os.close(fd_out)
with ThreadPoolExecutor(max_workers=16) as ex:
    list(ex.map(copy_range, tasks))
'
    STAGE_END_TS="$(date +%s.%N)"
    printf '{"phase":"loading_vllm_siglip","boot_ts":%s,"stage_start_ts":%s,"stage_end_ts":%s}\n' "$BOOT_TS" "$STAGE_START_TS" "$STAGE_END_TS" > /tmp/dgemma/warmup_state.json
    touch /tmp/dgemma/.ready
    echo "[init] Safetensors copy to /tmp/dgemma complete (stage_end_ts=$STAGE_END_TS)"
  ) &
  MODEL="/tmp/dgemma"
else
  printf '{"phase":"loading_vllm_siglip","boot_ts":%s,"stage_start_ts":%s,"stage_end_ts":%s}\n' "$BOOT_TS" "$BOOT_TS" "$BOOT_TS" > /tmp/dgemma/warmup_state.json
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

if [ "${COPY_TO_SHM:-0}" = "1" ] && [ "$MODEL" = "/tmp/dgemma" ]; then
  echo "[init] Waiting for /tmp/dgemma/.ready before launching vLLM..."
  while [ ! -f /tmp/dgemma/.ready ]; do
    sleep 1
  done
  echo "[init] /tmp/dgemma/.ready confirmed."
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
