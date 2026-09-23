---
title: "Vertex AI Endpoints (/invoke/*) vs. Cloud Run GPU"
description: "Architectural comparison of Google Cloud Vertex AI Dedicated Endpoints with arbitrary custom routes (invokeRoutePrefix=\"/*\") vs. Serverless Cloud Run GPU for DiffusionGemma (dgemma)."
---

# Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU

`dgem` supports **four execution infrastructures** using the exact same `.json.tmpl` decision policies, OpenTelemetry instrumentation, and `dgemma` container (`us-central1-docker.pkg.dev/genai-blackbelt-fishfooding/dgem/dgemma:latest`):

1. **Local Apple Silicon Metal (`diffgemma`)** — Developer laptop prototyping & offline eval.
2. **GCE VM (`g2-standard-8` L4 / `a2-highgpu-2g` A100)** — Raw VM benchmarking & custom kernel profiling.
3. **Serverless Cloud Run GPU (`dgemma`)** — **Default production backend** (`--min-instances=0`, scale-to-zero `$0/hr` idle cost, NVIDIA RTX Pro 6000 48GB or L4 24GB).
4. **Vertex AI Dedicated Endpoints (`dgemma-dedicated` with `invokeRoutePrefix: "/*"`)** — **Enterprise MLOps backend** exposing all `structured_server.py` and `vLLM` routes directly via `/invoke/*`.

---

## 1. Why We Use Vertex AI Arbitrary Custom Routes (`invokeRoutePrefix: "/*"`)

Standard Vertex AI Online Prediction (`:predict` and `:rawPredict`) binds a container deployment to a **single fixed HTTP path** (`AIP_PREDICT_ROUTE`). However, the `dgemma` container (`structured_server.py` on port `8080` fronting `vLLM EngineCore` on port `8000`) exposes **four distinct HTTP routes**:

- `POST /v1/chat/completions` — Structured 128-token diffusion decision envelope (`answers` + `diagnostics`).
- `POST /v1/raw/chat/completions` — Direct pass-through to `vLLM`'s raw `/v1/chat/completions`.
- `POST /v1/systemone` — Multipart image + JSON decision route (`JevBench` / `SystemOne`).
- `GET /health` — Live container warmup phase, staged GiB telemetry, and `vllm_ready` flag.

