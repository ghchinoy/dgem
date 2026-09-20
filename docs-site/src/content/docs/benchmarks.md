---
title: Benchmark Evaluation Report
description: Empirical benchmark metrics comparing Discrete Diffusion Slot Readout against Generative Autoregression across Apple Silicon Metal, GCE L4 (4-bit), and GCE Dual-A100 (16-bit bfloat16).
---

This report presents empirical benchmark metrics comparing **Discrete Diffusion Slot Readout** against traditional **Prompt-Mediated Generative Autoregression**, evaluated across:
1. **Local Apple Silicon Metal (M5, 32 GB)** — 4-bit native Metal kernels (`diffgemma-26b-a4b-it-q4`)
2. **Google Compute Engine (`g2-standard-8`, 1× NVIDIA L4 24 GB)** — 4-bit ModelOpt Marlin (`nvidia/diffusiongemma-26B-A4B-it-NVFP4`)
3. **Google Compute Engine (`a2-highgpu-2g`, 2× NVIDIA A100-40GB, `TP=2`)** — **16-bit Unquantized `bfloat16`** (`google/diffusiongemma-26B-A4B-it`)

---

## 1. Executive Summary

| Evaluation Metric | Target 1: Local Apple M5 Metal (`q4`) | Target 2: Cloud GCE 1× L4 (`NVFP4` 4-bit) | Target 3: Cloud GCE 2× A100 (`bfloat16` 16-bit Unquant) | Target 4: Generative Autoregression Baseline |
| :--- | :--- | :--- | :--- | :--- |
| **Model Evaluated** | `diffgemma-26b-a4b-it-q4` | `nvidia/diffusiongemma-26B-A4B-it-NVFP4` | `google/diffusiongemma-26B-A4B-it` | `diffgemma-26b-a4b-it-q4` (`think=false`) |
| **Weight Memory Footprint** | ~18.84 GiB (Unified RAM) | ~18.15 GiB VRAM | **50.14 GiB VRAM** (25.07 GiB / GPU) | ~18.84 GiB |
| **End-to-End Wall Latency** | **~1.83 s** (1 read) / **4.14 s** (multi-avg) | **1,968.7 ms** (min: **776 ms**) | **2,733.8 ms** (min: **1,002 ms**) | **17,486.6 ms** (~17.5 s) |
| **Speedup vs. Generative** | **4.2× – 9.7× faster** | **~8.9× faster** | **~6.4× faster** | Baseline (1.0×) |
| **Speedup vs. Local Metal** | Baseline (1.0×) | **~2.1× faster** | **~1.5× faster** | — |
| **Syntactic Reliability** | **100% Schema-Guaranteed** | **100% Schema-Guaranteed** | **100% Schema-Guaranteed** | **0% raw pass** (wrapped in markdown) |
| **Classification Accuracy** | **80.0%** (24 / 30 cases) | **73.3%** (22 / 30 cases) | **80.0%** (24 / 30 cases) | ~75 – 80% |

---

## 2. Experimental Design, Dataset & Sample Cases

The evaluation suite (`./bin/dgem bench`) tests structured multi-question decision accuracy and latency across a standardized 30-case dataset (`benchmarks/eval_dataset.jsonl`).

### 2.1 Multi-Domain Evaluation Corpus (30 Cases)
Each test item injects structured input variables into a domain-specific Go template and simultaneously evaluates **three distinct question types** (`boolean`, `choice`, and `score`) in a single request:

1. **Customer Support Triage (10 cases — `templates/support_triage.json.tmpl`)**:
   * `urgent` (`boolean`): Does this issue require immediate same-day escalation? (`yes` / `no`)
   * `team` (`choice`): Which department owns resolution? (`billing`, `support`, `engineering`)
   * `sentiment` (`score`): Customer distress level (`calm`, `frustrated`, `furious`)
2. **Code Review & PR Triage (10 cases — `templates/code_review.json.tmpl`)**:
   * `approved` (`boolean`): Is this change safe to merge without security or stability regressions? (`yes` / `no`)
   * `category` (`choice`): Primary classification of the change (`bugfix`, `feature`, `refactor`, `security`)
   * `risk` (`score`): Operational risk level (`low`, `medium`, `high`)
