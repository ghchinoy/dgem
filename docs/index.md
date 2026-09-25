---
title: DiffusionGemma — Zero-Shot Decision Model
description: Declarative Policy-as-Template engine, Lit WebComponents Decision Studio, Model Context Protocol (MCP) Server, HTTP Gateway API, and sub-second discrete diffusion slot readouts across Vertex AI Dedicated Endpoints (/invoke/*), Serverless Cloud Run GPU, Google Compute Engine, and Apple Silicon Metal.
---

# `dgem` — DiffusionGemma as a Zero-Shot Decision Model

**`dgem`** is a declarative **Policy-as-Template** engine, **Lit WebComponents Decision Studio (`dgem serve`)**, **Model Context Protocol (`MCP`) Server (`dgem mcp` & `/mcp`)**, **HTTP Gateway REST API (`/api/decide` & `/v1/systemone`)**, and **empirical benchmark harness** for Google DeepMind's **DiffusionGemma (`26B-A4B-it`)**.

**New here?** Start with **[The Journey to Decision Models](decision-models-primer.md)** (what a decision model is and why it can report a per-field uncertainty score), then **[Confidence Beyond Shannon (IDC)](confidence-beyond-shannon.md)** (why that score can be fooled by option order, and how `dgem` checks it).

---

## Multi-Environment Serving Matrix (4 Primary Serving Targets)

See **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU](vertex-ai-vs-cloudrun.md)** and the **[Experiment Authoring Guide](experiment-authoring-guide.md)** for complete architectural details and live 30-case benchmark receipts:

| Serving Target | Engine & Hardware Shape | Cold-Start / Wakeup | Verified GPU Denoise / Wall Time | Primary Use Case |
| :--- | :--- | :--- | :--- | :--- |
| **1. Vertex AI Dedicated Endpoint (`/invoke/*`)** | `structured_server.py` + vLLM (`TRITON_ATTN`) on **`g2-standard-16` (`1× NVIDIA L4` `24GB` VRAM, `64GB` RAM, ID `4217256562927861760`)** | **`0.0 s`** (`minReplicaCount=1`, permanently warm) | **`490.0 ms` avg GPU** (`536.0 ms` wall for `N=4`; `195 ms` for `N=1`) | **Recommended Primary Target (`vertex_first` default)**: Zero cold-start SLA, interactive agents, synchronous CI/CD gates, and multimodal `SigLIP` headroom (`64 GB` RAM). |
| **2. Serverless Cloud Run GPU (`dgemma`)** | `structured_server.py` + vLLM on **`1× NVIDIA RTX Pro 6000` (`48GB` VRAM, `80Gi` RAM)** or **`1× L4` (`24GB`)** | **`~121.8 s`** (`0 → 1` scale-from-zero, `$0.00/hr` idle) | **`427.3 ms` avg GPU** (`459.0 ms` warm wall) | **Scale-to-Zero & Auto-Failover Standby (`cloudrun`)**: Episodic batch evaluations, research experiments, and `$0.00/hr` idle sandboxes. |
| **3. Google Compute Engine VM (`1× L4` / `2× A100`)** | Raw vLLM + Triton Attention (`TRITON_ATTN`) on `g2-standard-8` (`1× L4` `NVFP4`) or `a2-highgpu-2g` (`2× A100` `bfloat16`) | **`0.0 s`** (dedicated VM) | **`~1,968.7 ms`** (`L4`) / **`~2,733 ms`** (`2× A100`) | High-throughput continuous batching (`Banking77` / `CLINC150`) and 16-bit unquantized `bfloat16` baselines. |
| **4. Local Apple Silicon (`Metal`)** | Native Rust Metal (`diffgemma`) with `diffgemma-26b-a4b-it-q4` (`~18.8 GB` Unified RAM) | **`0.0 s`** (local daemon) | **`892.0 ms` avg GPU** (`898.5 ms` wall; `210 ms` for `N=1`) | Local development, offline privacy, `$0.00/hr` cloud cost. |

---

## Documentation Navigation

* **[Decision Studio Web App, MCP Server & HTTP Gateway API](studio-mcp-api.md)**
* **[Experiment Authoring Guide & Backend Target Selection (`vertex_first` vs. `vertex` vs. `cloudrun`)](experiment-authoring-guide.md)**
* **[CLI, HTTP Gateway & MCP Reference (`--vertex-url`, `dgem serve`, `/v1/systemone`)](cli-reference.md)**
* **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU](vertex-ai-vs-cloudrun.md)**
* **[Custom Dataset & Experiment Cookbook](custom-dataset-guide.md)**
* **[The Journey to Decision Models](decision-models-primer.md)**
* **[Confidence Beyond Shannon: Invariant Decision Calibration (`IDC`)](confidence-beyond-shannon.md)**
* **[Benchmark Evaluation Report](benchmarks-report.md)**
* **[Experiments & Research Ledger (`EXP-01` – `EXP-14`)](experiments/README.md)**
