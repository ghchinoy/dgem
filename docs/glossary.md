# Glossary & Mental Models

New to **DiffusionGemma (`dgem`)**? Because `dgem` sits at the intersection of **Classical Search/Tabular ML**, **Formal Language Automata**, and **Discrete Diffusion Generative Models**, engineers arriving from different specialties often bring different terminology for overlapping ideas.

Use this page as a **Decoder Ring** to translate between disciplines.

---

## 1. Quick-Start Mental Model Matrix (By Reader Background)

| If You Come From... | Think of `dgem` (`steps=1, think=0`) As... | Think of `.json.tmpl` Templates As... | Think of `EXP-05b` Entropy Cascade As... |
| :--- | :--- | :--- | :--- |
| **Software / Platform Engineering** | A sub-second (`~460–712 ms`) type-safe RPC call that returns guaranteed JSON booleans, enums, and scores without hallucinated syntax. | **Policy-as-Code** (like OpenPolicyAgent `.rego` or JSON Schema, but for semantic natural language & code rules). | An automatic circuit breaker that routes `66%` of easy traffic on the fast path (`712 ms`) and escalates `34%` of hard edge cases. |
| **Search, Ranking & Tabular ML (`GTR`, `DeBERTa`, `TabPFN`)** | A **zero-shot multi-head Cross-Encoder** where all target heads (`slot_1 <-> slot_2`) mutually attend in 1 forward pass with zero training rows ($N=0$). | A declarative classification head compiler—changing a label set takes `0 seconds` instead of relabeling + retraining. | A scale-invariant **Selective Prediction / Abstention Gate** ($\tilde{H}_m = H_m / \ln|\mathcal{V}_m|$) paired with Bayesian prior forwarding. |
| **LLM / GenAI Infrastructure (`vLLM`, `CoT`)** | Replacing an $O(L)$ sequential token generation loop with an **$O(1)$ parallel block-diffusion logit readout** over masked slot positions. | Constrained single-token decoding (`[A–Z]`, `yes/no`, `1..5`) evaluated simultaneously across all slots. | **Adaptive Test-Time Compute (`think=0 -> think>0`)**: only spending reasoning scratchpad tokens when Pass-1 epistemic entropy spikes. |

---

## 2. Decision Model & Canvas Primitives

### Discrete Diffusion Slot Readout
* **In Plain English**: Reading the exact probability of every valid answer (`yes`/`no`, `A`–`Z`, `1`–`5`) directly from masked blanks in a single forward pass instead of generating words one by one.
* **Under the Hood**: Given a causal prompt prefix and a seeded canvas containing masked slot positions $\langle s_1, \dots, s_M \rangle$, the engine runs 1 denoising step (`steps=1, think=0`), slices the raw vocabulary logits $z_{m,k}$ restricted to the valid single-token options $k \in \mathcal{V}_m$, and normalizes via softmax.
* **Where You See It in `dgem`**: [`pkg/client/client.go`](../pkg/client/client.go) (`ParseStructuredContentWithLogprobs`) and `structured_server.py`.

### Bidirectional Canvas Attention
* **In Plain English**: Every input word directly inspects every policy rule and every decision blank—and the decision blanks inspect *each other* simultaneously (`slot_1 <-> slot_2`).
* **Under the Hood**: Standard LLM decoders apply a lower-triangular causal mask ($t$ can only see $<t$). DiffusionGemma uses a hybrid attention mask (`TRITON_ATTN` in vLLM): causal over the prompt prefix (for KV-cache reuse) and **all-to-all bidirectional** over the 256-token diffusion canvas.
* **Where You See It in `dgem`**: Enables joint 3-slot triage (`urgent` + `team` + `sentiment`) in **458.9 ms** (`EXP-01`).

### Policy-as-Template (`.json.tmpl`)
* **In Plain English**: A declarative JSON file where you define decision questions (`boolean`, `choice`, `score`) and natural-language rubrics that execute immediately with zero model training.
* **Under the Hood**: Go `text/template` files compiled by `dgem decide -t` into structured slot schemas and single-token option maps (`[A–Z]`).
* **Where You See It in `dgem`**: [`templates/`](../templates/) (`templates/calibration/*.json.tmpl`, `templates/secops_conditional_dag.json.tmpl`).

