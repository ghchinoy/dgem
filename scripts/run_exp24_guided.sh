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

# EXP-24: Gemini 3.x boxes and masks guided by a fast dgem pass, on RefCOCO + ScreenSpot (positives and
# Gemini-confirmed negatives from benchmarks/bbox_real_aspects.jsonl).
#
#   ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_PROJECT=<PROJECT> ./scripts/run_exp24_guided.sh
#   then masks from boxes (SAM) on a short-lived VM:
#   GCP_PROJECT=<PROJECT> BUCKET=<BUCKET> RUN_ID=<same run> SAM_BOXES=benchmarks/runs/<run>/sam_boxes.jsonl ./scripts/run_detectors_gce.sh
#   python3 scripts/analyze_guided.py benchmarks/runs/<run>/guided__*.json [--sam benchmarks/runs/<run>/sam_masks.jsonl]
#
# Needs fixtures/bbox_real/ (python3 scripts/fetch_bbox_real.py fetch).
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-exp24-guided}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:?ENDPOINT or DGEM_VERTEX_URL is required}}"
MODEL="${MODEL:-gemini-3.8-flash}"
WORKERS="${WORKERS:-12}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"
R="python3 scripts/bench_runs.py"
go build -o bin/dgem .
$R note --run "$RUN_ID" "EXP-24 guided cascade: dgem presence + grid cell, then $MODEL full/hint/crop/poly at default/medium/low thinking (MINIMAL is not supported by this model)."
DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite guided --config "${MODEL//[.-]/_}" -- \
  ./bin/dgem bench-guided -d benchmarks/bbox_real_aspects.jsonl --vertex-url "$ENDPOINT" --gcp-auth \
    --gemini-model "$MODEL" -w "$WORKERS" -o {out}
python3 scripts/analyze_guided.py benchmarks/runs/"$RUN_ID"/guided__"${MODEL//[.-]/_}".json \
  --export-sam-boxes benchmarks/runs/"$RUN_ID"/sam_boxes.jsonl \
  --detector-preds benchmarks/runs/20261003-prop18-detectors/detector_preds.jsonl
if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
