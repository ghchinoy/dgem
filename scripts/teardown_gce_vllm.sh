#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${GCP_PROJECT:-$(gcloud config get-value project 2>/dev/null || true)}"
ZONE="${GCP_ZONE:-us-central1-a}"
INSTANCE_NAME="${GCE_INSTANCE_NAME:-diffgemma-bench-l4}"

echo "================================================================================"
echo "  Tearing down GCE GPU benchmark infrastructure..."
echo "================================================================================"

if gcloud compute instances describe "$INSTANCE_NAME" --zone="$ZONE" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Deleting GCE instance $INSTANCE_NAME..."
  gcloud compute instances delete "$INSTANCE_NAME" --zone="$ZONE" --project="$PROJECT_ID" --quiet
  echo "Instance deleted."
else
  echo "Instance $INSTANCE_NAME already deleted."
fi

if gcloud compute firewall-rules describe allow-diffgemma-8080 --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "==> Deleting firewall rule allow-diffgemma-8080..."
  gcloud compute firewall-rules delete allow-diffgemma-8080 --project="$PROJECT_ID" --quiet
  echo "Firewall rule deleted."
fi

echo "================================================================================"
echo "  Teardown Complete. Zero idle cost."
echo "================================================================================"
