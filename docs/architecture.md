# Discrete Diffusion vs. Autoregression

Theoretical and mechanical breakdown of DiffusionGemma's discrete block diffusion canvas compared to sequential autoregression, Dual-Encoders (`GTR`) + `TabPFN`, and Test-Time Compute cascades.

---

## 1. The Bottleneck of Autoregressive LLMs

Standard Large Language Models (LLMs) operate under a sequential autoregressive factorization:

$$P(x_1, x_2, \dots, x_T) = \prod_{t=1}^T P(x_t \mid x_{<t})$$

Each forward pass generates exactly **one token**. Even if the model only needs to output a single boolean decision or three fields of a JSON object:
1. It must sequentially predict structural characters (`{`, `\n`, `"`, `k`, `e`, `y`, `"`, `:`, ` `).
2. It suffers from high memory bandwidth pressure: loading billions of parameters from memory to compute a single token's logits.
3. It takes dozens or hundreds of forward passes (2–17.5 seconds across cloud or consumer GPUs).

---

## 2. Discrete Block Diffusion & Multi-Canvas Sampling

**DiffusionGemma (`dgemma`)** breaks this sequential bottleneck by using **discrete block diffusion**:

* **Block-Autoregressive Canvas**: The decoder works on a **32-to-256-token canvas** with **bidirectional self-attention**.
* **Iterative Denoising**: The entire block of tokens begins as masked slots and is denoised in parallel across a single pass (`steps: 1`, `think: 0`) or a small number of steps.
* **Joint Multi-Slot Conditioning (`slot_1 <-> slot_2`)**: Unlike independent classification heads, all masked decision slots attend to the prompt and to *each other* simultaneously in $O(1)$ forward passes (`458.9 ms` on Cloud Run 1×L4).

---

## 3. How Discrete Diffusion Slot Readout Works

In a structured decision query (`steps: 1, think: 0`), no conversational prose is generated:

1. **Canvas Seeding**: The known policy template (e.g. `urgent: @\nteam: @\nsentiment: @`) is pre-seeded into the canvas, where `@` represents masked tokens at the candidate decision slots.
2. **Single-Pass Readout**: A single forward pass executes across the causal prompt prefix and bidirectional canvas (~458–880 ms).
3. **Restricted-Softmax Logit Readout**: Rather than decoding free-form text, the engine extracts the raw logits $z_{m,k}$ restricted to the valid single-token candidate vocabulary $\mathcal{V}_m$ (`{"yes","no"}` for `boolean`, `[A–Z]` for `choice`, `1..5` for `score`) and normalizes via softmax:

$$p_{m,k} = \frac{\exp(z_{m,k})}{\sum_{j \in \mathcal{V}_m} \exp(z_{m,j})}$$

```text
Seeded Canvas:
[<|channel>thought\n<channel|>urgent: @ \nteam: @ \nsentiment: @ ]
                                      ▲          ▲           ▲
                                   Slot 1     Slot 2      Slot 3
                              [p(yes), p(no)] [p(A)..p(Z)] [p(1)..p(5)]
                                |V_1| = 2      |V_2| = 26   |V_3| = 5
```

---

## 4. Empirical Uncertainty & Cardinality-Normalized Entropy (`EXP-05b`)

Single-pass restricted-softmax readout provides calibrated epistemic uncertainty at every decision slot $m$:

* **Raw Shannon Entropy ($H_m$)**:
  $$H_m = -\sum_{k \in \mathcal{V}_m} p_{m,k} \ln p_{m,k} \in [0, \ln|\mathcal{V}_m|]$$
* **Cardinality-Normalized Epistemic Entropy ($\tilde{H}_m$)**: Because maximum entropy scales logarithmically with option count ($\ln 2 = 0.693\text{ nats}$ for binary `boolean` vs. $\ln 26 = 3.258\text{ nats}$ for 26-way `choice`), `dgem` normalizes each slot's entropy by its theoretical ceiling $\ln|\mathcal{V}_m|$:
  $$\tilde{H}_m = \frac{H_m}{\ln|\mathcal{V}_m|} \in [0, 1]$$

<details class="term-aside">
<summary>💡 <strong>Concept Aside: Why does Raw Entropy ($H_m$) cause "Multi-Slot Scale Inversion" without $\ln|\mathcal{V}_m|$ normalization?</strong> <em>(click to expand)</em></summary>

