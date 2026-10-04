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

# EXP-26: guided boxes (bench-guided) and categorical image questions (bench-vision) on the domain image set,
# with gemini-3.8-flash and gemini-3.7-flash. Decision rule: docs/experiments/exp-26-domain-images.md.
#
#   ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_PROJECT=<PROJECT> ./scripts/run_exp26_domains.sh
#   python3 scripts/exp26_decide.py benchmarks/runs/<run_id>
#
# Needs fixtures/domain_images/ (python3 scripts/build_domain_images.py fetch).
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-exp26-domains}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:?ENDPOINT or DGEM_VERTEX_URL is required}}"
MODELS="${MODELS:-gemini-3.8-flash gemini-3.7-flash}"
STEPS="${STEPS:-guided vision}"
SUITE=benchmarks/domain_images.jsonl
TEMPLATE=templates/multimodal/vision_aspects_generic.json.tmpl
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT:-${GCP_PROJECT:-}}"
R="python3 scripts/bench_runs.py"
has() { [[ " $STEPS " == *" $1 "* ]]; }
go build -o bin/dgem .
$R note --run "$RUN_ID" "EXP-26 domain images ($SUITE): bench-guided full/hint x default/low per model ($MODELS); bench-vision dgem x2, blank, each Gemini model."
export DGEM_VERTEX_URL="$ENDPOINT" RECORD_VERTEX_STATE=1

if has guided; then
  for M in $MODELS; do
    $R exec --run "$RUN_ID" --suite guided --config "${M//[.-]/_}" -- \
      ./bin/dgem bench-guided -d "$SUITE" -t "$TEMPLATE" --vertex-url "$ENDPOINT" --gcp-auth --gemini-model "$M" \
        --conditions full@default,full@low,hint@default,hint@low --neg-conditions full@default,full@low -w 12 -o {out}
  done
fi
if has vision; then
  $R exec --run "$RUN_ID" --suite vision --config vertex_g4_x2 -- \
    ./bin/dgem bench-vision -d "$SUITE" -t "$TEMPLATE" --vertex-url "$ENDPOINT" --gcp-auth --repeat 2 -o {out}
  $R exec --run "$RUN_ID" --suite vision --config vertex_g4_blank -- \
    ./bin/dgem bench-vision -d "$SUITE" -t "$TEMPLATE" --vertex-url "$ENDPOINT" --gcp-auth --variant blank -o {out}
  for M in $MODELS; do
    $R exec --run "$RUN_ID" --suite vision --config "reference_${M//[.-]/_}" -- \
      ./bin/dgem bench-vision -d "$SUITE" -t "$TEMPLATE" --engine gemini --gemini-model "$M" -o {out}
  done
fi
if [[ -f scratch/leak-patterns.txt ]]; then
  python3 scripts/redact_receipt.py benchmarks/runs/"$RUN_ID"/*.json --in-place
fi
echo "Done: benchmarks/runs/$RUN_ID"
