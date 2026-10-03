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

# PROP-18 phase 3: open-vocabulary detector references (OWLv2, Grounding DINO, Grounding DINO + SAM)
# on a short-lived GCE GPU VM. The VM is always deleted on exit (success, failure or Ctrl-C).
#
#   GCP_PROJECT=<PROJECT> BUCKET=<BUCKET> ./scripts/run_detectors_gce.sh
#
# Flow (no SSH): bundle the manifests, images and scripts/detectors/ -> gs://$BUCKET/$PREFIX/bundle.tgz;
# create the VM with a startup script that waits for the GPU driver, runs the detectors and uploads
# preds.jsonl + run.log + DONE; poll for DONE; download into benchmarks/runs/$RUN_ID/; delete the VM and
# the GCS prefix; score every model with `dgem bench-bbox --engine predictions` and record the receipts.
#
# Inputs: benchmarks/bbox_sweep.jsonl (+ fixtures/bbox_sweep, scripts/generate_bbox_sweep.py) and
#         benchmarks/bbox_real.jsonl (+ fixtures/bbox_real, scripts/fetch_bbox_real.py fetch).
# The VM needs internet access (Hugging Face model downloads) and a service account that can read and
# write gs://$BUCKET (GCE_SA, default: the project's default compute service account).
set -euo pipefail
cd "$(dirname "$0")/.."

PROJECT="${GCP_PROJECT:?GCP_PROJECT is required}"
BUCKET="${BUCKET:?BUCKET is required (a bucket the VM service account can read and write)}"
ZONE="${GCP_ZONE:-us-central1-a}"
MACHINE="${GCE_MACHINE_TYPE:-g2-standard-8}"
ACCEL="${GCE_ACCELERATOR:-type=nvidia-l4,count=1}"
IMAGE_FAMILY="${IMAGE_FAMILY:-pytorch-2-9-cu129-ubuntu-2204-nvidia-580}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%d)-prop18-detectors}"
MANIFESTS="${MANIFESTS:-benchmarks/bbox_sweep.jsonl benchmarks/bbox_real.jsonl}"
NAME="dgem-detectors-$(date -u +%H%M%S)"
PREFIX="prop18-detectors/${RUN_ID}-${NAME}"
TIMEOUT_MIN="${TIMEOUT_MIN:-75}"
OUT="benchmarks/runs/$RUN_ID"
SA_FLAG=()
[[ -n "${GCE_SA:-}" ]] && SA_FLAG=(--service-account="$GCE_SA")

cleanup() {
  echo "==> Teardown: deleting VM $NAME and gs://$BUCKET/$PREFIX"
  gcloud compute instances delete "$NAME" --zone "$ZONE" --project "$PROJECT" --quiet >/dev/null 2>&1 || true
  gcloud storage rm -r "gs://$BUCKET/$PREFIX" --project "$PROJECT" >/dev/null 2>&1 || true
  if gcloud compute instances describe "$NAME" --zone "$ZONE" --project "$PROJECT" >/dev/null 2>&1; then
    echo "WARNING: VM $NAME still exists; delete it manually" >&2
  else
    echo "    VM deleted."
  fi
}
trap cleanup EXIT

mkdir -p "$OUT"
TMP="$(mktemp -d)"
echo "==> Bundling $MANIFESTS and their images"
python3 - "$TMP/files.txt" $MANIFESTS <<'EOF'
import json, sys
out, mans = sys.argv[1], sys.argv[2:]
paths = list(mans) + ["scripts/detectors/run_detectors.py", "scripts/detectors/requirements.txt"]
for m in mans:
    paths += [json.loads(l)["image_path"] for l in open(m) if l.strip()]
open(out, "w").write("\n".join(paths) + "\n")
EOF
tar -czf "$TMP/bundle.tgz" -T "$TMP/files.txt"
gcloud storage cp "$TMP/bundle.tgz" "gs://$BUCKET/$PREFIX/bundle.tgz" --project "$PROJECT" >/dev/null

