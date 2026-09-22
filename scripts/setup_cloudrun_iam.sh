#!/usr/bin/env bash
set -euo pipefail

PROJECT="${GCP_PROJECT:-genai-blackbelt-fishfooding}"
REGION="${GCP_REGION:-us-central1}"
GPU_SERVICE="${UPSTREAM_SERVICE:-dgemma}"
GATEWAY_SERVICE="${GATEWAY_SERVICE:-dgemma-gateway}"
BUCKET="${GCS_BUCKET:-dgem-weights-${PROJECT}}"
ALLOW_GROUP="${ALLOW_GROUP:-aaie-decision-model@google.com}"

GPU_SA_NAME="dgemma-gpu-sa"
GPU_SA="${GPU_SA_NAME}@${PROJECT}.iam.gserviceaccount.com"

GATEWAY_SA_NAME="dgemma-gateway-sa"
GATEWAY_SA="${GATEWAY_SA_NAME}@${PROJECT}.iam.gserviceaccount.com"

echo "================================================================================"
echo "  Configuring Zero-Trust Service Accounts & IAM / IAP Bindings"
echo "  Project:       ${PROJECT} (${REGION})"
echo "  GPU SA:        ${GPU_SA}"
echo "  Gateway SA:    ${GATEWAY_SA}"
echo "  Allowed Group: group:${ALLOW_GROUP}"
echo "================================================================================"

# 1. Create dedicated Service Accounts (idempotent)
if ! gcloud iam service-accounts describe "${GPU_SA}" --project="${PROJECT}" >/dev/null 2>&1; then
  echo "-> Creating Service Account ${GPU_SA}..."
  gcloud iam service-accounts create "${GPU_SA_NAME}" \
    --project="${PROJECT}" \
    --display-name="DiffusionGemma GPU Engine SA (Read-Only GCS Weights)" \
    --description="Least-privilege identity for the dgemma Cloud Run GPU service; only reads gs://${BUCKET}"
else
  echo "-> Service Account ${GPU_SA} already exists."
fi

if ! gcloud iam service-accounts describe "${GATEWAY_SA}" --project="${PROJECT}" >/dev/null 2>&1; then
  echo "-> Creating Service Account ${GATEWAY_SA}..."
  gcloud iam service-accounts create "${GATEWAY_SA_NAME}" \
    --project="${PROJECT}" \
    --display-name="DiffusionGemma HTTP Gateway SA (Invoker on dgemma GPU)" \
    --description="Least-privilege identity for dgemma-gateway; only invokes the dgemma Cloud Run service"
else
  echo "-> Service Account ${GATEWAY_SA} already exists."
fi

# 2. Scope dgemma-gpu-sa EXCLUSIVELY to roles/storage.objectViewer on the weights bucket
echo "-> Granting roles/storage.objectViewer on gs://${BUCKET} to ${GPU_SA}..."
gcloud storage buckets add-iam-policy-binding "gs://${BUCKET}" \
  --member="serviceAccount:${GPU_SA}" \
  --role="roles/storage.objectViewer" \
  --quiet >/dev/null

# 3. Attach dgemma-gpu-sa to the dgemma GPU service (if deployed) and bind invokers
if gcloud run services describe "${GPU_SERVICE}" --project="${PROJECT}" --region="${REGION}" >/dev/null 2>&1; then
  echo "-> Updating ${GPU_SERVICE} to run as ${GPU_SA}..."
  gcloud run services update "${GPU_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --service-account="${GPU_SA}" \
    --quiet

  echo "-> Granting roles/run.invoker on ${GPU_SERVICE} to ${GATEWAY_SA} and group:${ALLOW_GROUP}..."
  gcloud run services add-iam-policy-binding "${GPU_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --member="serviceAccount:${GATEWAY_SA}" \
    --role="roles/run.invoker" \
    --quiet >/dev/null

  gcloud run services add-iam-policy-binding "${GPU_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --member="group:${ALLOW_GROUP}" \
    --role="roles/run.invoker" \
    --quiet >/dev/null
fi

# 4. Attach dgemma-gateway-sa to dgemma-gateway and bind group:aaie-decision-model@google.com
if gcloud run services describe "${GATEWAY_SERVICE}" --project="${PROJECT}" --region="${REGION}" >/dev/null 2>&1; then
  echo "-> Updating ${GATEWAY_SERVICE} to run as ${GATEWAY_SA}..."
  gcloud run services update "${GATEWAY_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --service-account="${GATEWAY_SA}" \
    --quiet

  echo "-> Granting roles/run.invoker on ${GATEWAY_SERVICE} to group:${ALLOW_GROUP}..."
  gcloud run services add-iam-policy-binding "${GATEWAY_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --member="group:${ALLOW_GROUP}" \
    --role="roles/run.invoker" \
    --quiet >/dev/null
fi

# 5. Grant roles/iap.httpsResourceAccessor to group:${ALLOW_GROUP} so Cloud Run IAP allows group members
echo "-> Granting roles/iap.httpsResourceAccessor to group:${ALLOW_GROUP}..."
gcloud projects add-iam-policy-binding "${PROJECT}" \
  --member="group:${ALLOW_GROUP}" \
  --role="roles/iap.httpsResourceAccessor" \
  --condition=None \
  --quiet >/dev/null

echo "================================================================================"
echo " ✅ Zero-Trust IAM & IAP Setup Complete!"
echo "    • ${GPU_SERVICE} runs as:     ${GPU_SA} (read-only on gs://${BUCKET}, IAP OFF)"
echo "    • ${GATEWAY_SERVICE} runs as: ${GATEWAY_SA} (invoker on ${GPU_SERVICE}, IAP ON)"
echo "    • Authorized Group:           group:${ALLOW_GROUP} (run.invoker + iap.httpsResourceAccessor)"
echo "================================================================================"
