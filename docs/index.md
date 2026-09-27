---
title: "Documentation Overview & Index"
description: "Welcome to the DiffusionGemma (dgem) documentation: quickstarts, architecture, experiments, cloud deployment, and API references."
---

# DiffusionGemma (`dgem`) Documentation

**`dgem`** turns Google's **DiffusionGemma-26B** into an ultra-fast, zero-shot **Discrete Decision Model**. It evaluates multi-slot decision policies simultaneously in $O(1)$ forward passes (~57.5 ms on Vertex G4, ~210 ms on Apple Silicon Metal), with exact restricted-softmax probabilities, calibrated Shannon entropy ($H$), and Invariant Decision Calibration (IDC).

---

## 🚀 Getting Started

* **[5-Minute Quickstart](quickstart.md)** — Launch on Apple Silicon Metal, Docker with NVIDIA GPU, or cloud endpoints in minutes.
* **[Setup & Local Metal Engine](setup.md)** — Run on Apple Silicon Macs using native unified memory ($0/hr).
* **[dgem User Guide](user-guide.md)** — Comprehensive CLI reference, syntax rules, and automation examples.
* **[Decision Studio, MCP & HTTP API](studio-mcp-api.md)** — Interactive Lit WebComponents playground, Streamable HTTP MCP server, and REST gateway.
* **[CLI, HTTP Gateway & MCP Reference](cli-reference.md)** — Exhaustive flags, environment variables, and tool parameters reference.

---

## 🏛️ Core Architecture & Calibration

* **[The Journey to Decision Models](decision-models-primer.md)** — Why discrete diffusion models solve classification, triage, and routing better than autoregressive LLMs.
* **[Confidence Beyond Shannon: Invariant Decision Calibration (IDC)](confidence-beyond-shannon.md)** — Eliminating ballot-order biases ($p_0$) and measuring true epistemic uncertainty.
* **[Discrete Diffusion vs. Autoregression](architecture.md)** — Deep technical dive into bidirectional attention canvases, masked single-step denoise, and zero autoregressive overhead.
* **[Unclassified Grouping & Zero-Retraining Taxonomy Discovery](taxonomy-discovery.md)** — Automatically discovering missing classes via the `"think"` channel without fine-tuning.
* **[Glossary & Mental Models](glossary.md)** — Plain-English definitions for every technical, mathematical, and serving term.
* **[Template Catalog (Policy-as-Code)](templates.md)** — Reference guide for all 26+ built-in `.json.tmpl` executable decision policies.

---

## ☁️ Deployment & Cloud Serving

* **[Public Container Images & Quickstart](public-image.md)** — Official zero-auth container images on Google Artifact Registry (`dgem` and `dgem-weights`).
* **[Deploy on Your Own Cloud GPU](deploy-your-own-gpu.md)** — Step-by-step instructions for deploying to Serverless Cloud Run GPU and Vertex AI Dedicated Endpoints.
* **[Connecting dgem to Remote Endpoints](remote-endpoints.md)** — Authentication, `.dgem.yaml` configuration, and endpoint routing.
* **[Path to Production](path-to-production.md)** — Hardware tiers, concurrency scaling, and cost analysis.
* **[Vertex AI Dedicated Endpoints vs. Cloud Run GPU](vertex-ai-vs-cloudrun.md)** — Architectural comparison between persistent G4 Dedicated Endpoints and serverless scale-to-zero.
* **[Cloud Run Lessons Learned](cloudrun-lessons-learned.md)** — Historical post-mortem on containerization, CUDA ABI, and GCS FUSE weight streaming.
* **[OpenTelemetry Traces & Observability](observability-traces.md)** — Distributed tracing, Google Cloud Trace integration, and Studio Gantt waterfalls.

---

## 🧪 Experiments & Research Ledger

* **[Experiment Ledger (EXP-01 – EXP-17)](experiments/README.md)** — Full registry of formal empirical experiments and receipts.
* **[Proposed Experiments Register](experiments/proposed.md)** — Backlog of proposed research directions and architectures.
* **[Listwise Diffusion Reranking (EXP-10)](experiments/exp-10-listwise-diffusion-reranking.md)** — 300-passage listwise ranking and prompt-injection quarantine.
* **[JevBench Parity & Scoring (EXP-11)](experiments/exp-11-jevbench-parity.md)** — 4-axis benchmark parity across 231 complex reasoning cases.
* **[Permutation Invariance & Dual-Mirror Canvas (EXP-13)](experiments/exp-13-permutation-invariance.md)** — Canceling first-order positional bias in a single forward pass.
