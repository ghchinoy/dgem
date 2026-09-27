---
title: "Production Testing & GPU Evaluation How-To Guide"
description: "Standard operating procedure for the production testing agent: deploying public dgem container images to dedicated GPU hosting projects, evaluating against Decision Index and JevBench, and enforcing zero-idle-cost teardown."
---

# Production Testing & GPU Evaluation How-To Guide

> **Audience**: For the automated agent or team member managing the **Production Testing Project** (separate from the `dgem-diffusiongemma` distribution-only project).

---

## 1. Project Separation & Architecture

This repository maintains strict separation of concerns across Google Cloud projects:

| Project | Purpose & Billing Role | What Runs Here |
| :--- | :--- | :--- |
| **`dgem-diffusiongemma`** | **Distribution & Public Image Hosting ONLY** | Artifact Registry (`allUsers` public read-only), Cloud Build, BigQuery pull telemetry (`dgem_image_access`). **No running GPUs or workloads.** |
| **`<hosting-project>`** (e.g. `generative-bazaar-001` or dedicated test project) | **GPU Compute & Evaluation** | Cloud Run GPU services, Vertex AI Dedicated Endpoints (`/invoke/*`), and GCE GPU benchmark instances. Billed to testing account. |

---

## 2. Public Container Images in Artifact Registry

Images are published in `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/` with public read access (`allUsers`). No GCP authentication or service account key is needed to pull them:

