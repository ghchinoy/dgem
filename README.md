# dgem — DiffusionGemma as a Zero-Shot Decision Model

**`dgem`** is a declarative **Policy-as-Template** engine and **empirical benchmark harness** for Google DeepMind's **DiffusionGemma** (`26B-A4B-it`), supporting **Local Apple Silicon (macOS Metal)**, **Serverless Cloud Run GPU (`NVIDIA L4` & `NVIDIA RTX Pro 6000`)**, and **Google Compute Engine (`L4` / `2× A100`)** deployments.

There's also a web app **Decision Studio** (via `dgem serve`), **Model Context Protocol (`MCP`) Server** via (`dgem mcp` & `/mcp`), and **HTTP Gateway REST API** to facilicate usage.

---

## Four Ways to Use `dgem`

See **[Decision Studio Web App, MCP Server & HTTP Gateway API (`docs/studio-mcp-api.md`)](docs/studio-mcp-api.md)** for full details:

| Interaction Surface | Command / Endpoint | Description |
| :--- | :--- | :--- |
| **1. 🖥️ Decision Studio Web App** | `./bin/dgem serve --port 8090`<br>`http://localhost:8090/` | Embedded **Lit WebComponents** web application featuring all **26+ `.json.tmpl` decision policies** (`core`, `calibration`, `multimodal`, `rerank`), live **SigLIP 2D Bounding Box SVG overlays (`EXP-09`)**, one-click **Scale-to-Zero Cloud Run GPU warmup**, and **OpenTelemetry Trace Waterfall** inspection. |
| **2. 🤖 Model Context Protocol (`MCP`)** | `./bin/dgem mcp` (`stdio`)<br>`POST /mcp` (`Streamable HTTP`) | Native MCP server exposing **6 tools** (`decide_policy`, `locate_bounding_boxes`, `decide_custom_questions`, `list_policy_templates`, `get_health_and_gpu_status`, `warmup_gpu`) to **Gemini CLI**, **Claude Desktop**, **Cursor**, and cloud agent orchestrators. |
| **3. 🌐 HTTP Gateway REST API** | `POST /api/decide/{template}`<br>`GET /api/templates`, `POST /api/warmup` | Execute any `.json.tmpl` decision policy with a simple JSON variable map (`curl` / microservices) without installing `dgem` or managing local templates. Automatically handles GCP IAM/IAP auth and holds requests while scale-from-zero Cloud Run GPUs wake up. |
| **4. ⌨️ CLI & 6 Benchmark Suites** | `./bin/dgem decide` / `./bin/dgem bench-*` | Direct single-pass decisions (`--stats`) and six reproducible evaluation harnesses (`bench`, `bench-ecotone`, `bench-intents`, `bench-calibration`, `bench-bbox`, `bench-rerank`) backed by [`docs/experiments/`](docs/experiments/README.md) (`EXP-01` – `EXP-10`). |

### Why a "Decision Model"?

Historically, production engineering teams had to choose between two extremes for automated triage, routing, and guardrails:
1. **Discriminative Classifiers & Automata (BERT / DeBERTa / C++ WFSTs)**: Sub-10ms latency, but **rigid**. Adding a new policy rule or routing category requires curating labeled datasets, retraining weights, and redeploying model binaries.
2. **Autoregressive Generative LLMs (Gemini / GPT-4 / Gemma 4)**: Zero-shot flexible, but **architecturally mismatched for discrete decisions**—paying $O(T_{\text{output}})$ serial token generation latency (`2–17s`), vulnerable to markdown/JSON syntax drift, and lacking calibrated distribution entropy over the decision space.

**DiffusionGemma introduces a third architectural category: the Zero-Shot Decision Model.**
Instead of generating text left-to-right, `dgem` compiles declarative `.json.tmpl` templates into a **pre-allocated discrete diffusion canvas** (`32–256` tokens) with full bidirectional attention. Boolean gates, `[A-Z]` categorical choices, and ordinal rubrics are resolved simultaneously in a **single forward pass (`~125–450 ms`)**, returning **100% schema-guaranteed decisions**.

### Confidence Beyond Shannon: `dgem Invariant Decision Calibration (IDC)`

See **[Confidence Beyond Shannon: Invariant Decision Calibration (`docs/confidence-beyond-shannon.md`)](docs/confidence-beyond-shannon.md)** for the full architectural explainer:

