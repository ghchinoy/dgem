# dgem — DiffusionGemma as a Zero-Shot Decision Model

**`dgem`** is a declarative **Policy-as-Template** engine and **empirical benchmark harness** for Google DeepMind's **DiffusionGemma** (`26B-A4B-it`), supporting **four serving targets**:
1. **Vertex AI Dedicated Endpoints (`/invoke/*`, `g2-standard-16` `1× NVIDIA L4` `64 GB` RAM)** — `0.0 s` cold start, `~490 ms` GPU denoise (`~536 ms` wall time), enterprise MLOps (`vertex_first` default in `dgemma-gateway`).
2. **Serverless Cloud Run GPU (`NVIDIA RTX Pro 6000` `48GB` & `NVIDIA L4` `24GB`)** — Scale-to-zero (`$0.00/hr` idle cost), `~427 ms` GPU denoise (`~459 ms` warm wall latency).
3. **Google Compute Engine VM (`1× L4` `NVFP4` / `2× A100` `bfloat16`)** — Dedicated VM continuous batching and custom CUDA kernel profiling.
4. **Local Apple Silicon (`macOS Metal`)** — Native Rust Metal engine (`diffgemma`, `q4` unified memory) with zero cloud cost.

It ships with an embedded **Decision Studio Web App (`dgem serve`)**, **Model Context Protocol (`MCP`) Server (`dgem mcp` & `/mcp`)**, **HTTP Gateway REST API (`/api/decide` & `/v1/systemone`)**, and **Stage 2 Gemini Cascade (`gemini-3.8-flash` default)**.

---

## Four Ways to Use `dgem`

See **[Decision Studio Web App, MCP Server & HTTP Gateway API (`docs/studio-mcp-api.md`)](docs/studio-mcp-api.md)** and **[Experiment Authoring Guide (`docs/experiment-authoring-guide.md`)](docs/experiment-authoring-guide.md)** for full details:

| Interaction Surface | Command / Endpoint | Description |
| :--- | :--- | :--- |
| **1. 🖥️ Decision Studio Web App** | `./bin/dgem serve --port 8090`<br>`https://dgemma.aaie.cloud` | Embedded **Lit WebComponents** web application featuring all **26+ `.json.tmpl` decision policies** (`core`, `calibration`, `multimodal`, `rerank`), topbar **Backend Target selector (`vertex_first` \| `vertex` \| `cloudrun`)**, **Stage 2 Gemini Cascade (`gemini-3.8-flash`)**, live **SigLIP 2D Bounding Box SVG overlays (`EXP-09`)**, and **OpenTelemetry Trace Waterfall** inspection. |
| **2. 🤖 Model Context Protocol (`MCP`)** | `./bin/dgem mcp` (`stdio`)<br>`POST /mcp` (`Streamable HTTP`) | Native MCP server exposing **6 tools** (`decide_policy`, `locate_bounding_boxes`, `decide_custom_questions`, `list_policy_templates`, `get_health_and_gpu_status`, `warmup_gpu`) with `backend` (`vertex_first` \| `vertex` \| `cloudrun`) and Stage 2 Gemini Cascade support (`cascade_mode`, `cascade_threshold`, `cascade_model`). |
| **3. 🌐 HTTP Gateway REST API** | `POST /api/decide/{template}`<br>`POST /v1/systemone`, `GET /api/templates` | Execute any `.json.tmpl` decision policy or `/v1/systemone` schema with `X-DGem-Backend: vertex_first \| vertex \| cloudrun` (`X-DGem-Backend-Used` returned on every response) and optional Stage 2 `gemini-3.8-flash` cascade. |
| **4. ⌨️ CLI & 7 Benchmark Suites** | `./bin/dgem decide --vertex-url ...`<br>`./bin/dgem bench-*` | Direct single-pass decisions (`--stats`, `--vertex-url 4217256562927861760`) and seven reproducible evaluation harnesses (`bench`, `bench-ecotone`, `bench-intents`, `bench-calibration`, `bench-bbox`, `bench-rerank`, `bench-jev`) backed by [`docs/experiments/`](docs/experiments/README.md) (`EXP-01` – `EXP-13`). |

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
| **Inference Latency** | **125 – 490 ms** (1-pass Vertex AI L4 / Cloud Run GPU / Metal) | ~5 – 25 ms (single head) | **17,486.6 ms** (~17.5s for 3-slot JSON + CoT) | **1.35 – 8.68 ms** (`1.54 ms` p50 over UDS) |
| **Latency Scaling Law** | **$O(K_{\text{steps}})$ constant time** (1 or 12 joint slots take same pass) | $O(M_{\text{heads}})$ separate classifiers per attribute | **$O(T_{\text{output}})$ linear penalty** (serial token loop) | $O(N_{\text{chars}})$ graph traversal |
| **Joint Slot Conditioning** | **Bidirectional (`slot_1 <-> slot_2`)** in a single forward pass | Independent static classification heads | Unidirectional causal bias (`left -> right`) | Local sliding window (1–3 tokens) |
| **Epistemic Calibration (`IDC`)** | **Null-Prior + Dual-Mirror + $T^*$** (`0.0326` ECE, `0%` reversal flip, **8.0×** $H$ on `ChaosNLI`) | Overconfident logits out-of-distribution | Uncalibrated sequence-level logprobs | Static tropical semiring arc weights |
| **Guardrail & Policy Accuracy** | **100%** `AgentDrift` hijack, **100%** Prompt Injection, **100%** RAG Grounding | Narrow single-task scope (512–8k context) | High accuracy at 15–25× higher latency | **36.7%** on semiotic polysemy traps |

