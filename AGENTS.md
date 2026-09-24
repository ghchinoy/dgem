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

## 1. Serving Backends, Hybrid Routing (`vertex_first`) & Telemetry (`logprobs`)
- **Vertex AI Dedicated Endpoint (`/invoke/*`, Endpoint `4217256562927861760` — Primary Production Target)**:
  - Hosted on Dedicated DNS `4217256562927861760.us-central1-882920967572.prediction.vertexai.goog` using **Arbitrary Custom Routes (`/invoke/*`)**, mapping 1:1 to `structured_server.py` (`/invoke/v1/chat/completions`, `/invoke/v1/systemone`, `/invoke/v1/raw/chat/completions`, `/invoke/health`).
  - Delivers **`0.0 s` cold-start latency** and **`~490 ms` GPU denoise (`~536 ms` wall time)** with **`1.0000` top-1 agreement** (`0.0005` mean TVD) against Serverless Cloud Run (`benchmarks/results_head_to_head_vertex_vs_cloudrun.json`).
  - **Gateway & MCP `vertex_first` Auto-Failover**: `dgem serve` (`https://dgemma.aaie.cloud`) defaults to `--default-backend vertex_first` (`DGEM_DEFAULT_BACKEND`), routing requests to warm Vertex AI `4217256562927861760` first and automatically failing over to Serverless Cloud Run (`dgemma`) if Vertex is quiesced (`0` replicas) or updating. Per-request overrides are supported across all surfaces via `X-DGem-Backend: vertex_first | vertex | cloudrun`, `?backend=...`, JSON `"backend"`, or MCP `"backend"`, and every response includes `X-DGem-Backend-Used: vertex | cloudrun`.
  - **Strict Separation of Backend Readiness Telemetry (`RecordVertexReadoutLatency` vs. `MarkGPUWarm`)**: Never invoke `MarkGPUWarm()` from Vertex AI `/invoke/health` probes or Vertex `/invoke/*` completions, as doing so overwrites Cloud Run's `lastWarmTimestamp` (`180m TTL`) and falsely reports Cloud Run as warm when inspecting `GET /api/status?backend=cloudrun`. Use `RecordVertexReadoutLatency(ms)` for Vertex AI and `CheckHealthAndGPUStatusForBackend(ctx, userEmail, backend)` (`?backend=vertex_first|vertex|cloudrun`) so `.status-cluster` and MCP `get_health_and_gpu_status` report accurate, backend-specific telemetry.
  - **`dgem mcp` Automatic Pure-Go ADC & `--remote` Proxy (`cmd/root.go`, `cmd/mcp.go`)**: `fetchADCTokens()` reads `~/.config/gcloud/application_default_credentials.json` in pure Go (~80ms, zero `gcloud` PATH dependency) to mint both OAuth2 `access_token`s (for Vertex AI `/invoke/*`) and OIDC `id_token`s (for Cloud Run IAP `programmaticClients`). `dgem mcp --remote https://dgemma.aaie.cloud/mcp` bridges `stdio` JSON-RPC to the hosted Streamable HTTP MCP gateway with automatic ADC authentication and single-line NDJSON error synthesis.
  - **`POST /v1/systemone` Pass-Through Proxy**: `dgem serve` exposes `POST /v1/systemone` (supporting both `application/json` and `multipart/form-data` image uploads) and routes it via `vertex_first` to `/invoke/v1/systemone` or `/v1/systemone`.
- **Stage-2 Gemini 3.x Escalation Cascade (`cmd/cascade_gemini.go`)**:
  - **Strictly NO Gemini 2.x models** (`gemini-2.5-*` etc. are forbidden and automatically upgraded by `SanitizeCascadeModel`).
  - Supported Stage-2 models are **`gemini-3.8-flash`** *(default)*, **`gemini-3.7-flash`**, and **`gemini-3.5-flash-lite`**, configured via `--cascade-model` (`DGEM_CASCADE_MODEL`) and `--cascade-models` (`DGEM_CASCADE_MODELS`) and served dynamically via `GET / POST /api/backend-config`.
  - Available in `POST /api/decide`, MCP (`decide_policy`, `decide_custom_questions`), and Web Studio Batch Eval via `cascade_mode` (`"off" | "entropy" | "on_miss"`), `cascade_threshold` (default `0.35` nats), and `cascade_model`.
