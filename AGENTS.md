<!-- headroom:rtk-instructions -->
# RTK (Rust Token Killer) - Token-Optimized Commands

When running shell commands, **always prefix with `rtk`**. This reduces context
usage by 60-90% with zero behavior change. If rtk has no filter for a command,
it passes through unchanged — so it is always safe to use.

## Key Commands
```bash
# Git (59-80% savings)
rtk git status          rtk git diff            rtk git log

# Files & Search (60-75% savings)
rtk ls <path>           rtk read <file>         rtk grep <pattern>
rtk find <pattern>      rtk diff <file>

# Test (90-99% savings) — shows failures only
rtk pytest tests/       rtk cargo test          rtk test <cmd>

# Build & Lint (80-90% savings) — shows errors only
rtk tsc                 rtk lint                rtk cargo build
rtk prettier --check    rtk mypy                rtk ruff check

# Analysis (70-90% savings)
rtk err <cmd>           rtk log <file>          rtk json <file>
rtk summary <cmd>       rtk deps                rtk env

# GitHub (26-87% savings)
rtk gh pr view <n>      rtk gh run list         rtk gh issue list

# Infrastructure (85% savings)
rtk docker ps           rtk kubectl get         rtk docker logs <c>

# Package managers (70-90% savings)
rtk pip list            rtk pnpm install        rtk npm run <script>
```

## Rules
- In command chains, prefix each segment: `rtk git add . && rtk git commit -m "msg"`
- For debugging, use raw command without rtk prefix
- `rtk proxy <cmd>` runs command without filtering but tracks usage
<!-- /headroom:rtk-instructions -->

# DiffusionGemma (`dgem`) Project Guidelines

## 1. Serving Backends & Telemetry (`logprobs`)
- **Local Apple Silicon Metal (`diffgemma`)**: Returns native `StructuredDecisionResponse` JSON (`answers` + `diagnostics`) with restricted-softmax `probabilities`, `first_read_max_entropy`, and `reused_tokens`.
- **Cloud GPU vLLM (`/v1/chat/completions`)**:
  - Requires installing the nightly prebuilt wheel (`wheels.vllm.ai`) matching the base commit of vLLM PR #57250 (`133b71e0be`) to avoid C++ CUDA extension ABI symbol mismatches (`gptq_marlin_repack`, `moe_wna16_marlin_gemm`).
  - Must launch with `--attention-backend TRITON_ATTN --enforce-eager --max-model-len 32768` because FlashAttention-2 rejects DiffusionGemma's mixed causal-prompt / bidirectional-canvas attention masks.
  - `pkg/client` requests `logprobs: true, top_logprobs: 5` on every `Decide` call and uses `ParseStructuredContentWithLogprobs` to compute calibrated slot probability $\exp(\text{logprob})$, Shannon entropy $H = -\sum p_k \ln p_k$, and top-$k$ candidate probabilities.
- **Serverless Cloud Run GPU (`structured_server.py`)**:
  - Acts as a local reverse proxy in front of vLLM on port 8000, serving `POST /v1/chat/completions` on port 8080.
  - Returns structured envelope JSON `{"answers": {...}, "diagnostics": {...}}`. `pkg/client` supports envelope fallback parsing, handling polymorphic slice-based `entropy` and dictionary-based sampling policies.

## 2. Cloud GPU Provisioning & Zero-Idle-Cost Mandate
- **Serverless Cloud Run (1× NVIDIA L4, 24GB)**:
  - Container Image: Built in-repo from `deploy/cloudrun/Dockerfile` and pushed to Google Artifact Registry: `us-central1-docker.pkg.dev/$PROJECT/dgem/dgemma:latest` (`make cloudrun-build`).
  - Weights: Pre-staged in GCS: `gs://dgem-weights-$PROJECT/dgemma/` (`make cloudrun-stage`).
  - Service Name: Strictly **`dgemma`** (never `"djev"` or `"djev-dgemma"`).
  - Deploy: `make cloudrun-deploy` (mounts GCS bucket via Cloud Storage FUSE with `--safetensors-load-strategy prefetch`).
  - Query: `./bin/dgem decide -u "${URL}/v1" --gcp-auth ...`
  - **Mandatory Immediate Teardown**: Always run `make cloudrun-teardown` (`gcloud run services delete dgemma --region=us-central1 --quiet`) immediately after capturing test receipts.
- **4-bit (`NVFP4`) on GCE VM 1× NVIDIA L4 (`g2-standard-8`)**: Use project `ghchinoy-genai-sa` (`PRECISION=4 make gce-deploy`).
- **16-bit (`bfloat16`) on GCE VM 2× NVIDIA A100-40GB (`a2-highgpu-2g`, `TP=2`)**: Use project `genai-blackbelt-fishfooding` (`GCP_PROJECT=genai-blackbelt-fishfooding GCP_ZONE=us-central1-b PRECISION=16 make gce-deploy`).
- **Mandatory Immediate Teardown**: Always run `make gce-teardown` (`./scripts/teardown_gce_vllm.sh`) immediately after capturing benchmark receipts so zero cloud GPU resources remain active. Never commit `.env` (contains `HF_TOKEN`).

## 3. Benchmark Harnesses
- `dgem bench`: 30-case multi-domain decision suite (`benchmarks/eval_dataset.jsonl` across `support`, `code_review`, `security`). Receipts stored in `benchmarks/results_local_metal_slot.json`, `benchmarks/results_gce_l4.json`, and `benchmarks/results_cloudrun.json`.
- `dgem bench-ecotone`: 49-case Text Normalization comparison against the C++ `ecotone` Sparrowhawk/NeMo WFST sidecar (`unix:///tmp/ecotone.sock`) across `tn_semiotics.jsonl` (30 polysemy traps) and `tn_challenge_en.jsonl` (19 deterministic NSWs).
- `dgem bench-intents`: High-cardinality intent & Out-of-Scope (`oos`) evaluation for `PolyAI/banking77` and `DeepPavlov/clinc150`. Pass `--dataset banking77 --full --workers 16` (3,080 test items) or `--dataset clinc150 --full --workers 16` (5,500 test items) to run full upstream splits.


