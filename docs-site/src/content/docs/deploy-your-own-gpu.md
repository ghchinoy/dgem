---
title: "Deploy on Your Own Cloud GPU"
description: "Deploy dgem using public zero-credential container images to Google Cloud Run GPU or Vertex AI Dedicated Endpoints."
---

# Deploy dgem on Your Own Cloud GPU

DiffusionGemma (`dgem`) container images are published to Google Artifact Registry with public read access (`allUsers`). Anyone can pull and deploy them directly to their own Google Cloud project, on-premises Kubernetes cluster, or GPU workstation without creating a service account key or configuring Google Cloud credentials.

---

## 1. Public Container Images in Artifact Registry

Images are hosted in `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/`:

### A. Lean Variant (`dgem:56baadf` or `dgem:latest`)
* **Size**: `~10 GB` compressed.
* **Weights**: Downloads public, ungated NVFP4 weights from [`nvidia/diffusiongemma-26B-A4B-it-NVFP4`](https://huggingface.co/nvidia/diffusiongemma-26B-A4B-it-NVFP4) at container startup (or mounts local storage / GCS).
* **Pinned URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:56baadf@sha256:a7ace753973b6c3521dbc4c62ea4dfbea5384c582884e98a5ccae9f81b1f6dd9`

### B. Self-Contained Variant (`dgem-weights:56baadf` or `dgem-weights:latest`)
* **Size**: `~26 GB` compressed.
* **Weights**: NVFP4 weights pre-baked into `/opt/dgemma/weights`. **0.0s network download** on cold start.
* **Pinned URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:56baadf@sha256:893f45a29e774bcda67ec66574f6b084c878795f95ecd9301a9d424cd726d36a`

---

## 2. Prerequisite GPU Quotas

Before deploying to Cloud Run GPU or Vertex AI, verify that your Google Cloud project has active GPU quota in your chosen region (e.g. `us-central1`):

### Cloud Run GPU Quota Check
```bash
gcloud beta quotas info describe NvidiaRtxPro6000GpuAllocNoZonalRedundancyPerProjectRegion \
  --service=run.googleapis.com \
  --project="<your-project-id>" \
  --format="value(dimensionsInfos[0].details.value)"
```
* If `0` or unset: Request a quota increase in the Google Cloud Console under **IAM & Admin $\to$ Quotas $\to$ Cloud Run Admin API $\to$ `NvidiaRtxPro6000GpuAllocNoZonalRedundancyPerProjectRegion`** (or `NvidiaL4GpuAllocNoZonalRedundancyPerProjectRegion`).

### Vertex AI GPU Quota Check
```bash
gcloud beta quotas info describe CustomModelServingRTXPRO6000GPUsPerProjectPerRegion \
  --service=aiplatform.googleapis.com \
  --project="<your-project-id>" \
  --format="value(dimensionsInfos[0].details.value)"
```
*Note: In many newly activated GCP projects, Cloud Run GPU quota is 0 by default while Vertex AI custom model serving quota (`CustomModelServingRTXPRO6000GPUsPerProjectPerRegion` or `CustomModelServingL4GPUsPerProjectPerRegion`) is already enabled. If Cloud Run GPU quota is 0, Vertex AI Dedicated Endpoints offer the fastest path to running.*

---

## 3. Deploying to Serverless Cloud Run GPU

Deploy using the pre-baked weights image for rapid startup without Hugging Face downloads:

```bash
export GCP_PROJECT="<your-project-id>"
export GCP_REGION="us-central1"
export IMAGE="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:56baadf@sha256:893f45a29e774bcda67ec66574f6b084c878795f95ecd9301a9d424cd726d36a"

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

*For L4 GPUs (`--gpu-type=nvidia-l4 --memory=32Gi`), set `--set-env-vars="DISABLE_MM=1"` to disable the multimodal SigLIP vision tower if operating within the 32Gi RAM ceiling.*

### Evaluating on Decision Index (`ROLE=decision-index`)
To run wide-canvas multi-candidate evaluations (such as `apolinario/decision-index`), launch with `ROLE=decision-index`:

```bash
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

---

## 4. Deploying to Vertex AI Dedicated Endpoints

For persistent, low-latency production endpoints, deploy using the automated provisioning script:

```bash
export GCP_PROJECT="<your-project-id>"
export GCP_REGION="us-central1"
export IMAGE_URI="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:56baadf@sha256:a7ace753973b6c3521dbc4c62ea4dfbea5384c582884e98a5ccae9f81b1f6dd9"
export VERTEX_PROFILE="g4-rtxpro6000"

./scripts/deploy_vertex_endpoint.sh
```

This deploys a Blackwell G4 (`g4-standard-48` + 1× RTX PRO 6000) dedicated endpoint with custom routes (`/invoke/*`), delivering ~57.5 ms single-read denoise latencies.

---

## 5. Verification & Querying

Once deployed, retrieve the service URL and issue a test structured decision:

```bash
ENDPOINT_URL=$(gcloud run services describe dgemma --project="$GCP_PROJECT" --region="$GCP_REGION" --format="value(status.url)")

# Check health
curl -s "${ENDPOINT_URL}/health"

# Run a test policy evaluation using the dgem CLI
./bin/dgem decide -u "${ENDPOINT_URL}/v1" \
  -t templates/support_triage.json.tmpl \
  -v "ticket_text=Database connection pool exhausted"
```

---

## 6. Teardown (Zero Idle Cost)

To avoid incurring ongoing GPU charges when testing is complete:

### Cloud Run Teardown
```bash
gcloud run services delete dgemma \
  --project="$GCP_PROJECT" \
  --region="$GCP_REGION" \
  --quiet
```

### Vertex AI Teardown
```bash
GCP_PROJECT="<your-project-id>" make vertex-teardown
```
