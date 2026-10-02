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

# Scheduled regression matrix (T0 / T1) as a Cloud Run job. Built by scripts/deploy_bench_matrix_job.sh, which
# stages a static dgem binary and the repository subset the T0/T1 suites read. T2 is never scheduled.
FROM python:3.12-slim
WORKDIR /app
# T1 reads parquet for di_catchall (CLINC150) and rag_dev (RAGTruth)
RUN pip install --no-cache-dir 'pyarrow>=15'
COPY dgem /usr/local/bin/dgem
COPY app/ /app/
ENV PYTHONUNBUFFERED=1 DGEM_MATRIX_CACHE=/app/.cache
ENTRYPOINT ["sh", "-c", "exec python3 scripts/bench_matrix.py run --tier \"${MATRIX_TIER:-T0}\" --label \"scheduled-$(echo ${MATRIX_TIER:-T0} | tr A-Z a-z)-$(date -u +%H%M)\" --out-dir \"/tmp/run/$(date -u +%Y%m%d-%H%M)-${MATRIX_TIER:-T0}\" --dgem /usr/local/bin/dgem --log-json ${MATRIX_UPLOAD:+--upload \"$MATRIX_UPLOAD\"}"]
