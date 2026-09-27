---
title: Setup & Metal Engine Guide
description: Step-by-step instructions for running DiffusionGemma locally on Apple Silicon Metal.
---

# DiffusionGemma on Apple Silicon: Setup, Architecture, and Operational Guide

This guide covers setting up, running, and querying **DiffusionGemma 26B-A4B** locally on Apple Silicon (tested on Apple M5 with 32 GB unified memory) using **`diffgemma`** (a native Rust + Metal inference engine) and **`dgem`** (a Go-based CLI assistant for structured decisions and generative queries).

---

## 1. Overview & Architectural Concepts

**DiffusionGemma** (developed by Google DeepMind) is a 26B-parameter Mixture-of-Experts (MoE) foundation model with 3.8B active parameters (8 active experts out of 128 total + 1 shared expert).

Unlike traditional autoregressive language models that generate text token-by-token from left to right, DiffusionGemma utilizes **discrete block diffusion**:
* **Canvas-based generation**: It operates on a 256-token canvas with bidirectional cross-attention.
* **Parallel denoising**: It iteratively denoises blocks of tokens in parallel, generating 15–20 tokens per forward pass.
* **Fast structured decisions ("System-1" Judgment)**: By seeding a canvas with a predefined JSON template and leaving answer slots as noise, the model can evaluate classification, categorization, and scale choices in **a single forward pass (~210 ms on Apple Silicon Metal)** without generating conversational filler.

### Moving Beyond "Noul": The Boolean Decision

In early Jev publications (by TypeSafe AI) and experimental implementations, binary decisions were labeled with the neologism `"noul"`. In practical computing and schema design, this is simply a **boolean** (or binary predicate decision):
* You provide a question or proposition (e.g. *"Does this ticket require immediate escalation?"*).
* The model evaluates whether the proposition holds (`yes` vs. `no`, `true` vs. `false`), providing calibrated confidence, entropy, and error bars.
* The engine natively accepts `"boolean"` and `"bool"` interchangeably with `"noul"`.

---

## 2. Hardware & Memory Tuning for 32 GB Macs

DiffusionGemma's full-precision BF16 checkpoint is ~50 GB, which cannot fit into 32 GB of RAM. However, the 4-bit quantized pack (`mmastrac/diffgemma-26b-a4b-it-q4`) compresses the MoE experts into 4-bit affine blocks while keeping precision-sensitive attention, norms, and embeddings in 16-bit, resulting in an **18.84 GiB** resident footprint.

### Unified Memory Budgeting

By default, macOS allocates a maximum of ~66% of unified memory to the GPU. On a 32 GB machine, this limit is ~21.5 GB—leaving little headroom for the KV cache or other running processes.

To allocate up to 28 GB (87%) of memory to the Metal GPU, run:

```bash
# Set Metal alloc limit to 28 GB (requires sudo):
sudo sysctl iogpu.wired_mem_limit=28672
```

To make this persistent across reboots, add the setting to `/etc/sysctl.conf`.

---

## 3. Prerequisites

* **macOS 14.0+ (Sonoma or Sequoia)** running on Apple Silicon (M1/M2/M3/M4/M5).
* **32 GB unified memory minimum** for the 4-bit quantization pack (64 GB+ recommended for running unquantized weights or large KV caches).
* **Xcode Command Line Tools**:
  ```bash
  xcode-select --install
  ```
* **Rust toolchain (1.85+)**:
  ```bash
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  source "$HOME/.cargo/env"
  ```
* **Go toolchain (1.23+)** (for the `dgem` CLI tool):
  ```bash
  brew install go
  ```

---

## 4. Automated Setup via Makefile

The repository includes a complete automated setup pipeline:

```bash
# 1. Run the prerequisite check and install the diffgemma binary:
make setup

# 2. Download the 4-bit model checkpoint (~18.8 GB from Hugging Face):
make download

# 3. Compile the dgem Go CLI tool:
make build
```

---

## 5. Step-by-Step Manual Setup

If you prefer to configure components manually:

### Step 1: Install the `diffgemma` Binary
Install the official pre-compiled engine or compile from the upstream repository:

```bash
cargo install --git https://github.com/mmastrac/diffgemma.git --tag v0.1.0 diffgemma
```

Verify installation:
```bash
diffgemma --version
```

### Step 2: Download Model Weights
Run the built-in downloader to fetch the canonical 4-bit quantization pack from Hugging Face (`mmastrac/diffgemma-26b-a4b-it-q4`). Use `--jobs 8` to enable parallel chunk downloads:

```bash
# In your project directory (or 'make download'):
mkdir -p model/diffgemma-26b-a4b-it-q4
huggingface-cli download mmastrac/diffgemma-26b-a4b-it-q4 \
  --local-dir model/diffgemma-26b-a4b-it-q4 \
  --local-dir-use-symlinks False
```

