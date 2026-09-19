# Dockerfile for DiffusionGemma with Structured Reads (vLLM PR #57250) on Google Cloud Run
FROM vllm/vllm-openai:latest

LABEL maintainer="ghchinoy"
LABEL description="vLLM with DiffusionGemma discrete block diffusion structured reads overlay"

# Install git and overlay Matt Mastracci's PR #57250 branch
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && \
    git clone --depth 1 -b structured-reads-main https://github.com/mmastrac/vllm.git /tmp/vllm-pr && \
    VLLM_PATH=$(python3 -c "import vllm, os; print(os.path.dirname(vllm.__file__))") && \
    echo "Overlaying PR #57250 files into $VLLM_PATH..." && \
    for f in \
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
    rm -rf /tmp/vllm-pr /var/lib/apt/lists/* && \
    python3 -c "\
import os, vllm\n\
vp = os.path.dirname(vllm.__file__)\n\
vf = os.path.join(vp, 'config', 'vllm.py')\n\
with open(vf, 'r') as f:\n\
    c = f.read()\n\
hook = '''\n\
        if (\n\
            self.diffusion_config is not None\n\
            and self.scheduler_config.async_scheduling\n\
            and self.scheduler_config.scheduler_cls is None\n\
        ):\n\
            self.scheduler_config.scheduler_cls = (\n\
                \"vllm.v1.core.sched.diffusion_scheduler.DiffusionAsyncScheduler\"\n\
            )\n\
'''\n\
if 'DiffusionAsyncScheduler' not in c:\n\
    c = c.replace('current_platform.check_and_update_config(self)', 'current_platform.check_and_update_config(self)' + hook)\n\
    with open(vf, 'w') as f:\n\
        f.write(c)\n\
cf = os.path.join(vp, '_custom_ops.py')\n\
with open(cf, 'r') as f:\n\
    c = f.read()\n\
old_repack = 'return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, size_k, size_n, num_bits, is_a_8bit\n    )'\n\
new_repack = 'perm = torch.empty(0, dtype=torch.int, device=b_q_weight.device)\n    return torch.ops._C.gptq_marlin_repack(\n        b_q_weight, perm, size_k, size_n, num_bits, is_a_8bit\n    )'\n\
c = c.replace(old_repack, new_repack)\n\
with open(cf, 'w') as f:\n\
    f.write(c)\n\
print('vLLM patches applied successfully!')\n\
" && \
    python3 -c "import vllm; print('vllm successfully verified, version:', vllm.__version__)"

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