### Conditional Policy DAG (`depends_on` & `ask_if`)
* **In Plain English**: A multi-stage decision flowchart where follow-up questions are only evaluated if an upstream gate question resolves to `true` (or a specific option).
* **Under the Hood**: Topological sorting in `pkg/schema` partitions questions into stages. If Stage 1 (`is_prompt_injection`) evaluates to `false`, downstream forensic slots are pruned in **1 pass** (`682 ms`), saving 50% of compute on benign traffic.
* **Where You See It in `dgem`**: [Experiment `EXP-06`](experiments/exp-05-roadmap-cascades-and-dags.md#experiment-exp-06-conditional-policy-dags-depends_on--ask_if).

---

## 3. Uncertainty & Cascade Telemetry

### Relative Tie-Detection vs. Target-Domain Probability Calibration
* **In Plain English**: `dgem`'s single-pass logprob scores tell you **whether the model is torn between your template choices** (relative routing ambiguity), *not* the real-world base rate of how often a class appears in your database.
* **Why This Matters**: True statistical calibration ($P(\text{Gold}=A \mid \hat{p}=0.80) = 0.80$) depends on the target environment's class prior $P_{\text{target}}(Y)$ and always requires post-hoc target data (Platt scaling, temperature scaling, or conformal prediction). What `dgem` provides zero-shot in 1 forward pass is a **tie-detector over the user-supplied option letters (`A..Z`)**—eliminating the $10\times\text{–}50\times$ token-cost multiplier of multi-sample autoregressive confidence rollouts. Order bias can also make a tie look decisive; see [IDC](confidence-beyond-shannon.md) for the zero-label corrections `dgem` applies before gating.

### Distributional Discrete Regression (`score` Slots)
* **In Plain English**: Turning continuous regression (like a `1..5` severity score) into a probability histogram over discrete levels so that classification confidence and regression variance come out of the exact same softmax formula.
* **Under the Hood**: Instead of a point-estimate MSE head or quantile pinball loss, `dgem` evaluates the restricted-softmax probabilities $p_{m,1}, \dots, p_{m,L}$ over the discrete numeric bins $v_1, \dots, v_L$ (`pkg/client/client.go`), yielding the continuous expected value $\mathbb{E}[v] = \sum_k v_k p_{m,k}$, ordinal variance $\text{Var}(v) = \sum_k p_{m,k}(v_k - \mathbb{E}[v])^2$, and normalized entropy $\tilde{H}_m = H(p) / \ln L$ in 1 pass.
* **Where You See It in `dgem`**: Every `score` primitive in `.json.tmpl` templates (`sentiment`, `risk_score`, `severity`).

### Cardinality-Normalized Entropy
* **In Plain English**: A universal **`0.0` to `1.0` uncertainty meter** that adjusts for how many answer choices a question has (`2` options vs. `26` options).
* **Under the Hood**: Raw Shannon entropy $H_m = -\sum_{k \in \mathcal{V}_m} p_{m,k} \ln p_{m,k}$ has a theoretical maximum of $\ln|\mathcal{V}_m|$ (`0.693 nats` for binary vs. `3.258 nats` for 26-way `choice`). Dividing by $\ln|\mathcal{V}_m|$ yields the dimensionless normalized entropy:
  $$\tilde{H}_m = \frac{H_m}{\ln|\mathcal{V}_m|} \in [0, 1]$$
* **Where You See It in `dgem`**: `--normalize-entropy --cascade-threshold 0.16` in `dgem bench-calibration` (`cmd/bench_calibration.go`).
* **In Decision Studio**: shown as **Hesitation** (0–100%) on every answer: under 16% is *Clear*, 16–50% *Somewhat unsure*, above 50% *Very unsure*. Hover to see the raw entropy in nats. The 16% line matches the `EXP-05b` cascade gate.

### Multi-Slot Scale Inversion
* **In Plain English**: The bug that happens when you apply a single raw entropy cutoff (like `0.35 nats`) to questions with different numbers of choices—causing confident 26-choice questions to falsely escalate while uncertain 3-choice questions slip through!
* **Under the Hood**: On `b77-01` (`Banking77`, $|\mathcal{V}|=26$), tiny residual probabilities across 25 classes yield $H = 0.5162\text{ nats}$ even when `dgemma` is **88.6% confident and right** ($\tilde{H} = 0.158 < 0.160$). Meanwhile, on `anli-01` and `anli-02` ($|\mathcal{V}|=3$), entropy spikes **2.5×–3.3×** above baseline to $H = 0.1847$ and $0.2464\text{ nats}$—which is below `0.35 nats` in raw units, but **above `0.160` once normalized by $\ln(3)$** ($\tilde{H} = 0.168$ and $0.224$).
* **Where You See It in `dgem`**: Solved in [Experiment `EXP-05b`](experiments/exp-05-roadmap-cascades-and-dags.md), lifting `ANLI-R3` from `33.3%` $\to$ `100.0%` (`3/3`).

### Pass-1 Slot Prior Forwarding
* **In Plain English**: Handing Stage 2 not just the original question, but also Stage 1's exact probability breakdown (`{entailment: 94.2%, neutral: 4.9%, contradiction: 0.9%}`) as a diagnostic clue to double-check.
* **Under the Hood**: `formatTier1PriorBlock` (`cmd/bench_calibration.go`) injects `[TIER-1 DISCRETE DIFFUSION PRIOR TELEMETRY]` sorted by restricted-softmax probability descending, acting as a **cognitive counter-anchor** that forces Stage 2 to verify why `dgemma`'s entropy spiked before committing to a label.
* **Where You See It in `dgem`**: `benchmarks/results_calibration_cascade_normalized.json` (`98.0%` overall accuracy, `49/50`).

---

## 4. Order Bias & Invariant Decision Calibration (`IDC`)

These terms come from [Confidence Beyond Shannon: Invariant Decision Calibration (IDC)](confidence-beyond-shannon.md), which walks through them with a worked example.

### Invariant Decision Calibration (`IDC`)
* **In Plain English**: A set of cheap checks and corrections that make a confidence score reflect *the question*, not *where each answer happened to be listed*, and that flag decisions whose answer changes when the list order changes.
* **Why "Invariant"**: Reordering the options doesn't change the question, so an ideal decision (and its confidence) shouldn't change either.
* **Where You See It in `dgem`**: `--null-prior-debias`, `--dual-mirror` (`dgem decide`, `bench-calibration`, `bench-decision-index`), `bench-permutation` (`EXP-13`).

### Ballot-Order (Primacy) Bias / Null Prior $p_0$
* **In Plain English**: Like undecided voters who tick the first name on a ballot, the model leans toward whichever option is listed first (`A`). Given a blank question with meaningless options, `DiffusionGemma` still picks `A` 88% (2 options), 78% (3), or 49% (4) of the time.
* **Under the Hood**: $p_0(k)$ is the model's slot distribution on a content-free input. It estimates the position term $b_{\text{pos}}(k)$ in $z = s(\text{option}) + b_{\text{pos}}(k) + \epsilon$.

### Null-Prior De-Biasing ("Tare the Scale")
* **In Plain English**: Weigh the empty bowl first, then subtract it. `dgem` divides out the model's built-in preference for each slot before reporting confidence. It needs no labeled data.
* **Under the Hood**: $\tilde{p}_k \propto p_k / p_0(k)^{\alpha}$, with $\alpha \in [0,1]$ controlling correction strength (`--prior-alpha`, default `0.5`). Related prior work: *contextual calibration* (Zhao et al., 2021).
* **Caveat**: It removes the *average* slot habit, not input-specific order effects. In `EXP-13` it improved Brier score but increased flips under other orderings (12.5% → 25%). In `EXP-14` it helped on the 50-item suite but made calibration worse on the 231-item JevBench set, so treat it as suite-dependent.

### Dual-Mirror Canvas
* **In Plain English**: Print the ballot twice on the same page, once in reverse order, and check that both votes agree. Because a diffusion model fills every blank at once, the second copy costs no extra forward pass.
* **Under the Hood**: For each `choice` slot, `dgem` adds `<id>__rev` with options $[o_K \dots o_1]$, reads both in one pass, maps them back to option names, and merges them (currently 70% forward / 30% reversed with a forward-priority rule).
* **Caveats**: The two slots can see each other on the canvas, so they are not independent readings. Reversal is only one reordering: `perm_06` flips under a cyclic shift but passes the mirror check. In `EXP-14` the extra slot lowered forward accuracy on JevBench (189 → 163–169), and the original slot id `__mirror_rev` degraded readings further (renamed `__rev`). Treat it as a research diagnostic.

### Mirror TVD (Total Variation Distance)
* **In Plain English**: How far apart the forward and reversed readings are, from `0` (identical) to `1` (completely different). Near 0 means order didn't matter for this input. A large value means the confidence depends on the layout.
* **Under the Hood**: $\text{TVD} = \tfrac12 \sum_k |p^{\text{fwd}}_k - p^{\text{rev}}_k|$. Clear-cut `EXP-13` items typically score below `0.01`; `perm_08` scored `0.258` while its single reading claimed 99.9%.
* **Status**: Computed in `bench-permutation`. Not yet returned by `dgem decide`, `dgem serve`, MCP, or the Studio.

### Cyclic JSD (Permutation Mutual Information)
* **In Plain English**: The expensive, thorough version of the mirror: ask the question once per rotation of the option list and measure how much the answers disagree.
* **Under the Hood**: The Jensen–Shannon divergence across $K$ cyclic orderings estimates $I(Y; \Pi \mid X)$, the information the option order carries about the answer. It costs $K$ forward passes (`EXP-13A`).

### Temperature Scaling ("Humility Dial")
* **In Plain English**: One dial that makes over-confident scores more modest (or under-confident ones bolder) without changing which answer wins.
* **Under the Hood**: $p_k(T) \propto p_k^{1/T}$. $T > 1$ softens. $T^*$ is **fitted on labeled examples** (Guo et al., 2017), so the improvement is only trustworthy when measured on data not used for fitting.

### Expected Calibration Error (`ECE`)
* **In Plain English**: "When the model says 80%, is it right about 80% of the time?" ECE is the average gap between stated confidence and actual accuracy, so `0` is perfect.
* **Under the Hood**: Predictions are grouped into 10 confidence bins; ECE $= \sum_b \frac{n_b}{N}\,|\text{acc}_b - \text{conf}_b|$. It's noisy on small datasets, especially when most predictions fall in the top bin.

### Brier Score
* **In Plain English**: A penalty for being confidently wrong *and* for being needlessly unsure when right. Lower is better. A perfect, fully confident forecaster scores `0`.
* **Under the Hood**: $\text{Brier} = \frac1N \sum_i \sum_k (p_{i,k} - y_{i,k})^2$, where $y$ is the one-hot gold label.

---

## 5. Comparative ML Architectures

### Dual Encoder (`GTR` / `Sentence-T5`)
* **In Plain English**: A bi-encoder architecture that compresses the input text into one vector $u$ and the label description into another vector $v_k$ *independently*, then compares the two vectors at the very end.
* **Under the Hood**: Because $u = E_x(x) \in \mathbb{R}^d$ is computed **before** the model sees the policy rules or hypothesis $c_k$, token-to-token alignment (like checking whether a specific SQL argument matches an allowlist or comparing `50–75%` against `100%`) is lost during vector pooling (**Late Interaction Bottleneck**).
* **Where You See It in `dgem`**: Contrasted with `dgem`'s early all-to-all cross-attention in [Discrete Diffusion vs. Autoregression (§5)](architecture.md#5-architectural-faq-can-dual-encoders-gtr--tabpfn-replace-a-decision-model-or-do-you-need-test-time-compute).

### `TabPFN` & Tabular Foundation Models
* **In Plain English**: A foundation model pre-trained on millions of synthetic spreadsheets that predicts a missing target column by attending across labeled example rows (`in-context learning` for tables).
* **Under the Hood**: `TabPFN` approximates Bayesian posterior inference $P(y_{\text{test}} \mid X_{\text{test}}, X_{\text{train}}, y_{\text{train}})$ in a single forward pass. However, it requires **labeled support rows ($N_{\text{support}} > 0$)** in its context window and operates on pre-extracted tabular columns—meaning pairing `GTR + TabPFN` still suffers from `GTR`'s pooling bottleneck and cannot compile zero-shot (`N=0`) natural-language `.json.tmpl` policies.

### Fixed-Depth Circuits ($\mathsf{TC}^0$) vs. Test-Time Compute
* **In Plain English**: Why a single forward pass (`think=0`) can verify direct relational facts in `712 ms`, whereas multi-step mental arithmetic (`2015 + 4 = 2019 > 2018`) requires generating scratchpad tokens (`think > 0`).
* **Under the Hood**: A transformer with fixed layer depth $L$ and no scratchpad generation (`think=0`) is bounded by the circuit complexity class $\mathsf{TC}^0$. When a contradiction depends on an intermediate state not present in the input text (`anli-02`'s latent year `2019`), test-time compute (`--cascade-self-think 256` or Tier-2 reasoning) allocates working-memory tokens to materialize the intermediate state.

---

## 6. Spatial Grounding & Vision-Language Terminology (`EXP-09`)

### `DETR` Object Queries (Detection Transformer)
* **In Plain English**: Instead of scanning an image with thousands of sliding-window guesses and filtering duplicates afterward (`Non-Maximum Suppression`), `DETR` creates a fixed number of parallel "empty parking spots" (**Object Queries**—e.g., `obj1` and `obj2`). Because all query slots attend to the image and to **each other simultaneously**, `obj2` sees that `obj1` already claimed the left object and automatically claims the right object in a single pass.
* **Under the Hood**: In `dgem`, `templates/multimodal/bbox_multi_object_detr.json.tmpl` places `obj1_[ymin,xmin,ymax,xmax]` and `obj2_[ymin,xmin,ymax,xmax]` on the same bidirectional `[MASK]` canvas (`reads=1`), allowing the query slots to co-adapt without autoregressive left-to-right drift.
* **Where You See It in `dgem`**: `dgem bench-bbox` (`bbox-t3-01-detr-dual-buttons`, `bbox-t3-02-detr-stacked-banner-cta`).

### Softmax Expectation (`DFL` / Distribution Focal Loss) Sub-Bin Regression
* **In Plain English**: Turning 21 coarse `5%` coordinate bins (`00, 05, 10, ..., 100`) into a smooth, continuous coordinate (`32.4%`) by taking the **probability-weighted average** across all 21 bins rather than picking only the single winning bin (`argmax`).
* **Under the Hood**: When an edge lies at `32.5%`, `dgemma` splits probability mass between bin `30` (`P=0.50`) and bin `35` (`P=0.50`). Discrete `argmax` suffers a `2.5%` quantization penalty (or collapses narrow objects like `008.png` onto `xmin=55, xmax=55` $\rightarrow$ `0.000 IoU`), whereas Softmax Expectation:
  $$\hat{c}_m = \sum_{k=0}^{20} (5k) \cdot P(\text{slot}_m = \text{bin}_k)$$
  recovers the continuous coordinate (`+8.75%` `mIoU` across `EXP-09` and `0.000` $\rightarrow$ `0.504 IoU` on `008.png`).
* **Where You See It in `dgem`**: `cmd/bench_bbox.go` (`computeEdgeMetrics`).

### Per-Edge Occlusion Entropy ($\tilde{H}_{\text{edge}}$)
* **In Plain English**: Traditional object detectors give you a single confidence number for an entire box, hiding *which side* of the object is blocked. Because `dgem` evaluates `ymin`, `xmin`, `ymax`, and `xmax` as 4 independent 21-bin distributions, an object covering the bottom edge causes entropy to spike **specifically on `ymax`** (`1.37×` higher on live Cloud Run `dgemma`) while the 3 visible edges stay sharp.
* **Under the Hood**: Computed per edge $m \in \{\text{ymin}, \text{xmin}, \text{ymax}, \text{xmax}\}$ as $\tilde{H}_m = H_m / \ln(21) \in [0, 1]$.
* **Where You See It in `dgem`**: `dgem bench-bbox --annotate` and `scratch/render_bbox_results.py`.

