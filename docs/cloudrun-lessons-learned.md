# Lessons Learned: Deploying Experimental vLLM on Google Cloud Run with GPU

This document records the empirical findings, architectural trade-offs, and operational lessons learned while attempting to deploy Google DeepMind's **DiffusionGemma** on **Google Cloud Run with GPUs** using an experimental vLLM discrete block diffusion branch (PR #57250), along with a concrete blueprint for building custom CUDA C++ extensions in the future.

---

## 1. Executive Summary

Google Cloud Run with GPUs offers an attractive serverless platform: zero-to-N autoscaling, pay-per-second billing, and access to enterprise NVIDIA L4 (24 GB) and RTX Pro 6000 (48 GB) GPUs.

However, deploying **unmerged experimental branches of high-performance ML engines (like vLLM)** to Cloud Run reveals fundamental friction points between serverless container constraints and deep-learning software architecture:

1. **The Python-Over-Container Fallacy**: Overlaying pure Python files from an active PR onto a prebuilt official container image (`vllm/vllm-openai:latest`) fails due to C++ CUDA ABI mismatches at the PyTorch dispatcher boundary.
2. **Shared Egress NAT & Hugging Face Hub (HTTP 429)**: Unauthenticated model weight ingestion from Cloud Run shares public GCP egress IP pools that Hugging Face aggressively rate-limits.
3. **Startup Probe Deadlines vs. Cold Starts**: Downloading multi-gigabyte models on container cold start risks exceeding serverless health check timeouts.

---

## 2. Deep Dive: The C++ CUDA Extension ABI Mismatch

### The Core Mechanism
vLLM achieves high-throughput inference by compiling bespoke CUDA kernels for attention, quantization (Marlin, AWQ, GPTQ), and Mixture of Experts (MoE). These kernels are registered into PyTorch via `TORCH_LIBRARY_IMPL` and exposed through `torch.ops._C` and `torch.ops._moe_C`.

When Python calls a custom op, PyTorch's C++ dispatcher validates the types and argument positions on the **host CPU** before launching the CUDA kernel.

### The Failure Chain
Because Matt Mastracci's PR #57250 branch (`mmastrac/vllm:structured-reads-main`) was forked from a different commit of vLLM than the official Docker Hub release image (`vllm/vllm-openai:latest`), the Python wrappers and C++ kernel schemas diverged:

#### Failure 1: Marlin Quantization Repack (`gptq_marlin_repack`)
* **Compiled C++ Schema in Image**:
  ```cpp
  _C::gptq_marlin_repack(Tensor b_q_weight, Tensor perm, SymInt size_k, SymInt size_n, int num_bits, bool is_a_8bit) -> Tensor
  ```
* **PR Python Wrapper**:
  ```python
  torch.ops._C.gptq_marlin_repack(b_q_weight, size_k, size_n, num_bits, is_a_8bit)
  ```
* **Result**: `size_k` (integer `2816`) was passed positionally into argument 1 (`Tensor perm`), causing:
  `RuntimeError: _C::gptq_marlin_repack() Expected a value of type 'Tensor' for argument 'perm' but instead found type 'int'. Unable to cast 2816 to Tensor.`

#### Failure 2: Fused MoE Marlin GEMM (`moe_wna16_marlin_gemm`)
Even after patching `_custom_ops.py` to supply an empty permutation tensor for weight repacking, the engine crashed during model profiling:
```text
(EngineCore pid=102) Declaration: _moe_C::moe_wna16_marlin_gemm(..., Tensor num_tokens_past_padded, ...) -> Tensor
(EngineCore pid=102) Value: 64
(EngineCore pid=102) Position: 13
(EngineCore pid=102) RuntimeError: _moe_C::moe_wna16_marlin_gemm() Expected a value of type 'Tensor' for argument 'num_tokens_past_padded' but instead found type 'int'.
```
Position 13 had been refactored in the C++ extension to accept a Tensor, while the branch's Python caller passed an integer (`64`).

### Takeaway
**You cannot reliably overlay Python files from an experimental git branch on top of a precompiled vLLM Docker image.** In high-velocity ML frameworks, internal C++ kernel schemas change frequently. Experimental branches must be compiled in tandem with their matching C++ extensions.

---

## 3. Lesson: Hugging Face Hub Rate Limiting on Cloud Run

* **Symptom**: On container cold start, vLLM spent 4–8 minutes stuck in retry loops:
  ```text
  (APIServer pid=1) HTTP Error 429 thrown while requesting HEAD https://huggingface.co/.../config.json
  (APIServer pid=1) Rate limited. Waiting 253.0s before retry [Retry 1/5].
  ```
