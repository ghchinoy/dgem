#!/usr/bin/env bash
set -euo pipefail

MODEL="${MODEL:-/mnt/gcs/dgemma}"
CANVAS="${CANVAS:-128}"
PORT="${AIP_HTTP_PORT:-${PORT:-8080}}"
ENFORCE_EAGER="${ENFORCE_EAGER:-1}"
DISABLE_MM="${DISABLE_MM:-1}"
export VLLM_WORKER_MULTIPROC_METHOD="${VLLM_WORKER_MULTIPROC_METHOD:-fork}"
EXTRA_ARGS="${EXTRA_ARGS:-}"

# Vertex AI Online Prediction passes AIP_STORAGE_URI (e.g. gs://dgem-weights-.../dgemma) instead of a Cloud Run FUSE mount
if [ ! -d "$MODEL" ] && [ -n "${AIP_STORAGE_URI:-}" ]; then
  echo "[init] Vertex AI AIP_STORAGE_URI detected ($AIP_STORAGE_URI); staging weights to /tmp/dgemma..."
  mkdir -p /tmp/dgemma
  python3 -c '
import json, os, urllib.request
from concurrent.futures import ThreadPoolExecutor
uri = os.environ["AIP_STORAGE_URI"].rstrip("/")
assert uri.startswith("gs://"), f"Expected gs:// URI, got {uri}"
bucket, _, prefix = uri[5:].partition("/")
req = urllib.request.Request(
    "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token",
    headers={"Metadata-Flavor": "Google"}
)
token = json.loads(urllib.request.urlopen(req, timeout=5).read().decode())["access_token"]
list_url = f"https://storage.googleapis.com/storage/v1/b/{bucket}/o?prefix={prefix}/"
items = json.loads(urllib.request.urlopen(urllib.request.Request(list_url, headers={"Authorization": f"Bearer {token}"})).read().decode()).get("items", [])
def fetch_obj(item):
    name = item["name"]
    rel = name[len(prefix)+1:]
    if not rel or "/" in rel:
        return
    dst = os.path.join("/tmp/dgemma", rel)
    dl_url = f"https://storage.googleapis.com/{bucket}/{name}"
    with urllib.request.urlopen(urllib.request.Request(dl_url, headers={"Authorization": f"Bearer {token}"})) as r, open(dst, "wb") as f:
        while True:
            chunk = r.read(16 * 1024 * 1024)
            if not chunk:
                break
            f.write(chunk)
with ThreadPoolExecutor(max_workers=16) as ex:
    list(ex.map(fetch_obj, items))
'
  MODEL="/tmp/dgemma"
fi

mkdir -p /root/.cache/flashinfer /root/.triton /root/.cache/vllm

# If JIT cache archive exists in GCS, unpack it to eliminate runtime JIT compile overhead
JIT_CACHE_ARCHIVE="/mnt/gcs/jit-cache/jit-cache.tar.gz"
if [ -f "$JIT_CACHE_ARCHIVE" ]; then
  tar -xzf "$JIT_CACHE_ARCHIVE" -C /root/ || true
fi

# Record container boot and warmup stage telemetry in /tmp/dgemma/warmup_state.json
mkdir -p /tmp/dgemma
export BOOT_TS="$(date +%s.%N)"
export GCS_BUCKET="${GCS_BUCKET:-dgem-weights-genai-blackbelt-fishfooding}"
printf '{"phase":"mounting_gcs","boot_ts":%s,"bytes_staged_gb":0.0}\n' "$BOOT_TS" > /tmp/dgemma/warmup_state.json

# On large-memory instances (e.g. RTX Pro 6000 with 80GB RAM), stage weights into /tmp/dgemma via 64-stream Direct GCS HTTPS Range API overlapped with vLLM initialization
if [ "${COPY_TO_SHM:-0}" = "1" ] && [ -d "/mnt/gcs/dgemma" ] && [ -f "/mnt/gcs/dgemma/config.json" ]; then
  echo "[init] Overlapping 64-stream GCS HTTPS Range staging (/tmp/dgemma) with vLLM + CUDA initialization..."
  STAGE_START_TS="$(date +%s.%N)"
  export STAGE_START_TS
  printf '{"phase":"staging_tmpfs","boot_ts":%s,"stage_start_ts":%s,"bytes_staged_gb":0.0}\n' "$BOOT_TS" "$STAGE_START_TS" > /tmp/dgemma/warmup_state.json
  find /mnt/gcs/dgemma -maxdepth 1 -type f ! -name "*.safetensors" | xargs -I {} cp -f {} /tmp/dgemma/

  # Pre-copy first 4 MB (containing the complete safetensors JSON header) and truncate to exact size synchronously (<0.2s)
  # so VllmConfig header validation at t=38s succeeds immediately while tensor bodies stream in parallel
  python3 -c '
import glob, os
HDR_BYTES = 4 * 1024 * 1024
for sf in sorted(glob.glob("/mnt/gcs/dgemma/*.safetensors")):
    df = os.path.join("/tmp/dgemma", os.path.basename(sf))
    sz = os.path.getsize(sf)
    with open(sf, "rb") as f_in, open(df, "wb") as f_out:
        f_out.write(f_in.read(min(HDR_BYTES, sz)))
        f_out.truncate(sz)
'

  # Install sitecustomize.py hook on DefaultModelLoader.load_weights so vLLM runs all Python/Torch/Config/CUDA/NCCL init
  # in parallel and only waits right before copying tensor weights into GPU VRAM (t ~ 65s)
  cat << 'EOF' > /tmp/sitecustomize.py
