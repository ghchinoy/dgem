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
  - Requires patching `self._enable_mm_lora = False` in `Gemma4ForConditionalGeneration.__init__` (`vllm/model_executor/models/gemma4_mm.py`) so `DiffusionGemmaForConditionalGeneration` does not raise `AttributeError` during model load (automated in `scripts/startup_gce_vllm.sh` and `deploy/cloudrun/patch_vllm.py`).
  - `pkg/client` requests `logprobs: true, top_logprobs: 5` on every `Decide` call and uses `ParseStructuredContentWithLogprobs` to compute calibrated slot probability $\exp(\text{logprob})$, Shannon entropy $H = -\sum p_k \ln p_k$, and top-$k$ candidate probabilities.
- **Serverless Cloud Run GPU (`structured_server.py`)**:
  - Acts as a local reverse proxy in front of vLLM on port 8000, serving `POST /v1/chat/completions` on port 8080.
  - Returns structured envelope JSON `{"answers": {...}, "diagnostics": {...}}`. `pkg/client` supports envelope fallback parsing, handling polymorphic slice-based `entropy` and dictionary-based sampling policies.
  - **26-Option `[A-Z]` Slot Limit**: `structured_server.py` maps single-token `choice` slots to uppercase English letters `A`–`Z`, enforcing a maximum of **26 alternatives per `choice` question** (`HTTP 400: at most 26 alternatives`). High-cardinality taxonomies (>26 labels) must use a 26-option slice or two-stage hierarchical routing.
  - **`depends_on` / `ask_if` Boolean Format & 2-Stage Split (`reads=2`)**: In `structured_server.py`, boolean (`noul`) questions map to string choices `["yes", "no"]` (so `ask_if` must specify `["yes"]`, never JSON `[true]`). Adding `depends_on` / `ask_if` partitions questions into `level 0` and `level 1` in `schedule(qs)`, forcing **2 sequential forward passes (`reads=2`)**. To guarantee a strict **single forward pass (`reads=1`)** (e.g., for `EXP-09` bounding-box readout), omit `depends_on` / `ask_if` so all slots reside in `level 0` simultaneously and gate downstream interpretation in the client (`pkg/client` / `cmd/bench_bbox.go`).

## 2. Cloud GPU Provisioning & Zero-Idle-Cost Mandate
- **Serverless Cloud Run (`NVIDIA L4` 24GB or `NVIDIA RTX Pro 6000` 48GB)**:
  - Container Image: Built in-repo from `deploy/cloudrun/Dockerfile` and pushed to Google Artifact Registry: `us-central1-docker.pkg.dev/$PROJECT/dgem/dgemma:latest` (`make cloudrun-build`).
  - Weights: Pre-staged in GCS: `gs://dgem-weights-$PROJECT/dgemma/` (`make cloudrun-stage`).
  - Service Name: Strictly **`dgemma`** (never `"djev"` or `"djev-dgemma"`).
  - Deploy: `make cloudrun-deploy` (mounts GCS bucket via Cloud Storage FUSE with `--safetensors-load-strategy prefetch`).
  - **Multimodal `SigLIP` Vision Workloads (`DISABLE_MM=0`) & 32Gi Page-Cache OOM**: `scripts/deploy_cloudrun_vllm.sh` defaults to `DISABLE_MM=${DISABLE_MM:-0}` so Gemma 4's `SigLIP` vision tower is enabled (`--limit-mm-per-prompt '{"image":1,"video":0}'`). When `DISABLE_MM=0` is combined with `--safetensors-load-strategy prefetch` over GCS FUSE, Linux caches the 17.53 GiB safetensors in page cache while PyTorch allocates a second 17.53 GiB buffer + `SigLIP` warmup tensors, exceeding the `32Gi` container RAM ceiling on `nvidia-l4`. For multimodal `SigLIP` workloads on Cloud Run, deploy with `CLOUDRUN_GPU_TYPE=nvidia-rtx-pro-6000 DISABLE_MM=0` (`80Gi` RAM, `48GB` VRAM).
  - **Cold-Start Readiness Pattern**: On scale-from-zero (`--min-instances=0`), `structured_server.py` on port 8080 passes `/health` probes immediately while vLLM's `EngineCore` on port 8000 takes ~6–8 minutes to stream the 17.53 GiB checkpoint over GCS FUSE (`ConnectionRefusedError(111)`). Always poll with a single `-w 1` `dgem decide` readiness probe before launching concurrent `-w 4` benchmark workers to avoid `HTTP 429` / `HTTP 500` errors.
  - Query: `./bin/dgem decide -u "${URL}/v1" --gcp-auth ...`
  - **Mandatory Immediate Teardown**: Always chain `&& make cloudrun-teardown` (`gcloud run services delete dgemma --region=us-central1 --quiet`) immediately after capturing test receipts.
