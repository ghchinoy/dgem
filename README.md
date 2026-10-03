# dgem — DiffusionGemma as a Zero-Shot Decision Model

**`dgem`** turns Google DeepMind's **DiffusionGemma** (`26B-A4B-it`, a discrete-diffusion Gemma 4) into a **decision engine**. You describe a decision as a small template of typed questions (`boolean`, `choice`, `score`). `dgem` places every question on the model's bidirectional canvas and reads **a probability for every allowed answer, for every question, in one forward pass**. There is no free-text generation to parse, and every answer comes with a per-question uncertainty score you can use to decide when to act automatically and when to escalate.

📖 **Docs site:** [ghchinoy.github.io/dgem](https://ghchinoy.github.io/dgem/)

## What's in this repo

| Piece | What it does | Start here |
| :--- | :--- | :--- |
| **Decision engine** (`dgem decide`) | Compiles `.json.tmpl` *Policy-as-Template* files into one-pass multi-question readouts with per-option probabilities and Shannon entropy. | [The Journey to Decision Models](docs/decision-models-primer.md) · [Template Catalog](docs/policies/templates.md) |
| **Hesitation-gated decisions** | A probability for every option and a hesitation score from one pass; hesitant answers handed off; per-domain calibration and noise-judged release checks. | [Confidence beyond Shannon](docs/confidence/overview.md) · [Confidence and calibration](docs/confidence/index.md) |
| **Hesitation-gated cascade** | Answers low-hesitation decisions directly and hands the rest to Vertex AI `gemini-3.8-flash` with `dgem`'s probabilities attached. | [Authoring and cascades](docs/policies/authoring.md) · [EXP-18](docs/experiments/exp-18-mizan-judge-capability.md) |
| **Four surfaces** | CLI, HTTP gateway (`/api/decide`, `/v1/systemone`), MCP server (`dgem mcp`, `/mcp`), and the embedded Decision Studio web app (`dgem serve`). | [Studio, MCP & API](docs/reference/studio-mcp-api.md) · [CLI reference](docs/reference/cli.md) |
| **Benchmarks & research log** | 9 reproducible `dgem bench-*` harnesses with committed JSON receipts, an experiment ledger (`EXP-01`–`EXP-21`), and a register of pre-registered follow-up experiments. | [Experiment Ledger](docs/experiments/README.md) · [Proposed Experiments](docs/experiments/proposed.md) |
| **Serving** | Vertex AI Dedicated Endpoint on RTX PRO 6000 (primary), Cloud Run GPU (scale-to-zero failover), GCE VMs, and local Apple Silicon (Metal). | [From laptop to production](docs/deploy/index.md) |

## Quick Start

Full walkthrough: **[Run on your laptop](docs/deploy/laptop.md)**.

### 1. Start an Inference Backend
Choose the option matching your hardware:

* **Local Apple Silicon Mac (Metal)** — No Docker or cloud needed:
  ```bash
  make setup && make download && make local-up   # engine on :8080, Decision Studio on :8090
  ```
* **Any Linux/Windows Workstation with NVIDIA GPU (Docker)**:
  ```bash
  # Lean public image (downloads the public weights on first boot); tags and digests: docs/deploy/public-images.md
  docker run --gpus all -p 8080:8080 us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:v0.1.0@sha256:5fa4a866163169aaf91c3bdb727020ff3c86e26e84ffadad86bc859053d45b26
  ```
* **Remote Google Cloud Endpoint**:
  ```bash
  export DGEM_VERTEX_URL="<endpoint-id>" GCP_PROJECT_NUMBER="<project-number>" DGEM_GCP_AUTH=1
  ```

### 2. Build CLI & Execute Your First Decision
```bash
make build                                   # compiles ./bin/dgem
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed $500 twice for my annual renewal this morning!' --stats
```

### 3. Open Decision Studio (Web UI)
```bash
./bin/dgem serve --port 8090
```
Open **[http://localhost:8090](http://localhost:8090)** to inspect all 26+ templates, evaluate Stage 2 Gemini cascades, and visualize OpenTelemetry trace waterfalls.

**Next, pick your path:** [build, deploy and operate](docs/deploy/index.md) ·
[confidence and calibration](docs/confidence/index.md) · [write your first policy](docs/policies/first-policy.md).

---

## Confidence beyond Shannon: hesitation-gated decisions

> **In one sentence:** every `dgem` answer comes with a probability for every allowed option and a **hesitation** score; clear answers are returned in about 0.1 s and hesitant ones are handed to a larger model or a person. Full explanation with evidence: **[`docs/confidence/overview.md`](docs/confidence/overview.md)**.

**Hesitation-gating.** Hesitation is Shannon entropy scaled to 0–100% by the number of options. On 231 JevBench items it separates wrong answers from right ones with an AUROC of about 0.85 ([EXP-17](docs/experiments/exp-17-separate-pass-mirror.md)). With a 35% threshold fixed in advance, a live hand-off to `gemini-3.8-flash` reached 81/100 on safety and 86/100 on faithfulness while handing off 8–32% of items, at a median 105 ms vs 3.1 s and about $0.02 vs $1.06–2.25 per 1,000 judgements at full utilization ([EXP-18](docs/experiments/exp-18-mizan-judge-capability.md)).

**What keeps the probabilities trustworthy.**
1. **The answer template is part of the prompt.** Shared option letters across questions make the model copy answers (JevBench 154 vs a baseline band of 182–189, [EXP-15](docs/experiments/exp-15-letter-collision.md)), and loaded question ids act as instructions ([EXP-16](docs/experiments/exp-16-slot-names.md)). Both are now template rules.
2. **Prompt layout.** Putting the input before the questions (`document_first`, default since v0.2.0) gained 3–12 points on frozen held-out sets such as XNLI (0.662 → 0.699, n=4,500) and typed decisions (0.674 → 0.728, n=2,000), at the cost of lower out-of-scope recall ([prompt layout](docs/policies/prompt-layout.md)).
3. **Order bias, measured and monitored.** The model favours the first option when unsure; every release measures the extra flips under shuffled order. In-pass corrections were tested and did not help reliably (EXP-13 to EXP-17), so order is tracked, not corrected.
4. **Calibration by domain.** The best held-out temperature ranges from about 1.2 to 3.6 across suites, so fit one per policy ([calibrate your policy](docs/confidence/calibrate-your-policy.md)).
5. **Results judged against measured noise.** The [regression matrix](docs/operate/regression-matrix.md) measures run-to-run agreement (94–99%) in every session and judges serving changes against it, with versioned receipts.

The earlier "Invariant Decision Calibration (IDC)" framing, centred on a same-pass reversed-order check, is kept for reference in [`docs/history/`](docs/history/idc-confidence-beyond-shannon.md); the experiments that retired it are EXP-14 to EXP-17.

---

## Headline Results (measured, with sample sizes)

| Result | Value | Sample | Receipt |
| :--- | :--- | :---: | :--- |
| Single-pass accuracy, 11 public datasets (`dgem bench-calibration`) | 88.0% (44/50) | 50 | `benchmarks/results_calibration_cloudrun.json` |
| + Entropy cascade to `gemini-3.8-flash` ($\tilde H \ge 0.16$, 34% escalated) | **98.0% (49/50)**, 56% lower cost than Gemini on every item | 50 | `results_calibration_cascade_normalized.json` |
| JevBench v1.3.1: `dgem` single pass → entropy cascade (hesitation ≥ 16%, 39% escalated, offline) | 187–189 → **221** of 231 | 231 | `benchmarks/runs/20260925-vertex-idc/` |
| Live hesitation-gated hand-off, threshold fixed in advance (`EXP-18`) | 81/100 safety, 86/100 faithfulness at 8–32% handed off; p50 105 ms | 100 per suite | `docs/experiments/exp-18-mizan-judge-capability.md` |
| `document_first` layout (v0.2.0), frozen held-out sets | XNLI 0.662 → 0.699; typed decisions 0.674 → 0.728 | 4,500; 2,000 | `benchmarks/runs/20261002-v020-release-t2/` |
| Decision Index panel: bracket routing (> 26 options) + slot batching | 76.67 → 98.89, coverage 16/22 → 22/22 | 22 requests | `benchmarks/decision_index/` |
| Listwise reranking of 10 passages in one pass (`EXP-10`) | 0.9265 nDCG@10, 0% ties | 30 queries | `results_rerank_cloudrun.json` |
| Content-free Slot-A habit (`EXP-13B`) | 88.3% / 78.3% / 49.3% for K = 2 / 3 / 4 | probe | `results_permutation_cloudrun.json` |

Answers are not deterministic (about 5–7% of items change between identical runs), so compare runs with the [regression matrix](docs/operate/regression-matrix.md), which measures that noise floor in every run. The offline cascade thresholds were chosen on the evaluation items; treat those as directional. Details and caveats: [Benchmark Report](docs/benchmarks-report.md), [Experiment Ledger](docs/experiments/README.md).

---

## Four Ways to Use `dgem`

See **[Decision Studio, MCP and HTTP API](docs/reference/studio-mcp-api.md)** and **[Gateway and routing](docs/deploy/gateway.md)** for full details:

| Interaction Surface | Command / Endpoint | Description |
| :--- | :--- | :--- |
| **1. 🖥️ Decision Studio Web App** | `./bin/dgem serve --port 8090` | Embedded **Lit WebComponents** web application featuring all **26+ `.json.tmpl` decision policies** (`core`, `calibration`, `multimodal`, `rerank`), topbar **Backend Target selector (`vertex_first` \| `vertex` \| `cloudrun`)**, **Stage 2 Gemini Cascade (`gemini-3.8-flash`)**, live **SigLIP 2D Bounding Box SVG overlays (`EXP-09`)**, a plain-English **Concepts** tab (including a hesitation-gating walkthrough), and **OpenTelemetry Trace Waterfall** inspection. |
| **2. 🤖 Model Context Protocol (`MCP`)** | `./bin/dgem mcp` (`stdio`)<br>`POST /mcp` (`Streamable HTTP`) | Native MCP server exposing **6 tools** (`decide_policy`, `locate_bounding_boxes`, `decide_custom_questions`, `list_policy_templates`, `get_health_and_gpu_status`, `warmup_gpu`) with `backend` (`vertex_first` \| `vertex` \| `cloudrun`) and Stage 2 Gemini Cascade support (`cascade_mode`, `cascade_threshold`, `cascade_model`). |
| **3. 🌐 HTTP Gateway REST API** | `POST /api/decide/{template}`<br>`POST /v1/systemone`, `GET /api/templates` | Execute any `.json.tmpl` decision policy or `/v1/systemone` schema with `X-DGem-Backend: vertex_first \| vertex \| cloudrun` (`X-DGem-Backend-Used` returned on every response) and optional Stage 2 `gemini-3.8-flash` cascade. |
| **4. ⌨️ CLI & 9 Benchmark Harnesses** | `./bin/dgem decide --vertex-url ...`<br>`./bin/dgem bench-*` | Direct single-pass decisions (`--stats`, `--null-prior-debias`, `--dual-mirror`) and nine reproducible evaluation harnesses (`bench`, `bench-ecotone`, `bench-intents`, `bench-calibration`, `bench-bbox`, `bench-rerank`, `bench-jev`, `bench-decision-index`, `bench-permutation`) backed by [`docs/experiments/`](docs/experiments/README.md) (`EXP-01` – `EXP-21`). |

### Why a "Decision Model"?

Production teams have historically chosen between two extremes for automated triage, routing, and guardrails:
1. **Discriminative classifiers & automata (BERT / DeBERTa / C++ WFSTs)**: very fast, but **rigid**. A new policy rule or category means new labeled data, retraining, and redeployment.
2. **Autoregressive LLMs (Gemini / GPT / Gemma 4)**: zero-shot flexible, but they generate answers token by token (seconds per multi-field JSON answer), can drift from the output format, and their token probabilities are spread across formatting tokens rather than the decision itself.

A **zero-shot decision model** sits in between. `dgem` compiles a `.json.tmpl` template into a fixed diffusion canvas (`32–256` tokens) with bidirectional attention; boolean gates, `[A–Z]` choices, and ordinal scores are read together in **one forward pass**, and every answer is constrained to the allowed options.

| Architectural Dimension | Discrete Diffusion Decision Model (`dgem`) | Discriminative Encoder (DeBERTa-v3 / Llama-Guard) | Autoregressive LLM (Gemini / Gemma 4) | Compiled Rulebook (`ecotone` C++ WFST) |
| :--- | :--- | :--- | :--- | :--- |
| **Policy Adaptability** | **Zero-shot Policy-as-Template** (edit `.json.tmpl`) | Labeled dataset & retraining per label change | Zero-shot prompt engineering | Manual grammar authoring & compilation |
| **Inference Latency** | **~55 ms GPU / ~150 ms end to end** for a 3-question decision on RTX PRO 6000; ~1.4 s for a 12-slot rerank (measured on L4) | ~5 – 25 ms (single head) | **17,486.6 ms** (~17.5 s for 3-slot JSON + CoT) | **1.35 – 8.68 ms** (`1.54 ms` p50 over UDS) |
| **Passes per Request** | **1 forward pass** for all questions (cost grows with canvas length) | One classifier per attribute | One token per step ($O(T_{\text{output}})$) | $O(N_{\text{chars}})$ graph traversal |
| **Joint Slot Conditioning** | **Bidirectional (`slot_1 <-> slot_2`)** in a single pass | Independent heads | Left-to-right only | Local sliding window (1–3 tokens) |
| **Uncertainty & Calibration** | **Per-option probabilities + hesitation**; hesitation-gated hand-off; per-domain temperature; order bias monitored on every release | Often overconfident out-of-distribution | Sequence-level logprobs over formatting tokens | Static arc weights |
| **Guardrail Examples (50-item suite)** | `AgentDrift` 7/7, prompt injection 4/4, RAG grounding 2/2 | Narrow single-task scope | High accuracy, 15–25× higher latency | **36.7%** on semiotic polysemy traps |

---

## Serving: From Laptop to Production

Measured with the current serving image (3-question decision, `samples: 1`, p50;
[receipts](benchmarks/runs/20260927-image-parity/README.md)):

| Stage | Where | Cold start | GPU / end to end | Guide |
| :--- | :--- | :--- | :--- | :--- |
| Crawl | Apple Silicon (`diffgemma`) or a local NVIDIA GPU | None | ~0.9 s on Metal | [Run on your laptop](docs/deploy/laptop.md) |
| Walk | Cloud Run GPU, 1× RTX PRO 6000, scale to zero | 2.3–2.7 min | 61 / 153 ms | [Deploy on Cloud Run](docs/deploy/cloud-run.md) |
| Run | Vertex AI dedicated endpoint, `g4-standard-48` + 1× RTX PRO 6000 | None | **55 / 150 ms** | [Production on Vertex AI](docs/deploy/vertex.md) |

A gateway (`dgem serve`) in front routes to Vertex first and fails over to Cloud Run
([Gateway and routing](docs/deploy/gateway.md)). Capacity, runbook and observability:
[Latency and capacity](docs/operate/latency-capacity.md) · [Operations runbook](docs/operate/runbook.md).

---

## Installation & CLI Examples

```bash
# Clone the repository
git clone https://github.com/ghchinoy/dgem.git
cd dgem

# Compile dgem binary into bin/
make build
```

### 1. Single-Pass Discrete Decision (`dgem decide`)
Evaluate customer tickets, code changes, or security alerts in a single sub-second forward pass:
```bash
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed $500 twice for my annual renewal this morning!' \
  --stats
```

Output:
```
QUESTION         | TYPE       | VALUE / CHOICE       | CONFIDENCE | ENTROPY (H) | AGREEMENT 
-----------------------------------------------------------------------------------------
sentiment        | score      | frustrated           | 99.8%      | 0.002 nats  | 1.00      
team             | choice     | billing              | 100.0%     | 0.000 nats  | 1.00      
urgent           | boolean    | yes                  | 99.9%      | 0.001 nats  | 1.00      

──────────────────────────────── STATS ────────────────────────────────
  Model:             nvidia/diffusiongemma-26B-A4B-it-NVFP4
  Endpoint:          http://127.0.0.1:8080/v1/chat/completions
  Total Wall Time:   856 ms
  KV Cache Reused:   169 tokens (82.8% hit rate)
  Denoise Steps:     1 step (policy: samples=4)
───────────────────────────────────────────────────────────────────────
```

### 2. Generative Prompt Completion (`dgem ask`)
Standard chat completion with optional thinking mode:
```bash
./bin/dgem ask "Explain discrete block diffusion in two sentences."
```

### 3. Remote Cloud Routing with IAM Authentication
Connect to any remote GCE or Cloud Run GPU service:
```bash
./bin/dgem decide \
  -u "http://<EXTERNAL_IP>:8080/v1" \
  -m "nvidia/diffusiongemma-26B-A4B-it-NVFP4" \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Outage: production database cluster unreachable' \
  --stats
```

### 4. Multimodal Visual Assessment (`--image` / `-I`)
Attach local image paths (automatically base64 encoded) or remote URLs:
```bash
./bin/dgem decide -t templates/multimodal/ui_design_review.json.tmpl \
  -I fixtures/ui_component.svg \
  -v 'component=CheckoutCard' \
  --stats
```

---

## Benchmark Suites & Empirical Calibration

`dgem` includes nine benchmark harnesses (all tracked in [`docs/experiments/README.md`](docs/experiments/README.md)). Four of the most commonly used are below; the others are `bench-jev` (JevBench v1.3.1), `bench-decision-index` (Decision Index panel + `/v1/systemone`), `bench-permutation` (option-order sensitivity, `EXP-13`), `bench-rerank` (listwise reranking, `EXP-10`), `bench-bbox` (bounding boxes, `EXP-09`), and `bench-vision` (categorical image questions, `EXP-22`). For what dgem can and cannot do with images, see [`docs/policies/images.md`](docs/policies/images.md).

### 1. Public Dataset Policy & Epistemic Calibration Suite (`dgem bench-calibration`)
Evaluates 50 items across **11 public datasets** ([`benchmarks/calibration_suite.jsonl`](benchmarks/calibration_suite.jsonl)), testing declarative policy templates (`templates/calibration/*.json.tmpl`) across agent trajectory hijacking (`AgentDrift`), multilingual jailbreaks (`deepset/prompt-injections`), RAG fact grounding (`LLM-AggreFact`), retrieval relevance (`MS MARCO`), toxicity (`Jigsaw Civil Comments`), and human annotator disagreement (`ChaosNLI`):

```bash
./bin/dgem bench-calibration -u "https://<CLOUD_RUN_URL>/v1" --gcp-auth -w 4 \
  -o benchmarks/results_calibration_cloudrun.json
```

| Public Dataset / Policy Domain | Cases | Accuracy | Mean $P(y)$ | Mean Entropy $H$ | Avg Latency |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **`AgentDrift`** (`agent_step_drift.json.tmpl` — Hijack + 4-Way Step Localization) | 7 | **100.0% (7/7)** ⭐ | `0.997` | `0.0186 nats` | **693 ms** |
| **`deepset/prompt-injections`** (`prompt_injection.json.tmpl` — `en`/`de` Gate) | 4 | **100.0% (4/4)** ⭐ | `0.980` | `0.0817 nats` | **669 ms** |
| **`LLM-AggreFact` & `MS MARCO`** (RAG Grounding & Retrieval Relevance) | 4 | **100.0% (4/4)** ⭐ | `0.993` | `0.0403 nats` | **728 ms** |
| **`CLINC150`, `Banking77`, `GoEmotions`, `BoolQ`, `Yelp/SST-5`** | 20 | **100.0% (20/20)** ⭐ | `0.898` | `0.3263 nats` | **769 ms** |
| **`ChaosNLI` Crowd Consensus (`low-entropy`)** | 3 | **100.0% (3/3)** | `0.986` | **`0.0744 nats` (1.0×)** | **625 ms** |
| **`ChaosNLI` Crowd Split (`high-entropy`)** | 3 | 33.3% (1/3) | `0.759` | **`0.5932 nats` (8.0× higher; n=3)** | **731 ms** |
| **Stage 1 Alone: `DiffusionGemma` (`steps=1, think=0`)** | **50** | **88.0% (44/50)** | **`0.925`** | **`0.2279 nats`** | **712 ms** |
| **Raw Entropy Cascade (`EXP-05a`): `dgemma [H<0.35]` $\rightarrow$ `gemini-3.8-flash`** | **50** | **94.0% (47/50, `+6.0%`)** | **`0.959`** | **`0.1410 nats`** | **1,824 ms** (`72%` early-exit) |
| **Normalized + Prior-Guided Cascade (`EXP-05b`, $\tilde{H} < 0.16$, threshold tuned on these items)** | **50** | **98.0% (49/50, `+10.0%`)** ⭐ | **`0.960`** | **`0.1416 nats` ($\tilde{H}=0.106$)** | **2,105 ms** (`66%` early-exit) |
| **Stage 2 Alone: `gemini-3.8-flash` (100% Frontier LLM)** | **50** | **98.0% (49/50)** | **`0.959`** | **`0.1347 nats`** | `3,412 ms` (`4.8×` slower) |

### 2. Multi-Domain Operational Triage (`dgem bench`)
Evaluates 30 multi-field test cases (`boolean` + `choice` + `score` in a single pass) across `support`, `code_review`, and `security` ([`benchmarks/eval_dataset.jsonl`](benchmarks/eval_dataset.jsonl)):
```bash
./bin/dgem bench -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_cloudrun.json
```

### 3. Ecotone WFST vs. DiffusionGemma (`dgem bench-ecotone`)
Evaluates 49 Text Normalization cases comparing C++ `ecotone` (OpenFst / Sparrowhawk WFSTs over `unix:///tmp/ecotone.sock`) against DiffusionGemma across semiotic polysemy traps and deterministic NSWs:
```bash
./bin/dgem bench-ecotone -c benchmarks/ecotone/tn_semiotics.jsonl --samples 1 -o benchmarks/results_ecotone.json
```

### 4. High-Cardinality Intent & Out-of-Scope Routing (`dgem bench-intents`)
Evaluates 30-way to 151-way intent routing and Out-of-Scope (`oos`) rejection on **`PolyAI/banking77`** and **`DeepPavlov/clinc150`**:
```bash
./bin/dgem bench-intents --dataset banking77 --full --workers 16
./bin/dgem bench-intents --dataset clinc150 --full --workers 16
```

---

## Documentation

Published at [ghchinoy.github.io/dgem](https://ghchinoy.github.io/dgem/) ([index](docs/index.md)), organized by audience:

- **Build, deploy and operate:** [From laptop to production](docs/deploy/index.md) ·
  [Run on your laptop](docs/deploy/laptop.md) · [Use a remote GPU](docs/deploy/remote-gpu.md) ·
  [Deploy on Cloud Run](docs/deploy/cloud-run.md) · [Production on Vertex AI](docs/deploy/vertex.md) ·
  [Gateway and routing](docs/deploy/gateway.md) · [Latency and capacity](docs/operate/latency-capacity.md) ·
  [Operations runbook](docs/operate/runbook.md) · [Observability](docs/operate/observability.md)
- **Confidence and calibration:** [Overview](docs/confidence/index.md) ·
  [Calibrate your policy](docs/confidence/calibrate-your-policy.md) ·
  [Confidence beyond Shannon](docs/confidence/overview.md) ·
  [The journey to decision models](docs/decision-models-primer.md) · [Glossary](docs/glossary.md) ·
  [Benchmark report](docs/benchmarks-report.md) · [Experiment ledger](docs/experiments/README.md) ·
  [Proposed experiments](docs/experiments/proposed.md)
- **Policies and decisions:** [Your first decision policy](docs/policies/first-policy.md) ·
  [Authoring and Stage 2 cascades](docs/policies/authoring.md) · [Run a dataset](docs/policies/datasets.md) ·
  [Template catalog](docs/policies/templates.md) · [Real-world applications](docs/policies/applications.md) ·
  [Taxonomy discovery](docs/policies/taxonomy-discovery.md)
- **Reference:** [CLI, HTTP and MCP](docs/reference/cli.md) · [Studio, MCP and HTTP API](docs/reference/studio-mcp-api.md) ·
  [Public container images](docs/deploy/public-images.md) · [Vertex AI vs. Cloud Run](docs/reference/vertex-vs-cloud-run.md) ·
  [Apple Silicon engine](docs/reference/metal-engine.md) · [How the model decides in one pass](docs/confidence/architecture.md) ·
  [Ecotone comparison](docs/ecotone-comparison.md) · [Engineering history (archive)](docs/history/cloud-run-engineering-notes.md)

---

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for contributor license agreement (CLA) requirements and community guidelines, and [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) for our Code of Conduct. If you encounter a bug or have feedback on benchmark methodologies or templates, please open an [Issue](https://github.com/GoogleCloudPlatform/dgem/issues).

## License

This project is licensed under the [Apache-2.0 License](LICENSE).

## Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.
> This project is not eligible for the [Google Open Source Software Vulnerability Rewards Program](https://bughunters.google.com/open-source-security).
