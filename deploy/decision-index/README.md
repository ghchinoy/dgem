# `dgem` Decision Index Public Container

This directory provides the official public container for evaluating **DiffusionGemma (`dgem`)** against the **Decision Index** benchmark ([`apolinario/decision-index`](https://github.com/apolinario/decision-index) / [`multimodalart/jev-decision-index`](https://huggingface.co/spaces/multimodalart/jev-decision-index)).

The container exposes a standard **`POST /v1/systemone`** endpoint that wraps DiffusionGemma with `dgem`'s Wide-Canvas and Bracket Tournament Adapter, eliminating capacity refusals on high-cardinality questions ($K > 26$) and multi-question requests ($M > 8$).

---

## 1. Quickstart: Running with Docker

### Pull Public Image (No Google Cloud Auth Required)
```bash
# Public Artifact Registry image (open to allUsers)
docker pull us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-systemone:latest
```

### Launch on an NVIDIA GPU (RTX PRO 6000, L4, A100, etc.)
```bash
docker run --gpus all \
  -p 8080:8080 \
  -e TEMPERATURE=1.0 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-systemone:latest
```
On startup, the container automatically downloads the public, ungated weights from [`nvidia/diffusiongemma-26B-A4B-it-NVFP4`](https://huggingface.co/nvidia/diffusiongemma-26B-A4B-it-NVFP4) if no local volume is mounted.

To mount existing weights locally:
```bash
docker run --gpus all \
  -v /path/to/weights:/mnt/gcs/dgemma:ro \
  -p 8080:8080 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-systemone:latest
```

---

## 2. Running the Official Upstream Benchmark

Once the container is running and healthy (`curl http://localhost:8080/health`), run the official Decision Index evaluation:

```bash
# Test sample
python -m decision_index run \
  --engine http \
  --option base_url=http://127.0.0.1:8080 \
  --option model=dgem \
  --rows sample-100.jsonl.gz \
  --out runs/dgem-sample

# Full Edition 0.2.1 suite
python -m decision_index pipeline \
  --engine http \
  --option base_url=http://127.0.0.1:8080 \
  --option model=dgem \
  --out runs/dgem-0.2.1
```

---

## 3. Architecture & Features

```
Client (decision_index runner)
   │
   ▼ POST /v1/systemone (port 8080)
┌────────────────────────────────────────────────────────────────────────┐
│ dgem systemone adapter                                                 │
│  • Multi-Slot Canvas Batching: slices requests with M > 8 questions    │
│  • 2-Stage Bracket Tournaments: slices K > 26 choices into brackets    │
│  • Normalizes probabilities sum(p_k) = 1.0 (passes validate())         │
│  • Boolean / noul support (RAGTruth, PhishNChips)                      │
│  • Optional Bearer token authentication (-e API_KEY=...)               │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ POST /v1/chat/completions (port 8081)
┌──────────────────────────────────▼─────────────────────────────────────┐
│ vLLM + structured_server.py (PR #57250)                                │
│  • DiffusionGemma 26B-A4B-it NVFP4                                    │
│  • Bidirectional discrete canvas slot denoise                         │
└────────────────────────────────────────────────────────────────────────┘
```
