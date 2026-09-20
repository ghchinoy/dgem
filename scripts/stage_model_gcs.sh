#!/usr/bin/env bash
set -euo pipefail

# Stage DiffusionGemma Model Weights in Google Cloud Storage for Cloud Run GCS FUSE Mounting

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${GCP_REGION:-us-central1}"
BUCKET="${1:-${GCS_BUCKET:-}}"
MODEL_ID="${2:-${MODEL_ID:-nvidia/diffusiongemma-26B-A4B-it-NVFP4}}"

if [[ -z "$PROJECT_ID" ]]; then
  echo "Error: No GCP project detected. Set GCP_PROJECT=<project-id>."
  exit 1
fi

if [[ -z "$BUCKET" ]]; then
  BUCKET="dgem-weights-${PROJECT_ID}"
fi

echo "================================================================================"
echo "  Staging DiffusionGemma Model Weights in GCS for Cloud Run GCS FUSE"
echo "================================================================================"
echo "Project:  $PROJECT_ID"
echo "Region:   $REGION"
echo "Bucket:   gs://$BUCKET"
echo "Model:    $MODEL_ID"
echo "================================================================================"

# 1. Create regional bucket if it does not exist
if ! gcloud storage buckets describe "gs://${BUCKET}" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Creating regional GCS bucket gs://${BUCKET} in ${REGION}..."
  gcloud storage buckets create "gs://${BUCKET}" \
    --project="$PROJECT_ID" \
    --location="$REGION" \
    --uniform-bucket-level-access
fi

# 2. Check if weights already staged
if gcloud storage ls "gs://${BUCKET}/dgemma/config.json" >/dev/null 2>&1; then
  echo "==> Weights already present at gs://${BUCKET}/dgemma/:"
  gcloud storage ls -lh "gs://${BUCKET}/dgemma/"
  exit 0
fi

# 3. Download weights from Hugging Face Hub using python / huggingface_hub
echo "==> Downloading model weights from Hugging Face Hub ($MODEL_ID)..."
TMP_DIR=$(mktemp -d /tmp/dgemma-stage-XXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

python3 -c "
import os
from huggingface_hub import snapshot_download
token = os.environ.get('HF_TOKEN', None)
print(f'Downloading {sys.argv[1]} to {sys.argv[2]}...')
snapshot_download(repo_id=sys.argv[1], local_dir=sys.argv[2], token=token)
" "$MODEL_ID" "$TMP_DIR"

echo "==> Uploading weights to gs://${BUCKET}/dgemma/ via Google internal network..."
gcloud storage cp -r "$TMP_DIR/*" "gs://${BUCKET}/dgemma/"

echo ""
echo "================================================================================"
echo "  Model Weights Successfully Staged in GCS!"
echo "================================================================================"
echo "GCS Path: gs://${BUCKET}/dgemma/"
gcloud storage ls -lh "gs://${BUCKET}/dgemma/"
echo ""
echo "Now deploy to Cloud Run:"
echo "  GCS_BUCKET=\"$BUCKET\" make cloudrun-deploy"
echo "================================================================================"
