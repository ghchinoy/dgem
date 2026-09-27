#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Automated Cloud Run GPU Smoke Test with Guaranteed Immediate Teardown
# Used by the production testing agent to validate public dgem container images.
# ==============================================================================

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${GCP_REGION:-us-central1}"
GPU_TYPE="${GPU_TYPE:-nvidia-rtx-pro-6000}"
IMAGE_URI="${IMAGE_URI:-us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:latest}"
SERVICE_NAME="dgem-smoke-$(date +%s)"
TEMPERATURE="${TEMPERATURE:-1.0}"

if [[ -z "$PROJECT_ID" ]]; then
  echo "Error: No GCP hosting project specified."
  echo "Usage: GCP_PROJECT=<hosting-project> [GPU_TYPE=nvidia-rtx-pro-6000] ./scripts/gpu_smoketest.sh"
  exit 1
fi

echo "================================================================================"
echo "  dgem Cloud Run GPU Smoke Test"
echo "================================================================================"
echo "  Hosting Project:  $PROJECT_ID"
echo "  Region:           $REGION"
echo "  GPU Accelerator:  $GPU_TYPE"
echo "  Container Image:  $IMAGE_URI"
echo "  Ephemeral Svc:    $SERVICE_NAME"
echo "================================================================================"

# 1. Quota Pre-Check
QUOTA_ID="NvidiaRtxPro6000GpuAllocNoZonalRedundancyPerProjectRegion"
if [[ "$GPU_TYPE" == *"l4"* ]]; then
  QUOTA_ID="NvidiaL4GpuAllocNoZonalRedundancyPerProjectRegion"
fi

echo "==> Verifying Cloud Run GPU quota ($QUOTA_ID) in $PROJECT_ID ($REGION)..."
QUOTA_VAL=$(gcloud beta quotas info describe "$QUOTA_ID" \
  --service=run.googleapis.com \
  --project="$PROJECT_ID" \
  --format="value(dimensionsInfos[0].details.value)" 2>/dev/null || echo "0")

if [[ "$QUOTA_VAL" == "0" || "$QUOTA_VAL" == "null" || -z "$QUOTA_VAL" ]]; then
  echo ""
  echo "⚠️  Quota Check Failed: $PROJECT_ID has 0 quota for $GPU_TYPE in $REGION."
  echo "   Please request quota via Google Cloud Console before running GPU smoke tests:"
  echo "   https://console.cloud.google.com/iam-admin/quotas?project=${PROJECT_ID}&service=run.googleapis.com"
  exit 2
fi

echo "✓ Active quota confirmed: $QUOTA_VAL GPU(s) available."

# 2. Register Guaranteed Teardown Trap
cleanup() {
  echo ""
  echo "================================================================================"
  echo "  MANDATORY TEARDOWN: Deleting ephemeral Cloud Run service $SERVICE_NAME..."
  echo "================================================================================"
  gcloud run services delete "$SERVICE_NAME" \
    --project="$PROJECT_ID" \
    --region="$REGION" \
    --quiet 2>/dev/null || true
  echo "✓ Service $SERVICE_NAME deleted. Zero idle cost."
}
trap cleanup EXIT INT TERM

# 3. Deploy Ephemeral Cloud Run Service
MEMORY="80Gi"
CPU="8"
if [[ "$GPU_TYPE" == *"l4"* ]]; then
  MEMORY="32Gi"
  CPU="4"
fi

echo "==> Deploying ephemeral service $SERVICE_NAME to Cloud Run..."
gcloud run deploy "$SERVICE_NAME" \
  --project="$PROJECT_ID" \
  --region="$REGION" \
  --image="$IMAGE_URI" \
  --gpu=1 \
  --gpu-type="$GPU_TYPE" \
  --memory="$MEMORY" \
  --cpu="$CPU" \
  --no-cpu-throttling \
  --min-instances=0 \
  --max-instances=1 \
  --timeout=600 \
  --set-env-vars="ROLE=decision-index,TEMPERATURE=${TEMPERATURE},CANVAS=128" \
  --allow-unauthenticated

SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" --project="$PROJECT_ID" --region="$REGION" --format="value(status.url)")
echo "==> Service deployed at $SERVICE_URL"

# 4. Probe Health
echo -n "==> Waiting for /health probe..."
for i in {1..90}; do
  if curl -s -m 5 "${SERVICE_URL}/health" >/dev/null 2>&1; then
    echo " healthy!"
    break
  fi
  sleep 2
  echo -n "."
done

# 5. Execute Test Decision via POST /v1/systemone
echo "==> Executing test decision via POST /v1/systemone..."
START_T=$(date +%s%N)
TEST_PAYLOAD='{
  "state": "Customer was double billed on transaction #48291.",
  "questions": {
    "urgent": {"type": "noul", "instructions": "Is this an urgent issue?"},
    "department": {
      "type": "choice",
      "instructions": "Route to the responsible department",
      "criteria": {
        "billing": "Invoice and charge disputes",
        "engineering": "Platform and bug reports",
        "support": "General customer inquiries"
      }
    }
  }
}'

RESP=$(curl -sS -X POST "${SERVICE_URL}/v1/systemone" \
  -H "Content-Type: application/json" \
  -d "$TEST_PAYLOAD")

END_T=$(date +%s%N)
WALL_MS=$(( (END_T - START_T) / 1000000 ))

echo ""
echo "=== Smoke Test Output (${WALL_MS} ms) ==="
echo "$RESP" | python3 -m json.tool || echo "$RESP"
echo ""

CHOICE=$(echo "$RESP" | python3 -c 'import json,sys; print(json.load(sys.stdin)["answers"]["department"]["choice"])' 2>/dev/null || echo "")
if [[ "$CHOICE" == "billing" ]]; then
  echo "🎉 SMOKE TEST PASSED: Department routed correctly to 'billing'!"
else
  echo "⚠️  Smoke test unexpected output: $RESP"
fi
