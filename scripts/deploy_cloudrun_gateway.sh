#!/usr/bin/env bash
set -euo pipefail

PROJECT="${GCP_PROJECT:-genai-blackbelt-fishfooding}"
REGION="${GCP_REGION:-us-central1}"
GATEWAY_SERVICE="${GATEWAY_SERVICE:-dgemma-gateway}"
UPSTREAM_SERVICE="${UPSTREAM_SERVICE:-dgemma}"
IMAGE="us-central1-docker.pkg.dev/${PROJECT}/dgem/dgemma-gateway:latest"
ALLOW_GROUP="${ALLOW_GROUP:-}"

echo "================================================================"
echo " Deploying ${GATEWAY_SERVICE} (Go HTTP API & Web Studio Gateway)"
echo " Project: ${PROJECT} | Region: ${REGION}"
echo "================================================================"

UPSTREAM_URL="$(gcloud run services describe "${UPSTREAM_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --format='value(status.url)')"

if [ -z "${UPSTREAM_URL}" ]; then
  echo "ERROR: Upstream GPU service '${UPSTREAM_SERVICE}' not found in ${PROJECT}/${REGION}." >&2
  exit 1
fi
echo "-> Discovered Upstream GPU Service: ${UPSTREAM_URL}"

# Stage minimal build context (Go source + templates) for fast ~20s Cloud Build
TMP_CTX="$(mktemp -d)"
trap 'rm -rf "${TMP_CTX}"' EXIT
cp go.mod go.sum main.go "${TMP_CTX}/"
cp -R cmd pkg templates "${TMP_CTX}/"
cp deploy/gateway/Dockerfile "${TMP_CTX}/Dockerfile"

echo "-> Building ${IMAGE} via Cloud Build..."
gcloud builds submit "${TMP_CTX}" \
  --project="${PROJECT}" \
  --tag="${IMAGE}" \
  --quiet

echo "-> Deploying Cloud Run service ${GATEWAY_SERVICE}..."
gcloud run deploy "${GATEWAY_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --image="${IMAGE}" \
  --cpu=1 \
  --memory=512Mi \
  --min-instances=0 \
  --max-instances=10 \
  --concurrency=80 \
  --timeout=600 \
  --no-allow-unauthenticated \
  --set-env-vars="UPSTREAM_DGEMMA_URL=${UPSTREAM_URL}/v1,DGEM_GCP_AUTH=1" \
  --quiet

# Ensure the project's default compute service account can invoke the upstream dgemma service
PROJECT_NUM="$(gcloud projects describe "${PROJECT}" --format='value(projectNumber)')"
COMPUTE_SA="${PROJECT_NUM}-compute@developer.gserviceaccount.com"
gcloud run services add-iam-policy-binding "${UPSTREAM_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --member="serviceAccount:${COMPUTE_SA}" \
  --role="roles/run.invoker" \
  --quiet >/dev/null

# Optional: Grant a Google Group (e.g. ALLOW_GROUP=aaie-team@google.com) invoker & IAP access
if [ -n "${ALLOW_GROUP}" ]; then
  echo "-> Granting roles/run.invoker and roles/iap.httpsResourceAccessor to group:${ALLOW_GROUP}..."
  gcloud run services add-iam-policy-binding "${GATEWAY_SERVICE}" \
    --project="${PROJECT}" \
    --region="${REGION}" \
    --member="group:${ALLOW_GROUP}" \
    --role="roles/run.invoker" \
    --quiet
fi

GATEWAY_URL="$(gcloud run services describe "${GATEWAY_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --format='value(status.url)')"

echo "================================================================"
echo " ✅ ${GATEWAY_SERVICE} deployed: ${GATEWAY_URL}"
echo " -> Upstream GPU Backend: ${UPSTREAM_URL}/v1"
echo "================================================================"
