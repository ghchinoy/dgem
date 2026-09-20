# syntax=docker/dockerfile:1
# DiffusionGemma Cloud Run Serverless GPU Container
# Fully self-contained, reproducible multi-stage build.
#
# Lineage:
# 1. Base image: vLLM official nightly image pinned to commit dee37d89115
# 2. Structured reads fork: mmastrac/vllm pinned to commit 6591b093b
# 3. Cloud Run serverless runtime optimizations: eager-mode worker bypass,
#    GCS FUSE buffered streaming, and TypeSafe AI / Jev structured server proxy.

ARG BASE=vllm/vllm-openai:nightly-dee37d89115db4c94a820a79a78a7828e141c910

# --- Stage 1: The structured reads fork at pinned commit ----------------------
FROM alpine/git:latest AS fork
ARG VLLM_FORK=https://github.com/mmastrac/vllm.git
ARG VLLM_UPSTREAM=https://github.com/vllm-project/vllm.git
ARG VLLM_REF=6591b093b29536dd070c6af3628b734025c53e23
ARG VLLM_BASE=dee37d89115db4c94a820a79a78a7828e141c910

RUN git clone --filter=blob:none --quiet "${VLLM_FORK}" /fork \
    && cd /fork \
    && git checkout --quiet "${VLLM_REF}" \
    && git fetch --filter=blob:none --quiet "${VLLM_UPSTREAM}" "${VLLM_BASE}" \
    && mb=$(git merge-base "${VLLM_BASE}" HEAD) \
    && git diff --name-only "$mb" HEAD -- vllm > /fork/changed.txt \
    && git diff --name-only --diff-filter=A "$mb" HEAD -- vllm > /fork/added.txt \
    && cat /fork/changed.txt \
    && if ! git diff --quiet "$mb" "${VLLM_BASE}" -- $(cat /fork/changed.txt); then \
         echo "upstream changed overlaid files between branch base and ${VLLM_BASE}; rebase required" >&2; \
         exit 1; fi

# --- Stage 2: Production Cloud Run Runtime Image ------------------------------
FROM ${BASE}
ARG VLLM_BASE=dee37d89115db4c94a820a79a78a7828e141c910

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
ENV PORT=8080

# 1. Symlink CUDA headers where FlashInfer/CUTLASS JIT requires them
COPY patches/link_cuda_headers.sh /tmp/link_cuda_headers.sh
RUN bash /tmp/link_cuda_headers.sh && rm /tmp/link_cuda_headers.sh

# 2. Structured reads overlay from Stage 1
COPY patches/overlay_vllm.py /tmp/overlay_vllm.py
RUN --mount=type=bind,from=fork,source=/fork,target=/fork \
    python3 /tmp/overlay_vllm.py /fork "${VLLM_BASE}" && rm /tmp/overlay_vllm.py

# 3. Raise Dynamo recompile limit for multi-width canvas reads
COPY patches/raise_recompile_limit.py /tmp/raise_recompile_limit.py
RUN python3 /tmp/raise_recompile_limit.py && rm /tmp/raise_recompile_limit.py

# 4. Patch vLLM gpu_worker for eager mode on Cloud Run
COPY patch_vllm.py /tmp/patch_vllm.py
RUN python3 /tmp/patch_vllm.py && rm /tmp/patch_vllm.py

# 5. Install compressed-tensors for dynamic FP8 / INT8 quantization support
RUN pip install --no-cache-dir compressed-tensors

# 6. Bundle structured_server proxy and entrypoint
COPY server/structured_server.py /opt/dgemma/structured_server.py
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /opt/dgemma/structured_server.py \
    && python3 -m py_compile /opt/dgemma/structured_server.py

EXPOSE 8080 8000
ENTRYPOINT ["/entrypoint.sh"]
