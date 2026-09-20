#!/usr/bin/env bash
set -euo pipefail

# Build and Push DiffusionGemma Cloud Run Image to Google Artifact Registry
# Uses Cloud Build for 100% cloud-native, reproducible container building.

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${GCP_REGION:-us-central1}"
REPOSITORY="${AR_REPOSITORY:-dgem}"
IMAGE_NAME="${IMAGE_NAME:-djev-run}"
TAG="${IMAGE_TAG:-latest}"

if [[ -z "$PROJECT_ID" ]]; then
  echo "Error: No GCP project detected. Set GCP_PROJECT=<project-id>."
  exit 1
fi

AR_TARGET="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY}/${IMAGE_NAME}:${TAG}"

echo "================================================================================"
echo "  Building DiffusionGemma Cloud Run Image via Google Cloud Build"
echo "================================================================================"
echo "Project:    $PROJECT_ID"
echo "Region:     $REGION"
echo "Repository: $REPOSITORY"
echo "Target Tag: $AR_TARGET"
echo "Source:     deploy/cloudrun/"
echo "================================================================================"

# 1. Ensure Artifact Registry repository exists
if ! gcloud artifacts repositories describe "$REPOSITORY" --location="$REGION" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Creating Artifact Registry repository $REPOSITORY in $REGION..."
  gcloud artifacts repositories create "$REPOSITORY" \
    --repository-format=docker \
    --location="$REGION" \
    --project="$PROJECT_ID" \
    --description="DiffusionGemma Cloud Run container images"
fi

# 2. Submit build to Cloud Build
echo "==> Submitting build to Google Cloud Build (timeout: 20m)..."
gcloud builds submit deploy/cloudrun \
  --project="$PROJECT_ID" \
  --tag="$AR_TARGET" \
  --timeout=1200s

echo ""
echo "================================================================================"
echo "  Container Image Ready!"
echo "================================================================================"
echo "Image URI: $AR_TARGET"
echo ""
echo "Deploy to Cloud Run:"
echo "  CLOUDRUN_IMAGE=\"$AR_TARGET\" make cloudrun-deploy"
echo "================================================================================"
