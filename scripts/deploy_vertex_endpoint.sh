#!/usr/bin/env bash
# ==============================================================================
# Deploy DiffusionGemma (dgemma) to a Vertex AI Dedicated Endpoint using
# Arbitrary Custom Routes (invokeRoutePrefix: "/*")
#
# Reference:
#   https://docs.cloud.google.com/gemini-enterprise-agent-platform/machine-learning/predictions/use-arbitrary-custom-routes
#
# Exposed Custom Routes on the Dedicated Endpoint:
#   POST .../invoke/v1/chat/completions      -> structured_server.py (/v1/chat/completions)
#   POST .../invoke/v1/raw/chat/completions  -> raw vLLM pass-through (/v1/raw/chat/completions)
#   POST .../invoke/v1/systemone             -> native SystemOne/JevBench route (/v1/systemone)
#   GET  .../invoke/health                   -> container & vLLM readiness telemetry (/health)
# ==============================================================================
set -euo pipefail

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || echo "genai-blackbelt-fishfooding")}"
PROJECT_NUMBER="${GCP_PROJECT_NUMBER:-$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)' 2>/dev/null || echo "882920967572")}"
REGION="${GCP_REGION:-us-central1}"
IMAGE_URI="${IMAGE_URI:-${REGION}-docker.pkg.dev/${PROJECT_ID}/dgem/dgemma:latest}"
GCS_BUCKET="${GCS_BUCKET:-dgem-weights-${PROJECT_ID}}"
ARTIFACT_URI="${ARTIFACT_URI:-gs://${GCS_BUCKET}/dgemma}"

MODEL_DISPLAY_NAME="${VERTEX_MODEL_NAME:-dgemma-invoke}"
ENDPOINT_DISPLAY_NAME="${VERTEX_ENDPOINT_NAME:-dgemma-dedicated}"

# Hardware profile:
#   1x L4 (24GB VRAM, 64GB RAM for 17.53 GiB tmpfs + PyTorch load): MACHINE_TYPE=g2-standard-16, ACCELERATOR_TYPE=NVIDIA_L4, ACCELERATOR_COUNT=1
#   2x L4 (48GB VRAM, multimodal DISABLE_MM=0): MACHINE_TYPE=g2-standard-24, ACCELERATOR_TYPE=NVIDIA_L4, ACCELERATOR_COUNT=2
MACHINE_TYPE="${VERTEX_MACHINE_TYPE:-g2-standard-16}"
ACCELERATOR_TYPE="${VERTEX_ACCELERATOR_TYPE:-NVIDIA_L4}"
ACCELERATOR_COUNT="${VERTEX_ACCELERATOR_COUNT:-1}"
DISABLE_MM="${DISABLE_MM:-1}"
MIN_REPLICAS="${VERTEX_MIN_REPLICAS:-1}"
MAX_REPLICAS="${VERTEX_MAX_REPLICAS:-1}"
SERVICE_ACCOUNT="${VERTEX_SERVICE_ACCOUNT:-dgemma-gpu-sa@${PROJECT_ID}.iam.gserviceaccount.com}"

API_BASE="https://${REGION}-aiplatform.googleapis.com/v1beta1"
TOKEN="$(gcloud auth application-default print-access-token 2>/dev/null || gcloud auth print-access-token)"

echo "==> [1/4] Uploading Invoke-Enabled Model (${MODEL_DISPLAY_NAME}) with invokeRoutePrefix=\"/*\"..."
UPLOAD_PAYLOAD=$(cat <<EOF
{
  "model": {
    "displayName": "${MODEL_DISPLAY_NAME}",
    "containerSpec": {
      "imageUri": "${IMAGE_URI}",
      "invokeRoutePrefix": "/*",
      "healthRoute": "/health",
      "ports": [
        { "containerPort": 8080 }
      ],
      "env": [
        { "name": "DGEM_WEIGHTS_URI", "value": "${ARTIFACT_URI}" },
        { "name": "GCS_BUCKET", "value": "${GCS_BUCKET}" },
        { "name": "CANVAS", "value": "128" },
        { "name": "ENFORCE_EAGER", "value": "1" },
        { "name": "DISABLE_MM", "value": "${DISABLE_MM}" },
        { "name": "GPU_UTIL", "value": "0.85" }
      ]
    }
  }
}
EOF
)

