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

# Deploy the scheduled regression matrix: Cloud Run jobs + Cloud Scheduler triggers (docs/operate/regression-matrix.md).
#
#   dgem-matrix-t0   daily:  T0 smoke (health, contract, calibration, multilingual spot check), ~200 requests
#   dgem-matrix-t1   weekly: T1 gate (adds JevBench x3 on both prompt paths, calibration x3, intents, latency), ~2.7k requests
#
# Runs are single-target (compared with the reference ranges in benchmarks/matrix/matrix_v2.json). Each run uploads
# its directory (report.md, summary.json, redacted receipts) to MATRIX_BUCKET and writes JSON log lines
# (jsonPayload.matrix_event: dgem.matrix.start | suite | gate | done | uploaded). Nothing is committed. T2 is never
# scheduled.
#
# Required:
#   GCP_PROJECT        hosting project (never the image distribution project)
#   MATRIX_TARGET      serving base URL, e.g. the Vertex dedicated invoke base
#                      https://<ID>.<REGION>-<NUM>.prediction.vertexai.goog/v1/projects/<P>/locations/<R>/endpoints/<ID>/invoke
#   MATRIX_BUCKET      gs://<bucket>[/prefix] for run directories (created if the bucket does not exist)
# Optional:
#   GCP_REGION (us-central1)  MATRIX_TARGET_NAME (prod)  T0_SCHEDULE ("37 5 * * *")  T1_SCHEDULE ("37 6 * * 1")
#   SCHEDULE_TZ (Etc/UTC)  MATRIX_SA_NAME (dgem-matrix-sa)  MATRIX_T1=off (deploy T0 only)
set -euo pipefail

PROJECT="${GCP_PROJECT:?set GCP_PROJECT}"
REGION="${GCP_REGION:-us-central1}"
: "${MATRIX_TARGET:?set MATRIX_TARGET (serving base URL)}"
: "${MATRIX_BUCKET:?set MATRIX_BUCKET (gs://bucket[/prefix])}"
NAME="${MATRIX_TARGET_NAME:-prod}"
SA_NAME="${MATRIX_SA_NAME:-dgem-matrix-sa}"
SA="${SA_NAME}@${PROJECT}.iam.gserviceaccount.com"
T0_SCHEDULE="${T0_SCHEDULE:-37 5 * * *}"
T1_SCHEDULE="${T1_SCHEDULE:-37 6 * * 1}"
SCHEDULE_TZ="${SCHEDULE_TZ:-Etc/UTC}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAG="$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo latest)"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/dgem/dgem-matrix:${TAG}"
BUCKET="${MATRIX_BUCKET#gs://}"; BUCKET="${BUCKET%%/*}"

echo "==> Service account ${SA}"
gcloud iam service-accounts describe "${SA}" --project="${PROJECT}" >/dev/null 2>&1 || \
  gcloud iam service-accounts create "${SA_NAME}" --project="${PROJECT}" --display-name="dgem scheduled regression matrix"
for i in 1 2 3 4 5 6; do  # a new service account takes a few seconds to become visible to IAM
  gcloud projects add-iam-policy-binding "${PROJECT}" --member="serviceAccount:${SA}" \
    --role="roles/aiplatform.user" --condition=None --quiet >/dev/null 2>&1 && break
  [[ $i == 6 ]] && { echo "could not grant roles/aiplatform.user to ${SA}" >&2; exit 1; }
  sleep 10
done
if [[ "${MATRIX_TARGET}" == *".run.app"* ]]; then
  SVC="$(echo "${MATRIX_TARGET}" | sed -E 's#https://([a-z0-9-]+---)?([a-z0-9-]+)-[a-z0-9]+-[a-z]{2}\.a\.run\.app.*#\2#')"
  gcloud run services add-iam-policy-binding "${SVC}" --project="${PROJECT}" --region="${REGION}" \
    --member="serviceAccount:${SA}" --role="roles/run.invoker" --quiet >/dev/null
fi

echo "==> Bucket gs://${BUCKET}"
gcloud storage buckets describe "gs://${BUCKET}" --project="${PROJECT}" >/dev/null 2>&1 || \
  gcloud storage buckets create "gs://${BUCKET}" --project="${PROJECT}" --location="${REGION}" --uniform-bucket-level-access
gcloud storage buckets add-iam-policy-binding "gs://${BUCKET}" --member="serviceAccount:${SA}" \
  --role="roles/storage.objectCreator" --quiet >/dev/null

