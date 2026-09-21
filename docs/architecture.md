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

### How `dgem` Computes Confidence & Entropy from Template Logprobs (Step-by-Step)

In standard autoregressive LLM pipelines, estimating whether the model is confident on a custom task—without generating long Chain-of-Thought rationales or running $N=10\text{–}20$ Monte Carlo rollouts (self-consistency)—multiplies token costs by orders of magnitude.

`dgem` avoids that token-cost explosion by turning your `.json.tmpl` policy into a **single-token `logprob` tie-detector** across 4 concrete steps:

1. **Step 1 — Map User-Supplied Classes to Single Letters (`A`, `B`, `C`...) in the Prompt**:
   When you define custom classes in a template (even for a domain not in the model's training distribution), `dgem` formats them into a single-letter legend in the prompt prefix and places **one masked blank (`@`)** per question on the diffusion canvas:
   ```text
   Prompt Prefix:
     Slot 'intent' options:
       A = billing_dispute
       B = account_compromise
       C = feature_request

   Seeded Diffusion Canvas (1 token per slot):
     intent: @
   ```
2. **Step 2 — Run 1 Forward Pass (`think=0`) & Read `logprobs` of *Only* Those Letters**:
   Instead of generating free-form text, `dgem` executes **1 forward pass** (`~460–712 ms`) and inspects the raw token logits at that exact `@` blank. It discards the other ~255,997 words in the vocabulary and runs a softmax strictly over the valid letters (`A`, `B`, `C`) you supplied:
   $$p_A, p_B, p_C = \text{softmax}(z_A, z_B, z_C)$$
   *(For numeric `score` slots like `1..5`, it does the exact same thing over the digit tokens `'1'..'5'`, computing the weighted average $\mathbb{E}[v] = \sum_{k=1}^5 k \cdot p_k$ and spread from those 5 probabilities).*
3. **Step 3 — Measure Whether the Top Letters Are in a Close Race ($H_m$)**:
   - If the restricted probabilities are **`{A: 97.5%, B: 1.5%, C: 1.0%}`**, letter `A` dominates ($H_m = 0.13\text{ nats}$). We take `A` immediately and pay **zero** generation tokens.
   - If the restricted probabilities are **`{A: 54.0%, B: 42.0%, C: 4.0%}`**, the model's attention is **torn** between `A` and `B` ($H_m = 0.82\text{ nats}$).
4. **Step 4 — Divide by $\ln(\text{Number of Choices})$ to Scale the "Tie Meter" from `0.0` to `1.0` ($\tilde{H}_m$)**:
   Why can't we use the same raw entropy cutoff ($H_m$) for a 2-choice `yes/no` question and a 26-choice `A..Z` question? Because a flat dead tie between 2 choices has a maximum entropy of $\ln(2) = 0.693$, while a flat tie between 26 choices has a maximum entropy of $\ln(26) = 3.258$. Dividing by $\ln(\text{number of choices})$:
   $$\tilde{H}_m = \frac{H_m}{\ln(\text{number of choices})} \in [0, 1]$$
   scales our "tie meter" onto `0.0` (one letter dominates) to `1.0` (dead tie) regardless of how many classes you put in your template. When $\tilde{H}_m \ge 0.16$, `dgem` escalates the query to a reasoning pass (`think > 0` or Tier-2 LLM) **and passes along the Pass-1 letter breakdown (`{A: 54%, B: 42%}`)** so the reasoning model knows which two candidates to disambiguate.

> [!CAUTION]
> **Important Statistical Distinction: Relative Tie-Detection vs. Target-Domain Base-Rate Calibration**
> In statistics and classical ML, **true probability calibration** means that when a classifier outputs `0.80` for class `A`, class `A` empirically occurs `80%` of the time in your production environment.
>
> **No zero-shot model can provide true out-of-the-box base-rate calibration on an unseen task without target-domain data**, because the model does not know your production class priors $P_{\text{target}}(Y)$ (e.g., whether a rare clinical or fraud event happens in `0.1%` or `30%` of cases). If your application requires calibrated frequentist probabilities tied to production base rates, you still need a post-hoc calibration layer fit on labeled target examples (such as Platt/temperature scaling, isotonic regression, or conformal prediction).
>
> What `dgem`'s normalized entropy $\tilde{H}_m$ provides zero-shot is **Relative Routing Ambiguity (Tie-Detection)**—answering *"Given the options in the template, does one choice clearly win in 1 forward pass, or are the top choices competing?"*—allowing you to early-exit **66%** of traffic in a single pass (`712 ms`) and reserve expensive autoregressive reasoning tokens for the **34%** of inputs where the choices are in contention.

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
