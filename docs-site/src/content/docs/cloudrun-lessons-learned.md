---
title: Cloud Run Lessons Learned & Self-Contained Deployment Guide
description: Architectural comparison of experimental vLLM deployments on Google Cloud Run, comparing taeold/djev-run against dgem and providing a 100% self-contained Artifact Registry build pipeline.
---

# Lessons Learned: Deploying Experimental vLLM on Google Cloud Run with GPU

This document records the empirical findings, architectural trade-offs, and operational lessons learned while deploying Google DeepMind's **DiffusionGemma** on **Google Cloud Run with GPUs** using an experimental vLLM discrete block diffusion branch (PR #57250), comparing against Taehoon Lee's [**`taeold/djev-run`**](https://github.com/taeold/djev-run) and providing a 100% self-contained, reproducible container build and deployment pipeline.

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
  2. For production, never download weights over the public internet on container boot: pre-stage model weights in a **Google Cloud Storage (GCS) bucket** and mount it via Cloud Run volume mounts (GCS FUSE). This drops cold-start latency from minutes to under 20 seconds.

---

## 4. Lesson: Cloud Run Startup Probe Configuration

* **Initial Anti-Pattern**:
  ```bash
  --startup-probe tcpSocket.port=8080,initialDelaySeconds=240,failureThreshold=1,timeoutSeconds=240,periodSeconds=240
  ```
  This configuration forces Cloud Run to wait 4 full minutes before performing the first probe. If the container finishes in 60 seconds, it still sits idle. Worse, with `failureThreshold=1`, a single failed ping immediately terminates the container.
* **Best-Practice Pattern**:
  ```bash
  --startup-probe httpGet.path=/health,httpGet.port=8080,initialDelaySeconds=5,periodSeconds=2,timeoutSeconds=2,failureThreshold=120
  ```
  This begins probing after 5 seconds and polls every 2 seconds. The moment the server binds to port 8080, the instance is marked healthy immediately. With 120 retries, it provides a 4-minute readiness window for weight downloads and memory profiling.

---

## 5. The Proven Production Pattern: Learning from `taeold/djev-run` & `mmastrac/djev-spark`

A working, hardened deployment pattern for DiffusionGemma on Google Cloud Run with GPUs was proved by Google Cloud engineer Taehoon Lee in [**`taeold/djev-run`**](https://github.com/taeold/djev-run), building on Matt Mastracci's [**`mmastrac/djev-spark`**](https://github.com/mmastrac/djev-spark).

### The Container Lineage & Why Our Initial Overlay Failed
Tracing the source code across repositories reveals why our initial attempt diverged while `djev-spark` succeeded:

```
vllm-project/vllm (upstream)
      │  Pinned commit: dee37d89115db4c94a820a79a78a7828e141c910
      ▼
vllm/vllm-openai:nightly-dee37d89115db4c94a820a79a78a7828e141c910
      │  C++ CUDA extensions compiled matching Python interfaces
      ▼
mmastrac/djev-spark (Dockerfile)
      │  Clones mmastrac/vllm at pinned fork commit: 6591b093b
      │  patches/link_cuda_headers.sh (symlinks nvrtc.h for CUTLASS JIT)
      │  patches/overlay_vllm.py (verifies exact merge-base before copying)
      │  patches/raise_recompile_limit.py (sets Dynamo recompile limit to 64)
      ▼
taeold/djev-run (Cloud Run adaptation)
      │  patch_vllm.py (bypasses profile_run and kernel_warmup under eager mode)
      │  GCS FUSE volume mount (/mnt/gcs/dgemma) with enable-buffered-read=true
      │  Background weight staging into /dev/shm (on 80GB RAM instances)
      │  Cloud Run /health probe handler in structured_server.py
      ▼
dgem (100% Self-Contained in this Repository)
      • deploy/cloudrun/Dockerfile (fully reproducible multi-stage build from source)
      • scripts/build_cloudrun_image.sh (pushes to your own Google Artifact Registry)
      • scripts/stage_model_gcs.sh (one-command GCS weight staging)
      • scripts/deploy_cloudrun_vllm.sh (adaptive sizing for L4 and RTX Pro 6000)
```

---

## 6. GPU Selection on Cloud Run: NVIDIA L4 vs. RTX Pro 6000

| Dimension | Cost Tier: **NVIDIA L4 (24 GB)** | Performance Tier: **NVIDIA RTX Pro 6000 (48 GB)** |
| :--- | :--- | :--- |
| **GPU Architecture** | Ada Lovelace (24 GB VRAM) | Blackwell / Ada Workstation (48 GB VRAM) |
| **Cloud Run Allocation** | **8 vCPUs, 32 GB RAM** | **20 vCPUs, 80 GB RAM** |
| **Hourly Rate** | **~$0.70 / hour** | **~$2.20 / hour** |
| **Model Weight Staging** | **Direct GCS FUSE buffered streaming** (`COPY_TO_SHM=0`) | **Parallel `/dev/shm` RAM disk staging** (`COPY_TO_SHM=1`) |
| **Why the RAM Difference Matters** | Staging 17.5 GB into `/dev/shm` on a 32 GB RAM instance leaves only 14.5 GB for Python, risking OOM. Direct streaming is safe and fast. | With 80 GB RAM, 16 parallel threads copy weights from GCS into `/dev/shm` in background, leaving >60 GB for Python/vLLM. |
| **Canvas & Model Length** | `--canvas 32 --max-model-len 32768` | `--canvas 128 --max-model-len 4096` |
| **Regional Quota** | Standard regional Cloud Run GPU quota (`us-central1`, `europe-west4`, etc.) | Requires `--no-gpu-zonal-redundancy` flag |

---

## 7. How to Build Your Own Patched Image and Deploy (100% Self-Contained)

This repository includes a completely self-contained build and deployment pipeline:

```bash
# 1. Build Container in Your Own Google Artifact Registry
export GCP_PROJECT="your-gcp-project"
export GCP_REGION="us-central1"
make cloudrun-build

# 2. Pre-Stage Weights in Your Regional GCS Bucket
make cloudrun-stage

# 3. Deploy to Cloud Run on NVIDIA L4:
make cloudrun-deploy

# Or deploy on high-performance RTX Pro 6000:
CLOUDRUN_GPU_TYPE="nvidia-rtx-pro-6000" make cloudrun-deploy

# 4. Query & Benchmark via dgem
SERVICE_URL=$(gcloud run services describe dgemma --region=us-central1 --format="value(status.url)")
./bin/dgem decide -u "${SERVICE_URL}/v1" --gcp-auth -t templates/support_triage.json.tmpl -v 'ticket=Emergency' --stats
./bin/dgem bench -u "${SERVICE_URL}/v1" --gcp-auth -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_cloudrun.json
```