UPLOAD_OP=$(curl -sS -X POST "${API_BASE}/projects/${PROJECT_ID}/locations/${REGION}/models:upload" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${UPLOAD_PAYLOAD}")

OP_NAME=$(echo "$UPLOAD_OP" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("name",""))')
if [ -z "$OP_NAME" ]; then
  echo "ERROR uploading model: $UPLOAD_OP" >&2
  exit 1
fi
echo "    Waiting for model upload operation: ${OP_NAME}..."
while true; do
  TOKEN="$(gcloud auth print-access-token)"
  OP_RES=$(curl -sS -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/${OP_NAME}")
  DONE=$(echo "$OP_RES" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("done", False))')
  if [ "$DONE" = "True" ]; then
    MODEL_RESOURCE=$(echo "$OP_RES" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("response",{}).get("model",""))')
    break
  fi
  sleep 5
done
echo "    Uploaded Model Resource: ${MODEL_RESOURCE}"

echo "==> [2/4] Finding or Creating Dedicated Endpoint (${ENDPOINT_DISPLAY_NAME}, dedicatedEndpointEnabled=true)..."
EXISTING_EP=$(curl -sS -H "Authorization: Bearer ${TOKEN}" \
  "${API_BASE}/projects/${PROJECT_ID}/locations/${REGION}/endpoints" | \
  python3 -c "import json,sys; eps=json.load(sys.stdin).get('endpoints',[]); print(next((e['name'] for e in eps if e.get('displayName')=='${ENDPOINT_DISPLAY_NAME}' and e.get('dedicatedEndpointEnabled')), ''))")

if [ -n "$EXISTING_EP" ]; then
  ENDPOINT_RESOURCE="$EXISTING_EP"
  echo "    Reusing existing Dedicated Endpoint: ${ENDPOINT_RESOURCE}"
else
  CREATE_EP_OP=$(curl -sS -X POST "${API_BASE}/projects/${PROJECT_ID}/locations/${REGION}/endpoints" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"displayName\": \"${ENDPOINT_DISPLAY_NAME}\", \"dedicatedEndpointEnabled\": true}")
  EP_OP_NAME=$(echo "$CREATE_EP_OP" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("name",""))')
  while true; do
    TOKEN="$(gcloud auth print-access-token)"
    OP_RES=$(curl -sS -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/${EP_OP_NAME}")
    DONE=$(echo "$OP_RES" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("done", False))')
    if [ "$DONE" = "True" ]; then
      ENDPOINT_RESOURCE=$(echo "$OP_RES" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("response",{}).get("name",""))')
      break
    fi
    sleep 4
  done
  echo "    Created Dedicated Endpoint: ${ENDPOINT_RESOURCE}"
fi

ENDPOINT_ID="${ENDPOINT_RESOURCE##*/}"

echo "==> [3/4] Deploying ${MODEL_RESOURCE} to ${ENDPOINT_RESOURCE} (${MACHINE_TYPE}, ${ACCELERATOR_COUNT}x ${ACCELERATOR_TYPE})..."
DEPLOY_PAYLOAD=$(cat <<EOF
{
  "deployedModel": {
    "model": "${MODEL_RESOURCE}",
    "displayName": "${MODEL_DISPLAY_NAME}-deployment",
    "serviceAccount": "${SERVICE_ACCOUNT}",
    "dedicatedResources": {
      "machineSpec": {
        "machineType": "${MACHINE_TYPE}",
        "acceleratorType": "${ACCELERATOR_TYPE}",
        "acceleratorCount": ${ACCELERATOR_COUNT}
      },
      "minReplicaCount": ${MIN_REPLICAS},
      "maxReplicaCount": ${MAX_REPLICAS}
    }
  },
  "trafficSplit": {
    "0": 100
  }
}
EOF
)

DEPLOY_OP=$(curl -sS -X POST "${API_BASE}/${ENDPOINT_RESOURCE}:deployModel" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${DEPLOY_PAYLOAD}")

DEPLOY_OP_NAME=$(echo "$DEPLOY_OP" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("name",""))')
if [ -z "$DEPLOY_OP_NAME" ]; then
  echo "ERROR deploying model: $DEPLOY_OP" >&2
  exit 1
fi
echo "    Deploy operation started: ${DEPLOY_OP_NAME}"
echo "    (Vertex AI Dedicated Endpoint provisioning + 17.53 GiB safetensors load typically takes 12-18 minutes.)"

while true; do
  TOKEN="$(gcloud auth print-access-token)"
  OP_RES=$(curl -sS -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/${DEPLOY_OP_NAME}")
  DONE=$(echo "$OP_RES" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("done", False))')
  if [ "$DONE" = "True" ]; then
    break
  fi
  echo "    Still deploying (${ENDPOINT_ID})... sleeping 20s"
  sleep 20
done

echo "==> [4/4] Resolving Dedicated Endpoint DNS & Custom /invoke/* Routes..."
EP_INFO=$(curl -sS -H "Authorization: Bearer ${TOKEN}" "${API_BASE}/${ENDPOINT_RESOURCE}")
DEDICATED_DNS=$(echo "$EP_INFO" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("dedicatedEndpointDns",""))')
if [ -z "$DEDICATED_DNS" ]; then
  DEDICATED_DNS="${ENDPOINT_ID}.${REGION}-${PROJECT_NUMBER}.prediction.vertexai.goog"
fi

INVOKE_BASE_URL="https://${DEDICATED_DNS}/v1/projects/${PROJECT_ID}/locations/${REGION}/endpoints/${ENDPOINT_ID}/invoke"

cat <<EOF

================================================================================
✅ Vertex AI Dedicated Endpoint Deployed with Arbitrary Custom Routes (/*)
================================================================================
  Endpoint ID:            ${ENDPOINT_ID}
  Dedicated Endpoint DNS: ${DEDICATED_DNS}

  Mapped Custom Routes:
  • Structured Decision:  ${INVOKE_BASE_URL}/v1/chat/completions
  • Raw vLLM Pass-Thru:   ${INVOKE_BASE_URL}/v1/raw/chat/completions
  • SystemOne / JevBench: ${INVOKE_BASE_URL}/v1/systemone
  • Live Health Probe:    ${INVOKE_BASE_URL}/health

  Use with dgem CLI:
    ./bin/dgem decide -u "${INVOKE_BASE_URL}/v1" --gcp-auth -t templates/support_triage.json.tmpl -v ticket="Double billed"

  Use with dgemma-gateway (Header or Web Studio Settings):
    curl -sS "${GATEWAY_URL:-https://<your-dgem-gateway>}"/api/decide/support_triage \\
      -H "X-DGem-Backend: vertex" \\
      -H "X-DGem-Vertex-Url: ${ENDPOINT_ID}" \\
      -H "Content-Type: application/json" \\
      -d '{"variables":{"ticket":"My invoice was charged twice"}}'

  IMPORTANT (Zero-Idle-Cost Mandate):
  Vertex AI Dedicated Endpoints enforce minReplicaCount >= 1 (billed continuously).
  Always run 'make vertex-teardown' immediately after testing!
================================================================================
EOF