### A. Lean Variant (`dgem:latest` or `dgem:<git-sha>`)
* **Size**: `~10 GB` compressed.
* **Weights**: Downloads public, ungated weights from [`nvidia/diffusiongemma-26B-A4B-it-NVFP4`](https://huggingface.co/nvidia/diffusiongemma-26B-A4B-it-NVFP4) at container startup (or mounts local storage / GCS).
* **URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest`

### B. Self-Contained Variant (`dgem-weights:latest` or `dgem-weights:<git-sha>`)
* **Size**: `~29 GB` compressed.
* **Weights**: NVFP4 weights pre-baked into `/opt/dgemma/weights`. **0.0s network download** on cold start.
* **URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:latest`

---

## 3. Prerequisite GPU Quotas in the Hosting Project

Before deploying to Cloud Run GPU or Vertex AI, verify that the hosting project has active quota in `us-central1`:

### Cloud Run GPU Quota Check
```bash
gcloud beta quotas info describe NvidiaRtxPro6000GpuAllocNoZonalRedundancyPerProjectRegion \
  --service=run.googleapis.com \
  --project="<hosting-project>" \
  --format="value(dimensionsInfos[0].details.value)"
```
* If `0` or unset: Submit a quota increase via Google Cloud Console: **IAM & Admin $\to$ Quotas $\to$ Cloud Run Admin API $\to$ `NvidiaRtxPro6000GpuAllocNoZonalRedundancyPerProjectRegion`** (or `NvidiaL4GpuAllocNoZonalRedundancyPerProjectRegion`).

### Vertex AI GPU Quota Check
```bash
gcloud beta quotas info describe CustomModelServingNvidiaRtxPro6000GpusPerProjectRegion \
  --service=aiplatform.googleapis.com \
  --project="<hosting-project>" \
  --format="value(dimensionsInfos[0].details.value)"
```

---

## 4. Deploying to Cloud Run GPU (`1× NVIDIA RTX PRO 6000`)

### Mode 1: Decision Index Benchmark Evaluator (`ROLE=decision-index`)
Starts `dgem systemone serve` on port 8080 (handling multi-slot batching and wide-option tournaments) and internal `structured_server.py` on port 8081:

```bash
export GCP_PROJECT="<hosting-project>"
export GCP_REGION="us-central1"
export IMAGE="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest"

gcloud run deploy dgem-decision-eval \
  --project="$GCP_PROJECT" \
  --region="$GCP_REGION" \
  --image="$IMAGE" \
  --gpu=1 \
  --gpu-type=nvidia-rtx-pro-6000 \
  --memory=80Gi \
  --cpu=8 \
  --no-cpu-throttling \
  --min-instances=0 \
  --max-instances=1 \
  --timeout=3600 \
  --set-env-vars="ROLE=decision-index,TEMPERATURE=1.0,CANVAS=128" \
  --allow-unauthenticated
```

### Mode 2: Standard Production Serving (`ROLE=default`)
Serves standard `/v1/chat/completions`, `/v1/systemone` pass-through, and `/health`:
```bash
gcloud run deploy dgemma \
  --project="$GCP_PROJECT" \
  --region="$GCP_REGION" \
  --image="$IMAGE" \
  --gpu=1 \
  --gpu-type=nvidia-rtx-pro-6000 \
  --memory=80Gi \
  --cpu=8 \
  --no-cpu-throttling \
  --min-instances=0 \
  --max-instances=1 \
  --timeout=600 \
  --allow-unauthenticated
```

---

## 5. Deploying to Vertex AI Dedicated Endpoint (Blackwell G4)

Deploy to Vertex AI Dedicated DNS (`4423577720856772608` or custom endpoint) on `g4-standard-48` + 1× RTX PRO 6000:

```bash
export GCP_PROJECT="<hosting-project>"
export GCP_REGION="us-central1"
export IMAGE_URI="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest"
export VERTEX_PROFILE="g4-rtxpro6000"

./scripts/deploy_vertex_endpoint.sh
```

---

## 6. Running Benchmark Evaluations Against the Deployment

### A. Decision Index Official Pipeline (`apolinario/decision-index`)
```bash
ENDPOINT_URL=$(gcloud run services describe dgem-decision-eval --project="$GCP_PROJECT" --region="$GCP_REGION" --format="value(status.url)")

# 1. Quick smoke verification (100 rows)
python -m decision_index run \
  --engine http \
  --option base_url="$ENDPOINT_URL" \
  --option model=dgem \
  --rows sample-100.jsonl.gz \
  --out runs/dgem-smoke

# 2. Full official Edition 0.2.1 suite (120,340 requests)
python -m decision_index pipeline \
  --engine http \
  --option base_url="$ENDPOINT_URL" \
  --option model=dgem \
  --out runs/dgem-v021
```

### B. JevBench v1.4 Official Scoring & Leaderboard Comparison
```bash
# Evaluate against Cloud Run or Vertex AI with live leaderboard placement
./bin/dgem bench-jev \
  -u "${ENDPOINT_URL}/v1" \
  --endpoint-kind gpu \
  --compare-leaderboard \
  -o "benchmarks/runs/$(date +%Y%m%d)-gpu-run/jevbench.json"
```

### C. Public Dataset Calibration & Guardrail Suite
```bash
./bin/dgem bench-calibration \
  -u "${ENDPOINT_URL}/v1" \
  -w 4 \
  -o "benchmarks/runs/$(date +%Y%m%d)-gpu-run/calibration.json"
```

---

## 7. Mandatory Teardown & Zero-Idle-Cost Mandate

**CRITICAL**: Always tear down GPU instances immediately after benchmarking receipts are captured:

### Cloud Run GPU Teardown
```bash
gcloud run services delete dgem-decision-eval \
  --project="$GCP_PROJECT" \
  --region="$GCP_REGION" \
  --quiet
```

### Vertex AI Dedicated Endpoint Teardown
```bash
GCP_PROJECT="<hosting-project>" make vertex-teardown
```

---

## 8. Pull Telemetry & Download Statistics

To monitor public image downloads and adopter traffic in `dgem-diffusiongemma`:
Open Google Cloud Console $\to$ BigQuery in `dgem-diffusiongemma` and query `dgem_image_access`:

```sql
-- Daily Image Pulls by Digest and Client IP
SELECT
  DATE(timestamp) AS pull_date,
  protopayload_auditlog.methodName AS action,
  protopayload_auditlog.requestMetadata.callerIp AS client_ip,
  protopayload_auditlog.requestMetadata.callerSuppliedUserAgent AS user_agent,
  COUNT(1) AS request_count
FROM
  `dgem-diffusiongemma.dgem_image_access.cloudaudit_googleapis_com_data_access`
WHERE
  protopayload_auditlog.serviceName = "artifactregistry.googleapis.com"
GROUP BY 1, 2, 3, 4
ORDER BY pull_date DESC, request_count DESC;
```
