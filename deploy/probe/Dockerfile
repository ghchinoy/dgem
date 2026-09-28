# Scheduled health check (synthetic probe) for dgem: scripts/probe.py on a small Python image.
# Built and deployed by scripts/deploy_probe.sh as Cloud Run jobs triggered by Cloud Scheduler.
FROM python:3.12-slim
WORKDIR /app
COPY scripts/probe.py scripts/contract_diff.py scripts/serving_speed.py /app/scripts/
COPY fixtures/bbox/ /app/fixtures/bbox/
ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python3", "/app/scripts/probe.py"]