- **4-bit (`NVFP4`) on GCE VM 1× NVIDIA L4 (`g2-standard-8`)**: Use project `ghchinoy-genai-sa` (`PRECISION=4 make gce-deploy`).
- **16-bit (`bfloat16`) on GCE VM 2× NVIDIA A100-40GB (`a2-highgpu-2g`, `TP=2`)**: Use project `genai-blackbelt-fishfooding` (`GCP_PROJECT=genai-blackbelt-fishfooding GCP_ZONE=us-central1-b PRECISION=16 make gce-deploy`).
- **Mandatory Immediate Teardown**: Always run `make gce-teardown` (`./scripts/teardown_gce_vllm.sh`) immediately after capturing benchmark receipts so zero cloud GPU resources remain active. Never commit `.env` (contains `HF_TOKEN`).

## 3. Benchmark Harnesses
- `dgem bench`: 30-case multi-domain decision suite (`benchmarks/eval_dataset.jsonl` across `support`, `code_review`, `security`). Receipts stored in `benchmarks/results_local_metal_slot.json`, `benchmarks/results_gce_l4.json`, and `benchmarks/results_cloudrun.json`.
- `dgem bench-ecotone`: 49-case Text Normalization comparison against the C++ `ecotone` Sparrowhawk/NeMo WFST sidecar (`unix:///tmp/ecotone.sock`) across `tn_semiotics.jsonl` (30 polysemy traps) and `tn_challenge_en.jsonl` (19 deterministic NSWs).
- `dgem bench-intents`: High-cardinality intent & Out-of-Scope (`oos`) evaluation for `PolyAI/banking77` and `DeepPavlov/clinc150`. Pass `--dataset banking77 --full --workers 16` (3,080 test items) or `--dataset clinc150 --full --workers 16` (5,500 test items) to run full upstream splits.
- `dgem bench-calibration`: 50-case public dataset calibration, guardrail (`AgentDrift`, `deepset/prompt-injections`, `LLM-AggreFact`, `MS MARCO`), and `ChaosNLI` human-disagreement Shannon entropy $H$ correlation suite (`benchmarks/calibration_suite.jsonl`, `templates/calibration/`, receipt in `benchmarks/results_calibration_cloudrun.json`).
  - Supports **Entropy-Gated Escalation Cascades (`EXP-05`)**: pass `--cascade-from benchmarks/results_calibration_cloudrun.json --escalate-entropy 0.35` to early-exit low-entropy items ($H < 0.35\text{ nats}$) at Stage 1 (`dgemma`, `712 ms`) and escalate only high-entropy items ($H \ge 0.35\text{ nats}$, `28%` of traffic) to Vertex AI (`gemini-3.8-flash`), achieving `94.0%` overall accuracy (`benchmarks/results_calibration_cascade.json`) or `98.0%` with `--normalize-entropy --cascade-threshold 0.16` (`benchmarks/results_calibration_cascade_normalized.json`).
  - Supports **`JevBench v1.3.1` 4-Axis Parity & Slot Temperature Calibration (`EXP-11`)**: pass `--from-receipt <path> --auto-temperature` (`--temperature-scale <T>`) to compute Chance-Corrected Intelligence, 10-Bin ECE + ASCII Reliability Diagrams, Multi-Class Brier Score, Soft-Label TVD, and `USD per 1,000 decisions`.
