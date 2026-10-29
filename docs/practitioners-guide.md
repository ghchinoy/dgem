# Decision Models (`dgem`) Practitioner Guide

**Status**: Production Reference (`dgem v0.3.13`, `EXP-01` to `EXP-51`)  
**Repositories**: [`GoogleCloudPlatform/dgem`](https://github.com/ghchinoy/dgem) (SDK & CLI) | [`ghchinoy/decision-lab`](https://github.com/ghchinoy/decision-lab) (Public Benchmarks & Domain Reference Suite)

## 1. System-1 Readout Versus System-2 Generation

Standard autoregressive language models operate as deliberative `System 2` generators: they emit tokens sequentially across dozens of forward passes (`300 ms` to `8,000 ms`), consume KV-cache memory bandwidth proportional to output length, and produce poorly calibrated greedy probabilities when constrained to single-token classifications.

`dgem` converts `DiffusionGemma` into a single-pass `System 1` decision engine via discrete block-diffusion slot readout:

1. Single-Pass Multi-Slot Resolution (`25` to `62 ms` p50): A fixed 16-token diffusion block is seeded with up to 8 categorical slots (`questions`). In one bidirectional denoising step, the model resolves all 8 slots simultaneously.
2. Calibrated Token-Level Shannon Entropy ($\tilde{H}_{\text{raw}} \in [0, 1]$): Each slot extracts the top-5 option-token log-probabilities (`A`, `B`, `C`, ...) and computes normalized Shannon entropy (`ECE = 0.044`, `Brier = 0.097`, error-detection `AUROC = 0.80` to `0.93`).
3. Selective System-1 to System-2 Cascades: High-confidence reads ($\tilde{H}_{\text{raw}} < \tau$) execute immediately at `$0.011` per `1,000` decisions (`85` to `90` decisions/s on a single L4 or RTX PRO 6000 GPU), while only ambiguous or high-risk inputs ($\tilde{H}_{\text{raw}} \ge \tau$) escalate to `Gemini 3.8 Flash`.

![Four-Stage dgem Decision Funnel and Selective System-2 Cascade](images/fig1_dgem_system1_to_system2_pipeline.png)

*Figure 1: Four-stage dgem decision funnel combining Stage 0 shortlisting (`ShortlistPrepass`) and Two-RPC sketch extraction (`TwoRpcCompiler`), Stage 1 single-pass block-diffusion slot readout (`POST /v1/systemone?raw=1`), Stage 2 hesitation-gated cosine prior fusion (`ApplyPriorLogits`) and EM label-shift adaptation, and Stage 3 selective escalation with `MCKP` trajectory context pruning.*

## 2. Runtime Selection Decision Matrix

Across `EXP-23`, `EXP-35`, `EXP-39`, and `EXP-51`, we benchmarked five model families on the `/v1/systemone` contract (`N >= 479` pooled items across `jev_systemone`, `calib_systemone`, `gate_mixed_noul`, `tool_call_gate`, and `BFCL`). Use Figure 2 and Table 1 to select the right runtime for your workload.

![Runtime Selection Architecture and Public-Dataset Industry Domain Benchmarks](images/fig3_runtime_selection_and_domain_matrix.png)

*Figure 2: Empirical runtime comparison across Stock `dgem`, `DecisionGemma` (`r=16` LoRA), `Classy v0.3.4` (`Gemma-4-E2B` scratchpad), and `Gemini 3.8 Flash`, paired with the five public-dataset industry domain benchmarks.*

| Runtime Engine | p50 Latency (`1` to `8` Slots) | Pooled Accuracy / DI (`EXP-39`, `EXP-51`) | Error-Detection `AUROC` / `ECE` | Throughput & Unit Cost (`1k` dec) | Optimal Deployment Role |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **Stock `dgem` (`DiffusionGemma` + `?raw=1`)** | **`25` to `58 ms`** | `77.1%` (`DI 10.42`) | **`0.80` to `0.93` AUROC / `0.044` ECE** | **`85` to `90 dec/s` (`\$0.011`)** | High-volume stream triage, `<100ms` dialog guardrails, multimodal contact sheets, Stage-1 cascade gate |
| **`DecisionGemma` (`r=16 LoRA` + `?raw=1`)** | **`28` to `62 ms`** | `81.2%` (`+7.2 pp` tool gates) | `0.78` to `0.91` AUROC / `0.058` ECE | **`80` to `85 dec/s` (`\$0.012`)** | Agent tool routing (`workspace-tools-202`), 8-doc listwise reranking (`EXP-47`), `K=30` to `150` intent routing (`Banking77`, `CLINC150`) |
| **`Classy v0.3.4` (`Gemma-4-E2B` Scratchpad)** | `177` to `340 ms` | **`91.6%` (`DI 12.50`)** | `0.59` AUROC / `0.142` ECE *(over-confident)* | `15` to `25 dec/s` (`~\$0.045`) | Standalone single-model classifier where `~250ms` is acceptable and no Stage-2 cascade is needed |
| **Embedding Classifiers (`Clef` / `Jev` Dual-Encoder)** | `8` to `15 ms` | `68.0%` to `74.5%` *(zero cross-attention)* | `0.62` to `0.68` AUROC | `300+ dec/s` (`\$0.003`) | Stage-0 top-15 candidate generator feeding `dgem` `--shortlist-topk 15` (`EXP-45`) |
| **`Gemini 3.8 Flash` (`thinking=off` / `low`)** | `265` to `1,800 ms` | `88.5%` to `93.4%` | N/A *(Stage-2 oracle)* | API token billing (`\$0.15` to `\$0.60`) | Stage-2 escalation reasoner for the `15%` to `25%` of inputs where `dgem` hesitates ($\tilde{H}_{\text{raw}} \ge \tau$) |

## 3. Five Canonical Production Architectures

### 3.1 Pattern A: Single-Pass Multi-Slot Rubric (Up to 8 Slots)

Instead of issuing separate classification calls for intent, urgency, policy compliance, and routing action, pack up to 8 independent or complementary questions into a single `dgem` policy. Because `DiffusionGemma` denoises all 8 answer positions in one forward pass, going from 1 slot (`25 ms`) to 8 slots (`58 ms`) adds zero additional forward passes.

* Critical Requirement (`EXP-39`): Always target a serving image with native `/v1/systemone?raw=1` support (`v0.3.12+`, checked via [`FitsServingImage`](https://github.com/ghchinoy/dgem/blob/main/cmd/systemone_serve.go)). Bypassing chat-completion adapter re-wrapping improves pooled accuracy by **`+2.9 pp` (`+7.2 pp` on tool-call gates)**.

### 3.2 Pattern B: Selective Hesitation and Split Conformal Cascade

For workloads with strict precision floors (such as financial compliance, RAG security governance, or safety guardrails), use `dgem`'s raw normalized Shannon entropy ($\tilde{H}_{\text{raw}}$) or a calibrated Split Conformal prediction set (`LAC` / `RAPS`) to decide whether to commit the System-1 readout or escalate to `Gemini 3.8 Flash`:

$$\tilde{H}_{\text{raw}}(x) = -\frac{1}{\ln K} \sum_{k=1}^{K} P_{\text{dgem}}(y_k \mid x) \ln P_{\text{dgem}}(y_k \mid x)$$

* Fixed-Cardinality Gate ($K$ constant): Escalate when `slot.RawNormalizedEntropy` is at least `0.25` (captures `82%` to `93%` of errors while escalating only `15%` to `22%` of traffic in `EXP-34` and `EXP-44`).
* Variable-Cardinality Gate ($K \in [2, 50]$, `EXP-49` Track B): Normalized entropy is not scale-invariant across different option counts $K$. Calibrate a Least Ambiguous Set-Valued Classifier (`LAC`) quantile $\hat{q}_{1-\alpha}$ on `100` labeled examples per slot:

$$C_{1-\alpha}(x) = \{y_k : 1 - P_{\text{dgem}}(y_k \mid x) \le \hat{q}_{1-\alpha}\}$$

Escalate to `Gemini 3.8 Flash` if and only if $|C_{1-\alpha}(x)| > 1$. In `EXP-49` (`N=800` live slots across $K \in \{2, 4, 10, 50\}$), `LAC` achieved **`96.2%` overall empirical coverage (`1.18` mean set size)** and held per-cardinality coverage between **`89.3%` and `92.2%` (`0.029` cross-$K$ spread)** versus a `0.420` spread for fixed entropy.

### 3.3 Pattern C: Wide-Option Shortlist and Cosine Fusion

When a slot has more than 20 options (such as `30` to `150` service intents or document types), `dgem`'s default `BracketElimination` runs a 2-round tournament (`2.0` passes). If the caller already computes dense embeddings (such as `gemini-embedding-2-preview`), collapse the decision to **1 pass (`35.6 ms` p50, `-45.5%` latency)** using [`ShortlistPrepass`](https://github.com/ghchinoy/dgem/blob/main/pkg/decisionindex/shortlist.go) and [`ApplyPriorLogits`](https://github.com/ghchinoy/dgem/blob/main/pkg/decisionindex/prior_evidence.go):

1. Shortlist Top-15 (`--shortlist-topk 15`): Pass caller cosine similarities as `PriorLogits` so `PriorLogitsShortlist` prunes $K \to 15$ before prompt assembly (`98.0%` to `99.0%` top-15 recall on `PolyAI/banking77` and `DeepPavlov/clinc150` in `EXP-45`). Keep `ShortlistContrastive: false` (the default).
2. Z-Score Cosine Normalization (`NormalizeCosinePriorLogits`): Raw embedding cosines cluster tightly in $[0.55, 0.78]$ ($\sigma \approx 0.04$). Scale them to unit variance divided by temperature $\tau = 0.07$:

$$\ell_{\text{prior}}(y_k) = \frac{\text{cos}(q, c_k) - \mu_{\text{cos}}}{\max(\sigma_{\text{cos}}, 10^{-6}) \cdot \tau}$$

3. Hesitation-Gated Fusion (`--prior-gate-mode hesitant`): Fuse the normalized prior only when `dgem` is uncertain ($\tilde{H}_{\text{raw}} > 0.05$), and always drive downstream cascades from `RawNormalizedEntropy`:

$$\log \tilde{P}(y_k \mid x) \propto \log P_{\text{dgem}}(y_k \mid x) + \alpha \cdot \mathbb{I}[\tilde{H}_{\text{raw}} > 0.05] \cdot \ell_{\text{prior}}(y_k)$$

### 3.4 Pattern D: Two-RPC Sketch-Then-Decide (`TwoRpcCompiler`)

When a rubric requires extracting free-text entities (dates, identifiers, arithmetic totals) before scoring categorical rules, a single diffusion pass loses accuracy on multi-hop dependencies (`60.0%` in `EXP-28`). [`TwoRpcCompiler`](https://github.com/ghchinoy/dgem/blob/main/pkg/decisionindex/tworpc.go) splits execution into two fast RPCs:
* RPC 1 (`~210 ms`): A short generative sketch extracts the structured facts (`80` to `120` tokens).
* RPC 2 (`~26 ms`): `dgem` reads the compact sketch and resolves all 5 to 8 categorical rubric slots in one pass, lifting accuracy from **`60.0%` to `79.0%` at `240 ms` total** (`18.6x` faster than `Gemini 2.5 Flash` thinking at `4,461 ms`).

### 3.5 Pattern E: High-Throughput Batch Pre-Filtering

For offline or nearline backlogs (`1M` to `100M+` records/day), deploy `dgem` on Cloud Run Jobs with L4 or RTX PRO 6000 GPUs (`EXP-31`):
* Set `MAX_INFLIGHT=64` and reserve `12 GiB` KV cache per replica.
* Each GPU sustains **`85` to `90` decisions/s**, processing `1,000,000` records in `108 minutes` across 2 GPUs at a total compute cost of **`\$11.00` (`\$0.011` per `1,000` decisions)**.

## 4. Public-Dataset Industry Domain Benchmarks

Complete runnable policies and public benchmark runners for all five industry domains reside in `ghchinoy/decision-lab/examples/industry-domain-benchmarks/` (with zero client names or proprietary PII):

| Industry Domain Pattern | Directory in `decision-lab` | Public Benchmark Datasets | Core `dgem` Mechanism | Measured Public Benchmark Result |
| :--- | :--- | :--- | :--- | :--- |
| **1. High-Volume Stream Triage & Trajectory Pruning** | `examples/industry-domain-benchmarks/01_stream_triage_and_trajectory_pruning/` | `pruning_trajectories.jsonl` ($N=100$ coding/SQL turns), GitHub Issue/PR Spam ($N=200$) | 4-slot triage + `MCKP` trajectory pruning (`SelectContextPruningActions`, $\lambda=0.45$) | **`57.7%` prompt token reduction** at **`100.0%` critical-context recall** (`p = 6.98e-6`); `97.9%` spam precision at `\$0.011/1k` (`EXP-31`, `EXP-34`, `EXP-49`). |
| **2. Wide-Taxonomy Intent & Document Routing ($K=26\text{ to }150$)** | `examples/industry-domain-benchmarks/02_wide_taxonomy_intent_routing/` | `PolyAI/banking77` ($K=30$), `DeepPavlov/clinc150` ($K=26$ + OOS), `EXP-45` & `EXP-48` ($K=30\text{ to }60$) | `ShortlistPrepass` (`--shortlist-topk 15`) + `NormalizeCosinePriorLogits` ($\tau=0.07$, `hesitant`) + EM shift adaptation | **`86.7%` Banking77** and **`96.7%` CLINC150 (`100%` OOS)** in **1 pass (`35.6 ms` p50, `-45.5%` latency)**; **`+6.0 pp`** under 80/20 skew (`EXP-45`, `EXP-48`). |
| **3. Enterprise RAG Governance & Injection Quarantine** | `examples/industry-domain-benchmarks/03_rag_governance_and_quarantine/` | `exp50_rag_quarantine/test_frozen.jsonl` ($N=62$ slates, `372` passages from `docs_eval` & `BEIR`) | 8 co-denoised slots (`p1_rel`, `p1_sec`, ..., `p4_rel`, `p4_sec`) over `<untrusted_data>` with 4-tier `use`-vs-`mention` slot | **`100.0%` clean retrieval recall**, **`0.0%` benign security false-quarantine**, and **`0.0%` `ASR@1`** (`-9.7 pp`, `p = 0.0156`, `EXP-50`). |
| **4. Multi-Hop Rubric & Deterministic Tool Dispatch** | `examples/industry-domain-benchmarks/04_multihop_rubric_and_tool_dispatch/` | `workspace-tools-202` / `BFCL` ($N=202$), `JevBench` ($N=258$), `EXP-28` & `EXP-51` | 5-slot rubric fed by `TwoRpcCompiler` (`Sketch-Then-Decide`) or `Classy v0.3.4` (`r=16` LoRA) | **`98.5%` tool-gate accuracy** (`n=202`); `TwoRpcCompiler` lifts multi-hop rubrics from **`60.0%` to `79.0%`** at **`240 ms`**; `Classy v0.3.4` reaches **`84.0%`** on `relevance_few_detect`. |
| **5. Real-Time Dialog Guardrails & Turn-Taking** | `examples/industry-domain-benchmarks/05_realtime_dialog_guardrails/` | `calibration_suite.jsonl` (`CivilComments`, `DICES`, `MNLI`, $N=65$), `WildGuard`, `Aegis`, `JevBench` | 5 slots (`toxicity_and_harm`, `disclosure_compliance`, `factual_entailment`, `prosodic_turn_state`, `orchestrator_action`) | **`58.0 ms` p50 (`74.0 ms` p95)** 5-slot latency (`>220 ms` SLA margin under `<300 ms` voice budget); **`93.8%` `calibration_suite`** and **`94.6%` `JevBench`** cascade accuracy. |

## 5. Policy Authoring and Calibration Checklist

1. Keep Per-Pass Option Counts $K \le 20$ (`az` Single-Token Budget):
   * Standard single-letter labels (`A` through `T`) have uniform tokenizer priors and high error-detection `AUROC` (`0.84+`).
   * Wide 52-code label sets (`az_aa` in `PROP-29`) avoid tournament brackets up to $K=52$ but reduce out-of-scope entropy `AUROC` (`0.84` to `0.65`). Prefer caller embedding shortlisting (`--shortlist-topk 15`) whenever embeddings exist.
2. Write Mutually Exclusive, Positive Option Descriptions:
   * Define each option by what it *is*, not by referencing other option names.
   * Include an explicit catch-all or abstain option (`none_of_the_above` or `insufficient_evidence`) when inputs can fall outside the domain taxonomy.
3. Audit Order Bias on New Policies (`SwappedOrderAgreement`):
   * Run forward vs. reversed option order (`option_order=source` vs. `reversed`) on a 50-item dev slice. Policies with `SwappedOrderAgreement` below `92%` have overlapping option descriptions that should be tightened before production rollout.
4. Understand Concurrency Determinism (`PROP-32` & `PROP-34`):
   * Sequential `samples=1` requests are **100% deterministic (`0` flips across `2,731` questions in `PROP-32`)**.
   * Under concurrent load (`concurrency >= 8`), borderline slots ($\tilde{H}_{\text{raw}} > 0.25$) exhibit a `6%` to `9%` flip rate caused by vLLM dynamic batch-composition floating-point non-associativity (`PROP-34`). Any slot within that flip band already satisfies $\tilde{H}_{\text{raw}} \ge 0.25$ and should be routed to the Stage-2 cascade or evaluated with `samples=3`.

## 6. Empirically Falsified Anti-Patterns

Every anti-pattern below was tested on live `/v1/systemone` endpoints in `ghchinoy/decision-lab` and rejected based on negative empirical results:

| Anti-Pattern (Do Not Do This) | Measured Failure Mode (`decision-lab` Receipt) | Validated Production Alternative |
| :--- | :--- | :--- |
| **1. Appending `[vs siblings: ...]` contrastive suffixes to shortlisted options (`EXP-45`)** | Drops single-pass accuracy by **`-6.0 pp` on `K=30` (`82%` to `76%`)** and **`-7.0 pp` on `K=60` (`84%` to `77%`)** due to token cross-talk in the 16-token block. | Keep `ShortlistContrastive: false` (default in `dgem#198`). Let clean option descriptions stand alone. |
| **2. Using surface lexical overlap (`LexicalOverlapShortlist`) to shortlist paraphrased user intents (`EXP-41`)** | Top-15 shortlist recall drops to **`65.2%` to `69.5%`** on `PolyAI/banking77` and `DeepPavlov/clinc150`, capping end-to-end accuracy `-11.5 pp` below 2-round brackets. | Supply dense embedding cosines (`gemini-embedding-2-preview`) via `PriorLogitsShortlist` (**`98%` to `99%` recall**), or fall back to 20-way `BracketElimination`. |
| **3. Replacing 20-way tournament brackets with a 1-pass or 2-pass Hierarchical Taxonomy Trie (`EXP-46`)** | 1-pass joint parent+leaf trie drops accuracy by **`-38.1 pp`** (`43.4%` vs. `81.5%`); 2-pass top-2 beam trie still trails brackets by **`-17.3 pp`** due to irreversible root routing errors. | Use 1-pass embedding shortlists (`--shortlist-topk 15`) when embeddings exist, or standard 20-way `BracketElimination` (`2.0` passes) for cold-start catalogs. |
| **4. Gating downstream cascades on post-fusion `NormalizedEntropy` instead of `RawNormalizedEntropy` (`EXP-40`, `EXP-44`)** | A sharp external prior masks model uncertainty when the prior and `dgem` disagree, reducing error-detection `AUROC` by **`-0.041` (`-4.1 pp`)**. | Always read `slot.RawNormalizedEntropy` ($\tilde{H}_{\text{raw}}$) for cascade escalation decisions. |
| **5. Running a 2-pass self-prior cascade for cold-start 8-document reranking (`EXP-47`)** | Pass-1 coarse intent classification followed by Pass-2 reranking doubles latency (`2.0` passes) and **reduces `MRR@8` by `-0.025`** compared to a single-pass 8-slot co-read. | Score all 8 candidate documents jointly in **1 pass** (`0.619 MRR@8`, `0.680 nDCG@8` on `DecisionGemma`). |
| **6. Using binary keyword triggers for RAG prompt-injection quarantine (`EXP-50`)** | Causes a **`36.4%` to `56.4%` false-quarantine rate** on benign security or audit documentation that quotes attack phrases as educational examples. | Use the 4-tier `use`-vs-`mention` security taxonomy (`clean`, `benign_security_discussion`, `suspicious_framing`, `active_override_attempt` in `PROP-48`). |
