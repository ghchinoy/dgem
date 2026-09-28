---
title: "Public Container Images & Quickstart"
description: "How to pull and run official dgem DiffusionGemma container images directly from Google Artifact Registry without GCP credentials, and evaluate against Decision Index and JevBench."
---

`dgem` publishes official, public container images to Google Artifact Registry open to all users globally (`allUsers` read permissions). No Google Cloud account, IAM credentials, or service account keys are required to pull them.

---

## 1. Published Container Images

| Image Tag | Digest | Compressed Size | Weights Delivery | Recommended Use Case |
| :--- | :--- | :---: | :--- | :--- |
| **`dgem:4b1b809`** (also `:latest`) | `sha256:edc06728d2e8...` | `~10 GB` | Downloads the public weights from Hugging Face at boot, or mounts them (Cloud Storage / local path) | **Cloud Run GPU (recommended), Vertex AI, GCE VMs, local Docker** |
| **`dgem-weights:4b1b809`** (also `:latest`) | `sha256:7cfbbb9207f5...` | `~26 GB` | **NVFP4 weights baked in** at `/opt/dgemma/weights`; no download at boot, but a large pull (~8–12 minutes on a fresh Cloud Run instance) | **Offline or air-gapped hosts** |

Both images were validated side by side with the production serving image (API contract, latency, 0 errors at 32
concurrent clients) before publishing; see the [image parity run](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260927-image-parity/README.md)
and [Deploy on Cloud Run §7](/dgem/deploy/cloud-run/#7-cold-start-what-to-expect-and-your-options) for cold-start numbers.
Pin by digest in production; `:latest` moves.

For deploying to Google Cloud Run GPU or Vertex AI Dedicated Endpoints, see [Deploy on Cloud Run](/dgem/deploy/cloud-run/) or [Production on Vertex AI](/dgem/deploy/vertex/).

### Pulling the Image
```bash
# Pull lean image (pinned digest):
docker pull us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:4b1b809@sha256:edc06728d2e86c2e9408cc7ac046f2d261cfb2106c3523aa8d4f939f862abcfc

# Or pull self-contained image with pre-baked NVFP4 weights (pinned digest):
docker pull us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:4b1b809@sha256:7cfbbb9207f50cb4ef5d4432c88cde97a893770dc1a04329e8d4d0f6aa0d3c56
```

---

## 2. Running on Any GPU Host (NVIDIA Blackwell, Ada, Hopper, Ampere)

### A. Decision Index Certified Mode (`ROLE=decision-index`)
Starts `dgem systemone serve` on port `8080` (with multi-slot canvas batching and 2-stage bracket tournament routing) in front of internal `structured_server.py` on port `8081` and vLLM on port `8000`:

```bash
docker run --gpus all \
  -p 8080:8080 \
  -e ROLE=decision-index \
  -e TEMPERATURE=1.0 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:4b1b809@sha256:edc06728d2e86c2e9408cc7ac046f2d261cfb2106c3523aa8d4f939f862abcfc
```

Once running and healthy (`curl http://localhost:8080/health`), run the official upstream Decision Index pipeline:
```bash
python -m decision_index pipeline \
  --engine http \
  --option base_url=http://127.0.0.1:8080 \
  --option model=dgem \
  --out runs/dgem-0.2.1
```

### B. Standard OpenAI & Decision Studio Mode (`ROLE=default`)
Serves `/v1/chat/completions` (OpenAI format), `/v1/systemone` pass-through, and `/health`:

```bash
docker run --gpus all \
  -p 8080:8080 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:4b1b809@sha256:edc06728d2e86c2e9408cc7ac046f2d261cfb2106c3523aa8d4f939f862abcfc
```

Then point `dgem` CLI, Decision Studio, or MCP at `http://127.0.0.1:8080/v1`:
```bash
# Single decision via CLI
./bin/dgem decide -u http://127.0.0.1:8080/v1 \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Emergency database outage' --stats

# Full JevBench v1.4 evaluation with leaderboard comparison
./bin/dgem bench-jev -u http://127.0.0.1:8080/v1 --endpoint-kind gpu --compare-leaderboard
```

---

## 3. Environment Variables & Runtime Configuration

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| **`ROLE`** | `default` | `decision-index` (starts wide-canvas tournament adapter on `PORT`) or `default` (direct serving). |
| **`PORT`** | `8080` | External HTTP port to bind. |
| **`TEMPERATURE`** | `1.0` | Post-hoc slot logit temperature scaling $T^*$ ($1.0$ unscaled for official submissions). |
| **`API_KEY`** | `""` (open) | Optional secret key. When set, all incoming requests must supply `Authorization: Bearer <key>`. |
| **`CANVAS`** | `128` | Served diffusion canvas length in tokens. |
| **`MODEL`** | `/mnt/gcs/dgemma` | Weights path. Baked weights at `/opt/dgemma/weights` are used automatically; a mounted path is used if it has `config.json`; otherwise weights are staged from `DGEM_WEIGHTS_URI` (a `gs://` URI) or downloaded from `MODEL_HF`. |
| **`MODEL_HF`** | `nvidia/diffusiongemma-26B-A4B-it-NVFP4` | Hugging Face repo downloaded when no weights are baked, mounted or staged. |
| **`DEFAULT_SAMPLES`**, **`MAX_INFLIGHT`** | `1`, `8` | Samples for schemas without `samples`; decisions processed at once (the rest queue). |
| **`DISABLE_MM`** | `0` | `1` disables the vision tower (smaller GPUs). |

---

## 4. Container Lineage & Provenance

* **Base Image**: Official `vllm/vllm-openai:nightly` pinned by commit and digest (`vllm-project/vllm:main` commit `a9eafde59cb`).
* **PR Inclusions**: Upstream PR #57250 (*"[Core] structured generation mode for DiffusionGemma"*) and PR #58216 (*"[Perf] constrained reads"*) are natively included in the base image.
* **Patches**: Pinned in `deploy/cloudrun/server/dgem.patch` and tracked via `deploy/cloudrun/server/UPSTREAM`.