import os, time
try:
    from vllm.model_executor.model_loader.default_loader import DefaultModelLoader
    _orig_load_weights = DefaultModelLoader.load_weights
    def _waiting_load_weights(self, model, model_config):
        if str(getattr(model_config, "model", "")).startswith("/tmp/dgemma") and not os.path.exists("/tmp/dgemma/.ready"):
            print("[init] EngineCore reached DefaultModelLoader.load_weights; waiting for /tmp/dgemma/.ready...", flush=True)
            while not os.path.exists("/tmp/dgemma/.ready"):
                time.sleep(0.2)
            print("[init] /tmp/dgemma/.ready confirmed! Loading 17.53 GiB weights via zero-copy mmap...", flush=True)
        return _orig_load_weights(self, model, model_config)
    DefaultModelLoader.load_weights = _waiting_load_weights
except Exception:
    pass
EOF
  export PYTHONPATH="/tmp:${PYTHONPATH:-}"

  (
    set -e
    python3 -c '
import glob, json, os, time, urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

req = urllib.request.Request(
    "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token",
    headers={"Metadata-Flavor": "Google"}
)
with urllib.request.urlopen(req, timeout=5) as r:
    token = json.loads(r.read().decode())["access_token"]

bucket = os.environ.get("GCS_BUCKET", "dgem-weights-genai-blackbelt-fishfooding")
src_files = sorted(glob.glob("/mnt/gcs/dgemma/*.safetensors"))
hdr = 4 * 1024 * 1024
chunk = 64 * 1024 * 1024
tasks = []
pre_staged = 0
for sf in src_files:
    bn = os.path.basename(sf)
    df = os.path.join("/tmp/dgemma", bn)
    sz = os.path.getsize(sf)
    start_off = min(hdr, sz)
    pre_staged += start_off
    for off in range(start_off, sz, chunk):
        tasks.append((bucket, token, bn, df, off, min(chunk, sz - off)))

def download_range(t):
    bucket, token, bn, df, off, length = t
    end = off + length - 1
    url = f"https://storage.googleapis.com/{bucket}/dgemma/{bn}"
    for attempt in range(4):
        try:
            rreq = urllib.request.Request(url, headers={
                "Authorization": f"Bearer {token}",
                "Range": f"bytes={off}-{end}"
            })
            with urllib.request.urlopen(rreq, timeout=45) as resp:
                data = resp.read()
            fd_out = os.open(df, os.O_WRONLY)
            try:
                os.pwrite(fd_out, data, off)
            finally:
                os.close(fd_out)
            return len(data)
        except Exception:
            if attempt == 3:
                raise
            time.sleep(0.5 * (attempt + 1))

t0 = float(os.environ.get("STAGE_START_TS", time.time()))
boot_ts = float(os.environ.get("BOOT_TS", t0))
staged = pre_staged
last_write = 0.0
with ThreadPoolExecutor(max_workers=64) as ex:
    futs = [ex.submit(download_range, t) for t in tasks]
    for fut in as_completed(futs):
        staged += fut.result()
        now = time.time()
        if now - last_write >= 0.5 or staged >= 18800000000:
            last_write = now
            gb = round(staged / (1024**3), 2)
            try:
                with open("/tmp/dgemma/warmup_state.json.tmp", "w") as wf:
                    json.dump({
                        "phase": "staging_tmpfs",
                        "boot_ts": boot_ts,
                        "stage_start_ts": t0,
                        "bytes_staged_gb": gb
                    }, wf)
                os.replace("/tmp/dgemma/warmup_state.json.tmp", "/tmp/dgemma/warmup_state.json")
            except Exception:
                pass
dt = time.time() - t0
print(f"[init] 64-stream GCS HTTPS range copy of {round(staged/(1024**3),2)} GiB to /tmp/dgemma completed in {dt:.2f}s", flush=True)
'
    STAGE_END_TS="$(date +%s.%N)"
    printf '{"phase":"loading_vllm_siglip","boot_ts":%s,"stage_start_ts":%s,"stage_end_ts":%s,"bytes_staged_gb":17.53}\n' "$BOOT_TS" "$STAGE_START_TS" "$STAGE_END_TS" > /tmp/dgemma/warmup_state.json
    touch /tmp/dgemma/.ready
    echo "[init] Safetensors copy to /tmp/dgemma complete (stage_end_ts=$STAGE_END_TS)"
  ) &
  MODEL="/tmp/dgemma"
else
  printf '{"phase":"loading_vllm_siglip","boot_ts":%s,"stage_start_ts":%s,"stage_end_ts":%s,"bytes_staged_gb":17.53}\n' "$BOOT_TS" "$BOOT_TS" "$BOOT_TS" > /tmp/dgemma/warmup_state.json
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
  # Note: vLLM starts immediately and overlaps its ~65s Python/Torch/CUDA/NCCL init with /tmp/dgemma staging via /tmp/sitecustomize.py!
  # Reclaim 17.53 GiB tmpfs RAM as soon as vLLM moves weights to GPU VRAM
  (
    while ! python3 -c 'import urllib.request; urllib.request.urlopen("http://127.0.0.1:8000/health", timeout=1)' >/dev/null 2>&1; do
      sleep 2
    done
    rm -f /tmp/dgemma/*.safetensors
    printf '{"phase":"ready","bytes_staged_gb":17.53,"vllm_ready":true}\n' > /tmp/dgemma/warmup_state.json
    echo "[init] Reclaimed 17.53 GiB /tmp/dgemma RAM after vLLM GPU initialization."
  ) &
else
  VLLM_EXTRA_ARGS+=(--safetensors-load-strategy prefetch)
fi

echo "[init] Launching vLLM engine core on port 8000 (overlapped with /tmp/dgemma staging)..."
exec vllm serve "$MODEL" \
  --host 127.0.0.1 \
  --port 8000 \
  --served-model-name dgemma \
  --trust-remote-code \
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
