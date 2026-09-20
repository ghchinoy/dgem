# The Journey to Decision Models

For decades, software engineers and machine learning practitioners had to choose between two extremes when building classification and decision systems: **fast, rigid classical models** (like Naive Bayes, Logistic Regression, DeBERTa encoders, and Finite-State Transducers) or **slow, expensive autoregressive Large Language Models** (like GPT-4 and Gemini).

**DiffusionGemma** introduces a third paradigm: **Discrete Diffusion Decision Models**.

This primer explains the architectural journey from classical machine learning to discrete block diffusion, detailing why decision models are needed, how their mechanics differ from both statistical counting and next-token prediction, and how declarative `.json.tmpl` templates act as executable **Policy-as-Code**.

---

## 1. The Three Eras of Decision Systems

```
Era 1: Classical Statistical ML   --> Era 2: Classical Symbolic NLP --> Era 3: Autoregressive GenAI --> The Emerging Era: Decision Models
(Naive Bayes, SVM, XGBoost)           (N-grams, HMMs, WFSTs)            (GPT, Gemini, LLaMA)             (Discrete Block Diffusion / dgem)
Bag-of-Words / Zero Context           Local Sliding Window              Causal Sequential Next-Token     Non-Autoregressive Canvas Readout
Microsecond / Cheap CPU               Millisecond / C++ Rulebooks       Multi-Second / Heavy GPU Loops   Sub-Second / Guaranteed Schema
```

### Era 1: Classical Statistical ML & Discriminative Encoders (1960s–Present)
* **Core Algorithms**: Naive Bayes, Logistic Regression, SVMs, XGBoost, and Fine-Tuned Encoder Heads (`BERT`, `DeBERTa-v3`).
* **Strength**: Microsecond to 20 ms inference, minimal memory footprints, and fixed output schemas.
* **Fatal Flaw**: **Zero-Shot Rigidity & Independent Heads**. Classical statistical ML cannot model word order or negation (*"This is NOT an outage"*). Fine-tuned encoder classifiers understand context, but every policy change (adding a 4th severity level or a new department) requires curating thousands of labeled examples, retraining weights, and redeploying model binaries. Furthermore, evaluating 3 questions requires 3 separate classification heads that cannot attend to each other's predictions.

### Era 2: Classical Symbolic NLP & Automata (1990s–Present)
* **Core Algorithms**: Regular Grammars, Hidden Markov Models (HMMs), Weighted Finite-State Transducers (WFSTs like OpenFst, Google Sparrowhawk, NVIDIA NeMo).
* **Strength**: Extremely fast (1–8 ms in C++), deterministic, zero hallucinations, and mathematically verifiable.
* **Fatal Flaw**: **Local context horizon (1–3 token sliding window)**. WFSTs cannot build full-sentence dependency parse trees. As language complexity increases, grammar rulebooks explode into combinatorial conflicts (e.g., tuning a title rule for *"Dr. Smith"* causes street thoroughfares like *"Ocean Dr."* to expand to *"Ocean doctor"*).

### Era 3: Autoregressive Generative AI (2018–Present)
* **Core Algorithms**: Causal Transformer Decoders (GPT-4, Gemini, LLaMA, Claude).
* **Strength**: Deep semantic reasoning, world knowledge, and zero-shot instruction following.
* **Fatal Flaws for Structured Decisions**:
  1. **The Sequential Token Tax**: To output a single word like `"billing"`, the model must serially generate thought rationales, markdown tags, or JSON formatting loops, consuming **2,000–17,500 ms** per request.
  2. **Formatting & Syntax Drift**: Autoregressive models can emit invalid JSON, wrap responses in conversational filler, or succumb to prompt injections.
  3. **Strict Causal Masking**: Tokens generated early cannot attend to tokens that appear later in the sequence.

---

## 2. How Discrete Diffusion Decision Models Work

Rather than generating sequential text, DiffusionGemma treats a decision as a **discrete canvas denoising problem**:

```
[System Schema] + [User Input Data] + [Pre-Allocated Canvas: <slot_1> <slot_2> <slot_3>]
                                     │
                      ┌──────────────┴──────────────┐
                      │ Bidirectional Self-Attention │
                      │   (Prompt <---> Canvas)      │
                      └──────────────┬──────────────┘
                                     │
                  [Single-Pass Discrete Denoise / Readout]
                                     │
              ┌──────────────────────┼──────────────────────┐
              ▼                      ▼                      ▼
      Slot 1: "billing"       Slot 2: "yes"         Slot 3: "4 (frustrated)"
      (confidence: 99.1%)     (stderr: ±0.002)       (entropy: 0.04 nats)
```