---

## Supported Deployment Environments (4 Serving Targets)

See **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU (`docs/vertex-ai-vs-cloudrun.md`)](docs/vertex-ai-vs-cloudrun.md)** for the complete architectural comparison and live 30-case benchmark receipts:

| Serving Target | Hardware & Shape | Cold-Start / Wakeup | Avg GPU Denoise (`N=4`) | Avg End-to-End Wall Time | Cost Profile | Recommended Use Case |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **1. Vertex AI Dedicated Endpoint (`/invoke/*`)** | `g2-standard-16` (`1× NVIDIA L4` `24GB` VRAM, `64GB` RAM, ID `4217256562927861760`) | **`0.0 s`** (`minReplicaCount=1`, permanently warm) | **`490.0 ms`** (`195 ms` for `N=1`) | **`536.0 ms`** (`245 ms` for `N=1`) | `~$1.12/hr` while deployed (`$0/hr` after `make vertex-teardown`) | **Primary Production Target (`vertex_first` default)**: Zero cold-start SLA, interactive agents, CI/CD gates, and multimodal `SigLIP` headroom (`64 GB` RAM). |
| **2. Serverless Cloud Run GPU (`dgemma`)** | `1× NVIDIA RTX Pro 6000` (`48GB` VRAM, `80Gi` RAM) or `1× L4` (`24GB`) | **`~121.8 s`** (`0 → 1` scale-from-zero) | **`427.3 ms`** (`171 ms` for `N=1`) | **`459.0 ms`** (`199 ms` for `N=1`) | **`$0.00/hr` when idle** (`min-instances=0`) | **Scale-to-Zero & Auto-Failover Standby (`cloudrun`)**: Episodic batch jobs, research evaluations, and zero-idle-cost sandboxes. |
| **3. Cloud GPU on GCE VM** | `g2-standard-8` (`1× L4` `NVFP4`) or `a2-highgpu-2g` (`2× A100` `bfloat16`) | **`0.0 s`** (dedicated VM) | — | **`1,968.7 ms`** (`L4`) / **`2,733 ms`** (`2× A100`) | `~$0.70/hr` (`L4`) / `~$7.34/hr` (`2× A100`) | High-throughput raw `vLLM` continuous batching (`Banking77` / `CLINC150`) & `bfloat16` precision baselines. |
| **4. Local Apple Silicon (`Metal`)** | Apple M-Series (`diffgemma-26b-a4b-it-q4` unified RAM) | **`0.0 s`** (local daemon) | **`892.0 ms`** (`210 ms` for `N=1`) | **`898.5 ms`** | **`$0.00/hr`** (local hardware) | Offline laptop development, policy authoring, and local verification. |

### Option A: Vertex AI Dedicated Endpoint (`/invoke/*`, Recommended Primary)
Deploys the `dgemma` container with arbitrary custom routes (`invokeRoutePrefix: "/*"`) onto a `g2-standard-16` (`1× NVIDIA L4`, `64 GB` RAM) Vertex AI Dedicated Endpoint (`4217256562927861760`) so `/invoke/v1/chat/completions`, `/invoke/v1/systemone`, and `/invoke/health` are served with **`0.0 s` wakeup**:
```bash
# 1. Deploy dgemma to Vertex AI Dedicated Endpoint (g2-standard-16, 1× NVIDIA L4):
make vertex-deploy

# 2. Run single-pass decision or 30-case benchmark directly against /invoke/v1:
./bin/dgem decide --vertex-url 4217256562927861760 --gcp-auth \
  -t templates/support_triage.json.tmpl -v 'ticket=Emergency outage' --stats
./bin/dgem bench --vertex-url 4217256562927861760 --gcp-auth \
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

* **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU](docs/vertex-ai-vs-cloudrun.md)**: Architectural comparison, arbitrary custom route forwarding, `g2-standard-16` (`64 GB` RAM) sizing, and live 30-case benchmark receipts.
* **[Experiment Authoring Guide & Backend Target Selection](docs/experiment-authoring-guide.md)**: Choosing between `vertex_first`, `vertex`, and `cloudrun`, and configuring Stage 2 Gemini Cascades (`gemini-3.8-flash` default).
* **[CLI, HTTP Gateway & MCP Reference](docs/cli-reference.md)**: Complete flag and tool parameter reference (`--vertex-url`, `dgem serve --default-backend vertex_first`, `/v1/systemone`, and MCP tools).
* **[The Journey to Decision Models](docs/decision-models-primer.md)**: Architectural primer contrasting Classical ML, Symbolic WFSTs, Autoregressive LLMs, and Discrete Diffusion Decision Models.
* **[Experiments & Research Ledger (`docs/experiments/`)](docs/experiments/README.md)**: Structured log of completed empirical studies (`EXP-01` through `EXP-13`) and active architectural investigations (`EXP-05` Entropy-Gated Cascades, `EXP-06` Encoder Comparisons, `EXP-07` Conditional Policy DAGs).
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
