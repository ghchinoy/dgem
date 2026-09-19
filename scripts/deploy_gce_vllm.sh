#!/usr/bin/env bash
set -euo pipefail

# Load .env if present
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
elif [[ -f "../.env" ]]; then
  set -a
  source "../.env"
  set +a
fi

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
ZONE="${GCP_ZONE:-us-central1-a}"
INSTANCE_NAME="${GCE_INSTANCE_NAME:-diffgemma-bench-l4}"
MACHINE_TYPE="g2-standard-8"
IMAGE_FAMILY="pytorch-2-9-cu129-ubuntu-2204-nvidia-580"
IMAGE_PROJECT="deeplearning-platform-release"

if [[ -z "$PROJECT_ID" ]]; then
  echo "Error: No GCP project detected. Set GCP_PROJECT=<project-id>."
  exit 1
fi

echo "================================================================================"
echo "  Deploying DiffusionGemma to Google Compute Engine (g2-standard-8 with L4 GPU)"
echo "================================================================================"
echo "Project:       $PROJECT_ID"
echo "Zone:          $ZONE"
echo "Instance:      $INSTANCE_NAME"
echo "Machine Type:  $MACHINE_TYPE (1x NVIDIA L4, 8 vCPUs, 32GB RAM)"
echo "Base Image:    $IMAGE_FAMILY ($IMAGE_PROJECT)"
echo "================================================================================"

# 1. Ensure firewall rule exists
if ! gcloud compute firewall-rules describe allow-diffgemma-8080 --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Creating firewall rule allow-diffgemma-8080..."
  gcloud compute firewall-rules create allow-diffgemma-8080 \
    --project="$PROJECT_ID" \
    --allow=tcp:8080 \
    --target-tags=diffgemma-server \
    --description="Allow inbound traffic on port 8080 for DiffusionGemma benchmark"
fi

# 2. Check if instance already exists
if gcloud compute instances describe "$INSTANCE_NAME" --zone="$ZONE" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Instance $INSTANCE_NAME already exists."
else
  echo "==> Step 1: Provisioning GCE GPU instance..."
  gcloud compute instances create "$INSTANCE_NAME" \
    --project="$PROJECT_ID" \
    --zone="$ZONE" \
    --machine-type="$MACHINE_TYPE" \
    --accelerator=type=nvidia-l4,count=1 \
    --maintenance-policy=TERMINATE \
    --image-family="$IMAGE_FAMILY" \
    --image-project="$IMAGE_PROJECT" \
    --boot-disk-size=100GB \
    --boot-disk-type=pd-ssd \
    --tags=diffgemma-server \
    --metadata-from-file=startup-script=scripts/startup_gce_vllm.sh \
    --metadata="hf-token=${HF_TOKEN:-},install-nvidia-driver=True"
fi

EXTERNAL_IP=$(gcloud compute instances describe "$INSTANCE_NAME" \
  --zone="$ZONE" \
  --project="$PROJECT_ID" \
  --format="value(networkInterfaces[0].accessConfigs[0].natIP)")

echo ""
echo "Instance External IP: $EXTERNAL_IP"
echo "Target Endpoint:      http://${EXTERNAL_IP}:8080/v1"
echo ""
echo "==> Step 2: Waiting for vLLM DiffusionGemma to initialize on the VM..."
echo "(Startup script is installing vLLM and caching model weights. This takes ~3-5 minutes)"

START_TIME=$(date +%s)
TIMEOUT_SECS=600

while true; do
  CURRENT_TIME=$(date +%s)
  ELAPSED=$((CURRENT_TIME - START_TIME))

  if [[ $ELAPSED -gt $TIMEOUT_SECS ]]; then
    echo ""
    echo "Error: Timed out waiting for http://${EXTERNAL_IP}:8080/health after ${TIMEOUT_SECS}s."
    echo "Check serial console logs:"
    echo "  gcloud compute instances get-serial-port-output $INSTANCE_NAME --zone=$ZONE --project=$PROJECT_ID"
    exit 1
  fi

  if curl -s -f "http://${EXTERNAL_IP}:8080/health" >/dev/null 2>&1 || curl -s -f "http://${EXTERNAL_IP}:8080/v1/models" >/dev/null 2>&1; then
    echo ""
    echo "================================================================================"
    echo "  vLLM DiffusionGemma is LIVE and Healthy!"
    echo "================================================================================"
    echo "Service URL: http://${EXTERNAL_IP}:8080/v1"
    echo ""
    echo "Run the benchmark evaluation:"
    echo "  ./bin/dgem bench -u \"http://${EXTERNAL_IP}:8080/v1\" \\"
    echo "    -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_gce_l4.json"
    echo ""
    echo "When finished, tear down the instance:"
    echo "  make gce-teardown"
    echo "================================================================================"
    break
  fi

  printf "."
  sleep 10
done