* **In Plain English**: A 26-option banking classifier naturally leaks tiny $0.3\%$ probability crumbs across 25 runner-up classes even when it is **88.6% confident and right**, inflating its raw entropy (`0.516 nats`). Meanwhile, a 3-option NLI slot (`entailment` / `neutral` / `contradiction`) has a tiny maximum ceiling (`1.099 nats`), so a massive **3.3× epistemic spike** (`0.246 nats`) looks smaller in raw nats than the 26-way slot!
* **How $\tilde{H}_m = H_m / \ln|\mathcal{V}_m|$ Fixes It**: Dividing by $\ln|\mathcal{V}_m|$ puts every slot onto a universal $[0, 1]$ uncertainty scale (`--normalize-entropy --cascade-threshold 0.16`):
  * **`b77-01` (`Banking77`, $|\mathcal{V}|=26$, Correct Pass-1)**: $0.5162 / \ln(26) = \mathbf{0.158 < 0.160} \implies$ **Early-Exits in `754 ms`!**
  * **`anli-01` & `anli-02` (`ANLI-R3`, $|\mathcal{V}|=3$, Adversarial Traps)**: $0.1847 / \ln(3) = \mathbf{0.168 \ge 0.160}$ and $0.2464 / \ln(3) = \mathbf{0.224 \ge 0.160} \implies$ **Both Escalate to Stage 2 (`0% -> 100%`)!**
