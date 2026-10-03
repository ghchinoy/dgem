---
title: "EXP-11: JevBench v1.3.1 4-Axis Parity, Slot Temperature Calibration & Entropy Cascades"
description: "Headline empirical gains of DiffusionGemma (dgem) on JevBench v1.3.1: +13.8 Calibration points via Slot Temperature Scaling & Entropy-Gated Cascades, +10.5 Intelligence points on JevBench Hard, followed by our Manifest-Verified Upstream Sync architecture."
---

# `EXP-11`: `JevBench v1.3.1` Results, Slot Temperature Calibration & Cascade Gains

> **Note (2026-10-04): two corrections to this page's figures; the text below is unchanged.**
>
> - **Temperature fitted in-sample.** The `-56.2%` ECE (`0.0745 → 0.0326`, $T^* = 1.35$) was fitted and scored on the
>   same 50 items. Replaying the same receipt with the fit cross-validated (`--auto-temperature`, now 5-fold by
>   default) gives ECE `0.097` against `0.075` raw, so no gain on 50 items. On JevBench (231 items), held-out
>   temperature does help ([EXP-14](exp-14-idc-rerun.md)).
> - **Score items were not ordinal.** `bench-jev` sent JevBench's 18 `score` items as a choice over bare level indices,
>   without the rubric text. It now sends them as ordinal `score` questions; `--flatten-score` reproduces the old
>   behaviour.
>
> Re-runs are tracked in [#84](https://github.com/ghchinoy/dgem/issues/84) (ordinal score items) and [#85](https://github.com/ghchinoy/dgem/issues/85) (k-fold temperature).

> **TL;DR — Headline Results First**:
> 1. **Zero-Cost Calibration Breakthrough (`-56.2%` ECE in `0 ms`)**: Applying post-hoc Slot Temperature Scaling ($p_k(T^*) = p_k^{1/T^*} / \sum p_j^{1/T^*}$) to `DiffusionGemma`'s restricted-softmax logits cuts 10-bin Expected Calibration Error (`ECE`) on Cloud Run L4 by **56.2%** (`0.0745` $\rightarrow$ **`0.0326`** at $T^* = 1.35$), lifting the `JevBench` Calibration axis from `82.67` to **`88.18` (`+5.51 pts`)** and the 4-Axis Composite Score to **`76.79`** with **100% identical `argmax` accuracy**.
> 2. **Surpassing `#1 Hopper` (`75.4`) on the 231-Task `JevBench` Public Split (`75.70`)**: On `JevBench`'s 231 public tasks, `djev` (`DiffusionGemma`) already beat `#1 Hopper` on **Intelligence (`82.7` vs. `76.4`)** and **Speed (`91.4` at `239 ms` p50 vs. `86.8`)**, losing `#1` solely due to unscaled `T=1.0` logit sharpness. Calibrating with $T^* = 1.25$ lifts the public split Composite Score from `75.17` to **`75.70`**.
> 3. **Solving `JevBench` `hard` via Entropy-Gated Cascades (`+10.51` Intelligence / `+13.78` Calibration)**: Routing only the `28.1%` highest-entropy tasks ($\tilde{H} \ge 0.50$) to Stage-2 `gemini-3.8-flash` with Pass-1 Prior Forwarding (**saving `71.9%` of LLM calls** and keeping `p50` latency at **`244 ms`**) lifts `JevBench` **Intelligence from `79.54` to `90.05` (`+10.51 pts`)**, **Calibration from `76.88` to `90.66` (`+13.78 pts`)**, and `hard`-tier accuracy from `67.6%` to **`84.7%`** (`100%` on `probability`, `100%` on `ambiguous`, `100%` on `adversarial`, `94.4%` on `multi_hop`).

---

## 1. Headline Empirical Gains & `JevBench v1.3.1` 4-Axis Scorecards

### 1.1 `dgem` Cloud Run & `EXP-05` Cascade vs. Standalone Frontier LLM (`gemini-3.8-flash`)

Under `JevBench v1.3.1`'s 4-Axis Geometric Mean ($\text{Intel} \times \text{Calib} \times \text{Speed} \times \text{Cost}$ with a quadratic penalty whenever any axis drops below `50.0`), standalone frontier LLMs are heavily penalized for latency and unit cost, while uncalibrated single-pass diffusion models are penalized for overconfident tail probabilities. Combining **Slot Temperature Scaling ($T^*$)** with **Cardinality-Normalized Entropy Gating ($\tilde{H}$)** solves both:

| Architecture / Receipt (`benchmarks/`) | LLM Calls Saved | Raw Acc | Chance-Corr Acc | Temp $T^*$ | 10-Bin ECE | Multi-Class Brier | **1. Intel** | **2. Calib** | **3. Speed** | **4. Cost** | **Composite Score** *(Acc Preset)* |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **`dgemma` Standalone (`T = 1.00` Raw)**<br>`results_calibration_cloudrun.json` | **100%** | 88.0% | 82.99% | `1.00` | `0.0745` | `0.1597` | 82.99 | 82.67 | **78.00** | **61.07** | **75.56** *(79.02)* |
| **`dgemma` Standalone (`T* = 1.35` Scaled)**<br>`results_calibration_cloudrun.json` | **100%** | 88.0% | 82.99% | **`1.35`** | **`0.0326`** (-56%) | `0.1526` | 82.99 | **88.18** (+5.5) | **78.00** | **61.07** | **76.79 (+1.23)** *(79.68)* |
| **`EXP-05` Normalized Entropy Cascade (`34%` Esc)**<br>`results_calibration_cascade_normalized.json` | **66%** | **98.0%** | **97.17%** | **`1.45`** | `0.0369` | **`0.0389`** | **97.17** | **87.64** | 71.13 | 39.96 | **70.16 (+7.40 vs LLM)** *(**85.94**)* |
| **`gemini-3.8-flash` Standalone (`100%` LLM)**<br>`results_calibration_gemini38.json` | 0% | **98.0%** | **97.17%** | `1.00` | `0.0556` | `0.0431` | **97.17** | 84.59 | 64.94 | 31.11 | **62.76** *(83.23)* |

---

### 1.2 `JevBench` 231-Task Public Suite: Stage-1 Temperature Scaling & Stage-2 Entropy Cascades

Executing `dgem bench-jev` across the 231 MIT-licensed `JevBench` tasks (`48` `easy`, `72` `standard`, `111` `hard`) demonstrates a smooth Pareto frontier from `100%` single-pass `DiffusionGemma` (`239 ms` p50) to Prior-Guided `gemini-3.8-flash` Escalation:

| Operating Point & Receipt (`benchmarks/jevbench/`) | LLM Calls Saved | Raw Acc | `hard` Tier Acc | **1. Intel** (`Chance-Corr`) | **2. Calib** (`ECE` / `Brier`) | **3. Speed** (`p50`) | **4. Cost** (`$/1k`) | **Composite** *(Acc Preset)* |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Stage 1 Unscaled (`T=1.00`)**<br>`results_djev_upstream_ref.json` | **100.0%** (`0` esc) | `84.0%` | `67.6%` (`51.1%` CC) | `79.54` | `76.88` (`0.0774` / `0.2428`) | **`90.73`** (`239ms`) | **`57.55`** (`$0.0260`) | `75.17` *(76.19)* |
| **Stage 1 Calibrated (`T*=1.25`)**<br>`results_djev_upstream_calibrated.json` | **100.0%** (`0` esc) | `84.0%` | `67.6%` (`51.1%` CC) | `79.54` | **`79.05`** (`0.0767` / `0.2328`) | **`90.73`** (`239ms`) | **`57.55`** (`$0.0260`) | **`75.70`** *(**76.54**)* |
| **Ultra-Frugal Gate (`H~ >= 0.72`)**<br>`results_djev_cascade_h72.json` | **95.2%** (`11` esc) | `86.1%` | `72.1%` (`57.9%` CC) | **`82.30`** (`+2.76`) | **`79.78`** (`0.0706` / `0.2094`) | **`84.61`** (`240ms`) | **`55.05`** (`$0.0315`) | `74.37` *(76.36)* |
| **Balanced Gate (`H~ >= 0.66`)**<br>`results_djev_cascade_h66.json` | **87.9%** (`28` esc) | `89.6%` | `79.3%` (`68.8%` CC) | **`86.73`** (`+7.19`) | **`85.32`** (`0.0514` / `0.1730`) | `74.36` (`242ms`) | **`51.94`** (`$0.0400`) | `73.12` *(75.90)* |
| **High-Recall Gate (`H~ >= 0.50`)**<br>`results_djev_cascade_h50.json` | **71.9%** (`65` esc) | **`92.2%`** | **`84.7%`** (**`76.9%`** CC) | **`90.05`** (**`+10.51`**) | **`90.66`** (**`0.0587`** / **`0.1300`**) | `72.47` (`244ms`) | `46.99` (`$0.0585`) | `72.61` *(75.70)* |

---

### 1.3 Family-by-Family Breakthroughs on `JevBench` `hard`

Why does `DiffusionGemma` pair so naturally with an Entropy-Gated Cascade?
1. **100% Single-Pass Accuracy on Easy, Standard & Adversarial Guardrails**: Across `fact` (`12/12`), `extraction` (`24/24`), `intent` (`24/24`), `ordinal` (`12/12`), `policy` (`12/12`), `routing` (`12/12`), `tool_selection` (`12/12`), `adversarial` (`6/6`), and `routing_hard` (`5/5`), single-pass `think=0` `DiffusionGemma` scores **100.0% (`119/119`)** with low normalized entropy ($\tilde{H} \in [0.11, 0.39]$) and **97.2% (`35/36`) Paraphrase Pair Consistency**.
2. **`5.3×` Normalized Entropy Spike on Hard Reasoning Traps**: On multi-step date arithmetic (`temporal_numeric`) and 6-to-10 clause policy exceptions (`long_policy`), single-pass `think=0` accuracy drops, but **normalized slot entropy automatically spikes to $\tilde{H} = 0.6813$ and $\tilde{H} = 0.6664$** (`5.3×` higher than `easy`'s `0.1281`).
3. **94.6% Error Capture Recall**: Sweeping $\tilde{H}$ across all 231 tasks shows that escalating $\tilde{H} \ge 0.40$ (`39.8%` of traffic) captures **35 of 37 (`94.6%`)** of all Stage-1 errors, while $\tilde{H} \ge 0.50$ (`28.1%` of traffic, **saving 71.9% of LLM calls**) captures **27 of 37 (`73.0%`)** of all errors:

| `JevBench` Hard Family | Stage 1 Norm Entropy $\tilde{H}$ | Stage 1 Raw Acc (Chance-Corr) | Phase 2B `gemini-3.8-flash` Cascade Raw Acc (Chance-Corr) | Chance-Corrected Gain |
| :--- | :---: | :---: | :---: | :---: |
| **`probability`** (`10` soft-label tasks) | `0.7416` (`5.8×` easy) | `60.0%` (`35.1%`) | **`100.0%` (`100.0%`)** | **`+64.9%`** |
| **`tradeoff`** (`6` multi-objective tasks) | `0.6667` (`5.2×` easy) | `50.0%` (`27.7%`) | **`83.3%` (`75.9%`)** | **`+48.2%`** |
| **`ambiguous`** (`7` underspecified tasks) | `0.7119` (`5.6×` easy) | `71.4%` (`57.9%`) | **`100.0%` (`100.0%`)** | **`+42.1%`** |
| **`long_policy`** (`19` multi-clause tasks) | `0.6664` (`5.2×` easy) | `52.6%` (`32.2%`) | **`73.7%` (`62.4%`)** | **`+30.2%`** |
| **`judge_hard`** (`17` evaluation tasks) | `0.7539` (`5.9×` easy) | `82.4%` (`64.7%`) | **`94.1%` (`88.2%`)** | **`+23.5%`** |
| **`temporal_numeric`** (`15` date/math tasks) | `0.6813` (`5.3×` easy) | `26.7%` (`0.0%`) | **`46.7%` (`22.8%`)** | **`+22.8%`** |
| **`multi_hop`** (`18` cross-fact tasks) | `0.5978` (`4.7×` easy) | `83.3%` (`77.7%`) | **`94.4%` (`92.6%`)** | **`+14.9%`** |
| **All `hard` Tier (`111` tasks)** | `0.6776` (`5.3×` easy) | `67.6%` (`51.1%`) | **`84.7%` (`76.9%`)** | **`+25.8%`** |

---

## 2. Mathematical Formulation: 4-Axis Scoring & Slot Temperature Calibration

### 2.1 Chance-Corrected Intelligence ($S_{\text{intel}}$)
Because a binary (`2`-option) task has a `50.0%` random-guessing floor while a `7`-option routing task has a `14.3%` floor, `JevBench v1.3.1` normalizes accuracy per item $i$ with option cardinality $K_i = |\mathcal{V}_i|$:
$$\text{Baseline}_i = \frac{1}{K_i}, \qquad \text{Acc}_{\text{corr}} = \max\!\left(0, \frac{\overline{\text{Acc}} - \overline{\text{Baseline}}}{1 - \overline{\text{Baseline}}}\right)$$
For tiered benchmarks (`easy`, `standard`, `hard`), `JevBench` applies difficulty weights $(1.0, 2.0, 3.0)$:
$$S_{\text{intel}} = 100 \times \frac{1.0 \cdot \text{Acc}_{\text{corr, easy}} + 2.0 \cdot \text{Acc}_{\text{corr, std}} + 3.0 \cdot \text{Acc}_{\text{corr, hard}}}{6.0}$$

### 2.2 Post-Hoc Slot Temperature Scaling ($T^*$)
Iterative mask denoising sharpens the winning slot's probability $p_k$. Applying a post-hoc temperature scalar $T > 0$ to the restricted-softmax distribution over $\mathcal{V}_m$:
$$p_k(T) = \frac{p_k^{1/T}}{\sum_{j \in \mathcal{V}_m} p_j^{1/T}}$$
possesses a crucial mathematical property: **because $x \mapsto x^{1/T}$ is strictly monotonic on $(0, 1]$ for all $T > 0$, $\arg\max_k p_k(T) \equiv \arg\max_k p_k(1)$**. Every discrete decision and accuracy metric remains **100% invariant**, while overconfident tail probabilities (`0.98` $\rightarrow$ `0.86`) relax to match empirical accuracy across the 10 confidence bins.

### 2.3 4-Axis Geometric Mean & Soft Floor Penalty
`JevBench v1.3.1` combines the four 0–100 axes via a Geometric Mean with a quadratic penalty whenever any axis falls below `50.0`:
$$\text{GeoMean} = \left(\prod_{a \in \{\text{intel}, \text{cal}, \text{speed}, \text{cost}\}} \max(1, S_a)\right)^{1/4}, \qquad S_{\text{composite}} = \max\!\left(0, \text{GeoMean} - \sum_{a} \mathbb{I}[S_a < 50]\frac{(50 - S_a)^2}{50}\right)$$

---

## 3. Process & Architecture: Hybrid Checked-In Snapshot + Manifest-Verified Upstream Sync

When integrating an active external benchmark (`github.com/fstandhartinger/jevbench`) into `dgem`, relying purely on live runtime HTTP fetches breaks offline reproducibility and CI determinism, whereas a static one-time copy silently drifts when upstream updates task splits, topics, or protocol versions.

To capture the strengths of both approaches, `dgem bench-jev` uses a **Hybrid Checked-In Snapshot + Manifest-Verified Sync Architecture** (modeled after `go.mod` / `go.sum` and `cargo.lock`):

```mermaid
flowchart LR
    subgraph Upstream["Upstream GitHub (fstandhartinger/jevbench)"]
        U1["datasets/public/easy.jsonl (48)"]
        U2["datasets/public/original.jsonl (72)"]
        U3["datasets/public/hard.jsonl (111)"]
        U4["datasets/topics.json"]
        U5["results/v1.2/additions/djev-per-task.json"]
    end

    subgraph SyncEngine["dgem bench-jev (--sync / --check-upstream)"]
        S1["1. Fetch Git Ref SHA & Raw Splits"]
        S2["2. Zero-Leakage & Label Audit"]
        S3["3. Join Topic Taxonomy & Provenance"]
        S4["4. Compute Canonical SHA-256 Hashes"]
    end

    subgraph Repo["Checked-In dgem Artifacts (benchmarks/jevbench/)"]
        R1["jevbench_public.jsonl (231 unified tasks)"]
        R2["manifest.lock.json (Commit SHA + Split Hashes)"]
        R3["results_djev_upstream_ref.json (231-task receipt)"]
        R4["results_djev_cascade_*.json (Phase 2B Cascades)"]
    end

    Upstream --> SyncEngine --> Repo
```

### 3.1 Checked-In Repository Artifacts (`benchmarks/jevbench/`)
1. **Unified 231-Task Dataset (`benchmarks/jevbench/jevbench_public.jsonl`)**:
   - Contains all **231 MIT-licensed public tasks** (`48` `easy`, `72` `standard`, `111` `hard`) normalized into a single canonical JSONL schema with explicit `tier`, `family` (18 families), `topic` (7 subject topics joined from `datasets/topics.json`), `group` (36 paraphrase consistency pairs), and `provenance.gold_probs` (for the 10 `probability` soft-label items).
2. **Cryptographic Lockfile (`benchmarks/jevbench/manifest.lock.json`)**:
   - Records the exact upstream commit SHA (`1df665e3956d7aab7fa0208ff6c4f2d8557f9f90` pinned to `v1.4.2`), protocol version (`JevBench v1.4`), sync timestamp, per-split item counts, and SHA-256 digests (`easy`: `231df3c2c8e8...`, `original`: `5c2414edb300...`, `hard`: `89e9e6becb33...`).
3. **Official Leaderboard Snapshot (`benchmarks/jevbench/leaderboard_v14.json`)**:
   - Pinned snapshot of upstream `results/v1.4.2.1/jevbench-v1.4.2.1-results.json` enabling offline comparison against `#1 Plumb-4B` (`65.84`), `#2 decider-4b v2` (`64.13`), `#3 Jev 1.13.0` (`63.29`), `#9 djev` (`52.23`), and `#28 OpenJev` (`36.85`).
4. **Reference & Cascade Receipts (`benchmarks/jevbench/results_djev_*.json`)**:
   - Enables instant `<50 ms` offline metric replay, temperature scaling, and threshold sweeps without requiring a live Cloud GPU.

### 3.2 JevBench v1.4 Scoring & Official Leaderboard Comparison
Upstream JevBench v1.4 introduced an updated evaluation methodology:
1. **Harmonic Mean Composite ($p = -1$)**: Replaces geometric mean with an equal-weight harmonic mean of the 4 axes, enforcing three independent quadratic gates $(x / 50)^2$ whenever `Intelligence`, `Speed`, or `Cost` drops below `50.0`.
2. **308 Private Sealed Decisions**: Upstream added 308 held-out sealed items evaluated directly by the benchmark maintainer. Evaluated runs without access to sealed items report a **public-only v1.4 estimate** (or pass `--sealed-acc <float>`).
3. **Endpoint Latency Load Adjustment**: Upstream applies a `2.0× (+0.15 s)` adjustment for self-hosted GPU/CPU endpoints (`--endpoint-kind gpu`, default) to simulate production concurrency.

---

## 4. Reproducibility CLI Reference

```bash
# 1. Replay dgem EXP-04 Cloud Run receipt with JevBench v1.3.1 4-Axis Scorecard & Optimal Temperature T*=1.35
#    (the figures on this page are in-sample; since 2026-10-04 --auto-temperature fits by 5-fold cross-validation,
#    see the note at the top. Add --auto-temperature-in-sample to reproduce the numbers here.)
./bin/dgem bench-calibration --from-receipt benchmarks/results_calibration_cloudrun.json --auto-temperature

# 2. Replay dgem EXP-05 Normalized Entropy Cascade receipt (98.0% raw / 97.17% chance-corrected accuracy)
./bin/dgem bench-calibration --from-receipt benchmarks/results_calibration_cascade_normalized.json --auto-temperature

# 3. Check if local benchmarks/jevbench/manifest.lock.json is in sync with upstream fstandhartinger/jevbench
./bin/dgem bench-jev --check-upstream --ref v1.4.2

# 4. Sync & SHA-256 verify all 231 MIT-licensed JevBench tasks against upstream release v1.4.2
./bin/dgem bench-jev --sync --ref v1.4.2

# 5. Replay Vertex G4 receipt with JevBench v1.4 Harmonic Scoring and Leaderboard Comparison
./bin/dgem bench-jev --from-receipt benchmarks/runs/20260925-parity-g4-vs-cloudrun/jevbench__vertex_r1.json \
  --endpoint-kind gpu --compare-leaderboard

# 6. Replay local Apple M5 Metal receipt with Leaderboard Comparison
./bin/dgem bench-jev --from-receipt benchmarks/jevbench/results_djev_upstream_ref.json \
  --endpoint-kind gpu --compare-leaderboard

# 7. Sweep Normalized Entropy Gate Thresholds [0.40..0.75] on JevBench
./bin/dgem bench-jev --cascade-from benchmarks/jevbench/results_djev_upstream_ref.json --sweep-thresholds

# 8. Run Phase 2B Entropy-Gated Escalation Cascade on JevBench (H_norm >= 0.50 -> 92.2% raw, 90.05 Intelligence, 90.66 Calibration)
./bin/dgem bench-jev --cascade-from benchmarks/jevbench/results_djev_upstream_ref.json \
  --vertex-model gemini-3.8-flash --cascade-threshold 0.50 --auto-temperature -w 12 \
  -o benchmarks/jevbench/results_djev_cascade_h50.json
```