- **Unclassified Grouping & Zero-Retraining Taxonomy Discovery (`cmd/expand_taxonomy.go`, `templates/taxonomy_discovery.json.tmpl`)**:
  - `dgem decide --suggest-expansions` (`--expansion-entropy 0.35`), `POST /api/decide` (`"suggest_expansions": true` / `X-DGem-Suggest-Expansions: true`), MCP (`decide_policy`, `decide_custom_questions`), and the Web Studio (`Suggest Taxonomy Expansions` toggle + `➕ Add Option to Policy & Re-Run`) dynamically inject an `"other_unclassified"` option into `choice` slots (`< 26` options) via `InjectUnclassifiedCatchAll` and synthesize copy-pasteable `{"name", "description"}` options via `SynthesizeTaxonomyExpansions` (using DiffusionGemma's `"think"` channel `diagnostics.thought.text` with Stage-2 `gemini-3.8-flash` fallback) whenever a slot resolves to `other*` or exceeds `--expansion-entropy`.
- **4-Surface Decision Feature Parity (`CLI` $\leftrightarrow$ `HTTP Gateway + OTel` $\leftrightarrow$ `MCP` $\leftrightarrow$ `Web Studio`)**:
  - Whenever a new decision modifier or post-processing pass is added to `dgem decide` (`cmd/decide.go`), expose the corresponding fields in `GatewayDecideRequest` / `GatewayDecideResponse` (`cmd/serve.go`), `DecidePolicyToolInput` / `DecideCustomToolInput` (`cmd/mcp.go`), and the Web Studio (`studio/src/dgem-studio.ts`).
  - Any feature that can trigger an additional upstream model pass or Stage-2 call **must** wrap that execution in a dedicated OpenTelemetry child span under `dgem.gateway.decide` (e.g., `dgem.cascade.gemini`, `dgem.taxonomy.expand`) and color-code it in the Studio Gantt Waterfall so 100% of request wall time is attributed.
- **Serverless Cloud Run GPU (`structured_server.py`, Scale-to-Zero Failover & Batch Target)**:
  - Acts as a local reverse proxy in front of vLLM on port 8000, serving `POST /v1/chat/completions` and `POST /v1/systemone` on port 8080.
  - Returns structured envelope JSON `{"answers": {...}, "diagnostics": {...}}`. `pkg/client` supports envelope fallback parsing, handling polymorphic slice-based `entropy` and dictionary-based sampling policies.
  - **26-Option `[A-Z]` Slot Limit**: `structured_server.py` maps single-token `choice` slots to uppercase English letters `A`–`Z`, enforcing a maximum of **26 alternatives per `choice` question** (`HTTP 400: at most 26 alternatives`). High-cardinality taxonomies (>26 labels) must use a 26-option slice or two-stage hierarchical routing.
  - **`depends_on` / `ask_if` Boolean Format & 2-Stage Split (`reads=2`)**: In `structured_server.py`, boolean (`noul`) questions map to string choices `["yes", "no"]` (so `ask_if` must specify `["yes"]`, never JSON `[true]`). Adding `depends_on` / `ask_if` partitions questions into `level 0` and `level 1` in `schedule(qs)`, forcing **2 sequential forward passes (`reads=2`)**. To guarantee a strict **single forward pass (`reads=1`)** (e.g., for `EXP-09` bounding-box readout), omit `depends_on` / `ask_if` so all slots reside in `level 0` simultaneously and gate downstream interpretation in the client (`pkg/client` / `cmd/bench_bbox.go`).
- **Cloud GPU vLLM (`/v1/chat/completions`)**:
  - Requires installing the nightly prebuilt wheel (`wheels.vllm.ai`) matching the base commit of vLLM PR #57250 (`133b71e0be`) to avoid C++ CUDA extension ABI symbol mismatches (`gptq_marlin_repack`, `moe_wna16_marlin_gemm`).
  - Must launch with `--attention-backend TRITON_ATTN --enforce-eager --max-model-len 32768` because FlashAttention-2 rejects DiffusionGemma's mixed causal-prompt / bidirectional-canvas attention masks.
  - Requires patching `self._enable_mm_lora = False` in `Gemma4ForConditionalGeneration.__init__` (`vllm/model_executor/models/gemma4_mm.py`) so `DiffusionGemmaForConditionalGeneration` does not raise `AttributeError` during model load (automated in `scripts/startup_gce_vllm.sh` and `deploy/cloudrun/patch_vllm.py`).
  - `pkg/client` requests `logprobs: true, top_logprobs: 5` on every `Decide` call and uses `ParseStructuredContentWithLogprobs` to compute calibrated slot probability $\exp(\text{logprob})$, Shannon entropy $H = -\sum p_k \ln p_k$, and top-$k$ candidate probabilities.
- **Local Apple Silicon Metal (`diffgemma`)**: Returns native `StructuredDecisionResponse` JSON (`answers` + `diagnostics`) with restricted-softmax `probabilities`, `first_read_max_entropy`, and `reused_tokens`.

## 2. Cloud GPU Provisioning & Resource Lifecycle
- **Vertex AI Dedicated Endpoint (`4217256562927861760` — `dgemma-dedicated`, `us-central1`)**:
  - **Mandatory `g2-standard-16` (`64 GB` Host RAM) Machine Type**: `entrypoint.sh` stages `17.53 GiB` of `NVFP4` weights to `/tmp` (`tmpfs` in host RAM) while `vLLM` + `SigLIP` allocate another `~17.7 GB` (`~35.2 GB` total). Deploying on `g2-standard-8` (`32 GB` RAM) causes a silent host-RAM OOM crash after ~10 minutes. Always deploy with `--machine-type=g2-standard-16 --accelerator=type=nvidia-l4,count=1` and `--startup-probe-timeout-seconds=1800 --startup-probe-period-seconds=15 --startup-probe-failure-threshold=120` (`./scripts/deploy_vertex_endpoint.sh`).
  - **Overnight Scale-to-Zero (`HTTP 429`) vs. `AvailableReplicaCount` & Gateway Keepalive**: Even when `deployedModels` is non-empty on `4217256562927861760`, Vertex AI's internal autoscaler can scale replicas `1 → 0` (`availableReplicaCount: 0`, returning `HTTP 429 Resource Exhausted`) after idle periods. `inspectVertexEndpointState` (`cmd/serve.go`) **must** verify `dm.Status.AvailableReplicaCount > 0`, and `dgemma-gateway` must deploy with `--min-instances=1 --no-cpu-throttling` (`scripts/deploy_cloudrun_gateway.sh`) so its 4-minute `/invoke/health` keepalive loop runs 24/7.
  - **Lifecycle Management**: Managed via `GET /api/vertex/status`, `POST /api/vertex/deploy` (`1× L4`), and `POST /api/vertex/teardown` (`0` replicas = `$0.00/hr`) or in the Web Studio topbar (`Backend Target` menu). Leave `4217256562927861760` warm when requested for shared internal team usage.
- **Serverless Cloud Run (`NVIDIA L4` 24GB or `NVIDIA RTX Pro 6000` 48GB)**:
  - Container Image: Built in-repo from `deploy/cloudrun/Dockerfile` and pushed to Google Artifact Registry: `us-central1-docker.pkg.dev/$PROJECT/dgem/dgemma:latest` (`make cloudrun-build`).
  - Weights: Pre-staged in GCS: `gs://dgem-weights-$PROJECT/dgemma/` (`make cloudrun-stage`).
  - Service Name: Strictly **`dgemma`** (never `"djev"` or `"djev-dgemma"`), fronted by **`dgemma-gateway`** (`./scripts/deploy_cloudrun_gateway.sh`).
  - **Multimodal `SigLIP` Vision Workloads (`DISABLE_MM=0`) & 32Gi Page-Cache OOM**: `scripts/deploy_cloudrun_vllm.sh` defaults to `DISABLE_MM=${DISABLE_MM:-0}` so Gemma 4's `SigLIP` vision tower is enabled (`--limit-mm-per-prompt '{"image":1,"video":0}'`). When `DISABLE_MM=0` is combined with `--safetensors-load-strategy prefetch` over GCS FUSE, Linux caches the 17.53 GiB safetensors in page cache while PyTorch allocates a second 17.53 GiB buffer + `SigLIP` warmup tensors, exceeding the `32Gi` container RAM ceiling on `nvidia-l4`. For multimodal `SigLIP` workloads on Cloud Run, deploy with `CLOUDRUN_GPU_TYPE=nvidia-rtx-pro-6000 DISABLE_MM=0` (`80Gi` RAM, `48GB` VRAM).
  - **Cold-Start Readiness Pattern**: On scale-from-zero (`--min-instances=0`), `structured_server.py` on port 8080 passes `/health` probes immediately while vLLM's `EngineCore` on port 8000 takes ~6–8 minutes to stream the 17.53 GiB checkpoint over GCS FUSE (`ConnectionRefusedError(111)`). Always poll with a single `-w 1` `dgem decide` readiness probe before launching concurrent `-w 4` benchmark workers to avoid `HTTP 429` / `HTTP 500` errors.
- **Ad-Hoc GCE VMs (`g2-standard-8` 4-bit L4 or `a2-highgpu-2g` 16-bit 2× A100)**:
  - **Mandatory Immediate Teardown**: Always run `make gce-teardown` (`./scripts/teardown_gce_vllm.sh`) immediately after capturing benchmark receipts so zero unmanaged GCE VM GPU resources remain active. Never commit `.env` (contains `HF_TOKEN`).
- **Studio Embedded Asset Hygiene (`studio/dist`) & Multi-Agent Git Worktrees**:
  - `dgem serve` embeds `studio/dist` via `go:embed` (`studio/embed.go`). Whenever modifying `studio/src/*`, run `rtk npm --prefix studio run build && rtk go build -o bin/dgem . && rm -rf studio/dist/* && touch studio/dist/.gitkeep` before `git status` / `git commit` so `./bin/dgem` embeds the updated UI bundle while Vite build hashes remain untracked.
  - When multiple agents concurrently modify overlapping packages (`cmd/*.go`, `studio/src/*.ts`), isolate edits and builds in a dedicated git worktree (`git worktree add ../dgem-<feature> -b <branch>`) and `git cherry-pick` onto `main` after the peer agent completes.

## 3. Benchmark Harnesses
- `dgem bench`: 30-case multi-domain decision suite (`benchmarks/eval_dataset.jsonl` across `support`, `code_review`, `security`). Receipts stored in `benchmarks/results_local_metal_slot.json`, `benchmarks/results_gce_l4.json`, `benchmarks/results_cloudrun.json`, and `benchmarks/results_head_to_head_vertex_vs_cloudrun.json`.
- `dgem bench-ecotone`: 49-case Text Normalization comparison against the C++ `ecotone` Sparrowhawk/NeMo WFST sidecar (`unix:///tmp/ecotone.sock`) across `tn_semiotics.jsonl` (30 polysemy traps) and `tn_challenge_en.jsonl` (19 deterministic NSWs).
- `dgem bench-intents`: High-cardinality intent & Out-of-Scope (`oos`) evaluation for `PolyAI/banking77` and `DeepPavlov/clinc150`. Pass `--dataset banking77 --full --workers 16` (3,080 test items) or `--dataset clinc150 --full --workers 16` (5,500 test items) to run full upstream splits.
- `dgem bench-calibration`: 50-case public dataset calibration, guardrail (`AgentDrift`, `deepset/prompt-injections`, `LLM-AggreFact`, `MS MARCO`), and `ChaosNLI` human-disagreement Shannon entropy $H$ correlation suite (`benchmarks/calibration_suite.jsonl`, `templates/calibration/`, receipts in `benchmarks/results_calibration_cloudrun.json` and `benchmarks/results_calibration_vertex_l4.json`).
  - Supports **Entropy-Gated Escalation Cascades (`EXP-05`)**: pass `--cascade-from benchmarks/results_calibration_cloudrun.json --escalate-entropy 0.35` to early-exit low-entropy items ($H < 0.35\text{ nats}$) at Stage 1 (`dgemma`, `712 ms`) and escalate only high-entropy items ($H \ge 0.35\text{ nats}$, `28%` of traffic) to Vertex AI (`gemini-3.8-flash`), achieving `94.0%` overall accuracy (`benchmarks/results_calibration_cascade.json`) or `98.0%` with `--normalize-entropy --cascade-threshold 0.16` (`benchmarks/results_calibration_cascade_normalized.json`).
  - Supports **`JevBench v1.3.1` 4-Axis Parity & Slot Temperature Calibration (`EXP-11`)**: pass `--from-receipt <path> --auto-temperature` (`--temperature-scale <T>`) to compute Chance-Corrected Intelligence, 10-Bin ECE + ASCII Reliability Diagrams, Multi-Class Brier Score, Soft-Label TVD, and `USD per 1,000 decisions`.
- `dgem bench-bbox`: 12-case synthetic SVG/PNG spatial localization, `DETR` multi-object query, and per-edge occlusion suite (`EXP-09`, `benchmarks/bbox_suite.jsonl`, `fixtures/bbox/`, `templates/multimodal/bbox_*.json.tmpl`, receipts in `benchmarks/results_bbox_cloudrun.json` and `benchmarks/results_bbox_simulated.json`). Supports `--dir <path>` (loading custom images + `manifest.jsonl` or `index.txt`), `--annotate` (SVG overlays), and `scratch/render_bbox_results.py` (native-aspect-ratio PNG + self-contained HTML comparison gallery in `~/projects/tmp/dgem-bounding-boxes/bbox_gallery.html`).
- `dgem bench-rerank`: 30-query, 300-passage Listwise Diffusion Canvas Reranking, Softmax Expectation ($\hat{r}_i = \sum_{g=0}^3 g \cdot p_{i,g}$), and RAG Poison Quarantine suite (`EXP-10`, `benchmarks/rerank_suite.jsonl`, `templates/rerank/listwise_decision_rerank.json.tmpl`, receipts in `benchmarks/results_rerank_cloudrun.json` and `benchmarks/results_rerank_vertex_l4.json`). Supports `--from-receipt benchmarks/results_rerank_cloudrun.json` (`--json`) and live execution against Vertex AI or Cloud Run L4 (`0.9265 nDCG@10`, `0.9444 MRR@10`, `0.0%` tie rate, `100%` `NevIR` negation, `+0.7533` `FollowIR p-MRR`, `100%` prompt-injection quarantine).
- `dgem bench-jev`: 231-case `JevBench v1.3.1` (`fstandhartinger/jevbench`) zero-shot classification suite across `easy` (48), `standard` (72), and `hard` (111) tiers, 18 reasoning families, and 7 subject topics (`EXP-11`, `benchmarks/jevbench/jevbench_public.jsonl`, `benchmarks/jevbench/manifest.lock.json`, `templates/jevbench_generic.json.tmpl`, receipts in `benchmarks/jevbench/results_djev_upstream_ref.json` and `benchmarks/jevbench/results_djev_upstream_calibrated.json`). Supports `--sync` (SHA-256 verified upstream sync + zero-leakage check), `--check-upstream` (drift detection against GitHub), `--from-receipt` + `--auto-temperature`, `--flip-options`, and `--multi-slot-evidence`.

## 4. Decision Model Taxonomy, Experiment Ledger & Docs Symmetry
- **Core Framing**: Always position **DiffusionGemma (`dgemma`)** as a **Zero-Shot Decision Model** (joint multi-slot readout in $O(1)$ forward passes with calibrated epistemic Shannon entropy $H$) and **`dgem` `.json.tmpl` files** as **Executable Decision Policies (`Policy-as-Template` / `Policy-as-Code`)**, including conditional policy DAGs (`depends_on` & `ask_if` in `templates/secops_conditional_dag.json.tmpl`) and zero-retraining taxonomy discovery (`templates/taxonomy_discovery.json.tmpl`).
- **Experiment Ledger (`docs/experiments/`)**: Whenever a new benchmark harness, dataset, or cascade experiment is added, register it with an `EXP-XX` identifier in `docs/experiments/README.md` (linking the `.json.tmpl` policy templates, `benchmarks/*.jsonl` dataset, CLI command, and `benchmarks/results_*.json` receipt).
- **Mirroring Mandate (`docs/` $\leftrightarrow$ `docs-site/`)**: Keep markdown files in `docs/` (including `docs/experiments/`, `docs/decision-models-primer.md`, `docs/taxonomy-discovery.md`, `docs/templates.md`, `docs/vertex-ai-vs-cloudrun.md`, and `docs/benchmarks-report.md`) synchronized with `docs-site/src/content/docs/`, update `docs-site/astro.config.mjs` when adding new pages, and verify static site compilation via `rtk npm run build` in `docs-site/`.




