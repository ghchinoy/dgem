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

# PROP-18 phase 4 pilot: multi-aspect image decisions (templates/multimodal/vision_aspects.json.tmpl) on the
# generated sweep: presence, 3x3 grid cell, element count, spatial relation, image quality, occlusion.
#
#   ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_PROJECT=<PROJECT> ./scripts/run_prop18_phase4.sh
#
# Steps (STEPS="dgem blank gemini"): dgem xREPEAT; dgem on blank images (prompt prior); Gemini reference per MODELS.
# Needs fixtures/bbox_sweep/ (python3 scripts/generate_bbox_sweep.py).
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-prop18-vision}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:-}}"
REPEAT="${REPEAT:-2}"
MODELS="${MODELS:-gemini-3.8-flash}"
SUITE="${SUITE_FILE:-benchmarks/bbox_sweep.jsonl}"
STEPS="${STEPS:-dgem blank gemini}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"

R="python3 scripts/bench_runs.py"
D="./bin/dgem"
go build -o bin/dgem .
has() { [[ " $STEPS " == *" $1 "* ]]; }
$R note --run "$RUN_ID" "PROP-18 phase 4 pilot: bench-vision on $SUITE: dgem x$REPEAT, dgem blank-image prior, Gemini reference ($MODELS)."

if has dgem || has blank; then
  [[ -n "$ENDPOINT" ]] || { echo "Error: ENDPOINT or DGEM_VERTEX_URL is required" >&2; exit 1; }
fi
if has dgem; then
  DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite vision --config "vertex_g4_x${REPEAT}" -- \
    $D bench-vision -d "$SUITE" --vertex-url "$ENDPOINT" --gcp-auth --repeat "$REPEAT" -o {out}
fi
if has blank; then
  DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite vision --config "vertex_g4_blank" -- \
    $D bench-vision -d "$SUITE" --vertex-url "$ENDPOINT" --gcp-auth --variant blank -o {out}
fi
if has gemini; then
  for M in $MODELS; do
    $R exec --run "$RUN_ID" --suite vision --config "reference_${M//[.-]/_}" -- \
      $D bench-vision -d "$SUITE" --engine gemini --gemini-model "$M" -o {out}
  done
fi
if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