- `dgem bench-bbox`: 12-case synthetic SVG/PNG spatial localization, `DETR` multi-object query, and per-edge occlusion suite (`EXP-09`, `benchmarks/bbox_suite.jsonl`, `fixtures/bbox/`, `templates/multimodal/bbox_*.json.tmpl`, receipts in `benchmarks/results_bbox_cloudrun.json` and `benchmarks/results_bbox_simulated.json`). Supports `--dir <path>` (loading custom images + `manifest.jsonl` or `index.txt`), `--annotate` (SVG overlays), and `scratch/render_bbox_results.py` (native-aspect-ratio PNG + self-contained HTML comparison gallery in `~/projects/tmp/dgem-bounding-boxes/bbox_gallery.html`).
- `dgem bench-rerank`: 30-query, 300-passage Listwise Diffusion Canvas Reranking, Softmax Expectation ($\hat{r}_i = \sum_{g=0}^3 g \cdot p_{i,g}$), and RAG Poison Quarantine suite (`EXP-10`, `benchmarks/rerank_suite.jsonl`, `templates/rerank/listwise_decision_rerank.json.tmpl`, receipt in `benchmarks/results_rerank_cloudrun.json`). Supports `--from-receipt benchmarks/results_rerank_cloudrun.json` (`--json`) and live execution against Cloud Run L4 (`0.9265 nDCG@10`, `0.9444 MRR@10`, `0.0%` tie rate, `100%` `NevIR` negation, `+0.7533` `FollowIR p-MRR`, `100%` prompt-injection quarantine).
- `dgem bench-jev`: 231-case `JevBench v1.3.1` (`fstandhartinger/jevbench`) zero-shot classification suite across `easy` (48), `standard` (72), and `hard` (111) tiers, 18 reasoning families, and 7 subject topics (`EXP-11`, `benchmarks/jevbench/jevbench_public.jsonl`, `benchmarks/jevbench/manifest.lock.json`, `templates/jevbench_generic.json.tmpl`, receipts in `benchmarks/jevbench/results_djev_upstream_ref.json` and `benchmarks/jevbench/results_djev_upstream_calibrated.json`). Supports `--sync` (SHA-256 verified upstream sync + zero-leakage check), `--check-upstream` (drift detection against GitHub), `--from-receipt` + `--auto-temperature`, `--flip-options`, and `--multi-slot-evidence`.

## 4. Decision Model Taxonomy, Experiment Ledger & Docs Symmetry
- **Core Framing**: Always position **DiffusionGemma (`dgemma`)** as a **Zero-Shot Decision Model** (joint multi-slot readout in $O(1)$ forward passes with calibrated epistemic Shannon entropy $H$) and **`dgem` `.json.tmpl` files** as **Executable Decision Policies (`Policy-as-Template` / `Policy-as-Code`)**, including conditional policy DAGs (`depends_on` & `ask_if` in `templates/secops_conditional_dag.json.tmpl`).
- **Experiment Ledger (`docs/experiments/`)**: Whenever a new benchmark harness, dataset, or cascade experiment is added, register it with an `EXP-XX` identifier in `docs/experiments/README.md` (linking the `.json.tmpl` policy templates, `benchmarks/*.jsonl` dataset, CLI command, and `benchmarks/results_*.json` receipt).
- **Mirroring Mandate (`docs/` $\leftrightarrow$ `docs-site/`)**: Keep markdown files in `docs/` (including `docs/experiments/`, `docs/decision-models-primer.md`, `docs/templates.md`, and `docs/benchmarks-report.md`) synchronized with `docs-site/src/content/docs/`, update `docs-site/astro.config.mjs` when adding new pages, and verify static site compilation via `rtk npm run build` in `docs-site/`.