3. **SecOps Alert Containment (10 cases — `templates/security_incident.json.tmpl`)**:
   * `data_compromise` (`boolean`): Does this alert indicate active data exfiltration or credential compromise? (`yes` / `no`)
   * `action` (`choice`): Immediate automated response (`monitor`, `revoke_key`, `isolate_host`, `page_oncall`)
   * `severity` (`score`): Incident severity (`informational`, `elevated`, `critical`)

### 2.2 Difficulty Tiers & Representative Samples
To stress-test bidirectional attention and uncertainty calibration, the 30 cases are stratified across four difficulty tiers:

| Tier | Share | Purpose | Verbatim Sample from `eval_dataset.jsonl` | Ground Truth |
| :--- | :--- | :--- | :--- | :--- |
| **Unambiguous** | 60% (18) | Baseline speed & single-pass certainty | **`sup-01`**: *"EMERGENCY: Production API gateway returning 500 internal server error across all US-East nodes. Customer traffic failing."* | `team: engineering`<br>`urgent: yes` |
| **Ambiguous** | 20% (6) | Overlapping intents requiring multi-sample noise averaging | **`sup-04`**: *"YOUR PLATFORM IS DOWN AGAIN AND MY CLIENTS ARE FURIOUS! CANCEL MY SUBSCRIPTION AND REFUND EVERYTHING RIGHT NOW!"* | `team: billing`<br>`urgent: yes` |
| **Negation** | 15% (5) | Catching explicit negative constraints (`NOT`, `do not`) | **`sup-05`**: *"This is NOT an outage or a billing problem. I just want to know if you support exporting reports to CSV or Excel."* | `team: support`<br>`urgent: no` |
| **Complex** | 5% (1) | Architectural & multi-system reasoning | **`code-08`**: *"Rewrite core Raft consensus state machine to improve throughput under heavy network partitions."* | `category: refactor`<br>`approved: yes` |

---

## 3. Cloud GPU Findings: 16-Bit Unquantized `bfloat16` (GCE `a2-highgpu-2g`, 2× NVIDIA A100-40GB)

Evaluated live on Google Compute Engine (`us-central1-b`) in project `genai-blackbelt-fishfooding` running Google DeepMind's official unquantized 16-bit weights (`google/diffusiongemma-26B-A4B-it`, 50.14 GiB across 2× A100 GPUs via NCCL Tensor Parallelism `TP=2`, with 10.62 GiB KV cache per GPU, `benchmarks/results_gce_a100_bf16.json`):

```text
================================================================================
  DIFFUSIONGEMMA: DISCRETE DIFFUSION BENCHMARK EVALUATION
================================================================================
Target Server: http://127.0.0.1:8080/v1
Model:         google/diffusiongemma-26B-A4B-it
Mode:          slot
Test Cases:    30 items

• Slot Readout Accuracy:        80.0% (24 of 30 matched expected)
• Average End-to-End Wall Time: 2,733.8 ms (~2.73s per decision)
• Fastest Case:                 1,002 ms (code-05)
• Prefix Cache Hit Rate:        73.4%
• Speedup vs Generative:        ~6.4× faster wall time
```

### Precision vs. Latency Analysis (`bfloat16` vs. `NVFP4` vs. Metal `q4`):
1. **Full Precision Restores Classification Accuracy (80.0% vs. 73.3%)**:
   * Under 4-bit weight-only Marlin compression (`NVFP4` on L4), two borderline code review cases (`code-04` and `code-09`) suffered quantization drift, dropping accuracy from 80.0% to **73.3% (22/30)**.
   * Running full **16-bit unquantized `bfloat16`** on 2× A100 GPUs resolved both `code-04` (`2,660 ms | PASS`) and `code-09` (`2,363 ms | PASS`), restoring the full **80.0% (24/30)** accuracy ceiling!
2. **Memory Bandwidth & Tensor Parallelism**:
   * Even while moving **50.14 GiB of unquantized `bfloat16` weights** across 2× A100 GPUs (`TP=2` with NCCL all-reduce), the dual-A100 setup achieved an average end-to-end latency of **2,733.8 ms**—**1.5× faster** than the 4-bit model on Apple M5 Metal (`4,143.3 ms`) and **6.4× faster** than generative autoregression (`17,486.6 ms`).
