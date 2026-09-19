#!/usr/bin/env bash
set -euo pipefail

# Configuration
PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="${CLOUDRUN_SERVICE_NAME:-diffusiongemma-vllm}"
GPU_TYPE="${CLOUDRUN_GPU_TYPE:-nvidia-rtx-pro-6000}"
IMAGE_TAG="gcr.io/${PROJECT_ID}/vllm-diffgemma:pr57250"
MODEL_ID="nvidia/diffusiongemma-26B-A4B-it-NVFP4"

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
echo "  Deploying DiffusionGemma vLLM (PR #57250) to Google Cloud Run with GPU"
echo "================================================================================"
echo "Project:      $PROJECT_ID"
echo "Region:       $REGION"
echo "Service Name: $SERVICE_NAME"
echo "GPU Type:     $GPU_TYPE"
echo "Image:        $IMAGE_TAG"
echo "Model:        $MODEL_ID"
echo "================================================================================"

echo ""
echo "==> Step 1: Submitting Cloud Build for vLLM overlay container..."
gcloud builds submit \
  --project "$PROJECT_ID" \
  --tag "$IMAGE_TAG" \
  -f deploy/cloudrun/Dockerfile.vllm \
  deploy/cloudrun

echo ""
echo "==> Step 2: Deploying container to Cloud Run with GPU..."
CONTAINER_ARGS=(
  "serve"
  "$MODEL_ID"
  "--diffusion-config={\"canvas_length\":32}"
  "--max-logprobs=32"
  "--enable-prefix-caching"
  "--max-num-seqs=32"
  "--port=8080"
  "--host=0.0.0.0"
)

gcloud beta run deploy "$SERVICE_NAME" \
  --project "$PROJECT_ID" \
  --region "$REGION" \
  --image "$IMAGE_TAG" \
  --execution-environment gen2 \
  --no-allow-unauthenticated \
  --cpu 20 \
  --memory 80Gi \
  --gpu 1 \
  --gpu-type "$GPU_TYPE" \
  --no-gpu-zonal-redundancy \
  --no-cpu-throttling \
  --max-instances 2 \
  --concurrency 32 \
  --timeout 300 \
  --startup-probe tcpSocket.port=8080,initialDelaySeconds=240,failureThreshold=1,timeoutSeconds=240,periodSeconds=240 \
  --command "vllm" \
  --args="$(IFS=','; echo "${CONTAINER_ARGS[*]}")"

SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" --project "$PROJECT_ID" --region "$REGION" --format="value(status.url)")

echo ""
echo "================================================================================"
echo "  Deployment Successful!"
echo "================================================================================"
echo "Service URL: $SERVICE_URL"
echo ""
echo "To query using dgem with automatic IAM authentication:"
echo "  ./bin/dgem decide -u \"${SERVICE_URL}/v1\" --gcp-auth \\"
echo "    -t templates/support_triage.json.tmpl \\"
echo "    -v 'ticket=Emergency: cluster outage' --stats"
echo ""
echo "Or using gcloud run proxy:"
echo "  gcloud run services proxy $SERVICE_NAME --project $PROJECT_ID --region $REGION --port 8080"
echo "  ./bin/dgem decide -t templates/support_triage.json.tmpl -v 'ticket=Emergency' --stats"
echo "================================================================================"
