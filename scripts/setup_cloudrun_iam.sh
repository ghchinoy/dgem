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
  CURRENT_GPU_SA=$(gcloud run services describe "${GPU_SERVICE}" --project="${PROJECT}" --region="${REGION}" --format="value(spec.template.spec.serviceAccountName)" 2>/dev/null || true)
  if [[ "$CURRENT_GPU_SA" != "$GPU_SA" ]]; then
    echo "-> Updating ${GPU_SERVICE} to run as ${GPU_SA}..."
    gcloud run services update "${GPU_SERVICE}" \
      --project="${PROJECT}" \
      --region="${REGION}" \
      --service-account="${GPU_SA}" \
      --quiet
  else
    echo "-> ${GPU_SERVICE} already runs as ${GPU_SA} (skipping revision update to preserve warm GPU)."
  fi

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
echo "-> Granting roles/cloudtrace.agent on project ${PROJECT} to ${GATEWAY_SA} for OpenTelemetry export..."
gcloud projects add-iam-policy-binding "${PROJECT}" \
  --member="serviceAccount:${GATEWAY_SA}" \
  --role="roles/cloudtrace.agent" \
  --condition=None \
  --quiet >/dev/null || true

if gcloud run services describe "${GATEWAY_SERVICE}" --project="${PROJECT}" --region="${REGION}" >/dev/null 2>&1; then
  CURRENT_GW_SA=$(gcloud run services describe "${GATEWAY_SERVICE}" --project="${PROJECT}" --region="${REGION}" --format="value(spec.template.spec.serviceAccountName)" 2>/dev/null || true)
  if [[ "$CURRENT_GW_SA" != "$GATEWAY_SA" ]]; then
    echo "-> Updating ${GATEWAY_SERVICE} to run as ${GATEWAY_SA}..."
    gcloud run services update "${GATEWAY_SERVICE}" \
      --project="${PROJECT}" \
      --region="${REGION}" \
      --service-account="${GATEWAY_SA}" \
      --quiet
  else
    echo "-> ${GATEWAY_SERVICE} already runs as ${GATEWAY_SA} (skipping redundant revision update)."
  fi

  echo "-> Granting roles/run.invoker on ${GATEWAY_SERVICE} to group:${ALLOW_GROUP}..."
  gcloud run services add-iam-policy-binding "${GATEWAY_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --member="group:${ALLOW_GROUP}" \
    --role="roles/run.invoker" \
    --quiet >/dev/null
fi

# 5. Configure Cloud Run Direct IAP on dgemma-gateway (Browser Sign-In + Programmatic CLI Access)
echo "-> Granting roles/iap.httpsResourceAccessor on ${GATEWAY_SERVICE} to group:${ALLOW_GROUP}..."
gcloud projects add-iam-policy-binding "${PROJECT}" \
  --member="group:${ALLOW_GROUP}" \
  --role="roles/iap.httpsResourceAccessor" \
  --condition=None \
  --quiet >/dev/null

gcloud iap web add-iam-policy-binding \
  --project="${PROJECT}" \
  --resource-type=cloud-run \
  --region="${REGION}" \
  --service="${GATEWAY_SERVICE}" \
  --member="group:${ALLOW_GROUP}" \
  --role="roles/iap.httpsResourceAccessor" \
  --quiet >/dev/null

gcloud iap web add-iam-policy-binding \
  --project="${PROJECT}" \
  --resource-type=cloud-run \
  --region="${REGION}" \
  --service="${GATEWAY_SERVICE}" \
  --member="serviceAccount:${GATEWAY_SA}" \
  --role="roles/iap.httpsResourceAccessor" \
  --quiet >/dev/null

# Allow gcloud CLI (32555940559.apps.googleusercontent.com) and custom IAP clients to authenticate programmatically through IAP
IAP_SETTINGS_TMP="$(mktemp)"
cat <<EOF >"${IAP_SETTINGS_TMP}"
accessSettings:
  oauthSettings:
    programmaticClients:
      - 32555940559.apps.googleusercontent.com
EOF
if [ -n "${IAP_CLIENT_ID:-}" ]; then
  echo "      - ${IAP_CLIENT_ID}" >>"${IAP_SETTINGS_TMP}"
fi
echo "-> Configuring IAP programmaticClients (enabling --gcp-auth / gcloud auth print-identity-token through IAP)..."
gcloud iap settings set "${IAP_SETTINGS_TMP}" \
  --project="${PROJECT}" \
  --resource-type=cloud-run \
  --region="${REGION}" \
  --service="${GATEWAY_SERVICE}" \
  --quiet >/dev/null
rm -f "${IAP_SETTINGS_TMP}"

echo "================================================================================"
echo " ✅ Zero-Trust IAM & IAP Setup Complete!"
echo "    • ${GPU_SERVICE} runs as:     ${GPU_SA} (read-only on gs://${BUCKET}, IAP OFF)"
echo "    • ${GATEWAY_SERVICE} runs as: ${GATEWAY_SA} (invoker on ${GPU_SERVICE}, IAP ON)"
echo "    • Authorized Group:           group:${ALLOW_GROUP} (run.invoker + iap.httpsResourceAccessor + CLI programmaticClients)"
echo "================================================================================"
