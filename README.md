# dgem (DiffusionGemma CLI & Assistant)

High-performance CLI assistant, evaluation harness, and automation engine for Google DeepMind's **DiffusionGemma** (26B-A4B-it), supporting both **Local Apple Silicon (macOS Metal)** and **Cloud GPU (Google Compute Engine NVIDIA L4, A100, and H100)** deployments.

`dgem` evaluates discrete diffusion slot readouts—evaluating structured propositions, categorical choices, and multi-field routing decisions in a single forward pass without the latency or syntax failure of autoregressive text generation.

---

## Architecture: Discrete Diffusion Slot Readout vs. Autoregression

DiffusionGemma operates on a **256-token canvas** with bidirectional attention. Instead of sequentially generating JSON token-by-token across hundreds of autoregressive forward passes, `dgem` pre-seeds the prompt and canvas, evaluating logits directly at token slots:

| Evaluation Dimension | Discrete Diffusion Slot Readout (`dgem`) | Autoregressive LLM (Gemma 4 / Gemini) | Compiled Rulebook (`ecotone` C++ WFST) |
| :--- | :--- | :--- | :--- |
| **Inference Latency** | **425 – 960 ms** (1-slot Metal) / **458.9 ms** (Cloud Run L4) / **1,968 ms** (GCE L4) | **17,486.6 ms** (~17.5s for 3-slot JSON + CoT) | **1.35 – 8.68 ms** (`1.54 ms` p50 over UDS) |
| **Latency Scaling Law** | **$O(K_{\text{steps}})$ constant time** (1 or 5 slots take same time) | **$O(T_{\text{output}})$ linear penalty** (token-by-token bottleneck) | $O(N_{\text{chars}})$ graph traversal |
| **Output Reliability** | **100% Schema-Guaranteed** (0% parse errors) | Vulnerable to syntax drift & markdown wrappers | Deterministic pattern replacement |
| **Semiotic Polysemy** | **90.0% – 93.3%** (resolves *"123 St. Mark St."* and *"Ocean Dr."*) | ~90% (at 15× higher latency) | **36.7%** (collapses to default arc or drops `St.`) |
| **High-Cardinality NLU** | **96.7%** on CLINC150, **86.7%** on Banking77, **100% OOS** | Prone to left-to-right shared-prefix bias | Requires manual rulebooks per category |
| **Uncertainty Telemetry** | Calibrated $\exp(\text{logprob})$ confidence & Shannon entropy $H$ | Uncalibrated sequence logprobs | Static arc weights |

---

## Supported Deployment Environments

`dgem` is client-agnostic and connects to any OpenAI-compatible or native Jev endpoint:

```
                  ┌───────────────────────────────┐
                  │           dgem CLI            │
                  └───────────────┬───────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         ▼                        ▼                        ▼
Local Apple Silicon (Metal)  Cloud Run Serverless GPU   Cloud GPU on GCE VM
• diffgemma serve (:8080)    • 1× NVIDIA L4 (24GB)      • 1× L4 (NVFP4) / 2× A100 (bf16)
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

## Benchmark Suites

`dgem` includes three rigorous empirical benchmark harnesses:

### 1. Multi-Domain Triage Benchmark (`dgem bench`)
Evaluates 30 multi-field test cases across `support`, `code_review`, and `security` ([`benchmarks/eval_dataset.jsonl`](benchmarks/eval_dataset.jsonl)):
```bash
./bin/dgem bench -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results.json
```

### 2. Ecotone WFST vs. DiffusionGemma (`dgem bench-ecotone`)
Evaluates 49 Text Normalization cases comparing C++ `ecotone` (OpenFst / Sparrowhawk WFSTs over `unix:///tmp/ecotone.sock`) against DiffusionGemma across semiotic polysemy traps and deterministic NSWs:
```bash
./bin/dgem bench-ecotone \
  -c benchmarks/ecotone/tn_semiotics.jsonl \
  --samples 1 \
  -o benchmarks/results_ecotone.json
```

### 3. High-Cardinality Intent & Out-of-Scope Routing (`dgem bench-intents`)
Evaluates 30-way to 151-way intent routing and Out-of-Scope (`oos`) rejection on **`PolyAI/banking77`** and **`DeepPavlov/clinc150`**:
```bash
# Curated high-collision evaluation:
./bin/dgem bench-intents --dataset banking77 -o benchmarks/results_b77.json
./bin/dgem bench-intents --dataset clinc150 -o benchmarks/results_c150.json

# Run the FULL 3,080-item Banking77 test split with 16 parallel workers:
./bin/dgem bench-intents --dataset banking77 --full --workers 16

# Run the FULL 5,500-item CLINC150 test + OOS split:
./bin/dgem bench-intents --dataset clinc150 --full --workers 16
```

---

## Documentation

* **[Benchmark Evaluation Report](docs/benchmarks-report.md)**: Full empirical receipts comparing Apple M5 Metal, GCE 1× L4, GCE 2× A100 `bfloat16`, Banking77, and CLINC150.
* **[Ecotone (WFST) vs. DiffusionGemma](docs/ecotone-comparison.md)**: Semiotic polysemy taxonomy, head-to-head findings, and the hybrid Cascaded Normalizer architecture.
* **[Architecture: Discrete Diffusion vs. Autoregression](docs/architecture.md)**: Mechanical breakdown of 256-token canvas denoising, bidirectional slot readout, and terminology history.
* **[Cloud Run Lessons Learned & Native CUDA Build Guide](docs/cloudrun-lessons-learned.md)**: C++ ABI compatibility findings, Hugging Face Hub egress rate limiting, and blueprint for custom CUDA C++ builds.
* **[Remote Endpoints & Cloud Deployment](docs/remote-endpoints.md)**: Pointing `dgem` to Google Cloud GCE GPU instances, Vertex AI, and hosted vLLM clusters.
* **[Real-World Applications & Production Patterns](docs/applications.md)**: Production architectures for agentic dispatch, DevSecOps git hooks, and SIEM alert triage.
* **[User Guide](docs/user-guide.md)**: Full CLI reference, template authoring, interpreting `--stats`, and CI/CD integration.
* **[Setup & Metal Engine Guide](docs/setup.md)**: Hardware requirements, memory budgeting (`--ctx 32768`), and local Metal serving.

---

## Contributing

Issues, bug reports, and feature discussions are welcome! However, **we are not accepting pull requests (PRs) at this time**. If you encounter a bug or have feedback on benchmark methodologies or templates, please open an [Issue](https://github.com/ghchinoy/dgem/issues).

## License

This project is licensed under the [Apache-2.0 License](LICENSE).

## Disclaimer

> [!CAUTION]
> This is **not** an officially supported Google product.
> This project is not eligible for the [Google Open Source Software Vulnerability Rewards Program](https://bughunters.google.com/open-source-security).