* **Root Cause**: Cloud Run instances route egress through Google Cloud shared NAT IP ranges. Because many users run anonymous queries from GCP, Hugging Face Hub enforces severe IP-based rate limiting on unauthenticated requests.
* **Solution**:
  1. Always supply a Hugging Face User Access Token via `HF_TOKEN` in the environment.
  2. For production, never download weights over the public internet on container boot: pre-stage model weights in a **Google Cloud Storage (GCS) bucket** and mount it via Cloud Run volume mounts (GCS FUSE). This drops cold-start latency from minutes to under 15 seconds.

---

## 4. Lesson: Cloud Run Startup Probe Configuration

* **Initial Anti-Pattern**:
  ```bash
  --startup-probe tcpSocket.port=8080,initialDelaySeconds=240,failureThreshold=1,timeoutSeconds=240,periodSeconds=240
  ```
  This configuration forces Cloud Run to wait 4 full minutes before performing the first probe. If the container finishes in 60 seconds, it still sits idle. Worse, with `failureThreshold=1`, a single failed ping immediately terminates the container.
* **Best-Practice Pattern**:
  ```bash
  --startup-probe tcpSocket.port=8080,initialDelaySeconds=10,periodSeconds=10,failureThreshold=60,timeoutSeconds=4
  ```
  This begins probing after 10 seconds and polls every 10 seconds. The moment the server binds to port 8080, the instance is marked healthy immediately. With 60 retries, it provides a 10-minute readiness window for weight downloads and memory profiling.

---

## 5. Next Steps: Building Native CUDA C++ Extensions for Cloud Run

If you wish to deploy PR #57250 to Cloud Run in the future with full C++ ABI integrity, use a **multi-stage CUDA compilation Dockerfile**:

### Blueprint: `Dockerfile.cuda-build`

```dockerfile
# ==============================================================================
# Stage 1: Build CUDA C++ Extensions from Source
# ==============================================================================
FROM nvidia/cuda:12.4.1-devel-ubuntu22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
# Restrict architectures to reduce compile time (sm_89 = L4 / RTX 6000 Ada)
ENV TORCH_CUDA_ARCH_LIST="8.9"
ENV MAX_JOBS=16

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-pip python3-dev git build-essential ninja-build ccache && \
    rm -rf /var/lib/apt/lists/*

RUN pip3 install --no-cache-dir -U pip setuptools wheel torch==2.4.0 --index-url https://download.pytorch.org/whl/cu124

# Clone PR #57250 branch
WORKDIR /workspace
RUN git clone --depth 1 -b structured-reads-main https://github.com/mmastrac/vllm.git /workspace/vllm

# Build custom vLLM wheel with matching C++ extensions
WORKDIR /workspace/vllm
RUN python3 setup.py bdist_wheel --dist-dir /workspace/dist

# ==============================================================================
# Stage 2: Lean Production Runtime Image
# ==============================================================================
FROM nvidia/cuda:12.4.1-runtime-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PORT=8080
ENV HOST=0.0.0.0
ENV PYTHONUNBUFFERED=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-pip python3 ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

RUN pip3 install --no-cache-dir -U pip torch==2.4.0 --index-url https://download.pytorch.org/whl/cu124

# Install compiled wheel from builder stage
COPY --from=builder /workspace/dist/*.whl /tmp/
RUN pip3 install --no-cache-dir /tmp/*.whl && rm -rf /tmp/*.whl

EXPOSE 8080
ENTRYPOINT ["vllm", "serve"]
CMD ["nvidia/diffusiongemma-26B-A4B-it-NVFP4", \
     "--diffusion-config", "{\"canvas_length\": 32}", \
     "--max-logprobs", "32", \
     "--enable-prefix-caching", \
     "--port", "8080", \
     "--host", "0.0.0.0"]
```

### Cloud Build Submission
Building CUDA extensions requires significant CPU and memory:
```bash
gcloud builds submit deploy/cloudrun \
  --project "$GCP_PROJECT" \
  --tag "gcr.io/${GCP_PROJECT}/vllm-diffgemma:compiled" \
  --machine-type=e2-highcpu-32 \
  --disk-size=200GB \
  --timeout=7200s
```

---

## 6. The GCE Alternative for Experimental Research

For evaluating experimental forks, **Google Compute Engine (GCE)** provides a much faster and more flexible iteration cycle than Cloud Run:

1. **Native Compilation on Host GPU**: Run `pip install -e .` on a `g2-standard-8` instance with preinstalled CUDA 12.9 drivers (`deeplearning-platform-release`).
2. **Persistent Disk Caching**: Model weights and wheels persist on a persistent SSD disk (`pd-ssd`), eliminating cold-start download cycles.
3. **No Container Kill Switches**: The VM runs without arbitrary 4–8 minute health-check timeouts.
4. **Automation in this Repo**:
   - Deploy: `make gce-deploy`
   - Teardown: `make gce-teardown`