### 1. The Pre-Allocated Canvas & Bidirectional Cross-Attention
Instead of starting an open-ended token generation loop, `dgem` sets aside a fixed-width discrete canvas (32 to 256 tokens). While autoregressive LLMs apply a causal mask, DiffusionGemma applies **bidirectional self-attention** across the entire prompt and canvas:
* The slot tokens attend to all prompt tokens simultaneously.
* Crucially, **the slot tokens attend to each other (`slot_1 <-> slot_2`)**. The model's classification of `team = engineering` directly informs its confidence on `urgent = yes` during the exact same forward pass.

### 2. Dual-Mode Uncertainty Telemetry & Epistemic Calibration (`ChaosNLI`)
Evaluated on [`ChaosNLI`](benchmarks-report.md) (100 human annotators per item), DiffusionGemma's single-pass restricted-softmax Shannon entropy $H = -\sum p_k \ln p_k$ correlates monotonically with human disagreement:

| Human Annotator Consensus Tier | Accuracy | Mean Confidence $P(y)$ | Mean Shannon Entropy $H$ | Entropy Multiplier |
| :--- | :---: | :---: | :---: | :---: |
| **`low-entropy` (`ChaosNLI` Consensus)** | **100.0% (3/3)** | `0.986` | **`0.0744 nats`** | **1.0× (Baseline)** |
| **`easy` (In-Domain Guardrail & Triage)** | **100.0% (17/17)** | `0.983` | **`0.0892 nats`** | **1.2×** |
| **`ambiguous` (Borderline Rater Splits)** | **77.8% (7/9)** | `0.861` | **`0.4061 nats`** | **5.5× higher $H$** |
| **`high-entropy` (`ChaosNLI` 3-Way Crowd Split)** | **33.3% (1/3)** | `0.759` | **`0.5932 nats`** | **8.0× higher $H$** ⭐ |

When human annotators agree, DiffusionGemma resolves the slot with **100% accuracy** and near-zero entropy (`0.0744 nats`). When the human crowd splits evenly, internal entropy spikes **8.0× higher (`0.5932 nats`)**, giving engineers a deterministic threshold ($H > 0.30\text{ nats}$) to trigger abstention or escalate to a Tier-3 reasoning model.

### 3. Templates as Executable Decision Policies (`Policy-as-Code`)
In `dgem`, a declarative `.json.tmpl` file **is** the classifier head:
* **Zero-Shot Policy Compilation**: Evaluating 50 items across 11 public benchmarks (`dgem bench-calibration`) without a single fine-tuned weight achieved **100% accuracy** on `AgentDrift` trajectory hijacks (`7/7`), `deepset/prompt-injections` (`4/4`), `LLM-AggreFact` RAG grounding (`2/2`), and `MS MARCO` passage relevance (`2/2`) in **664–793 ms**.
* **Joint Multi-Slot Conditioning**: A single `.json.tmpl` policy evaluates boolean gates, `[A-Z]` categorical choices, and ordinal scores simultaneously in one forward pass (**458.9 ms** on Cloud Run L4).

---

## 3. Summary

| Question | Classical ML / Encoders | Autoregressive LLMs | Discrete Diffusion Decision Models (`dgem`) |
| :--- | :--- | :--- | :--- |
| **How does it decide?** | Feature weights / static linear heads | Token-by-token sequential prediction | **Bidirectional canvas slot denoising** |
| **How are policies updated?** | Relabel dataset & retrain weights | Prompt engineering + output parser | **Declarative `.json.tmpl` (`Policy-as-Code`)** |
| **How fast is it?** | Microseconds – 20 ms | 2 – 17.5 seconds | **458.9 – 712 ms (1 forward pass)** |
| **Can slots attend to each other?** | No (independent heads) | Unidirectional (`left -> right` only) | **Yes (`slot_1 <-> slot_2` bidirectionally)** |
| **Does it know when it's unsure?** | Overconfident out-of-domain | Uncalibrated sequence logprobs | **Yes (8.0× $H$ spike on `ChaosNLI` splits)** |
| **Can it handle vision?** | Separate vision classifiers | Yes (multimodal autoregression) | **Yes (native SigLIP vision canvas)** |