While raw single-pass **Shannon entropy ($H = -\sum p_k \ln p_k$)** rises **8.0×** when human annotators disagree (`ChaosNLI`), relying on raw token entropy alone suffers from **Ballot-Order (`Box A`) Primacy Bias** ($p_0 = 88.3\%$ on 2-way, $78.3\%$ on 3-way, $49.3\%$ on 4-way blank prompts) and **unscaled diffusion logit overconfidence**. **`dgem Invariant Decision Calibration (IDC)`** wraps `DiffusionGemma`'s `125 ms` snapshot in a zero-overhead calibration pipeline:
1. **Null-Prior De-Biasing ("Tare the Scale", `--null-prior-debias`, `EXP-13B`)**: Divides out the model's content-free `Box A` bias in logit space, cutting Multi-Class Brier error by **90.2%** (`0.0173` $\rightarrow$ `0.0017`) and making **`>90%`-confidence decisions `100.0%` accurate (`31/31`)** on our 50-case calibration suite (`78.84` JevBench Composite).
2. **$O(1)$ Dual-Mirror Canvas (`--dual-mirror`, `EXP-13C`)**: Evaluates forward (`A..D`) and reversed (`D..A`) option orderings simultaneously on the **same bidirectional diffusion canvas (`0 ms` extra latency)**, eliminating option-reversal answer flipping (`0.0%` flip rate) and exposing live `Mirror TVD` (`66.8×` spike on `ChaosNLI`) to catch hidden toss-ups.
3. **Slot Temperature Scaling ($T^* = 1.25\text{–}1.35$, `EXP-11`) & Wide-Canvas Bracket Routing (`EXP-12`)**: Cuts 10-Bin Expected Calibration Error (`ECE`) by **56.2%–86.1%** (`0.0326` on `JevBench`, `0.0332` on `jev-decision-index`) and lifts structural coverage to **100.0% (`98.89` Headline Decision Index)** across up to 255 options and 32+ simultaneous slots.

| Architectural Dimension | Discrete Diffusion Decision Model (`dgem`) | Discriminative Encoder (DeBERTa-v3 / Llama-Guard) | Autoregressive LLM (Gemini / Gemma 4) | Compiled Rulebook (`ecotone` C++ WFST) |
| :--- | :--- | :--- | :--- | :--- |
| **Policy Adaptability** | **Zero-Shot Policy-as-Template** (edit `.json.tmpl` in seconds) | Requires labeled dataset & weight retraining per label change | Zero-shot prompt engineering | Manual grammar authoring & compilation |
| **Inference Latency** | **125 – 458 ms** (1-pass Cloud Run GPU / Metal) | ~5 – 25 ms (single head) | **17,486.6 ms** (~17.5s for 3-slot JSON + CoT) | **1.35 – 8.68 ms** (`1.54 ms` p50 over UDS) |
| **Latency Scaling Law** | **$O(K_{\text{steps}})$ constant time** (1 or 12 joint slots take same pass) | $O(M_{\text{heads}})$ separate classifiers per attribute | **$O(T_{\text{output}})$ linear penalty** (serial token loop) | $O(N_{\text{chars}})$ graph traversal |
| **Joint Slot Conditioning** | **Bidirectional (`slot_1 <-> slot_2`)** in a single forward pass | Independent static classification heads | Unidirectional causal bias (`left -> right`) | Local sliding window (1–3 tokens) |
| **Epistemic Calibration (`IDC`)** | **Null-Prior + Dual-Mirror + $T^*$** (`0.0326` ECE, `0%` reversal flip, **8.0×** $H$ on `ChaosNLI`) | Overconfident logits out-of-distribution | Uncalibrated sequence-level logprobs | Static tropical semiring arc weights |
| **Guardrail & Policy Accuracy** | **100%** `AgentDrift` hijack, **100%** Prompt Injection, **100%** RAG Grounding | Narrow single-task scope (512–8k context) | High accuracy at 15–25× higher latency | **36.7%** on semiotic polysemy traps |

---

## Supported Deployment Environments

`dgem` connects to any OpenAI-compatible or native Jev endpoint and exposes Decision Studio, MCP, REST API, and CLI surfaces:

```
    ┌──────────────────────┬──────────────────────┬──────────────────────┐
    │ 1. Decision Studio   │ 2. MCP Server        │ 3. REST API & CLI    │
    │ Browser Web UI (:8090│ dgem mcp & POST /mcp │ POST /api/decide/... │
    └──────────────────────┴──────────┬───────────┴──────────────────────┘
                                      │
          ┌───────────────────────────┼───────────────────────────┐
          ▼                           ▼                           ▼
Local Apple Silicon (Metal)  Cloud Run Serverless GPU   Cloud GPU on GCE VM
• diffgemma serve (:8080)    • 1× L4 (24GB) / RTX 6000  • 1× L4 (NVFP4) / 2× A100 (bf16)
• 4-bit Q4 Unified Memory    • Self-contained container • vLLM PR #57250 nightly wheel
• 32k KV Context             • GCS FUSE weight mount    • 32k context + Triton Attn
• Zero cloud cost            • 459ms avg wall latency   • Direct raw completions
```

