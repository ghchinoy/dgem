# Scheduled regression matrix (T0 / T1) as a Cloud Run job. Built by scripts/deploy_bench_matrix_job.sh, which
# stages a static dgem binary and the repository subset the T0/T1 suites read. T2 is never scheduled.
FROM python:3.12-slim
WORKDIR /app
COPY dgem /usr/local/bin/dgem
COPY app/ /app/
ENV PYTHONUNBUFFERED=1 DGEM_MATRIX_CACHE=/app/.cache
ENTRYPOINT ["sh", "-c", "exec python3 scripts/bench_matrix.py run --tier \"${MATRIX_TIER:-T0}\" --label \"scheduled-$(echo ${MATRIX_TIER:-T0} | tr A-Z a-z)-$(date -u +%H%M)\" --out-dir \"/tmp/run/$(date -u +%Y%m%d-%H%M)-${MATRIX_TIER:-T0}\" --dgem /usr/local/bin/dgem --log-json ${MATRIX_UPLOAD:+--upload \"$MATRIX_UPLOAD\"}"]
