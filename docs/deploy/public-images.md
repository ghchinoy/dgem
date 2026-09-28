---
title: "Public Container Images & Quickstart"
description: "How to pull and run official dgem DiffusionGemma container images directly from Google Artifact Registry without GCP credentials, and evaluate against Decision Index and JevBench."
---

# Public Container Images & Quickstart

`dgem` publishes official, public container images to Google Artifact Registry open to all users globally (`allUsers` read permissions). No Google Cloud account, IAM credentials, or service account keys are required to pull them.

---

## 1. Published Container Images

| Image Tag | Digest | Compressed Size | Weights Delivery | Recommended Use Case |
| :--- | :--- | :---: | :--- | :--- |
| **`dgem:latest`** (or `dgem:56baadf`) | `sha256:a7ace753973b...` | `~10 GB` | Public HF download at container boot (`0.0s` if mounted or pre-cached) | **Ephemeral Cloud Run GPU, GCE VMs, Local Docker** |
| **`dgem-weights:latest`** (or `dgem-weights:56baadf`) | `sha256:893f45a29e77...` | `~26 GB` | **Pre-baked NVFP4 weights** in `/opt/dgemma/weights` (no weight download at boot, but a ~12-minute first image pull on Cloud Run; see Path to Production §5) | **Dedicated Vertex AI Endpoints, Offline Pods** |

For deploying to Google Cloud Run GPU or Vertex AI Dedicated Endpoints, see [Deploy on Cloud Run](cloud-run.md) or [Production on Vertex AI](vertex.md).

### Pulling the Image
```bash
# Pull lean image (pinned digest):
docker pull us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:56baadf@sha256:a7ace753973b6c3521dbc4c62ea4dfbea5384c582884e98a5ccae9f81b1f6dd9

# Or pull self-contained image with pre-baked NVFP4 weights (pinned digest):
docker pull us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:56baadf@sha256:893f45a29e774bcda67ec66574f6b084c878795f95ecd9301a9d424cd726d36a
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
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest
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
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest
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
| **`WEIGHTS_SOURCE`** | `hf` (or `baked`) | `baked` (uses `/opt/dgemma/weights`), `hf` (downloads from Hugging Face), `gcs`, or `local`. |
| **`MODEL_HF`** | `nvidia/...NVFP4` | Public Hugging Face repo ID to download when `WEIGHTS_SOURCE=hf`. |

---

## 4. Container Lineage & Provenance

* **Base Image**: Official `vllm/vllm-openai:nightly` pinned by commit and digest (`vllm-project/vllm:main` commit `a9eafde59cb`).
* **PR Inclusions**: Upstream PR #57250 (*"[Core] structured generation mode for DiffusionGemma"*) and PR #58216 (*"[Perf] constrained reads"*) are natively included in the base image.
* **Patches**: Pinned in `deploy/cloudrun/server/dgem.patch` and tracked via `deploy/cloudrun/server/UPSTREAM`.
