#!/usr/bin/env bash
set -euo pipefail

PROJECT="${GCP_PROJECT:-genai-blackbelt-fishfooding}"
REGION="${GCP_REGION:-us-central1}"
GATEWAY_SERVICE="${GATEWAY_SERVICE:-dgemma-gateway}"
UPSTREAM_SERVICE="${UPSTREAM_SERVICE:-dgemma}"
IMAGE="us-central1-docker.pkg.dev/${PROJECT}/dgem/dgemma-gateway:latest"
ALLOW_GROUP="${ALLOW_GROUP:-aaie-decision-model@google.com}"
GATEWAY_SA_NAME="dgemma-gateway-sa"
GATEWAY_SA="${GATEWAY_SA_NAME}@${PROJECT}.iam.gserviceaccount.com"

echo "================================================================"
echo " Deploying ${GATEWAY_SERVICE} (Go HTTP API & Web Studio Gateway)"
echo " Project: ${PROJECT} | Region: ${REGION}"
echo " Gateway SA: ${GATEWAY_SA} | Group: ${ALLOW_GROUP}"
echo "================================================================"

if ! gcloud iam service-accounts describe "${GATEWAY_SA}" --project="${PROJECT}" >/dev/null 2>&1; then
  echo "-> Creating Service Account ${GATEWAY_SA}..."
  gcloud iam service-accounts create "${GATEWAY_SA_NAME}" \
    --project="${PROJECT}" \
    --display-name="DiffusionGemma HTTP Gateway SA (Invoker on dgemma GPU)"
fi

UPSTREAM_URL="$(gcloud run services describe "${UPSTREAM_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --format='value(status.url)')"

if [ -z "${UPSTREAM_URL}" ]; then
  echo "ERROR: Upstream GPU service '${UPSTREAM_SERVICE}' not found in ${PROJECT}/${REGION}." >&2
  exit 1
fi
echo "-> Discovered Upstream GPU Service: ${UPSTREAM_URL}"

# Stage minimal build context (Go source + studio Lit app + templates) for fast ~20s Cloud Build
TMP_CTX="$(mktemp -d)"
trap 'rm -rf "${TMP_CTX}"' EXIT
cp go.mod go.sum main.go "${TMP_CTX}/"
cp -R cmd pkg templates "${TMP_CTX}/"
mkdir -p "${TMP_CTX}/studio"
cp studio/package*.json studio/tsconfig.json studio/vite.config.ts studio/index.html studio/embed.go "${TMP_CTX}/studio/"
cp -R studio/src "${TMP_CTX}/studio/"
if [ -d "studio/dist" ]; then
  cp -R studio/dist "${TMP_CTX}/studio/"
fi
cp deploy/gateway/Dockerfile "${TMP_CTX}/Dockerfile"

echo "-> Building ${IMAGE} via Cloud Build..."
gcloud builds submit "${TMP_CTX}" \
  --project="${PROJECT}" \
  --tag="${IMAGE}" \
  --suppress-logs \
  --quiet

GPU_IDLE_TTL="${GPU_IDLE_TTL:-3h}"

echo "-> Deploying Cloud Run service ${GATEWAY_SERVICE} (GPU_IDLE_TTL=${GPU_IDLE_TTL})..."
gcloud run deploy "${GATEWAY_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --image="${IMAGE}" \
  --service-account="${GATEWAY_SA}" \
  --cpu=1 \
  --memory=512Mi \
  --min-instances=1 \
  --no-cpu-throttling \
  --max-instances=10 \
  --concurrency=80 \
  --timeout=600 \
  --no-allow-unauthenticated \
  --set-env-vars="UPSTREAM_DGEMMA_URL=${UPSTREAM_URL}/v1,DGEM_GCP_AUTH=1,DGEM_GPU_IDLE_TTL=${GPU_IDLE_TTL}" \
  --quiet

# Run Zero-Trust IAM & IAP bindings for dgemma-gpu-sa, dgemma-gateway-sa, and group:${ALLOW_GROUP}
GCP_PROJECT="${PROJECT}" GCP_REGION="${REGION}" ALLOW_GROUP="${ALLOW_GROUP}" ./scripts/setup_cloudrun_iam.sh

GATEWAY_URL="$(gcloud run services describe "${GATEWAY_SERVICE}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --format='value(status.url)')"

echo "================================================================"
echo " ✅ ${GATEWAY_SERVICE} deployed: ${GATEWAY_URL}"
echo " -> Upstream GPU Backend: ${UPSTREAM_URL}/v1"
echo " -> Authorized Group:     group:${ALLOW_GROUP}"
echo "================================================================"