### Option A: Local Apple Silicon (Metal)
Runs fully offline on M-series Macs using the native Rust Metal engine ([`diffgemma`](https://github.com/mmastrac/diffgemma)):
```bash
# 1. Install diffgemma engine
make setup

# 2. Download the 4-bit model pack (mmastrac/diffgemma-26b-a4b-it-q4)
make download

# 3. Launch background Metal server on port 8080
make serve

# 4. Stop when finished
make stop
```

### Option B: Cloud GPU on Google Compute Engine (NVIDIA L4 / A100)
Provisions automated, production-grade GCE instances with the nightly vLLM wheel (`wheels.vllm.ai`, matching PR #57250 base commit `133b71e0be`) and Triton attention:
```bash
# 4-bit NVFP4 on 1× NVIDIA L4 (g2-standard-8, ~$0.70/hr):
export GCP_PROJECT="your-gcp-project"
PRECISION=4 make gce-deploy

# 8-bit FP8-dynamic on 1× NVIDIA A100-40GB (a2-highgpu-1g, ~$3.67/hr):
PRECISION=8 make gce-deploy

# 16-bit unquantized bfloat16 on 2× NVIDIA A100-40GB (a2-highgpu-2g, TP=2, ~$7.34/hr):
export GCP_ZONE="us-central1-b"
PRECISION=16 make gce-deploy

# Mandatory immediate teardown to eliminate idle costs:
make gce-teardown
```

### Option C: Serverless Cloud GPU on Google Cloud Run (1× NVIDIA L4)
Builds and deploys a self-contained container image to Google Artifact Registry and runs on Cloud Run with GCS FUSE weight streaming:
```bash
export GCP_PROJECT="your-gcp-project"
export GCP_REGION="us-central1"

# 1. Build self-contained image in Artifact Registry via Cloud Build:
make cloudrun-build

# 2. Pre-stage 17.57 GB NVFP4 weights to GCS:
make cloudrun-stage

# 3. Deploy dgemma service on Cloud Run (1× NVIDIA L4, 24GB):
make cloudrun-deploy

# 4. Run discrete decisions or 30-case benchmark:
SERVICE_URL=$(gcloud run services describe dgemma --region=$GCP_REGION --format="value(status.url)")
./bin/dgem decide -u "${SERVICE_URL}/v1" --gcp-auth -t templates/support_triage.json.tmpl -v 'ticket=Emergency outage'
./bin/dgem bench -u "${SERVICE_URL}/v1" --gcp-auth -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_cloudrun.json

# 5. Mandatory immediate teardown to eliminate idle costs:
make cloudrun-teardown
```

---

## Installation & Quick Start

```bash
# Clone the repository
git clone https://github.com/ghchinoy/dgem.git
cd dgem

# Compile dgem binary into bin/
make build
```

### 1. Single-Pass Discrete Decision (`dgem decide`)
Evaluate customer tickets, code changes, or security alerts in a single ~750 ms forward pass:
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

`dgem` includes four first-class empirical benchmark harnesses (tracked in [`docs/experiments/README.md`](docs/experiments/README.md)):

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
| **`ChaosNLI` Crowd Split (`high-entropy`)** | 3 | 33.3% (1/3) | `0.759` | **`0.5932 nats` (8.0× spike)** ⭐ | **731 ms** |
| **Stage 1 Alone: `DiffusionGemma` (`steps=1, think=0`)** | **50** | **88.0% (44/50)** | **`0.925`** | **`0.2279 nats`** | **712 ms** |
| **Raw Entropy Cascade (`EXP-05a`): `dgemma [H<0.35]` $\rightarrow$ `gemini-3.8-flash`** | **50** | **94.0% (47/50, `+6.0%`)** | **`0.959`** | **`0.1410 nats`** | **1,824 ms** (`72%` early-exit) |
| **Normalized + Prior-Guided Cascade (`EXP-05b`, $\tilde{H} < 0.16$)** | **50** | **98.0% (49/50, `+10.0%`)** ⭐ | **`0.960`** | **`0.1416 nats` ($\tilde{H}=0.106$)** | **2,105 ms** (`66%` early-exit) |
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

* **[The Journey to Decision Models](docs/decision-models-primer.md)**: Architectural primer contrasting Classical ML, Symbolic WFSTs, Autoregressive LLMs, and Discrete Diffusion Decision Models.
* **[Experiments & Research Ledger (`docs/experiments/`)](docs/experiments/README.md)**: Structured log of completed empirical studies (`EXP-01` through `EXP-04`) and active architectural investigations (`EXP-05` Entropy-Gated Cascades, `EXP-06` Encoder Comparisons, `EXP-07` Conditional Policy DAGs).
* **[Benchmark Evaluation Report](docs/benchmarks-report.md)**: Full empirical receipts comparing Apple M5 Metal, Cloud Run 1× L4, GCE 1× L4, GCE 2× A100 `bfloat16`, `ChaosNLI`, Banking77, and CLINC150.
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
