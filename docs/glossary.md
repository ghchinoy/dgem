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

### Cardinality-Normalized Entropy
* **In Plain English**: A universal **`0.0` to `1.0` uncertainty meter** that adjusts for how many answer choices a question has (`2` options vs. `26` options).
* **Under the Hood**: Raw Shannon entropy $H_m = -\sum_{k \in \mathcal{V}_m} p_{m,k} \ln p_{m,k}$ has a theoretical maximum of $\ln|\mathcal{V}_m|$ (`0.693 nats` for binary vs. `3.258 nats` for 26-way `choice`). Dividing by $\ln|\mathcal{V}_m|$ yields the dimensionless normalized entropy:
  $$\tilde{H}_m = \frac{H_m}{\ln|\mathcal{V}_m|} \in [0, 1]$$
* **Where You See It in `dgem`**: `--normalize-entropy --cascade-threshold 0.16` in `dgem bench-calibration` (`cmd/bench_calibration.go`).

### Multi-Slot Scale Inversion
* **In Plain English**: The bug that happens when you apply a single raw entropy cutoff (like `0.35 nats`) to questions with different numbers of choices—causing confident 26-choice questions to falsely escalate while uncertain 3-choice questions slip through!
* **Under the Hood**: On `b77-01` (`Banking77`, $|\mathcal{V}|=26$), tiny residual probabilities across 25 classes yield $H = 0.5162\text{ nats}$ even when `dgemma` is **88.6% confident and right** ($\tilde{H} = 0.158 < 0.160$). Meanwhile, on `anli-01` and `anli-02` ($|\mathcal{V}|=3$), entropy spikes **2.5×–3.3×** above baseline to $H = 0.1847$ and $0.2464\text{ nats}$—which is below `0.35 nats` in raw units, but **above `0.160` once normalized by $\ln(3)$** ($\tilde{H} = 0.168$ and $0.224$).
* **Where You See It in `dgem`**: Solved in [Experiment `EXP-05b`](experiments/exp-05-roadmap-cascades-and-dags.md), lifting `ANLI-R3` from `33.3%` $\to$ `100.0%` (`3/3`).

### Pass-1 Slot Prior Forwarding
* **In Plain English**: Handing Stage 2 not just the original question, but also Stage 1's exact probability breakdown (`{entailment: 94.2%, neutral: 4.9%, contradiction: 0.9%}`) as a diagnostic clue to double-check.
* **Under the Hood**: `formatTier1PriorBlock` (`cmd/bench_calibration.go`) injects `[TIER-1 DISCRETE DIFFUSION PRIOR TELEMETRY]` sorted by restricted-softmax probability descending, acting as a **cognitive counter-anchor** that forces Stage 2 to verify why `dgemma`'s entropy spiked before committing to a label.
* **Where You See It in `dgem`**: `benchmarks/results_calibration_cascade_normalized.json` (`98.0%` overall accuracy, `49/50`).

---

## 4. Comparative ML Architectures

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
