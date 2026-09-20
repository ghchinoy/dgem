#!/usr/bin/env bash
set -euo pipefail

echo "================================================================================"
echo "  Starting DiffusionGemma GCE Startup Script"
echo "================================================================================"

# Get HF_TOKEN from metadata if provided
HF_TOKEN=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/attributes/hf-token" -H "Metadata-Flavor: Google" || true)

# Wait for apt lock if necessary
while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
  echo "Waiting for package manager..."
  sleep 5
done

# Install vLLM nightly matching PR #57250 base commit
echo "==> Installing matching vLLM nightly wheel..."
pip install -U pip
pip install https://wheels.vllm.ai/36fa72d2d0d2f86c7c83e1e99c9012b7bd26463b/vllm-0.29.1rc1.dev410%2Bg36fa72d2d-cp38-abi3-manylinux_2_28_x86_64.whl huggingface_hub
pip uninstall -y torchaudio

# Clone PR #57250 branch
echo "==> Cloning mmastrac/vllm structured-reads-main..."
rm -rf /tmp/vllm-pr
git clone --depth 1 -b structured-reads-main https://github.com/mmastrac/vllm.git /tmp/vllm-pr

VLLM_PATH=$(python3 -c "import vllm, os; print(os.path.dirname(vllm.__file__))")
echo "==> Overlaying PR #57250 files into $VLLM_PATH..."
for f in \
    model_executor/models/diffusion_gemma.py \
    sampling_params.py \
    v1/core/sched/diffusion_scheduler.py \
    v1/engine/input_processor.py \
    v1/worker/gpu/model_runner.py \
    v1/worker/gpu/model_states/interface.py \
    v1/worker/gpu/spec_decode/utils.py; do
  if [ -f "/tmp/vllm-pr/vllm/$f" ]; then
    mkdir -p "$VLLM_PATH/$(dirname "$f")"
    cp "/tmp/vllm-pr/vllm/$f" "$VLLM_PATH/$f"
  fi
done

mkdir -p /opt/diffusion_reads
cp -r /tmp/vllm-pr/examples/features/diffusion_reads/* /opt/diffusion_reads/

# Find vllm executable path
VLLM_BIN=$(which vllm || echo "/usr/local/bin/vllm")

MODEL_ID=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/attributes/model-id" -H "Metadata-Flavor: Google" 2>/dev/null || echo "nvidia/diffusiongemma-26B-A4B-it-NVFP4")
if [[ -z "$MODEL_ID" ]]; then
  MODEL_ID="nvidia/diffusiongemma-26B-A4B-it-NVFP4"
fi

TP_SIZE=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/attributes/tensor-parallel-size" -H "Metadata-Flavor: Google" 2>/dev/null || echo "1")
if [[ -z "$TP_SIZE" ]]; then
  TP_SIZE="1"
fi

DTYPE=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/attributes/dtype" -H "Metadata-Flavor: Google" 2>/dev/null || echo "auto")
if [[ -z "$DTYPE" ]]; then
  DTYPE="auto"
fi

# Create systemd service for vllm
echo "==> Creating systemd service for vllm ($MODEL_ID, TP=$TP_SIZE, dtype=$DTYPE)..."
cat <<EOF > /etc/systemd/system/vllm.service
[Unit]
Description=vLLM DiffusionGemma Service
After=network.target

[Service]
Type=simple
User=root
Environment="HF_TOKEN=$HF_TOKEN"
Environment="PYTHONUNBUFFERED=1"
ExecStart=$VLLM_BIN serve $MODEL_ID \
  --tensor-parallel-size $TP_SIZE \
  --dtype $DTYPE \
  --diffusion-config '{"canvas_length":32}' \
  --max-model-len 32768 \
  --max-logprobs 32 \
  --enable-prefix-caching \
  --attention-backend TRITON_ATTN \
  --enforce-eager \
  --port 8080 \
  --host 0.0.0.0
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now vllm.service

echo "==> vLLM service started successfully."
