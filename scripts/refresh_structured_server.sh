#!/usr/bin/env bash
set -euo pipefail

# refresh_structured_server.sh
# Fetches upstream structured_server.py from vllm-project/vllm:main at a pinned commit,
# records provenance in UPSTREAM, and applies dgem.patch.

COMMIT="${1:-a9eafde59cbd55182dc2265cc398b7186f0a0eaa}"
TARGET_DIR="deploy/cloudrun/server"
UPSTREAM_URL="https://raw.githubusercontent.com/vllm-project/vllm/${COMMIT}/examples/features/structured_diffusion/structured_server.py"

echo "==> Fetching upstream structured_server.py @ ${COMMIT:0:10}..."
curl -fsSL "$UPSTREAM_URL" -o "${TARGET_DIR}/structured_server.upstream.py"

cat <<EOF > "${TARGET_DIR}/UPSTREAM"
repository: https://github.com/vllm-project/vllm
commit: ${COMMIT}
file: examples/features/structured_diffusion/structured_server.py
pulled_at: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

echo "==> Applying dgem.patch to produce ${TARGET_DIR}/structured_server.py..."
cp "${TARGET_DIR}/structured_server.upstream.py" "${TARGET_DIR}/structured_server.py"
patch "${TARGET_DIR}/structured_server.py" "${TARGET_DIR}/dgem.patch"

python3 -m py_compile "${TARGET_DIR}/structured_server.py"
echo "==> Successfully refreshed and verified ${TARGET_DIR}/structured_server.py!"
