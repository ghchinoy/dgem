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

# EXP-09 re-baseline (phase 1 of the image-metrics validation plan).
# Writes every receipt into benchmarks/runs/$RUN_ID/ and records it in manifest.json.
#
#   ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> ./scripts/run_exp09_rebaseline.sh
#   STEPS="legacy simulate" ./scripts/run_exp09_rebaseline.sh   # offline steps only
#
# Steps:
#   legacy    re-analyze the 2026-09-21 Cloud Run receipt (baselines, intervals, error split)
#   simulate  offline harness check with simulated distributions, every variant
#   live      Vertex G4: original + every probe variant, REPEAT runs each (default 3)
#
# Requires Application Default Credentials with access to the Vertex endpoint for the live step.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-exp09-rebaseline}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:-}}"
REPEAT="${REPEAT:-3}"
WORKERS="${WORKERS:-4}"
VARIANTS="${VARIANTS:-all}"
STEPS="${STEPS:-legacy simulate live}"

R="python3 scripts/bench_runs.py"
D="./bin/dgem"
go build -o bin/dgem .
has() { [[ " $STEPS " == *" $1 "* ]]; }

$R note --run "$RUN_ID" "EXP-09 re-baseline: harness fixes (no ground-truth fill), image-free baselines, bootstrap CIs, probe variants ($VARIANTS) x$REPEAT on Vertex G4."

if has legacy; then
  $R exec --run "$RUN_ID" --suite bbox --config legacy_cloudrun_20260921_reanalyzed -- \
    $D bench-bbox --from-receipt benchmarks/results_bbox_cloudrun.json -o {out}
fi

if has simulate; then
  $R exec --run "$RUN_ID" --suite bbox --config simulated_all_variants -- \
    $D bench-bbox --simulate --variants all -o {out}
fi

if has live; then
  if [[ -z "$ENDPOINT" ]]; then
    echo "Error: ENDPOINT or DGEM_VERTEX_URL is required for the live step" >&2
    exit 1
  fi
  DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite bbox --config "vertex_g4_${VARIANTS//,/_}_x${REPEAT}" -- \
    $D bench-bbox --vertex-url "$ENDPOINT" --gcp-auth --variants "$VARIANTS" --repeat "$REPEAT" -w "$WORKERS" -o {out}
fi

# Receipts record the endpoint URL; strip internal identifiers before committing.
if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
