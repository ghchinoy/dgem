# Dockerfile for DiffusionGemma with Structured Reads (vLLM PR #57250) on Google Cloud Run
FROM vllm/vllm-openai:latest

LABEL maintainer="ghchinoy"
LABEL description="vLLM with DiffusionGemma discrete block diffusion structured reads overlay"

# Install git and overlay Matt Mastracci's PR #57250 branch (pure Python changes)
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && \
    git clone --depth 1 -b structured-reads-main https://github.com/mmastrac/vllm.git /tmp/vllm-pr && \
    VLLM_PATH=$(python3 -c "import vllm, os; print(os.path.dirname(vllm.__file__))") && \
    echo "Overlaying PR #57250 files into $VLLM_PATH..." && \
    for f in \
        config/scheduler.py \
        config/vllm.py \
        model_executor/models/diffusion_gemma.py \
        sampling_params.py \
        v1/core/sched/diffusion_scheduler.py \
        v1/engine/input_processor.py \
        v1/worker/gpu/model_runner.py \
        v1/worker/gpu/model_states/interface.py \
        v1/worker/gpu/spec_decode/utils.py; do \
      mkdir -p "$VLLM_PATH/$(dirname "$f")" && \
      cp "/tmp/vllm-pr/vllm/$f" "$VLLM_PATH/$f"; \
    done && \
    mkdir -p /opt/diffusion_reads && \
    cp -r /tmp/vllm-pr/examples/features/diffusion_reads/* /opt/diffusion_reads/ && \
    rm -rf /tmp/vllm-pr /var/lib/apt/lists/*

# Environment settings for Cloud Run
ENV PORT=8080
ENV HOST=0.0.0.0
ENV PYTHONUNBUFFERED=1
EXPOSE 8080

# Default serving configuration using 4-bit NVFP4 weights optimized for RTX Pro 6000 / L4
ENTRYPOINT ["vllm", "serve"]
CMD ["nvidia/diffusiongemma-26B-A4B-it-NVFP4", \
     "--diffusion-config", "{\"canvas_length\": 32}", \
     "--max-logprobs", "32", \
     "--enable-prefix-caching", \
     "--port", "8080", \
     "--host", "0.0.0.0"]
