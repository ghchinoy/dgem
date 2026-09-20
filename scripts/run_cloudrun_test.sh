#!/usr/bin/env bash
set -euo pipefail

# Complete End-to-End Cloud Run Test Runner
# 1. Waits for Artifact Registry image build
# 2. Deploys service dgemma to Cloud Run with 1x NVIDIA L4 GPU & GCS FUSE
# 3. Runs single-pass decision and 30-case benchmark
# 4. Automatically deletes Cloud Run service to eliminate idle cost

source .env 2>/dev/null || true

PROJECT_ID="${GCP_PROJECT:-ghchinoy-genai-sa}"
REGION="${GCP_REGION:-us-central1}"
BUILD_ID="${1:-e32ba260-44f7-4abc-b074-3cb057d7e779}"
IMAGE_TAG="${REGION}-docker.pkg.dev/${PROJECT_ID}/dgem/dgemma:latest"
SERVICE_NAME="dgemma"
BUCKET="dgem-weights-${PROJECT_ID}"

echo "================================================================================"
echo "  End-to-End Cloud Run Test Pipeline (Self-Contained Artifact Registry Image)"
echo "================================================================================"
echo "Project:      $PROJECT_ID"
echo "Region:       $REGION"
echo "Build ID:     $BUILD_ID"
echo "Image:        $IMAGE_TAG"
echo "Service:      $SERVICE_NAME"
echo "Weights GCS:  gs://${BUCKET}/dgemma/"
echo "================================================================================"

echo "==> Step 1: Waiting for Cloud Build $BUILD_ID to complete..."
for i in {1..180}; do
  STATUS=$(gcloud builds describe "$BUILD_ID" --project="$PROJECT_ID" --format="value(status)" 2>/dev/null || echo "UNKNOWN")
  if [[ "$STATUS" == "SUCCESS" ]]; then
    echo "Cloud Build SUCCESS: Image ready in Artifact Registry!"
    break
  elif [[ "$STATUS" == "FAILURE" || "$STATUS" == "CANCELLED" || "$STATUS" == "TIMEOUT" ]]; then
    echo "Error: Cloud Build ended with status $STATUS."
    exit 1
  fi
  sleep 10
done

echo ""
echo "==> Step 2: Deploying service $SERVICE_NAME to Cloud Run on 1x NVIDIA L4 GPU..."
GCP_PROJECT="$PROJECT_ID" \
GCP_REGION="$REGION" \
CLOUDRUN_SERVICE_NAME="$SERVICE_NAME" \
CLOUDRUN_IMAGE="$IMAGE_TAG" \
CLOUDRUN_GPU_TYPE="nvidia-l4" \
GCS_BUCKET="$BUCKET" \
./scripts/deploy_cloudrun_vllm.sh

SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" --project="$PROJECT_ID" --region="$REGION" --format="value(status.url)")
echo "Service URL: $SERVICE_URL"

echo ""
echo "==> Step 3: Executing Single-Pass Decision on Cloud Run..."
./bin/dgem decide \
  -u "${SERVICE_URL}/v1" \
  --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v "ticket=Outage: production database cluster unreachable in us-east1" \
  --stats

echo ""
echo "==> Step 4: Running 30-Case Multi-Domain Benchmark Suite on Cloud Run..."
./bin/dgem bench \
  -u "${SERVICE_URL}/v1" \
  --gcp-auth \
  -d benchmarks/eval_dataset.jsonl \
  -M slot \
  -o benchmarks/results_cloudrun.json

echo ""
echo "================================================================================"
echo "  Step 5: Tearing Down Cloud Run Service $SERVICE_NAME (Zero Idle Cost)"
echo "================================================================================"
gcloud run services delete "$SERVICE_NAME" --project="$PROJECT_ID" --region="$REGION" --quiet
echo "Cloud Run service $SERVICE_NAME deleted successfully. Zero ongoing cost."
echo "Benchmark receipt saved to benchmarks/results_cloudrun.json."
