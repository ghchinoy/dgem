# dgem — DiffusionGemma as a Zero-Shot Decision Model

**`dgem`** turns Google DeepMind's **DiffusionGemma** (`26B-A4B-it`, a discrete-diffusion Gemma 4) into a **decision engine**. You describe a decision as a small template of typed questions (`boolean`, `choice`, `score`). `dgem` places every question on the model's bidirectional canvas and reads **a probability for every allowed answer, for every question, in one forward pass**. There is no free-text generation to parse, and every answer comes with a per-question uncertainty score you can use to decide when to act automatically and when to escalate.

📖 **Docs site:** [ghchinoy.github.io/dgem](https://ghchinoy.github.io/dgem/)

## What's in this repo

| Piece | What it does | Start here |
| :--- | :--- | :--- |
| **Decision engine** (`dgem decide`) | Compiles `.json.tmpl` *Policy-as-Template* files into one-pass multi-question readouts with per-option probabilities and Shannon entropy. | [The Journey to Decision Models](docs/decision-models-primer.md) · [Template Catalog](docs/templates.md) |
| **Invariant Decision Calibration (IDC)** — *the novel part* | Checks whether a decision depends on **where options were listed**: divides out the model's "pick option A" habit and reads a **reversed ballot on the same canvas, in the same forward pass**. | [Confidence Beyond Shannon (IDC)](docs/confidence-beyond-shannon.md) |
| **Entropy-gated cascade** | Answers low-uncertainty decisions directly and escalates the rest to Vertex AI `gemini-3.8-flash` with the Stage-1 probabilities attached. | [EXP-05](docs/experiments/exp-05-roadmap-cascades-and-dags.md) |
| **Four surfaces** | CLI, HTTP gateway (`/api/decide`, `/v1/systemone`), MCP server (`dgem mcp`, `/mcp`), and the embedded Decision Studio web app (`dgem serve`). | [Studio, MCP & API](docs/studio-mcp-api.md) · [CLI reference](docs/cli-reference.md) |
| **Benchmarks & research log** | 9 reproducible `dgem bench-*` harnesses with committed JSON receipts, an experiment ledger (`EXP-01`–`EXP-13`), and a register of pre-registered follow-up experiments. | [Experiment Ledger](docs/experiments/README.md) · [Proposed Experiments](docs/experiments/proposed.md) |
| **Serving** | Vertex AI Dedicated Endpoint (primary), Cloud Run GPU (scale-to-zero failover), GCE VMs, and local Apple Silicon (Metal). | [Vertex AI vs. Cloud Run](docs/vertex-ai-vs-cloudrun.md) |

## Quick start

```bash
make build                                   # builds ./bin/dgem
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed $500 twice for my annual renewal this morning!' --stats
# Add --null-prior-debias to divide out the first-option habit (validate on your data first; see EXP-14).
```

Point it at a backend with `-u <url>/v1` (Cloud Run / GCE / local Metal) or `--vertex-url <endpoint-id>` (Vertex AI); see [Supported Deployment Environments](#supported-deployment-environments-4-serving-targets).

---

## Confidence Beyond Shannon: Invariant Decision Calibration (IDC)

> **In one sentence:** IDC makes a `dgem` confidence score reflect *the question*, not *the position where each answer was listed*, and flags decisions whose answer depends on the list order. Plain-English guide with a worked example: **[`docs/confidence-beyond-shannon.md`](docs/confidence-beyond-shannon.md)**.

**The problem.** Per-question Shannon entropy ($H = -\sum p_k \ln p_k$) is a useful escalation signal: it rises when human annotators disagree. But DiffusionGemma, like other language models, has a strong **ballot-order ("Box A") habit**: on blank, content-free questions it puts $88.3\%$ / $78.3\%$ / $49.3\%$ of its probability on the first slot (2 / 3 / 4 options). On a borderline input that habit can make a toss-up look 99.9% certain, and the entropy gate waves it through.

**What IDC does.**
1. **Null-Prior De-Biasing** (`--null-prior-debias`, no labeled data): divides out the measured slot habit. On the 50-item calibration suite it improved Brier from 0.175–0.193 (three same-session baselines) to 0.147; on the 231-item JevBench set it did **not** help (186 vs 187 correct, worse calibration). Suite-dependent, so validate before enabling.
2. **Dual-Mirror Canvas** (`--dual-mirror`): adds a reversed-order copy of each choice question **to the same canvas**, so the forward and reversed readings come from **one forward pass**, and their gap (`Mirror TVD`) flags order-dependent answers. A research diagnostic for now: a slot-naming bug (`__mirror_rev`, fixed to `__rev`) degraded readings, and even after the fix the extra slot lowers forward accuracy on JevBench (189 → 163–169), so it is not recommended in production ([EXP-14](docs/experiments/exp-14-idc-rerun.md)).
3. **Slot Temperature Scaling** (`EXP-11`): softens over-sharp scores. Needs labeled data. Fitted on held-out folds it cut ECE by 24–33% on 231 JevBench items ($T^* \approx 1.5$) but gave no reliable gain on the 50-item suite.

**What's new.** Removing a content-free prior (*contextual calibration*, Zhao et al. 2021), permutation debiasing (e.g. PriDe, Zheng et al. 2023), and temperature scaling (Guo et al. 2017) are known techniques. The part specific to a diffusion decision model is **checking a reversed ballot on every request without a second forward pass**, which turns order sensitivity from an offline audit into a per-request signal.

**Status.** A same-session re-run on 50 + 231 items ([EXP-14](docs/experiments/exp-14-idc-rerun.md), versioned receipts in `benchmarks/runs/`) gave mixed results: the order-bias *problem* is real and reproducible, but the corrections are suite-dependent and the same-canvas mirror needs redesign. IDC is CLI-only today. See [IDC §6](docs/confidence-beyond-shannon.md#6-the-evidence-so-far-with-sample-sizes) for every number and [Proposed Experiments](docs/experiments/proposed.md) (`PROP-00`–`PROP-10`) for what comes next.

---

## Headline Results (measured, with sample sizes)

| Result | Value | Sample | Receipt |
| :--- | :--- | :---: | :--- |
| Single-pass accuracy, 11 public datasets (`dgem bench-calibration`) | 88.0% (44/50) | 50 | `benchmarks/results_calibration_cloudrun.json` |
| + Null-prior de-biasing (IDC), same session | 45/50, Brier 0.147 vs 0.175–0.193 (3 baselines) | 50 | `benchmarks/runs/20260925-vertex-idc/` |
| + Entropy cascade to `gemini-3.8-flash` ($\tilde H \ge 0.16$, 34% escalated) | **98.0% (49/50)**, 56% lower cost than Gemini on every item | 50 | `results_calibration_cascade_normalized.json` |
| JevBench v1.3.1: `dgem` single pass → entropy cascade (hesitation ≥ 16%, 39% escalated, offline) | 187–189 → **221** of 231 | 231 | `benchmarks/runs/20260925-vertex-idc/` |
| JevBench v1.3.1: null-prior de-biasing | 186 of 231, Brier 0.293 vs 0.264 (no gain) | 231 | `benchmarks/runs/20260925-vertex-idc/` |
| Decision Index panel: bracket routing (> 26 options) + slot batching | 76.67 → 98.89, coverage 16/22 → 22/22 | 22 requests | `benchmarks/decision_index/` |
| Listwise reranking of 10 passages in one pass (`EXP-10`) | 0.9265 nDCG@10, 0% ties | 30 queries | `results_rerank_cloudrun.json` |
| Content-free Slot-A habit (`EXP-13B`) | 88.3% / 78.3% / 49.3% for K = 2 / 3 / 4 | probe | `results_permutation_cloudrun.json` |

Run-to-run noise is about ±1 item on 50 and ±2 on 231. Cascade thresholds were chosen on the evaluation items; treat these as directional. Compare runs with `python3 scripts/bench_runs.py compare`. Details and caveats: [Benchmark Report](docs/benchmarks-report.md), [Experiment Ledger](docs/experiments/README.md).

---

## Four Ways to Use `dgem`

See **[Decision Studio Web App, MCP Server & HTTP Gateway API (`docs/studio-mcp-api.md`)](docs/studio-mcp-api.md)** and **[Experiment Authoring Guide (`docs/experiment-authoring-guide.md`)](docs/experiment-authoring-guide.md)** for full details:

| Interaction Surface | Command / Endpoint | Description |
| :--- | :--- | :--- |
| **1. 🖥️ Decision Studio Web App** | `./bin/dgem serve --port 8090` | Embedded **Lit WebComponents** web application featuring all **26+ `.json.tmpl` decision policies** (`core`, `calibration`, `multimodal`, `rerank`), topbar **Backend Target selector (`vertex_first` \| `vertex` \| `cloudrun`)**, **Stage 2 Gemini Cascade (`gemini-3.8-flash`)**, live **SigLIP 2D Bounding Box SVG overlays (`EXP-09`)**, a plain-English **Concepts** tab (including an IDC walkthrough), and **OpenTelemetry Trace Waterfall** inspection. |
| **2. 🤖 Model Context Protocol (`MCP`)** | `./bin/dgem mcp` (`stdio`)<br>`POST /mcp` (`Streamable HTTP`) | Native MCP server exposing **6 tools** (`decide_policy`, `locate_bounding_boxes`, `decide_custom_questions`, `list_policy_templates`, `get_health_and_gpu_status`, `warmup_gpu`) with `backend` (`vertex_first` \| `vertex` \| `cloudrun`) and Stage 2 Gemini Cascade support (`cascade_mode`, `cascade_threshold`, `cascade_model`). |
| **3. 🌐 HTTP Gateway REST API** | `POST /api/decide/{template}`<br>`POST /v1/systemone`, `GET /api/templates` | Execute any `.json.tmpl` decision policy or `/v1/systemone` schema with `X-DGem-Backend: vertex_first \| vertex \| cloudrun` (`X-DGem-Backend-Used` returned on every response) and optional Stage 2 `gemini-3.8-flash` cascade. |
| **4. ⌨️ CLI & 9 Benchmark Harnesses** | `./bin/dgem decide --vertex-url ...`<br>`./bin/dgem bench-*` | Direct single-pass decisions (`--stats`, `--null-prior-debias`, `--dual-mirror`) and nine reproducible evaluation harnesses (`bench`, `bench-ecotone`, `bench-intents`, `bench-calibration`, `bench-bbox`, `bench-rerank`, `bench-jev`, `bench-decision-index`, `bench-permutation`) backed by [`docs/experiments/`](docs/experiments/README.md) (`EXP-01` – `EXP-13`). |

### Why a "Decision Model"?

Production teams have historically chosen between two extremes for automated triage, routing, and guardrails:
1. **Discriminative classifiers & automata (BERT / DeBERTa / C++ WFSTs)**: very fast, but **rigid**. A new policy rule or category means new labeled data, retraining, and redeployment.
2. **Autoregressive LLMs (Gemini / GPT / Gemma 4)**: zero-shot flexible, but they generate answers token by token (seconds per multi-field JSON answer), can drift from the output format, and their token probabilities are spread across formatting tokens rather than the decision itself.

A **zero-shot decision model** sits in between. `dgem` compiles a `.json.tmpl` template into a fixed diffusion canvas (`32–256` tokens) with bidirectional attention; boolean gates, `[A–Z]` choices, and ordinal scores are read together in **one forward pass**, and every answer is constrained to the allowed options.

| Architectural Dimension | Discrete Diffusion Decision Model (`dgem`) | Discriminative Encoder (DeBERTa-v3 / Llama-Guard) | Autoregressive LLM (Gemini / Gemma 4) | Compiled Rulebook (`ecotone` C++ WFST) |
| :--- | :--- | :--- | :--- | :--- |
| **Policy Adaptability** | **Zero-shot Policy-as-Template** (edit `.json.tmpl`) | Labeled dataset & retraining per label change | Zero-shot prompt engineering | Manual grammar authoring & compilation |
| **Inference Latency** | **~125 ms** (1 short question) to **~1.4 s** (12-slot rerank) per pass on Cloud Run / Vertex L4 | ~5 – 25 ms (single head) | **17,486.6 ms** (~17.5 s for 3-slot JSON + CoT) | **1.35 – 8.68 ms** (`1.54 ms` p50 over UDS) |
| **Passes per Request** | **1 forward pass** for all questions (cost grows with canvas length) | One classifier per attribute | One token per step ($O(T_{\text{output}})$) | $O(N_{\text{chars}})$ graph traversal |
| **Joint Slot Conditioning** | **Bidirectional (`slot_1 <-> slot_2`)** in a single pass | Independent heads | Left-to-right only | Local sliding window (1–3 tokens) |
| **Uncertainty & Calibration** | **Per-option probabilities + entropy**; label-free order-bias correction (null-prior) and same-pass reversed-ballot check (IDC) | Often overconfident out-of-distribution | Sequence-level logprobs over formatting tokens | Static arc weights |
| **Guardrail Examples (50-item suite)** | `AgentDrift` 7/7, prompt injection 4/4, RAG grounding 2/2 | Narrow single-task scope | High accuracy, 15–25× higher latency | **36.7%** on semiotic polysemy traps |

---

## Supported Deployment Environments (4 Serving Targets)

See **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU (`docs/vertex-ai-vs-cloudrun.md`)](docs/vertex-ai-vs-cloudrun.md)** for the complete architectural comparison and live 30-case benchmark receipts:

Warm p50 latencies for a 3-question decision, from [`benchmarks/runs/20260925-serving-speed`](benchmarks/runs/20260925-serving-speed/README.md) (Vertex/Cloud Run) and older receipts (GCE, Metal).

| Serving Target | Hardware & Shape | Cold-Start / Wakeup | Avg GPU Denoise (`N=4`) | Avg End-to-End Wall Time | Cost Profile | Recommended Use Case |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **1. Vertex AI Dedicated Endpoint (`/invoke/*`)** | `g4-standard-48` + `1× NVIDIA RTX PRO 6000` (ID `4423577720856772608`; SigLIP on). Legacy: `g2-standard-16` + `1× L4` (`4217256562927861760`) | **`0.0 s`** (min 1 replica, autoscale to 2) | **`97.9 ms`** (`57.5 ms` for `N=1`) | **`181 ms`** (`143 ms` for `N=1`) | Billed per replica-hour while deployed | **Primary production target (`vertex_first` default)**: always warm, IAM, autoscaling, multimodal. |
| **2. Serverless Cloud Run GPU (`dgemma`)** | `1× NVIDIA RTX PRO 6000` (`80Gi` RAM) or `1× L4` | **`~90–120 s`** (`0 → 1` scale-from-zero) | **`107.7 ms`** (`65.0 ms` for `N=1`) | **`187 ms`** (`144 ms` for `N=1`) | **`$0.00/hr` when idle** (`min-instances=0`) | **Scale-to-zero failover and batch (`cloudrun`)**: episodic jobs, research evaluations, sandboxes. |
| **3. Cloud GPU on GCE VM** | `g2-standard-8` (`1× L4` `NVFP4`) or `a2-highgpu-2g` (`2× A100` `bfloat16`) | **`0.0 s`** (dedicated VM) | — | **`1,968.7 ms`** (`L4`) / **`2,733 ms`** (`2× A100`) | `~$0.70/hr` (`L4`) / `~$7.34/hr` (`2× A100`) | High-throughput raw `vLLM` continuous batching (`Banking77` / `CLINC150`) & `bfloat16` precision baselines. |
| **4. Local Apple Silicon (`Metal`)** | Apple M-Series (`diffgemma-26b-a4b-it-q4` unified RAM) | **`0.0 s`** (local daemon) | **`892.0 ms`** (`210 ms` for `N=1`) | **`898.5 ms`** | **`$0.00/hr`** (local hardware) | Offline laptop development, policy authoring, and local verification. |

### Option A: Vertex AI Dedicated Endpoint (`/invoke/*`, Recommended Primary)
Deploys the `dgemma` container with arbitrary custom routes (`invokeRoutePrefix: "/*"`) onto a Vertex AI Dedicated Endpoint (default G4: `g4-standard-48` + RTX PRO 6000, `4423577720856772608`) so `/invoke/v1/chat/completions`, `/invoke/v1/systemone`, and `/invoke/health` are served with **`0.0 s` wakeup**:
```bash
# 1. Deploy dgemma to a Vertex AI Dedicated Endpoint on G4 (RTX PRO 6000), pinned image tag:
VERTEX_PROFILE=g4-rtxpro6000 IMAGE_URI=us-central1-docker.pkg.dev/$GCP_PROJECT/dgem/dgemma:<sha> make vertex-deploy

# 2. Run single-pass decision or 30-case benchmark directly against /invoke/v1:
./bin/dgem decide --vertex-url 4423577720856772608 --gcp-auth \
  -t templates/support_triage.json.tmpl -v 'ticket=Emergency outage' --stats
./bin/dgem bench --vertex-url 4423577720856772608 --gcp-auth \
  -d benchmarks/eval_dataset.jsonl -o benchmarks/results_vertex_l4_invoke.json

# 3. Teardown replica when zero-idle-cost ($0.00/hr) is desired:
make vertex-teardown
```

### Option B: Serverless Cloud GPU on Google Cloud Run (`1× NVIDIA RTX Pro 6000` / `1× L4`)
Builds and deploys a self-contained container image to Google Artifact Registry and runs on Cloud Run with scale-to-zero (`--min-instances=0`):
```bash
export GCP_PROJECT="your-gcp-project"
export GCP_REGION="us-central1"

# 1. Build self-contained image in Artifact Registry via Cloud Build:
make cloudrun-build

# 2. Pre-stage 17.57 GB NVFP4 weights to GCS:
make cloudrun-stage

# 3. Deploy dgemma service on Cloud Run:
make cloudrun-deploy

# 4. Run discrete decisions or 30-case benchmark:
SERVICE_URL=$(gcloud run services describe dgemma --region=$GCP_REGION --format="value(status.url)")
./bin/dgem decide -u "${SERVICE_URL}/v1" --gcp-auth -t templates/support_triage.json.tmpl -v 'ticket=Emergency outage'
./bin/dgem bench -u "${SERVICE_URL}/v1" --gcp-auth -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_cloudrun.json

# 5. Mandatory immediate teardown to eliminate idle costs:
make cloudrun-teardown
```

### Option C: Cloud GPU on Google Compute Engine (`NVIDIA L4` / `2× A100`)
Provisions automated GCE instances with the nightly vLLM wheel (`wheels.vllm.ai`, matching PR #57250 base commit `133b71e0be`) and Triton attention:
```bash
# 4-bit NVFP4 on 1× NVIDIA L4 (g2-standard-8, ~$0.70/hr):
export GCP_PROJECT="your-gcp-project"
PRECISION=4 make gce-deploy

# 16-bit unquantized bfloat16 on 2× NVIDIA A100-40GB (a2-highgpu-2g, TP=2, ~$7.34/hr):
export GCP_ZONE="us-central1-b"
PRECISION=16 make gce-deploy

# Mandatory immediate teardown to eliminate idle costs:
make gce-teardown
```

### Option D: Local Apple Silicon (Metal)
Runs fully offline on M-series Macs using the native Rust Metal engine ([`diffgemma`](https://github.com/mmastrac/diffgemma)):
```bash
make setup && make download && make serve
# Stop when finished:
make stop
```

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
  Endpoint:          http://34.121.236.110:8080/v1/chat/completions
  Total Wall Time:   856 ms
  KV Cache Reused:   169 tokens (82.8% hit rate)
  Denoise Steps:     1 step (policy: samples=1)
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

`dgem` includes nine benchmark harnesses (all tracked in [`docs/experiments/README.md`](docs/experiments/README.md)). Four of the most commonly used are below; the others are `bench-jev` (JevBench v1.3.1), `bench-decision-index` (Decision Index panel + `/v1/systemone`), `bench-permutation` (option-order sensitivity and IDC, `EXP-13`), `bench-rerank` (listwise reranking, `EXP-10`), and `bench-bbox` (bounding boxes, `EXP-09`).

### 1. Public Dataset Policy & Epistemic Calibration Suite (`dgem bench-calibration`)
Evaluates 50 items across **11 public datasets** ([`benchmarks/calibration_suite.jsonl`](benchmarks/calibration_suite.jsonl)), testing declarative policy templates (`templates/calibration/*.json.tmpl`) across agent trajectory hijacking (`AgentDrift`), multilingual jailbreaks (`deepset/prompt-injections`), RAG fact grounding (`LLM-AggreFact`), retrieval relevance (`MS MARCO`), toxicity (`Jigsaw Civil Comments`), and human annotator disagreement (`ChaosNLI`):

```bash
./bin/dgem bench-calibration -u "${SERVICE_URL}/v1" -m "/mnt/gcs/dgemma" --gcp-auth -w 4 \
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

## Documentation & Research Ledger

All pages below are also published on the docs site: [ghchinoy.github.io/dgem](https://ghchinoy.github.io/dgem/).

* **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU](docs/vertex-ai-vs-cloudrun.md)**: Architectural comparison, arbitrary custom route forwarding, the G4 (RTX PRO 6000) default, and the crawl-walk-run serving recommendation.
* **[Experiment Authoring Guide & Backend Target Selection](docs/experiment-authoring-guide.md)**: Choosing between `vertex_first`, `vertex`, and `cloudrun`, and configuring Stage 2 Gemini Cascades (`gemini-3.8-flash` default).
* **[CLI, HTTP Gateway & MCP Reference](docs/cli-reference.md)**: Complete flag and tool parameter reference (`--vertex-url`, `dgem serve --default-backend vertex_first`, `/v1/systemone`, and MCP tools).
* **[The Journey to Decision Models](docs/decision-models-primer.md)**: Architectural primer contrasting Classical ML, Symbolic WFSTs, Autoregressive LLMs, and Discrete Diffusion Decision Models.
* **[Confidence Beyond Shannon: Invariant Decision Calibration (IDC)](docs/confidence-beyond-shannon.md)**: Why a raw confidence score can be fooled by option order, how IDC checks it in one pass, what it does not fix, and the evidence with sample sizes.
* **[Glossary & Mental Models](docs/glossary.md)**: Plain-English definitions (entropy, null prior, Mirror TVD, ECE, Brier) and translations across ML specialties.
* **[Experiments & Research Ledger (`docs/experiments/`)](docs/experiments/README.md)**: Structured log of completed empirical studies (`EXP-01` through `EXP-13`) and active architectural investigations (`EXP-05` Entropy-Gated Cascades, `EXP-06` Encoder Comparisons, `EXP-07` Conditional Policy DAGs).
* **[Proposed Experiments Register](docs/experiments/proposed.md)**: Pre-registered hypotheses, designs, and decision criteria for upcoming work (`PROP-00`–`PROP-10`).
* **[Benchmark Evaluation Report](docs/benchmarks-report.md)**: Full empirical receipts comparing Vertex AI `1× L4`, Cloud Run `1× RTX Pro 6000` / `1× L4`, Apple M5 Metal, GCE `1× L4`, GCE `2× A100` `bfloat16`, `ChaosNLI`, Banking77, and CLINC150.
* **[Template Catalog (`Policy-as-Code`)](docs/templates.md)**: Complete reference of declarative `.json.tmpl` decision schemas across triage, guardrails, NLU, and multimodal vision.
* **[Ecotone (WFST) vs. DiffusionGemma](docs/ecotone-comparison.md)**: Semiotic polysemy taxonomy, head-to-head findings, and the hybrid Cascaded Normalizer architecture.
* **[Architecture: Discrete Diffusion vs. Autoregression](docs/architecture.md)**: Mechanical breakdown of 256-token canvas denoising, bidirectional slot readout, and terminology history.
* **[Cloud Run Lessons Learned & Native CUDA Build Guide](docs/cloudrun-lessons-learned.md)**: Self-contained Artifact Registry build, GCS FUSE prefetching, and envelope unmarshaling architecture.
* **[Remote Endpoints & Cloud Deployment](docs/remote-endpoints.md)**: Pointing `dgem` to Google Cloud Run, GCE GPU instances, Vertex AI, and hosted vLLM clusters.
* **[Real-World Applications & Production Patterns](docs/applications.md)**: Production architectures for agentic dispatch, DevSecOps git hooks, and SIEM alert triage.
* **[User Guide](docs/user-guide.md)** & **[Setup & Metal Engine Guide](docs/setup.md)**: Full CLI reference and local Apple Silicon serving.

---

## Contributing

Issues, bug reports, and feature discussions are welcome! However, **we are not accepting pull requests (PRs) at this time**. If you encounter a bug or have feedback on benchmark methodologies or templates, please open an [Issue](https://github.com/ghchinoy/dgem/issues).

## License

This project is licensed under the [Apache-2.0 License](LICENSE).

## Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.
> This project is not eligible for the [Google Open Source Software Vulnerability Rewards Program](https://bughunters.google.com/open-source-security).
