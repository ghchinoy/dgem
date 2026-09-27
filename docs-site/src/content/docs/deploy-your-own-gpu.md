---
title: "Deploy on Your Own Cloud GPU"
description: "Deploy dgem using public zero-credential container images to Google Cloud Run GPU or Vertex AI Dedicated Endpoints."
---

# Deploy dgem on Your Own Cloud GPU

DiffusionGemma (`dgem`) container images are published to Google Artifact Registry with public read access (`allUsers`). Anyone can pull and deploy them directly to their own Google Cloud project, on-premises Kubernetes cluster, or GPU workstation without creating a service account key or configuring Google Cloud credentials.

---

## 1. Public Container Images in Artifact Registry

Images are hosted in `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/`:

### A. Lean Variant (`dgem:latest` or `dgem:<git-sha>`)
* **Size**: `~10 GB` compressed.
* **Weights**: Downloads public, ungated NVFP4 weights from [`nvidia/diffusiongemma-26B-A4B-it-NVFP4`](https://huggingface.co/nvidia/diffusiongemma-26B-A4B-it-NVFP4) at container startup (or mounts local storage / GCS).
* **Image URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest`

### B. Self-Contained Variant (`dgem-weights:latest` or `dgem-weights:<git-sha>`)
* **Size**: `~26 GB` compressed.
* **Weights**: NVFP4 weights pre-baked into `/opt/dgemma/weights`. **0.0s network download** on cold start.
* **Image URI**: `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:latest`

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
gcloud beta quotas info describe CustomModelServingNvidiaRtxPro6000GpusPerProjectRegion \
  --service=aiplatform.googleapis.com \
  --project="<your-project-id>" \
  --format="value(dimensionsInfos[0].details.value)"
```

---

## 3. Deploying to Serverless Cloud Run GPU

Deploy using the pre-baked weights image for rapid startup without Hugging Face downloads:

```bash
export GCP_PROJECT="<your-project-id>"
export GCP_REGION="us-central1"
export IMAGE="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:latest"

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
export IMAGE_URI="us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest"
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
