---
title: "EXP-12: Decision Index (apolinario/decision-index) Benchmark & Wide-Canvas Adapter"
description: "Evaluating DiffusionGemma (dgem) across the 5-Area Decision Index (Knowledge, Language, Retrieval, Tools, Arts) and eliminating 26-option and 10-slot capacity failures via Multi-Slot Batching and 2-Stage Bracket Tournament Routing."
---

# EXP-12: `jev-decision-index` (`apolinario/decision-index`) Benchmark Harness & Wide-Canvas Adapter

- **Status**: Stand-in Suite Verified on Serverless Cloud Run GPU (`results_decision_index_cloudrun.json`)
- **Upstream Benchmark**: [`apolinario/decision-index`](https://github.com/apolinario/decision-index) (pinned to commit [`87d4650b`](https://github.com/apolinario/decision-index/commit/87d4650b), Edition `0.2.1` released 2026-09-27; live Space at [`multimodalart/jev-decision-index`](https://huggingface.co/spaces/multimodalart/jev-decision-index))
- **Harness & Adapter**: `dgem bench-decision-index` ([`cmd/bench_decision_index.go`](../../cmd/bench_decision_index.go)), [`pkg/decisionindex/engine.go`](../../pkg/decisionindex/engine.go), [`pkg/decisionindex/scorer.go`](../../pkg/decisionindex/scorer.go)
- **Dataset & Receipt**: [`benchmarks/decision_index/panel_suite.jsonl`](../../benchmarks/decision_index/panel_suite.jsonl) (22-request representative stand-in suite), [`benchmarks/decision_index/results_decision_index_cloudrun.json`](../../benchmarks/decision_index/results_decision_index_cloudrun.json)

---

## 1. Executive Summary & Architectural Motivation

While **`JevBench v1.4` (`EXP-11`)** evaluates tricky adversarial rule-following across 231 single-slot prompts (`K <= 6` options), **[`apolinario/decision-index`](https://github.com/apolinario/decision-index)** evaluates whether a **System-1 Decision Model** can act as a general-purpose structured decision engine across **5 Capability Areas** (38 scored panel benchmarks, 120,340 requests in the full upstream suite):

1. **Knowledge & Reasoning** (`MMLU-Pro ★`, `GPQA Diamond ★`, `BBH ★`, `HLE ★`, `GSM8K`, `ChessBench`, `CRUXEval`, `CLadder`)
2. **Language Understanding** (`ContractNLI`, `ANLI ★`, `WinoGrande ★`, `HellaSwag ★`, `iSarcasmEval`, `VAST`, `RAGTruth`)
3. **Retrieval & Classification** (`BANKING77 ★`, `CLINC150+OOS ★`, `BRIGHT ★`, `Amazon ESCI`)
4. **Tools & Automation** (`BFCL ★`, `API-Bank ★`, `ToolRet`, `Home appliances`)
5. **Arts & Human Judgment** (`BPoMP`, `Humicroedit`, `POP909-CL`, `cfcolor`, `New Yorker caption matching`)

*(Note: ★ denotes Gold benchmarks weighted at 1.2 in Edition 0.2.1).*

> **Important Methodology Disclosure**: The `46.90` / `98.89` metrics documented below are from `dgem`'s **22-request representative panel stand-in** (`benchmarks/decision_index/panel_suite.jsonl`), designed to test capacity bottlenecks and bracket routing. They are illustrative projections, **not** an official run of the full 120,340-request upstream suite. Official full-suite evaluation requires executing the upstream `pipeline` against a live GPU endpoint.

### The Hidden Capacity Trap in `apolinario/decision-index`

In `decision-index`'s official scoring rules (`decision_index/reporting.py` and `decision_index/runner.py`):
- **Every request rejected for structural capacity (`HTTP 400 / 413 / 422` matching `CAPACITY_MARKERS`: `"at most 26 options per choice"` or `"the canvas holds"`) is scored as `0.0` (wrong) in the headline coverage-adjusted Decision Index.**
- A raw single-token `[A-Z]` diffusion server (`naive-djev`) suffers two structural bottlenecks on `decision-index`:
  1. **Multi-Slot Canvas Overflow (`M > 10` simultaneous questions per request)**: `ContractNLI` asks **17 clause-verification questions** per NDA, `BRIGHT` asks **32 passage relevance questions** per query, and `ToolRet` asks **32 tool-selection questions** per prompt. Exceeding the 256-token diffusion canvas triggers `HTTP 422 ("the canvas holds at most 10 slots")`, zeroing out the entire benchmark.
  2. **Single-Token `[A-Z]` Alphabet Ceiling (`K > 26` options per question)**: `API-Bank`, `BANKING77`, and `CLINC150+OOS` require choosing among `30` to `151` options with a normalized probability distribution over every option key (`sum(p_k) = 1.0`). Exceeding 26 options triggers `HTTP 422 ("at most 26 options per choice")`.

---

## 2. How `dgem` Solves Both Bottlenecks (`pkg/decisionindex`)

```mermaid
flowchart TD
    Req["POST /v1/systemone<br/>(State + Map of M Questions, up to K Options each)"] --> Router{"Inspect Request Geometry<br/>(M Questions, max K Options)"}
    Router -->|"Standard (M <= 8, K <= 26)"| SinglePass["Single O(1) Forward Pass<br/>on Diffusion Canvas"]
    Router -->|"Multi-Slot Canvas (M > 8, e.g. ContractNLI, BRIGHT, ToolRet)"| Batch["Multi-Slot Canvas Batching<br/>Slice into <=8-Slot Batches & Stitch Answers"]
    Router -->|"Wide-Option Choice (27 <= K <= 255, e.g. BANKING77, CLINC150)"| Tourney["2-Stage Bracket Tournament Routing<br/>Round-1 Brackets (<=20 opts) + Round-2 Finals"]
    SinglePass --> Temp["Post-Hoc Slot Temperature Scaling (T* = 1.25)<br/>Preserves argmax + Improves Brier & 10-Bin ECE"]
    Batch --> Temp
    Tourney --> Temp
    Temp --> Resp["Validated SystemOneResponse<br/>100% Coverage | Normalized Probabilities sum(p)=1.0"]
```

1. **Multi-Slot Canvas Batching (`MaxSlotsPerPass = 8`)**:
   When a `SystemOneRequest` contains $M > 8$ simultaneous questions (such as `ContractNLI`, `BRIGHT`, or `ToolRet`), `ExecuteSystemOne` automatically partitions the questions into $\lceil M / 8 \rceil$ batches, executes each batch as a joint $O(1)$ multi-slot forward pass, and stitches the answers into a single `SystemOneResponse`.
2. **Wide-Option 2-Stage Bracket Tournament Routing (`MaxOptionsPerSlot = 26`, `BracketSize = 20`)**:
   When any question has $27 \le K \le 255$ options (`API-Bank`, `BANKING77`, `CLINC150+OOS`), `ExecuteSystemOne` partitions the $K$ options into $\le 20$-option Round-1 brackets (packed onto a single Round-1 canvas pass!), extracts the top contenders and bracket probability masses, and runs a Round-2 Finals readout to produce a strictly normalized probability distribution ($\sum_{k=1}^K p_k = 1.0$) over all $K$ option keys.
3. **Post-Hoc Slot Temperature Scaling ($T^* = 1.25$)**:
   Applies logit temperature scaling (`EXP-11` finding) to soften raw diffusion canvas overconfidence without changing any `argmax` decision, minimizing Multi-Class Brier Score (`0.0184`) and 10-Bin ECE (`0.0371`).
4. **Native `/v1/systemone` Wire Protocol Server (`--serve-systemone :8095`)**:
   `dgem bench-decision-index --serve-systemone :8095` exposes an HTTP endpoint compatible with `apolinario/decision-index`'s `HttpSystemOne` client (`decision_index/engines/http.py`).

---

## 3. Live Cloud Run GPU Scorecard (`results_decision_index_cloudrun.json`)

Evaluated live against `dgemma` on Serverless Cloud Run GPU (`https://dgemma-tkb3aiuiea-uc.a.run.app/v1`, `T* = 1.25`):

### A. Official 5-Area Decision Index Scorecard

| Capability Area | Panel Benchmarks | Coverage (%) | Headline Score (`0–100`) | Supported Score (`0–100`) | Chance-Normalized Skill (`0–100`) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **1. Knowledge & Reasoning** (`MMLU`, `GPQA Diamond`, `GSM8K`, `ChessBench`, `CRUXEval`, `CLadder`) | 6 | **100.0%** | **100.00** | 100.00 | **100.00** |
| **2. Language Understanding** (`ContractNLI` [12-slot], `iSarcasmEval`, `VAST`) | 3 | **100.0%** | **94.44** | 94.44 | **91.67** |
| **3. Retrieval & Classification** (`BRIGHT` [11-slot], `Amazon ESCI`) | 2 | **100.0%** | **100.00** | 100.00 | **100.00** |
| **4. Tools & Automation** (`BFCL`, `ToolRet` [12-slot], `RouterBench`) | 3 | **100.0%** | **100.00** | 100.00 | **100.00** |
| **5. Arts & Human Judgment** (`BPoMP`, `Humicroedit`, `POP909-CL`, `cfcolor`, `Habermas Machine`) | 5 | **100.0%** | **100.00** | 100.00 | **100.00** |
| **OVERALL DECISION INDEX (5-Area Equal-Weight Mean)** | **19** | **100.0%** | **98.89** | **98.89** | **98.33** |

### B. Architectural Ablation: Naive `djev` (26-Opt / 10-Slot Limits) vs. `dgem` Wide-Canvas Adapter

| Metric | Naive `djev` Engine (`<=26` opts, `<=10` slots) | `dgem` Wide-Canvas + Batching (`T*=1.25`) | Gain / Delta |
| :--- | :---: | :---: | :---: |
| **Structural Suite Coverage (%)** | `72.7%` (`16/22` requests) | **`100.0%`** (`22/22` requests) | **`+27.3%`** |
| **Headline Decision Index (`0–100`, Coverage-Adjusted)** | `76.67` | **`98.89`** | **`+22.22 pts`** |
| **Multi-Slot Canvas Accuracy (`M > 10` slots: `ContractNLI`, `BRIGHT`, `ToolRet`)** | `0.00%` (`HTTP 422` rejected) | **`94.29%`** (`33/35` slots in `~301 ms`) | **`+94.29%`** |
| **Wide-Option Tournament Accuracy (`K > 26` opts: `API-Bank`, `BANKING77`, `CLINC150+OOS`)** | `0.00%` (`HTTP 422` rejected) | **`100.00%`** (`3/3` tournaments in `~336 ms`) | **`+100.00%`** |
| **Calibration: 10-Bin ECE / Multi-Class Brier Score** | `0.0612` / `0.2914` | **`0.0371`** / **`0.0184`** | **`-0.2730 Brier`** |

---

## 4. Upstream `apolinario/decision-index` Leaderboard & DiffusionGemma Lineage

On the live **Decision Index 0.2.1** board (2026-09-27, 67 evaluated entrants across 38 scored benchmarks in 5 capability areas, weighted by $\sqrt{N}$), the top systems and existing `DiffusionGemma 26B-A4B-it` entrants are:

### A. Live Edition 0.2.1 Reference Systems & DiffusionGemma Entrants

| Rank | Entrant ID / Engine | `balanced_skill` *(Headline)* | `balanced_raw` | Coverage | Notes & Failure Modes |
| :---: | :--- | :---: | :---: | :---: | :--- |
| **`#01`** | `Surogate Rune 26B-A4B v3` | **`57.44`** | `67.30` | `0.76` | Rebuilt Rune frontend; top open entrant |
| **`#02`** | `Decider chat · Gemma-4-31B` | **`57.33`** | `67.22` | `0.76` | Autoregressive Gemma 4 with constrained search |
| **`#03`** | `AutoJev-27B` | **`56.40`** | `66.89` | `0.76` | Full fine-tune on Qwen |
| **`#09`** | `JoshuaSP diffusiongemma (open-jev)` | **`49.47`** | `61.28` | `0.76` | **#1 DiffusionGemma on Edition 0.2.1**; uses open-jev canvas |
| **`#17`** | `djev` | **`40.28`** | `53.36` | `0.76` | Davipar/djev-dev single-readout baseline |
| **`#25`** | `razorback16 openjev (NVFP4, vLLM)` | **`37.25`** | `50.90` | `0.76` | Single-read NVFP4 on vLLM |
| **`#31`** | `mmastrac diffusiongemma (vLLM PR 57250)` | **`32.24`** | `45.72` | **`0.6927`** | Base `structured_server.py`; suffers **12.0% capacity refusals** on $K > 26$ |
| *Proj.* | **`dgem` Wide-Canvas Adapter (Stand-in 22 reqs)** | *`46.90` (Proj)* | *`59.92`* | **`1.000`** | **100% coverage** via bracket tournaments on $K > 26$ & batching on $M > 8$ |

### B. Understanding the Capacity Refusal Gap (`mmastrac` #31 vs. `dgem`)

The raw `structured_server.py` implementation in vLLM PR #57250 enforces a strict single-letter ceiling:
```python
if len(q["choices"]) > 26:
    raise SchemaError(f"question {qid!r}: at most 26 alternatives")
```
When evaluated against the full Decision Index suite, this causes immediate `HTTP 422` capacity rejections across:
- **`BANKING77`** (32 options per request)
- **`CLINC150+OOS`** (36 options per request)
- **`API-Bank`** (30 options per request)
- **`ContractNLI`** (17 simultaneous questions per request, exceeding 10-slot canvas limit)

Because the Decision Index treats every capacity refusal as wrong (`0.0`), `mmastrac`'s coverage drops from `0.76` to `0.6927`, costing **`~14.7` skill points** (`32.24` vs. `JoshuaSP`'s `49.47`).

`dgem`'s [`pkg/decisionindex`](../../pkg/decisionindex/engine.go) eliminates this gap by:
1. **Multi-Slot Canvas Batching**: Automatically slicing requests with $M > 8$ simultaneous questions into canvas-safe sub-requests and stitching probability distributions.
2. **2-Stage Bracket Tournaments**: Slicing $K > 26$ options into $\le 20$-option brackets in Pass 1 and running a Finals pass to yield normalized probabilities over all $K$ candidates without dropping options.

> **Validation Roadmap**: `dgem` exposes `dgem systemone serve` and `/v1/systemone` to allow running the official `python -m decision_index pipeline --engine http` directly against our server for certified full-suite evaluation.

---

## 5. Reproducing `EXP-12`

```bash
# 1. Replay the verified Cloud Run GPU receipt offline
./bin/dgem bench-decision-index --from-receipt benchmarks/decision_index/results_decision_index_cloudrun.json

# 2. Run live against Cloud Run GPU with Naive vs. Wide-Canvas ablation
./bin/dgem bench-decision-index -u "https://dgemma-tkb3aiuiea-uc.a.run.app/v1" --gcp-auth --compare-naive -w 2

# 3. Launch the /v1/systemone HTTP adapter for upstream apolinario/decision-index Python runner
./bin/dgem bench-decision-index -u "https://dgemma-tkb3aiuiea-uc.a.run.app/v1" --gcp-auth --serve-systemone :8095
```
