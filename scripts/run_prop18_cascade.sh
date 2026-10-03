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

# PROP-18 live hesitation cascade for image questions: dgem answers, slots with entropy >= T nats go to Gemini 3.x
# WITH the image (ExecuteStage2GeminiCascadeWithImages, the same code the gateway and MCP use).
#
#   ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_PROJECT=<PROJECT> ./scripts/run_prop18_cascade.sh
#
# Configs: all six questions per image (THRESHOLDS), and --only-scored (ask only what the item is scored on).
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-prop18-cascade}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:?ENDPOINT or DGEM_VERTEX_URL is required}}"
SUITE="${SUITE_FILE:-benchmarks/bbox_sweep.jsonl}"
TEMPLATE="${TEMPLATE:-templates/multimodal/vision_aspects.json.tmpl}"
THRESHOLDS="${THRESHOLDS:-0.35 0.1}"
MODEL="${MODEL:-gemini-3.8-flash}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"
R="python3 scripts/bench_runs.py"
go build -o bin/dgem .
$R note --run "$RUN_ID" "PROP-18 live image cascade on $SUITE: thresholds $THRESHOLDS nats -> $MODEL with the image; all questions and --only-scored."

for T in $THRESHOLDS; do
  for scope in all scored; do
    flag=(); [[ $scope == scored ]] && flag=(--only-scored)
    DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1 $R exec --run "$RUN_ID" --suite vision_cascade --config "${scope}_t${T/./p}" -- \
      ./bin/dgem bench-vision -d "$SUITE" -t "$TEMPLATE" --vertex-url "$ENDPOINT" --gcp-auth "${flag[@]}" \
        --cascade-threshold "$T" --cascade-model "$MODEL" -w 8 -o {out}
  done
done
if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