echo "==> Building ${IMAGE}"
CTX="$(mktemp -d)"; trap 'rm -rf "${CTX}"' EXIT
(cd "${REPO_ROOT}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${CTX}/dgem" .)
A="${CTX}/app"; mkdir -p "${A}/scripts" "${A}/benchmarks" "${A}/fixtures"
cp -R "${REPO_ROOT}/scripts/matrix" "${A}/scripts/"
cp "${REPO_ROOT}"/scripts/{bench_matrix.py,contract_diff.py,serving_speed.py} "${A}/scripts/"
cp -R "${REPO_ROOT}"/benchmarks/{jevbench,intents,matrix} "${A}/benchmarks/"
cp "${REPO_ROOT}/benchmarks/calibration_suite.jsonl" "${A}/benchmarks/"
cp -R "${REPO_ROOT}/templates" "${A}/"
cp -R "${REPO_ROOT}/fixtures/bbox" "${A}/fixtures/"
# Vision suites (EXP-22): the seeded synthetic sweep, generated here so the job image needs no Pillow.
if [[ ! -f "${REPO_ROOT}/fixtures/bbox_sweep/geo-1x1-grid-00.png" ]]; then
  python3 "${REPO_ROOT}/scripts/generate_bbox_sweep.py" >/dev/null || {
    echo "Error: fixtures/bbox_sweep/ missing; pip install -r scripts/requirements-vision.txt" >&2; exit 1; }
fi
cp -R "${REPO_ROOT}/fixtures/bbox_sweep" "${A}/fixtures/"
cp "${REPO_ROOT}/benchmarks/bbox_sweep.jsonl" "${A}/benchmarks/"
# Bake the verified datasets T0/T1 read (MASSIVE validation spot languages + their test label sets) into the image:
# anonymous Hugging Face downloads from cloud egress get rate-limited.
DGEM_MATRIX_CACHE="${A}/.cache" python3 - "${REPO_ROOT}" <<'PY'
import sys
sys.path.insert(0, sys.argv[1] + "/scripts")
from matrix import cases, datasets
for lg in cases.SPOT_LANGS:
    datasets.path(datasets.MASSIVE, f"validation/{lg}.json.gz")
    datasets.path(datasets.MASSIVE, f"test/{lg}.json.gz")
for f in ("validation/en.json.gz", "test/en.json.gz"):  # di_wide (Decision Index adapter track)
    datasets.path(datasets.MASSIVE, f)
datasets.path(datasets.CLINC, "plus/validation-00000-of-00001.parquet")  # di_catchall
datasets.path(datasets.RAGTRUTH, "data/train-00000-of-00001.parquet")   # rag_dev
PY
cp "${REPO_ROOT}/deploy/bench-matrix/Dockerfile" "${CTX}/Dockerfile"
gcloud builds submit "${CTX}" --project="${PROJECT}" --tag="${IMAGE}" --suppress-logs --quiet

deploy_job() {  # job tier timeout
  local job="$1" tier="$2" timeout="$3"
  echo "==> Cloud Run job ${job} (${tier})"
  gcloud run jobs deploy "${job}" --project="${PROJECT}" --region="${REGION}" --image="${IMAGE}" \
    --service-account="${SA}" --tasks=1 --max-retries=0 --task-timeout="${timeout}" --cpu=2 --memory=2Gi \
    --set-env-vars="^|^MATRIX_TIER=${tier}|MATRIX_TARGETS=${NAME}=${MATRIX_TARGET}|MATRIX_UPLOAD=${MATRIX_BUCKET%/}/${tier}" \
    --quiet >/dev/null
  gcloud run jobs add-iam-policy-binding "${job}" --project="${PROJECT}" --region="${REGION}" \
    --member="serviceAccount:${SA}" --role="roles/run.invoker" --quiet >/dev/null
}

schedule() {  # name job cron
  local name="$1" job="$2" cron="$3"
  local uri="https://run.googleapis.com/v2/projects/${PROJECT}/locations/${REGION}/jobs/${job}:run"
  local args=(--project="${PROJECT}" --location="${REGION}" --schedule="${cron}" --time-zone="${SCHEDULE_TZ}"
              --uri="${uri}" --http-method=POST --oauth-service-account-email="${SA}")
  if gcloud scheduler jobs describe "${name}" --project="${PROJECT}" --location="${REGION}" >/dev/null 2>&1; then
    gcloud scheduler jobs update http "${name}" "${args[@]}" --quiet >/dev/null
  else
    gcloud scheduler jobs create http "${name}" "${args[@]}" --quiet >/dev/null
  fi
  echo "    scheduled ${name}: '${cron}' (${SCHEDULE_TZ})"
}

deploy_job dgem-matrix-t0 T0 1800s
schedule dgem-matrix-t0 dgem-matrix-t0 "${T0_SCHEDULE}"
if [[ "${MATRIX_T1:-on}" != "off" ]]; then
  deploy_job dgem-matrix-t1 T1 7200s
  schedule dgem-matrix-t1 dgem-matrix-t1 "${T1_SCHEDULE}"
fi

cat <<EOF

Done. Run once now:
  gcloud run jobs execute dgem-matrix-t0 --project=${PROJECT} --region=${REGION} --wait
Reports: ${MATRIX_BUCKET%/}/<tier>/<run>/report.md. Logs: jsonPayload.matrix_event (dgem.matrix.gate with verdict=FAIL to alert on).
EOF