### Step 3: Launch the Engine Daemon
Start the server in background with a 32k context window on port 8080:

```bash
# Using the repository script:
make serve
# Or: ./scripts/serve.sh
```

Or run directly:
```bash
diffgemma serve \
  -m model/diffgemma-26b-a4b-it-q4 \
  --ctx 32768 \
  --addr 127.0.0.1:8080 > server.log 2>&1 &
```

### Verifying Server Health
```bash
curl -s http://127.0.0.1:8080/v1/models | jq .
```
Expected response:
```json
{
  "data": [
    {"id": "diffgemma-26b-a4b-it-q4", "object": "model", "owned_by": "local"},
    {"id": "diffgemma-26b-a4b-it-q4:think", "object": "model", "owned_by": "local"},
    {"id": "diffgemma-26b-a4b-it-q4:think=false", "object": "model", "owned_by": "local"}
  ],
  "object": "list"
}
```

> **Cloud Deployment**: If you want to host DiffusionGemma on Google Cloud Run with an NVIDIA RTX Pro 6000 or L4 GPU instead of running locally, see the [Remote Endpoints & Cloud Deployment Guide](/dgem/remote-endpoints/) or run `make cloudrun-deploy`.

---

## 6. Discrete Diffusion Slot Readout (Single-Pass Decisions)

A request containing a JSON question schema in the `system` role automatically triggers the discrete diffusion slot-readout pathway (informally termed "Jev-style" in early 2026 community benchmarks). The user message provides the target state or text.

### Supported Question Types

1. **`boolean` (or `bool` / `noul`)**:
   Binary proposition (`yes` / `no`). Returns `probabilities.yes`, `probabilities.no`, and confidence.
2. **`choice`**:
   Categorical selection from a list of named options (up to 26 single-token labels `A`, `B`, `C`...).
3. **`score`**:
   Ordered scalar evaluation (e.g. `["calm", "frustrated", "furious"]`). Returns the expected numerical level and the top level.

### Example cURL Request

```bash
curl -s http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "{\"instructions\": \"Triage customer tickets\", \"questions\": [{\"id\": \"urgent\", \"type\": \"boolean\", \"instructions\": \"Immediate escalation required?\"}, {\"id\": \"team\", \"type\": \"choice\", \"instructions\": \"Owning team\", \"options\": [{\"name\": \"billing\"}, {\"name\": \"support\"}, {\"name\": \"engineering\"}]}, {\"id\": \"sentiment\", \"type\": \"score\", \"instructions\": \"Customer distress\", \"levels\": [\"calm\", \"frustrated\", \"furious\"]}]}"
      },
      {
        "role": "user",
        "content": "{\"ticket\": \"Our production database is returning 500 across all cluster nodes!\"}"
      }
    ]
  }' | jq .
```

### Understanding the Response Diagnostics
The response includes `answers` and a comprehensive `diagnostics` object:
* **`confidence`**: Mean probability of the winning choice.
* **`stderr`**: Standard error computed across noise draws.
* **`agreement`**: Proportion of noise draws that agreed on the winning label.
* **`timing.reused_tokens`**: Indicates that the system prompt schema was reused directly from the KV cache (zero-recomputation prefill).
* **`samples.policy`**: When set to `"auto"`, the engine stops at 1 read if entropy is low (<0.10 nats), or automatically gathers up to 4 reads if ambiguity is detected.

---

## 7. Connecting to Opencode

To use your local `diffgemma` model directly within `opencode`:

```bash
OPENCODE_CONFIG_CONTENT='{
  "provider": {
    "diffgemma": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "diffgemma (local)",
      "options": { "baseURL": "http://127.0.0.1:8080/v1", "apiKey": "unused" },
      "models": { "diffgemma-26b-a4b-it-q4": { "name": "DiffGemma 26B-A4B q4" } }
    }
  }
}' opencode -m diffgemma/diffgemma-26b-a4b-it-q4
```

---

## 8. The `dgem` CLI Tool

`dgem` is the dedicated Go CLI for managing templates, running structured reads, and inspecting execution telemetry.

### Build and Install
```bash
make build
```

### Basic Usage
```bash
# Run a structured decision with detailed stats
./bin/dgem decide -t templates/support_triage.json.tmpl -v ticket="System is completely down" --stats

# Run a generative prompt
./bin/dgem ask "Explain the difference between autoregression and diffusion." --stats

# Render a template without executing
./bin/dgem template render -t templates/code_review.json.tmpl -v diff="added auth middleware"

# Run the comparative benchmark
./bin/dgem bench
# Or: make bench

# Deploy to Google Cloud Run with GPU
make cloudrun-deploy
```
