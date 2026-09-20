#!/usr/bin/env bash
set -euo pipefail

# Load .env if present
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
elif [[ -f "../.env" ]]; then
  set -a
  source "../.env"
  set +a
fi

# Configuration
PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="${CLOUDRUN_SERVICE_NAME:-djev-dgemma}"
GPU_TYPE="${CLOUDRUN_GPU_TYPE:-nvidia-l4}" # nvidia-l4 or nvidia-rtx-pro-6000
BUCKET="${GCS_BUCKET:-dgem-weights-${PROJECT_ID}}"

# Image resolution: Prefer Artifact Registry image in user's project, fallback to ghcr.io/taeold/djev-run:latest
DEFAULT_AR_IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/dgem/djev-run:latest"
if [[ -n "${CLOUDRUN_IMAGE:-}" ]]; then
  IMAGE_TAG="$CLOUDRUN_IMAGE"
elif gcloud artifacts docker images describe "$DEFAULT_AR_IMAGE" --project="$PROJECT_ID" >/dev/null 2>&1; then
  IMAGE_TAG="$DEFAULT_AR_IMAGE"
else
  IMAGE_TAG="ghcr.io/taeold/djev-run:latest"
fi

if [[ -z "$PROJECT_ID" ]]; then
  echo "Error: No Google Cloud Project ID detected."
  echo "Set GCP_PROJECT=<project-id> or run 'gcloud config set project <project-id>'."
  exit 1
fi

if ! command -v gcloud >/dev/null 2>&1; then
  echo "Error: gcloud CLI not found. Please install the Google Cloud SDK."
  exit 1
fi

echo "================================================================================"
echo "  Deploying DiffusionGemma (djev) to Google Cloud Run with GPU"
echo "================================================================================"
echo "Project:      $PROJECT_ID"
echo "Region:       $REGION"
echo "Service Name: $SERVICE_NAME"
echo "GPU Type:     $GPU_TYPE"
echo "Image:        $IMAGE_TAG"
echo "Weights GCS:  gs://${BUCKET}/dgemma/"
echo "================================================================================"

# 1. Ensure GCS bucket and weights exist
if ! gcloud storage ls "gs://${BUCKET}/dgemma/config.json" >/dev/null 2>&1; then
  echo "==> Weights not found at gs://${BUCKET}/dgemma/."
  echo "==> Automatically staging weights via scripts/stage_model_gcs.sh..."
  ./scripts/stage_model_gcs.sh "$BUCKET" "$REGION"
fi

# 2. Adaptive CPU, Memory, and SHM Staging based on GPU tier
if [[ "$GPU_TYPE" == "nvidia-rtx-pro-6000" ]]; then
  CPU="20"
  MEMORY="80Gi"
  COPY_SHM="1"
  CANVAS_LEN="128"
  MAX_MODEL_LEN="4096"
else
  # Default: 1x NVIDIA L4 (24GB VRAM, 32GB host RAM)
  CPU="8"
  MEMORY="32Gi"
  COPY_SHM="0" # Direct GCS FUSE buffered streaming without filling 32GB RAM
  CANVAS_LEN="32"
  MAX_MODEL_LEN="32768"
fi

ENV_VARS="MODEL=/mnt/gcs/dgemma,CANVAS=${CANVAS_LEN},MAX_SEQS=32,MAX_MODEL_LEN=${MAX_MODEL_LEN},GPU_UTIL=0.40,KV_CACHE_GB=2,ATTN=TRITON_ATTN,ENFORCE_EAGER=1,DISABLE_MM=1,TORCH_COMPILE_DISABLE=1,VLLM_WORKER_MULTIPROC_METHOD=fork,CUDA_MODULE_LOADING=LAZY,COPY_TO_SHM=${COPY_SHM}"
if [[ -n "${HF_TOKEN:-}" ]]; then
  ENV_VARS="${ENV_VARS},HF_TOKEN=${HF_TOKEN}"
fi

DEPLOY_FLAGS=(
  "--project" "$PROJECT_ID"
  "--region" "$REGION"
  "--image" "$IMAGE_TAG"
  "--execution-environment" "gen2"
  "--no-allow-unauthenticated"
  "--cpu" "$CPU"
  "--memory" "$MEMORY"
  "--gpu" "1"
  "--gpu-type" "$GPU_TYPE"
  "--no-gpu-zonal-redundancy"
  "--no-cpu-throttling"
  "--min-instances" "0"
  "--max-instances" "1"
  "--concurrency" "32"
  "--port" "8080"
  "--add-volume=name=weights,type=cloud-storage,bucket=${BUCKET},readonly=false,mount-options=enable-buffered-read=true"
  "--add-volume-mount=volume=weights,mount-path=/mnt/gcs"
  "--startup-probe=httpGet.path=/health,httpGet.port=8080,initialDelaySeconds=5,periodSeconds=2,timeoutSeconds=2,failureThreshold=120"
  "--set-env-vars=${ENV_VARS}"
)

echo "==> Deploying to Cloud Run..."
gcloud beta run deploy "$SERVICE_NAME" "${DEPLOY_FLAGS[@]}"

SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" --project "$PROJECT_ID" --region "$REGION" --format="value(status.url)")

echo ""
echo "================================================================================"
echo "  Deployment Successful! Zero-to-N Autoscaling Active"
echo "================================================================================"
echo "Service URL: $SERVICE_URL"
echo ""
echo "Query using dgem with automatic IAM identity authentication:"
echo "  ./bin/dgem decide -u \"${SERVICE_URL}/v1\" --gcp-auth \\"
echo "    -t templates/support_triage.json.tmpl \\"
echo "    -v 'ticket=Emergency: cluster outage' --stats"
echo ""
echo "Run the decision benchmark suite against Cloud Run:"
echo "  ./bin/dgem bench -u \"${SERVICE_URL}/v1\" --gcp-auth \\"
echo "    -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_cloudrun.json"
echo ""
echo "View web UI / snake demo (with gcloud proxy):"
echo "  gcloud run services proxy $SERVICE_NAME --project $PROJECT_ID --region $REGION --port 8080"
echo "  open http://localhost:8080/snake"
echo "================================================================================"