cat > "$TMP/startup.sh" <<EOF
#!/bin/bash
# Log to a file; the metadata script runner forwards stdout to the serial console. (Writing to /dev/ttyS0
# directly can fail with EIO, which kills tee and stalls the script.) Always upload the log and an exit code.
exec > >(tee -a /var/log/dgem-detectors.log) 2>&1
set -x
finish() { rc=\$?; cp /var/log/dgem-detectors.log /opt/work/startup.log 2>/dev/null || cp /var/log/dgem-detectors.log /tmp/startup.log;
  gcloud storage cp /opt/work/startup.log gs://$BUCKET/$PREFIX/out/ 2>/dev/null || gcloud storage cp /tmp/startup.log gs://$BUCKET/$PREFIX/out/;
  echo \${RC:-\$rc} | gcloud storage cp - gs://$BUCKET/$PREFIX/out/DONE; }
trap finish EXIT
for i in \$(seq 1 90); do nvidia-smi && break; sleep 10; done
mkdir -p /opt/work && cd /opt/work
# Bounded, non-interactive copies: one attempt once hung without output.
for i in 1 2 3; do timeout 300 gcloud storage cp gs://$BUCKET/$PREFIX/bundle.tgz . < /dev/null > /dev/null && break; sleep 10; done
tar -xzf bundle.tgz || exit 1
PY=\$(command -v python3)
[ -x /opt/conda/bin/python ] && PY=/opt/conda/bin/python
\$PY -m pip install -q -r scripts/detectors/requirements.txt < /dev/null
\$PY scripts/detectors/run_detectors.py --manifest $MANIFESTS -o preds.jsonl > run.log 2>&1
rc=\$?
\$PY -c "import torch, transformers; print('torch', torch.__version__, 'transformers', transformers.__version__)" >> run.log 2>&1
nvidia-smi --query-gpu=name,driver_version --format=csv >> run.log 2>&1
for i in 1 2 3; do timeout 300 gcloud storage cp preds.jsonl run.log gs://$BUCKET/$PREFIX/out/ < /dev/null > /dev/null && break; sleep 10; done
RC=\$rc
EOF

echo "==> Creating $NAME ($MACHINE, $ACCEL, $ZONE)"
gcloud compute instances create "$NAME" --project "$PROJECT" --zone "$ZONE" \
  --machine-type "$MACHINE" --accelerator "$ACCEL" --maintenance-policy TERMINATE \
  --image-family "$IMAGE_FAMILY" --image-project deeplearning-platform-release \
  --boot-disk-size 150GB --scopes cloud-platform "${SA_FLAG[@]}" \
  --metadata install-nvidia-driver=True --metadata-from-file startup-script="$TMP/startup.sh" \
  --labels purpose=dgem-prop18-detectors >/dev/null

echo "==> Waiting for results (up to $TIMEOUT_MIN min)"
deadline=$(( $(date +%s) + TIMEOUT_MIN * 60 ))
until gcloud storage ls "gs://$BUCKET/$PREFIX/out/DONE" --project "$PROJECT" >/dev/null 2>&1; do
  if (( $(date +%s) > deadline )); then
    echo "Timed out; serial console tail:" >&2
    gcloud compute instances get-serial-port-output "$NAME" --zone "$ZONE" --project "$PROJECT" 2>/dev/null | tail -40 >&2 || true
    exit 1
  fi
  sleep 30
done
gcloud storage cp "gs://$BUCKET/$PREFIX/out/*" "$OUT/" --project "$PROJECT" >/dev/null
[[ -f "$OUT/startup.log" ]] && mv "$OUT/startup.log" "$OUT/detector_startup.log"
rc="$(cat "$OUT/DONE")"; rm -f "$OUT/DONE"
if [[ "$rc" != 0 || ! -f "$OUT/preds.jsonl" ]]; then
  echo "Detector run failed (exit $rc); startup log tail:" >&2
  tail -40 "$OUT/detector_startup.log" >&2 || true
  [[ -f "$OUT/run.log" ]] && tail -30 "$OUT/run.log" >&2
  exit 1
fi
mv "$OUT/preds.jsonl" "$OUT/detector_preds.jsonl"
mv "$OUT/run.log" "$OUT/detector_run.log"
echo "==> Detector run exit code $rc; $(wc -l < "$OUT/detector_preds.jsonl") prediction rows"

go build -o bin/dgem .
for M in owlv2 grounding_dino gdino_sam; do
  for S in $MANIFESTS; do
    kind="$(basename "$S" .jsonl)"
    python3 scripts/bench_runs.py exec --run "$RUN_ID" --suite "$kind" --config "$M" -- \
      ./bin/dgem bench-bbox -d "$S" --engine predictions --predictions "$OUT/detector_preds.jsonl" --pred-model "$M" -o {out} >/dev/null
  done
done
echo "Done: $OUT"
