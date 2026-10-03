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

# PROP-18 phase 3: dgem and a Gemini 3.x reference on
#   SUITE=sweep  the generated bounding-box sweep (aspect ratio x grid, occlusion dose-response,
#                degradation, absent targets): benchmarks/bbox_sweep.jsonl
#   SUITE=real   RefCOCO + ScreenSpot sample: benchmarks/bbox_real.jsonl
#
#   SUITE=sweep ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_PROJECT=<PROJECT> ./scripts/run_prop18_phase3.sh
#
# Steps (STEPS="generate dgem gemini"):
#   generate  sweep: regenerate fixtures/bbox_sweep/ (seeded); real: download and hash-check fixtures/bbox_real/
#             (both need scripts/requirements-vision.txt; PY= points at that environment's python)
#   dgem      bench-bbox on Vertex G4, REPEAT runs
#   gemini    bench-bbox --engine gemini, one run per model in MODELS
set -euo pipefail
cd "$(dirname "$0")/.."

SUITE_KIND="${SUITE:-sweep}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-prop18-${SUITE_KIND}}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:-}}"
REPEAT="${REPEAT:-2}"
MODELS="${MODELS:-gemini-3.8-flash}"
PY="${PY:-python3}"
STEPS="${STEPS:-generate dgem gemini}"
SUITE_FILE="benchmarks/bbox_${SUITE_KIND}.jsonl"
NAME="bbox_${SUITE_KIND}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"

R="python3 scripts/bench_runs.py"
D="./bin/dgem"
go build -o bin/dgem .
has() { [[ " $STEPS " == *" $1 "* ]]; }

if has generate; then
  if [[ "$SUITE_KIND" == sweep ]]; then $PY scripts/generate_bbox_sweep.py; else $PY scripts/fetch_bbox_real.py fetch; fi
fi
$R note --run "$RUN_ID" "PROP-18 phase 3: $SUITE_FILE ($(wc -l < $SUITE_FILE) items, seed 18): dgem x$REPEAT on Vertex G4, Gemini reference ($MODELS)."

if has dgem; then
  [[ -n "$ENDPOINT" ]] || { echo "Error: ENDPOINT or DGEM_VERTEX_URL is required" >&2; exit 1; }
  DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite "$NAME" --config "vertex_g4_x${REPEAT}" -- \
    $D bench-bbox -d "$SUITE_FILE" --vertex-url "$ENDPOINT" --gcp-auth --repeat "$REPEAT" -w 8 -o {out}
fi
if has gemini; then
  for M in $MODELS; do
    $R exec --run "$RUN_ID" --suite "$NAME" --config "reference_${M//[.-]/_}" -- \
      $D bench-bbox -d "$SUITE_FILE" --engine gemini --gemini-model "$M" -w 8 -o {out}
  done
fi

if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
