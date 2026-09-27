# syntax=docker/dockerfile:1
# DiffusionGemma Production & Decision Index Container
# Built cleanly on official vLLM nightly pinned by digest (commit a9eafde59cb).
# Upstream PR #57250 and PR #58216 are natively included in the base image.

ARG BASE=vllm/vllm-openai:nightly-a9eafde59cbd55182dc2265cc398b7186f0a0eaa@sha256:896b7652343b6230cf788f4f1744563539f461f0454f30c5c7bb96ee9f8a34be

FROM ${BASE} AS base
ARG VLLM_COMMIT=a9eafde59cbd55182dc2265cc398b7186f0a0eaa

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
ENV PORT=8080

# 1. Symlink CUDA headers where FlashInfer/CUTLASS JIT requires them
COPY patches/link_cuda_headers.sh /tmp/link_cuda_headers.sh
RUN bash /tmp/link_cuda_headers.sh && rm /tmp/link_cuda_headers.sh

# 2. Install compressed-tensors and huggingface_hub
RUN pip install --no-cache-dir compressed-tensors huggingface_hub

# 3. Bundle dgem binary
COPY dgem /usr/local/bin/dgem
RUN chmod +x /usr/local/bin/dgem

# 4. Bundle patched structured_server proxy and entrypoint
COPY server/structured_server.py /opt/dgemma/structured_server.py
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /opt/dgemma/structured_server.py \
    && python3 -m py_compile /opt/dgemma/structured_server.py

EXPOSE 8080 8081 8000
ENTRYPOINT ["/entrypoint.sh"]
