---
title: "Path to Production: Serving dgem at Scale"
description: "Crawl, walk, run: how to take dgem from a laptop to a production Vertex AI endpoint, with measured latency and concurrency scaling, hardware recommendations, and a deploy/verify/rollback checklist."
---

# Path to Production: Serving `dgem` at Scale

This page describes how to take `dgem` from a laptop to production, what each stage is for, and how it
performs. All numbers are measured, from one client, in one session, with 50 requests per cell unless stated
(see [Method](#method)). Identifiers such as `<PROJECT>`, `<REGION>` and `<ENDPOINT_ID>` are placeholders.

## TL;DR

- **Production target:** a Vertex AI Dedicated Endpoint on **`g4-standard-48` + 1× NVIDIA RTX PRO 6000**
  (Blackwell). A 3-question decision takes **~58 ms of GPU time and ~143 ms end to end** (p50), images included.
- **Why this GPU:** the DiffusionGemma checkpoint is **NVFP4** (4-bit). Blackwell GPUs execute FP4 natively;
  L4 (Ada) and A100 (Ampere) do not, and the L4 was about 3× slower on every request type we tried.
- **Use `"samples": 1` by default** (one read). Opt in to `4` only where agreement/stderr matter; never use
  `"auto"` on latency-sensitive paths.
- **Cap concurrent decisions per replica** (`MAX_INFLIGHT`, default 8) so bursts queue instead of overloading
  the engine, and let the endpoint autoscale replicas on GPU duty cycle.
- **Keep a scale-to-zero Cloud Run GPU service** as failover and batch capacity.

## 1. Crawl → Walk → Run

| Stage | Platform | Use it for | Cost model | Cold start | Warm latency (1 sample, p50) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Crawl** | Apple Silicon (`diffgemma`, Metal, q4) | Offline development, writing templates, privacy | Your laptop | None (local daemon) | ~0.9 s (different engine; numbers don't transfer to GPU serving) |
| **Walk** | Cloud Run GPU (1× RTX PRO 6000) | Batch evaluation, research, bursty internal use, failover | Pay per instance-second; **$0 when idle** | Minutes (weights staged from Cloud Storage) | 65 ms GPU / 144 ms end to end |
| **Run** | Vertex AI Dedicated Endpoint, `g4-standard-48` + 1× RTX PRO 6000 | Production: always warm, IAM, autoscaling replicas, monitoring, multimodal | Per replica-hour while deployed | **None** (min replicas ≥ 1) | **58 ms GPU / 143 ms end to end** |

A thin gateway (`dgem serve`) in front of both runtime tiers routes each request to Vertex first and fails
over to Cloud Run automatically (`--default-backend vertex_first`). The same container image, templates,
CLI, HTTP API and MCP server work against every tier.

## 2. Measured performance

### 2.1 Latency by request type

GPU time is the server-side denoise time reported by the engine; end-to-end includes network and routing from
a client in the same region. The L4 column is the previous Vertex configuration, kept for comparison.

| Request type | L4 GPU / e2e | Cloud Run RTX PRO 6000 GPU / e2e | **Vertex G4 RTX PRO 6000 GPU / e2e** |
| :--- | :--- | :--- | :--- |
| 3 questions, `samples: 1` | 188 / 271 ms | 65 / 144 ms | **58 / 143 ms** |
| 3 questions, `samples: 4` (one batch) | 323 / 405 ms | 108 / 187 ms | **98 / 181 ms** |
| 3 questions, `samples: "auto"` (two batches) | 494 / 579 ms | 168 / 248 ms | **151 / 235 ms** |
| 3 questions + reversed-order check, `samples: 1` | 200 / 282 ms | 67 / 147 ms | **62 / 147 ms** |
| Image + 2 questions, `samples: 1` | not supported (vision off) | 79 / 186 ms | **67 / 168 ms** |
| 120-token reasoning trace + 2 questions | 1,425 / 1,507 ms | 408 / 493 ms | **393 / 476 ms** |

Takeaways:

- The RTX PRO 6000 is **~3× faster** than the L4 on every request type, and the Vertex and Cloud Run
  deployments of the same GPU perform the same.
- A fixed `samples: 4` costs **~40 ms more** than a single read on G4; `"auto"` costs ~50 ms more again because
  it usually runs a second, sequential batch.

### 2.2 Concurrency scaling (one replica)

Clients send requests back to back; each cell has `max(50, 4 × clients)` requests.

| Concurrent clients | G4, `samples: 1` throughput / p50 / p95 | G4, `samples: 4` throughput / p50 / p95 | Errors (G4) |
| ---: | :--- | :--- | ---: |
| 1 | 6.9/s · 140 · 151 ms | 5.5/s · 181 · 193 ms | 0 |
| 4 | 21.3/s · 180 · 191 ms | 12.9/s · 234 · 1,126 ms | 0 |
| 8 | 35.3/s · 199 · 210 ms | 23.6/s · 302 · 419 ms | 0 |
| 16 | 60.4/s · 239 · 336 ms | 34.8/s · 438 · 531 ms | 0 |
| 32 | 62.6/s · 480 · 595 ms | 35.7/s · 867 · 964 ms | 0 |

- **One G4 replica sustains ~60 decisions/s** at one sample and ~36/s at four, with no errors at 32 concurrent
  clients: beyond ~16 clients, requests queue (latency rises, throughput plateaus).
- **For more throughput, add replicas** (the endpoint autoscales on GPU duty cycle).
- **Without an in-flight cap the engine can fall over.** The L4 configuration had no cap; four-sample
  requests from 8 concurrent clients crashed the engine, and the replica took about 2 minutes to restart.
  The current container caps concurrent decisions (`MAX_INFLIGHT=8`) and queues the rest for up to 30 s
  before returning `503`.

## 3. Recommendations

**Hardware**
- Use a Blackwell GPU (RTX PRO 6000 on `g4-standard-48`) for NVFP4 weights. It also has room for the vision
  tower, so image decisions work on the same endpoint.
- Avoid splitting the model across several smaller GPUs; it adds inter-GPU traffic without a measured benefit.

**Request design**
- `samples: 1` by default. Use `samples: 4` (one parallel batch) when you need agreement and standard error,
  for example to feed a human-review queue. Avoid `"auto"` when latency matters.
- Put several questions about the same input in **one request**: they share one forward pass.
- Keep answer lists at 26 options or fewer per question; wider lists use a two-stage bracket (two passes).

**Serving configuration**
- Minimum replicas ≥ 1 for zero cold start; maximum replicas sized for peak load (~60 one-sample decisions/s
  per replica); autoscale on GPU duty cycle (~70%).
- Cap concurrent decisions per replica (`MAX_INFLIGHT`) and retry `429`/`503` with backoff in clients.
- Pin container images to an immutable tag or digest; never deploy `:latest` to production.
- Warm up on boot: the container sends one-sample, four-sample and image requests to itself before serving, so
  the first real request doesn't pay kernel compilation.
- Keep secrets (for example a Hugging Face token) in Secret Manager, never in plain environment variables.
- Route through a gateway with automatic failover to a scale-to-zero tier; keep a periodic health probe so an
  idle endpoint doesn't scale to zero unexpectedly.

**Calibration features**
- Per-question hesitation (normalized entropy) is cheap and useful for escalation.
- The reversed-order mirror check (`--dual-mirror`) is a research diagnostic, not a production default; see
  [EXP-14](experiments/exp-14-idc-rerun.md).

## 4. Deploy, verify, roll back

```bash
# 1. Build the serving image with an immutable tag
gcloud builds submit deploy/cloudrun --project=<PROJECT> \
  --tag=<REGION>-docker.pkg.dev/<PROJECT>/dgem/dgemma:<GIT_SHA>

# 2. Deploy to a new Vertex AI Dedicated Endpoint on G4 (RTX PRO 6000), 1-2 replicas, vision on
VERTEX_PROFILE=g4-rtxpro6000 \
IMAGE_URI=<REGION>-docker.pkg.dev/<PROJECT>/dgem/dgemma:<GIT_SHA> \
GCP_PROJECT=<PROJECT> ./scripts/deploy_vertex_endpoint.sh     # ~10 minutes to serving

# 3. Verify: readiness, a text decision, an image decision
curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke/health
#   -> {"vllm_ready": true, "warmed": true, ...}
./bin/dgem decide --vertex-url <ENDPOINT_ID> --gcp-auth -t templates/support_triage.json.tmpl -v ticket="Charged twice" --stats
./bin/dgem decide --vertex-url <ENDPOINT_ID> --gcp-auth -t templates/multimodal/bbox_localization.json.tmpl \
  -I fixtures/bbox/bbox-t1-03-offgrid-card.png -v target_object=checkout_summary_card

# 4. Measure before switching traffic
python3 scripts/serving_speed.py modes --target new=<INVOKE_URL>/v1/chat/completions -n 50 -o modes.json
python3 scripts/serving_speed.py sweep --target new=<INVOKE_URL>/v1/chat/completions -o sweep.json

# 5. Point the gateway at the new endpoint (and the CLI default via DGEM_VERTEX_URL)
DGEM_VERTEX_URL=<ENDPOINT_ID> ./scripts/deploy_cloudrun_gateway.sh
```

**Rollback:** gateways and Cloud Run services roll back by shifting traffic to the previous revision
(`gcloud run services update-traffic <SERVICE> --to-revisions=<PREVIOUS>=100`). For Vertex, deploy the previous
image as a second model on the endpoint, move the traffic split to it, then undeploy the new one.

## Method

- Tool: [`scripts/serving_speed.py`](https://github.com/ghchinoy/dgem/blob/main/scripts/serving_speed.py).
  Every request records HTTP status and error text; failed requests are counted as errors, never as zero-latency.
- Latency cells: 50 sequential requests after 3 warm-up requests; p50 reported. Concurrency cells:
  `max(50, 4 × clients)` requests with a fixed number of client threads.
- Engine settings differed slightly between targets (cache memory, in-flight cap, vision tower); single-request
  latency is dominated by the GPU. Raw receipts:
  [`benchmarks/runs/20260925-serving-speed`](https://github.com/ghchinoy/dgem/tree/main/benchmarks/runs/20260925-serving-speed).
- Single client in the same region; these are service-side characteristics, not end-user internet latencies.
