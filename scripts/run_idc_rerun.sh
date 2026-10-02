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

# Same-session IDC re-run (PROP-02 offline gates, PROP-05, PROP-10, EXP-13 reproduction, JevBench with dgem).
# Every receipt is written into benchmarks/runs/$RUN_ID/ and recorded in its manifest.json.
#
#   RUN_ID=20260925-vertex-idc ./scripts/run_idc_rerun.sh            # all steps
#   STEPS="calibration permutation" ./scripts/run_idc_rerun.sh        # subset
#
# Requires Application Default Credentials with access to the Vertex endpoint and Gemini.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-vertex-idc}"
ENDPOINT="${ENDPOINT:-${DGEM_VERTEX_URL:-}}"
if [[ -z "${ENDPOINT}" ]]; then
  echo "Error: ENDPOINT or DGEM_VERTEX_URL is required" >&2
  exit 1
fi
GEMINI="${GEMINI:-gemini-3.8-flash}"
WORKERS="${WORKERS:-4}"
STEPS="${STEPS:-calibration permutation jevbench gemini}"
export GOOGLE_CLOUD_PROJECT="${GOOGLE_CLOUD_PROJECT_ENDPOINT:-${GCP_PROJECT:-}}"
VPROJ="${VERTEX_PROJECT:-${GCP_PROJECT:-}}"

R="python3 scripts/bench_runs.py"
D="./bin/dgem"
T="--vertex-url $ENDPOINT --gcp-auth"

make build >/dev/null 2>&1 || go build -o bin/dgem .
$R note --run "$RUN_ID" "Same-session IDC re-run against Vertex endpoint $ENDPOINT; Stage 2 = $GEMINI. Baselines interleaved to measure drift."

has() { [[ " $STEPS " == *" $1 "* ]]; }

if has calibration; then
  cal() { local cfg="$1"; shift; $R exec --run "$RUN_ID" --suite calibration --config "$cfg" -- $D bench-calibration $T -w "$WORKERS" -o {out} "$@"; }
  cal baseline_r1
  cal null_prior --null-prior-debias
  cal dual_mirror --dual-mirror
  cal baseline_r2
  cal dual_mirror_realnames --dual-mirror --mirror-alias-names=false
  cal null_prior_dual_mirror --null-prior-debias --dual-mirror
  cal baseline_r3
fi

if has permutation; then
  $R exec --run "$RUN_ID" --suite permutation --config exp13_vertex -- $D bench-permutation $T -w 2 --out {out}
fi

if has jevbench; then
  jev() { local cfg="$1"; shift; $R exec --run "$RUN_ID" --suite jevbench --config "$cfg" -- $D bench-jev $T -w "$WORKERS" -o {out} "$@"; }
  jev baseline
  jev null_prior --null-prior-debias
  jev dual_mirror --dual-mirror
  jev dual_mirror_realnames --dual-mirror --mirror-alias-names=false
fi

if has gemini; then
  # Stage-2 answers for every item (threshold 0 escalates all), prior-guided by the baseline run, so any
  # gate can be simulated offline with scripts/analyze_idc.py gates.
  $R exec --run "$RUN_ID" --suite jevbench --config gemini_all_prior_guided -- \
    $D bench-jev --cascade-from "benchmarks/runs/$RUN_ID/jevbench__baseline.json" --cascade-threshold 0 \
    --vertex-model "$GEMINI" --vertex-project "$VPROJ" -w "$WORKERS" -o {out}
  $R exec --run "$RUN_ID" --suite calibration --config gemini38_standalone -- \
    $D bench-calibration --vertex-model "$GEMINI" --vertex-project "$VPROJ" -w "$WORKERS" -o {out}
fi

$R compare
