#!/usr/bin/env bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# PROP-18 phase 2: Gemini 3.x as reference localizer and as visual judge on the EXP-09 suite.
# Receipts go to benchmarks/runs/$RUN_ID/ and are recorded in its manifest.json.
#
#   GCP_PROJECT=<PROJECT> DGEM_RECEIPT=benchmarks/runs/20261003-exp09-rebaseline/bbox__vertex_g4_all_x3.json \
#     ./scripts/run_prop18_phase2.sh
#
# Steps (STEPS="reference judge"):
#   reference  bench-bbox --engine gemini on the image variants, REPEAT runs, for each model in MODELS
#   judge      bench-bbox-judge: calibration boxes (ground truth with shifted edges) + the dgem receipt's boxes
#
# Requires Application Default Credentials with Vertex AI Gemini access.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-prop18-gemini}"
MODELS="${MODELS:-gemini-3.8-flash gemini-3.7-flash}"
REPEAT="${REPEAT:-3}"
WORKERS="${WORKERS:-8}"
VARIANTS="${VARIANTS:-original,blank,hflip,vflip,pad_right,pad_left,pad_bottom}"
DGEM_RECEIPT="${DGEM_RECEIPT:-benchmarks/runs/20261003-exp09-rebaseline/bbox__vertex_g4_all_x3.json}"
STEPS="${STEPS:-reference judge}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"

R="python3 scripts/bench_runs.py"
D="./bin/dgem"
go build -o bin/dgem .
has() { [[ " $STEPS " == *" $1 "* ]]; }

$R note --run "$RUN_ID" "PROP-18 phase 2: Gemini reference boxes ($VARIANTS x$REPEAT) and Gemini judge validation (calibration + $DGEM_RECEIPT). Models: $MODELS."

for M in $MODELS; do
  tag="${M//[.-]/_}"
  if has reference; then
    $R exec --run "$RUN_ID" --suite bbox --config "reference_${tag}_x${REPEAT}" -- \
      $D bench-bbox --engine gemini --gemini-model "$M" --variants "$VARIANTS" --repeat "$REPEAT" -w "$WORKERS" -o {out}
  fi
  if has judge; then
    $R exec --run "$RUN_ID" --suite bbox_judge --config "judge_${tag}" -- \
      $D bench-bbox-judge --judge-model "$M" --calibrate --receipt "$DGEM_RECEIPT" -w "$WORKERS" -o {out}
  fi
done

if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
