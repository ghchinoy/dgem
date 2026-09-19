#!/usr/bin/env bash
set -euo pipefail

JOBS="${1:-8}"
MODEL_DIR="model/diffgemma-26b-a4b-it-q4"

if ! command -v diffgemma >/dev/null 2>&1; then
  echo "Error: diffgemma command not found on PATH."
  echo "Run ./scripts/setup_diffgemma.sh or 'make setup' first."
  exit 1
fi

echo "==> Downloading DiffusionGemma 4-bit pack (mmastrac/diffgemma-26b-a4b-it-q4) with $JOBS concurrent jobs..."
diffgemma download --jobs "$JOBS"

if [[ -f "$MODEL_DIR/model.dgq.bin" ]]; then
  echo "==> Model pack downloaded and verified: $MODEL_DIR"
  ls -lh "$MODEL_DIR"
else
  echo "Error: Model file $MODEL_DIR/model.dgq.bin missing after download."
  exit 1
fi