By uploading the model with **[`invokeRoutePrefix: "/*"`](https://docs.cloud.google.com/gemini-enterprise-agent-platform/machine-learning/predictions/use-arbitrary-custom-routes)** and deploying it to a **Vertex AI Dedicated Endpoint** (`dedicatedEndpointEnabled: true`), Vertex AI forwards any non-root path under `/invoke/<path>` **verbatim** as `/<path>` to `structured_server.py`:

```mermaid
flowchart LR
  subgraph Client["dgem CLI / dgemma-gateway / Web Studio"]
    GW["dgemma-gateway\n(X-DGem-Backend: vertex)"]
  end

  subgraph Vertex["Vertex AI Dedicated Endpoint (*.prediction.vertexai.goog)"]
    INV["/v1/projects/.../endpoints/{ID}/invoke/*"]
  end

  subgraph Container["dgemma Container (port 8080 -> vLLM 8000)"]
    R1["POST /v1/chat/completions\n(Structured Decision Envelope)"]
    R2["POST /v1/raw/chat/completions\n(Raw vLLM Pass-Through)"]
    R3["POST /v1/systemone\n(SystemOne / JevBench)"]
    R4["GET /health\n(Warmup & vLLM Readiness)"]
  end

  GW -->|"Bearer <OAuth2 cloud-platform>"| INV
  INV -->|"/invoke/v1/chat/completions"| R1
  INV -->|"/invoke/v1/raw/chat/completions"| R2
  INV -->|"/invoke/v1/systemone"| R3
  INV -->|"/invoke/health"| R4
```

---

## 2. Side-by-Side Comparison: Cloud Run GPU vs. Vertex AI Dedicated Endpoints

| Dimension | Serverless Cloud Run GPU (`dgemma`) | Vertex AI Dedicated Endpoint (`dgemma-dedicated`) |
| :--- | :--- | :--- |
| **Scale-to-Zero (`min=0`)** | **Yes (`--min-instances=0`)** — Automatically scales down after 15 min idle (`$0.00/hr` when idle). | **No (`minReplicaCount >= 1`)** — Bills continuously (`~$0.95/hr` for 1× L4 or `~$2.30/hr` for 2× L4) until undeployed (`make vertex-teardown`). |
| **Cold-Start / Provisioning** | **~89s** from `0 → 1` (64-stream GCS HTTPS Range API into `/tmp/dgemma` overlapped with `vLLM` + `SigLIP` init). | **~12–18 min** initial VM + container provisioning; once `Deployed`, replicas stay permanently warm (`0s` wakeup). |
| **Supported GPU Shapes** | `1× NVIDIA L4` (`24GB VRAM`, `32Gi RAM`) or `1× NVIDIA RTX Pro 6000 Blackwell` (`48GB VRAM`, `80Gi RAM`). | `1..8× NVIDIA L4` (`g2-standard-8` to `g2-standard-96`), `1..8× NVIDIA A100` (`a2-highgpu-*`), `8× NVIDIA H100` (`a3-highgpu-8g`). |
| **Multimodal (`bfloat16` + `SigLIP`)** | Single-GPU `RTX Pro 6000` (`48GB VRAM`, `80Gi RAM`) runs full `bfloat16` + `SigLIP` with zero tensor parallelism. | Requires `g2-standard-24` (`2× NVIDIA L4 = 48GB VRAM`, `--tensor-parallel-size 2`) or `a2-highgpu-1g` (`A100 40GB`). |
| **Routing Protocol** | Direct HTTPS to `https://dgemma-*.a.run.app/v1/chat/completions` | Arbitrary Custom Routes (`invokeRoutePrefix: "/*"`) via `https://<id>.<region>-<proj_num>.prediction.vertexai.goog/v1/projects/.../endpoints/<id>/invoke/v1/chat/completions` |
| **Authentication Token** | **OIDC Identity Token** (`gcloud auth print-identity-token`, audience = Cloud Run URL) | **OAuth2 Access Token** (`gcloud auth print-access-token`, scope = `cloud-platform`) |
| **Payload Size Limit** | **`32 MiB`** (HTTP/1.1) / Unlimited streaming (HTTP/2) | **`10–32 MiB`** on Dedicated Endpoints (`*.prediction.vertexai.goog`); bypasses the `1.5 MiB` shared `:predict` limit |
| **Health Probe Behavior** | Polls `GET /health` (`200 OK` immediately so gateway can read live staging telemetry during scale-from-zero). | Polls `GET /vertex-health` (`503` while `vLLM` loads weights, `200 OK` once `vllm_ready: true`). |
| **Enterprise MLOps Features** | Revision traffic splitting, Direct Cloud Run IAP, Cloud Logging & Monitoring. | Model Registry versioning, Private Service Connect (PSC) endpoints, traffic-split canary rollouts (`deployedModels/{id}/invoke/*`), DCGM GPU AutoMetrics. |

---

## 2.1 Empirical 30-Case Benchmark Comparison (`dgem bench`)

We evaluated the exact same 30-case multi-domain decision suite ([`benchmarks/eval_dataset.jsonl`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/eval_dataset.jsonl) across `support`, `code_review`, and `security`) on our live **Vertex AI Dedicated Endpoint (`4217256562927861760`, `g2-standard-16` · `1× NVIDIA L4` · `/invoke/v1`)** and **Serverless Cloud Run GPU (`dgemma`)**:

| Backend Target | Hardware Profile | Cold-Start / Wakeup | Single-Pass (`N=1`) Denoise | 4-Sample (`N=4`) Avg GPU Denoise | Avg End-to-End Wall Time (`30 cases`) | Proxy / Network Overhead | Multi-Domain Slot Accuracy | Receipt |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Vertex AI Dedicated Endpoint (`/invoke/v1`)** | `g2-standard-16` (`1× NVIDIA L4` 24GB VRAM, `64GB` RAM) | **`0.0 s`** (`minReplicaCount=1`, always warm) | **`195 ms`** (`245 ms` wall) | **`490.0 ms`** (`~122.5 ms/read`) | **`536.0 ms`** | **`46.0 ms`** | **`76.7%`** (`23/30`) | `benchmarks/results_vertex_l4_invoke.json` |
| **Serverless Cloud Run GPU (`dgemma`)** | `1× NVIDIA RTX Pro 6000` (`48GB VRAM`, `80Gi` RAM) | **`~121.8 s`** (`0 → 1` scale-from-zero, `$0.00/hr` idle) | **`171 ms`** (`199 ms` wall) | **`427.3 ms`** (`~106.8 ms/read`) | **`459.0 ms`** | **`31.7 ms`** | **`80.0%`** (`24/30`) | `benchmarks/results_cloudrun.json` |
| **GCE VM (Raw vLLM without `structured_server.py`)** | `g2-standard-8` (`1× NVIDIA L4` 24GB VRAM, `32GB` RAM) | **`0.0 s`** (dedicated VM) | — | — | **`1,968.7 ms`** (`3.67×` slower) | — | **`73.3%`** (`22/30`) | `benchmarks/results_gce_l4.json` |
| **Local Apple Silicon Metal (`diffgemma`)** | Apple M-Series (`q4` unified memory) | **`0.0 s`** (local daemon) | **`210 ms`** | **`892.0 ms`** | **`898.5 ms`** | **`6.5 ms`** | **`80.0%`** (`24/30`) | `benchmarks/results_local_metal_slot.json` |

### Key Engineering Takeaways from Live Deployment
1. **Why `g2-standard-16` (`64 GB` RAM) is Required for `1× NVIDIA L4` on Vertex AI**:
   - `g2-standard-8` (`1× NVIDIA L4`) provides only **`32 GB` of host RAM**. Staging the `17.53 GiB` `dgemma` safetensors into `/tmp/dgemma` (`tmpfs` in RAM) while `vLLM`/PyTorch allocates a `17.53 GiB` CPU load buffer requires **`~35.1 GiB` of RAM**, causing a container OOM kill on `g2-standard-8`.
   - **`g2-standard-16`** attaches the **exact same single `1× NVIDIA L4` GPU** (`24 GB` VRAM) with **`64 GB` of host RAM** and `16 vCPUs`, eliminating the `32 GB` RAM bottleneck for negligible incremental CPU cost.
2. **2-Second `/health` Startup for Vertex AI Health Probes**:
   - Vertex AI Dedicated Endpoints fail `deploy-model` if port `:8080/health` does not respond during container initialization. [`deploy/cloudrun/entrypoint.sh`](https://github.com/ghchinoy/dgem/blob/main/deploy/cloudrun/entrypoint.sh) stages only the `<20 MB` tokenizer + `4 MB` safetensors headers synchronously (`<2 seconds`), launches `structured_server.py` on `:8080` immediately so `/health` returns `200 OK`, and streams the `17.53 GiB` tensor bodies in the background across 64 HTTPS Range streams while `_wait_for_upstream()` holds early inference requests until `vLLM` on `:8000` is ready.

---

## 3. Choosing Between Endpoints Across Web Studio, HTTP API, `/v1/systemone`, and MCP

`dgemma-gateway` defaults to **`vertex_first` (Vertex AI Dedicated Endpoint Primary + Cloud Run Serverless GPU Auto-Failover)** and supports three routing modes across every interface:

| Mode (`backend` / `X-DGem-Backend`) | Routing Behavior | Recommended Use Case |
| :--- | :--- | :--- |
| **`vertex_first` (Default)** | Routes to **Vertex AI Dedicated Endpoint (`4217256562927861760`)** whenever `deployed` (`0s` cold start). Automatically fails over to **Cloud Run GPU (`dgemma`)** if Vertex AI is `deploying` or `quiesced`. | **Internal Teams & Production Agents** (Guaranteed `0s` cold start when Vertex is up; zero downtime during maintenance). |
| **`vertex` (Strict Pin)** | Routes strictly to **Vertex AI (`/invoke/*`)**. Never falls back to Cloud Run. | **Pure Vertex AI Benchmarking** (`dgem bench`). |
| **`cloudrun` (Strict Pin)** | Routes strictly to **Serverless Cloud Run GPU (`dgemma`)**. | **Pure Cloud Run Benchmarking** & scale-to-zero testing. |

### A. In the Web Studio UI (`https://dgemma.aaie.cloud`)
1. Click the **`Vertex First (Auto)` / `Cloud Run GPU` / `Vertex AI (/invoke/*)`** selector in the top header bar to open the solid opaque Backend Target panel.
2. Choose **Vertex First · Cloud Run Failover (Recommended)**, **Cloud Run GPU (Strict)**, or **Vertex AI Strict (`/invoke/*`)**.
3. The panel also displays the live replica status of **Vertex AI Dedicated Endpoint (`dgemma-dedicated` · `4217256562927861760`)** with 1-click **Provision Vertex GPU (1× L4)** and **Teardown Replica ($0/hr)** buttons.

### B. Via HTTP API (`/api/decide`, `/v1/systemone`, `/v1/chat/completions`, `/v1/raw/chat/completions`)
Pass `X-DGem-Backend: vertex_first | vertex | cloudrun` (or query parameter `?backend=vertex` or JSON body field `"backend": "vertex"`) on any gateway route. Every response includes `X-DGem-Backend-Used: vertex | cloudrun`:

```bash
# 1. Policy-as-Template Decision (/api/decide)
curl -sS "https://dgemma.aaie.cloud/api/decide/support_triage" \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  -H "Content-Type: application/json" \
  -H "X-DGem-Backend: vertex_first" \
  -d '{
    "variables": {
      "ticket": "I was charged twice for my Pro subscription and need an immediate refund."
    }
  }' | jq .

# 2. Direct SystemOne Ad-Hoc Schema Evaluation (/v1/systemone)
curl -sS "https://dgemma.aaie.cloud/v1/systemone?backend=vertex" \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "Production database CPU is at 100% and enterprise checkout is failing.",
    "questions": {
      "urgent": {"type": "boolean", "question": "Is this an urgent outage?"},
      "team": {"type": "choice", "question": "Which team owns this?", "criteria": {"engineering": "Infra/DB", "billing": "Invoices"}}
    }
  }' | jq .
```

### C. Via MCP Server (`https://dgemma.aaie.cloud/mcp`)
All three MCP inference tools (**`decide_policy`**, **`decide_custom_questions`** — the MCP equivalent of `/v1/systemone`, and **`locate_bounding_boxes`**) accept an optional `backend` parameter (`"vertex_first"` | `"vertex"` | `"cloudrun"`):

```json
{
  "name": "decide_custom_questions",
  "arguments": {
    "backend": "vertex_first",
    "context": "Production database CPU is at 100% and enterprise checkout is failing.",
    "questions": [
      { "id": "urgent", "type": "boolean", "question": "Is this an urgent outage?" },
      { "id": "team", "type": "choice", "question": "Which team owns this?", "options": ["engineering", "billing", "support"] }
    ]
  }
}
```

### D. Via `dgem` CLI (`--vertex-url`)
Pass `--vertex-url 4217256562927861760` on any `dgem` CLI command (`decide`, `bench`, `bench-calibration`, `bench-rerank`, `bench-jev`):

```bash
# Single decision against Vertex AI Dedicated Endpoint 4217256562927861760:
./bin/dgem decide --vertex-url 4217256562927861760 --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v ticket="I was charged twice for my Pro subscription." -s

# Full 30-case benchmark suite against Vertex AI Dedicated Endpoint:
./bin/dgem bench --vertex-url 4217256562927861760 --gcp-auth \
  -d benchmarks/eval_dataset.jsonl \
  -o benchmarks/results_vertex_l4_invoke.json
```

---

## 4. Deploying & Tearing Down a Vertex AI Dedicated Endpoint

```bash
# 1. Upload dgemma with invokeRoutePrefix="/*" and deploy to a Dedicated Endpoint (g2-standard-16 + 1x L4)
make vertex-deploy

# 2. Undeploy replicas when zero-idle-cost ($0.00/hr) is desired
make vertex-teardown
```
