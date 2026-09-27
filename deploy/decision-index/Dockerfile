# syntax=docker/dockerfile:1
# Decision Index Certified Server with Wide-Canvas Tournament Adapter
# Speaks POST /v1/systemone (compatible with python -m decision_index run --engine http)

ARG BASE=ghcr.io/taeold/djev-run:latest
FROM ${BASE}

USER root

# Install huggingface_hub for standalone weight downloads
RUN pip install --no-cache-dir huggingface_hub

# Bundle dgem binary
COPY dgem /usr/local/bin/dgem
RUN chmod +x /usr/local/bin/dgem

# Copy dedicated entrypoint
COPY entrypoint.sh /entrypoint-di.sh
RUN chmod +x /entrypoint-di.sh

EXPOSE 8080 8081 8000
ENTRYPOINT ["/entrypoint-di.sh"]