* **Full Reference**: See [Experiment `EXP-05b`](experiments/exp-05-roadmap-cascades-and-dags.md) and the [Glossary entry on Multi-Slot Scale Inversion](glossary.md#multi-slot-scale-inversion).

</details>

* **Adaptive Sampling & Prior-Guided Escalation**:
  - **Low Normalized Entropy ($\tilde{H}_m < 0.160$, `66%` of suite)**: The decision is decisive. `dgem` early-exits immediately after **1 forward pass** (`712 ms` mean latency) with **100.0% early-exit precision (`33/33`)**.
  - **High Normalized Entropy ($\tilde{H}_m \ge 0.160$, `34%` of suite)**: `dgem` forwards the **Pass-1 Slot Prior Distribution (`[TIER-1 DISCRETE DIFFUSION PRIOR TELEMETRY]`)** to Stage 2—either an **Intra-Model Self-Cascade** (`--cascade-self-think 256` on the same `dgemma` GPU) or a **Cross-Model Cascade** (`gemini-3.8-flash`), lifting overall accuracy from **`86.0%` $\to$ `98.0%` (`49/50`)**.

### Why `dgem` Uncertainty Quantification is Task-Agnostic (Unifying Classification & Regression)

Practitioners frequently raise a classic challenge: *"Uncertainty quantification—whether confidence scores for classification or variances/quantiles for regression—is notoriously task-specific. How can one threshold work across tasks?"*

> [!NOTE]
> **The 2-Sentence Elevator Pitch**
> Rather than just computing raw entropy and "punting" to an LLM, `dgem` makes uncertainty **task-agnostic** in three steps: **(1)** it projects each masked canvas slot onto its policy-valid token set (`{yes,no}`, `[A–Z]`, or discrete score bins `1..5`, turning regression into a histogram expectation $\mathbb{E}[v] = \sum v_k p_k$ + spread), **(2)** it **normalizes Shannon entropy by slot capacity ($\tilde{H}_m = H_m / \ln|\mathcal{V}_m| \in [0, 1]$)** so a 2-way guardrail, a 5-point regressor, and a 26-way classifier all live on the exact same $[0, 1]$ uncertainty scale, and **(3)** when $\tilde{H}_m \ge 0.16$, it **forwards the Pass-1 probability distribution as a Bayesian prior** to guide test-time compute (`think > 0`) in disambiguating the top competing candidates.

Four concrete mechanisms in [`pkg/client/client.go`](../pkg/client/client.go) and [`cmd/bench_calibration.go`](../cmd/bench_calibration.go) eliminate task-specific calibration:

1. **Subspace Projection Eliminates Lexical/Phrasing Noise**:
   Autoregressive LLMs compute entropy over 256,000 BPE tokens, mixing *semantic decision ambiguity* with *phrasing synonyms* (`"Yes"` vs. `"True"` vs. `"Certainly"`). `dgem` slices the canvas logits strictly over the policy-allowed single-token options $\mathcal{V}_m$, ensuring 100% of $H_m$ measures epistemic competition between the declared choices.
2. **[Distributional Discrete Regression](glossary.md#distributional-discrete-regression-score-slots) Unifies `score` (Regression) and `choice` (Classification)**:
   Instead of fitting a separate Gaussian variance head $(\mu, \sigma^2)$ or pinball-loss quantile head for regression, `dgem` evaluates every `score` slot (`1..5` or `0.0..0.9`) as a **Histogram Distribution** over its numeric scale levels $v_1, \dots, v_L$:
   $$\hat{y}_m = \mathbb{E}[v] = \sum_{k=1}^L v_k \cdot p_{m,k}, \qquad \text{Var}(v) = \sum_{k=1}^L p_{m,k}\bigl(v_k - \mathbb{E}[v]\bigr)^2$$
   Both the continuous expected value $\mathbb{E}[v]$, the ordinal variance $\text{Var}(v)$, and the normalized entropy $\tilde{H}_m = H(p) / \ln L$ are derived from the **exact same single-pass softmax vector** $p_{m,k}$.
3. **Capacity Normalization ($\tilde{H}_m = H_m / \ln|\mathcal{V}_m| \in [0, 1]$) Removes Class-Count Drift**:
   Dividing $H_m$ by the slot's maximum possible entropy $\ln|\mathcal{V}_m|$ converts task-dependent nats into a dimensionless $[0, 1]$ information-efficiency ratio. In `EXP-05b`, **one universal threshold ($\tau = 0.16$)** applied across **11 heterogeneous public datasets** (`|V| = 2` boolean gates, `|V| = 3` NLI, `|V| = 12` toxicity, `|V| = 26` banking/emotions) achieved **100.0% early-exit precision (`33/33`)** and **98.0% overall accuracy (`49/50`)** with zero per-task threshold tuning.
4. **Prior-Conditioned Escalation (A Bayesian Proposal, Not a Blind "Punt")**:
   When $\tilde{H}_m \ge 0.16$, Stage 1 does not discard its computation. It injects its restricted-softmax distribution (`{entailment: 94.2%, neutral: 4.9%, contradiction: 0.9%}`) into Stage 2—either `--cascade-self-think 256` on the **same `dgemma` checkpoint** or a frontier model—turning open-ended generation into targeted verification of the surviving candidates.

---

## 5. Architectural FAQ: Can Dual-Encoders (`GTR`) + `TabPFN` Replace a Decision Model, or Do You Need Test-Time Compute?

Engineers from search, retrieval, and tabular ML backgrounds frequently ask a foundational design question:

> *"Could the goal of a fast, reasoning-capable classifier be achieved without a generative model—specifically by pairing a **GTR-style Dual Encoder** (`Sentence-T5`) with a **TabPFN / TabFM** zero-shot tabular classification foundation model? Or do you strictly need a decoder and test-time compute (`think > 0`) to pull off reasoning?"*

> [!TIP]
> **TL;DR: The 30-Second Architectural Answer**
> 1. **Why `GTR Dual-Encoder + TabPFN` Hits an Early Information Wall**: A Dual Encoder compresses your entire input document into a single fixed vector $u \in \mathbb{R}^d$ **before** reading your policy rules or hypothesis. That pooling step permanently destroys token-to-token relational alignment (such as negation scope, numerical bounds like `50–75% < 100%`, or SQL parameter tampering in `AgentDrift`). Feeding those pooled embeddings into `TabPFN` cannot recover fine-grained relational bindings already lost in $u$—and `TabPFN` further requires **labeled support rows ($N_{\text{support}} > 0$)** rather than zero-shot instructions.
> 2. **When You Do *NOT* Need a Decoder (`think=0` in $O(1)$ — `86%–90%` of Tasks)**: Full **token-level cross-attention** across a 26B-A4B model (`dgem` with `steps=1, think=0`) solves 1-hop relational policy grounding (`AgentDrift` `100%`, `deepset/prompt-injections` `100%`, `LLM-AggreFact` `100%`, `MS MARCO` `100%`) in **a single `458–712 ms` forward pass** without generating a single scratchpad token.
> 3. **When You *DO* Need a Decoder + Test-Time Compute (`think > 0` — `anli-01..03`)**: When a decision hinges on synthesizing an **unwritten intermediate variable** (such as computing `2015 + 4 = 2019` and checking `2019 > 2018` in `anli-02`), constant-depth circuit bounds ($\mathsf{TC}^0$) prevent *any* single-pass model (`GTR`, `TabPFN`, or `dgemma [think=0]`) from reliably chaining the arithmetic. Because `dgemma` is a unified architecture, its **normalized entropy gate ($\tilde{H}_m \ge 0.160$)** detects those exact multi-hop traps and triggers `think > 0` (conditioned on Pass-1 priors) only on the **34% of queries** that need a scratchpad—reaching **`98.0%` accuracy (`49/50`)**.

### The Operative Decoder Ring (5 Core Concepts in Plain English)

| Term | 10-Word Plain-English Mental Model | Canonical Example |
| :--- | :--- | :--- |
| **[Dual Encoder (`GTR` / `T5`)](glossary.md#dual-encoder-gtr--sentence-t5)** | Compresses input and label into two separate vectors, then compares. | Semantic search & topical intent (`Banking77` similarity). |
| **[Tabular FM (`TabPFN` / `TabFM`)](glossary.md#tabpfn--tabular-foundation-models)** | Predicts a spreadsheet column by attending to labeled example rows. | Few-shot classification over numerical/embedding feature grids. |
| **[Cross-Attention Canvas (`dgem`)](glossary.md#bidirectional-canvas-attention)** | Every input word directly inspects every policy rule and slot. | Zero-shot `AgentDrift` security audit & `LLM-AggreFact` grounding. |
| **[Normalized Entropy ($\tilde{H}_m$)](glossary.md#cardinality-normalized-entropy)** | A universal `0.0–1.0` uncertainty gauge adjusted for option count. | Early-exiting `b77-01` ($\tilde{H}=0.158$) while escalating `anli-01` ($\tilde{H}=0.168$). |
| **[Test-Time Compute (`think > 0`)](glossary.md#fixed-depth-circuits-tc0-vs-test-time-compute)** | Scratchpad tokens generated only when a problem needs multi-step math. | Solving `2015 + 4 = 2019 > 2018` in `anli-02` (`--cascade-self-think 256`). |

### Architectural Comparison Matrix

| Capability / Dimension | `GTR Dual-Encoder` + `TabPFN / TabFM` | Fine-Tuned Cross-Encoder (`DeBERTa-v3`) | `dgem` Single-Pass Canvas (`steps=1, think=0`) | `dgem` Prior-Guided Cascade (`EXP-05b`, `think=0 → think>0`) |
| :--- | :--- | :--- | :--- | :--- |
| **Token-to-Policy Cross-Attention** | ❌ **No** (Late pooling into $u \in \mathbb{R}^d$) | ✅ **Yes** (Full cross-attention) | ✅ **Yes** (Full 26B-A4B cross-attention) | ✅ **Yes** (Full 26B-A4B cross-attention) |
| **Zero-Shot Policy Onboarding** | ⚠️ **Partial** (`TabPFN` requires $N>0$ support rows) | ❌ **No** (Requires fine-tuning per head) | ✅ **Yes** (`0 s` via `.json.tmpl`, $N=0$ rows) | ✅ **Yes** (`0 s` via `.json.tmpl`, $N=0$ rows) |
| **Joint Multi-Slot Readout (`s_1 <-> s_2`)** | ❌ **No** (1 target column per pass) | ❌ **No** (Independent linear heads) | ✅ **Yes** (`boolean` + `choice` + `score` in 1 pass) | ✅ **Yes** (`boolean` + `choice` + `score` in 1 pass) |
| **1-Hop Relational Grounding (`AgentDrift`, `AggreFact`)** | ⚠️ **Brittle** (Pooling loses parameter/negation scope) | ✅ **Strong** (If fine-tuned on domain) | ✅ **100.0%** (`7/7` `AgentDrift`, `2/2` `AggreFact`) | ✅ **100.0%** (`7/7` `AgentDrift`, `2/2` `AggreFact`) |
| **Latent Multi-Hop Arithmetic (`ANLI-R3` `anli-01..03`)** | ❌ **Fails** (Fixed circuit depth, no scratchpad) | ❌ **Fails** (Fixed circuit depth, no scratchpad) | ❌ **0.0% (`0/3`)** (Single-pass $\mathsf{TC}^0$ limit) | ✅ **100.0% (`3/3`)** ($\tilde{H}_m \ge 0.16$ triggers `think>0` + Priors) |
| **Overall 50-Case Calibration Suite Accuracy** | — | — | **86.0% (`43/50`)** | **98.0% (`49/50`)** ⭐ |
| **Mean Wall-Clock Latency (Cloud Run L4)** | `~15–45 ms` | `~15–30 ms` | **`712 ms`** (`458.9 ms` 3-slot triage) | **`1,259 ms` blended** (`66%` exit @ `712 ms`) |
