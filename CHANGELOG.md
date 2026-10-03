# Changelog

Releases of the `dgem` serving images (`dgem`, `dgem-weights`), gateway and CLI. One version covers all of them.
Versioning: `vMAJOR.MINOR.PATCH`. **Major:** breaking API or response changes. **Minor:** vLLM runtime or model
changes, new features. **Patch:** fixes. Images are published to
`us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/` as `:<version>` and `:<commit>`; `:latest` moves only after
a release is validated. Release process: [runbook](docs/operate/runbook.md#release-a-new-version). Gateway/CLI-only releases
(`make release-gateway`) reuse the previous serving images.

## Unreleased

- **Gateway and MCP image references are restricted** to data: URIs and http(s) URLs on public hosts. `dgem serve`
  previously accepted a local path in `image`/`images`, read it from the gateway's filesystem and sent it to the model
  (and, with #70, to the Stage-2 cascade). The CLI and stdio `dgem mcp` still accept local files.

- **Docs: hesitation-gating replaces "IDC" as the product name.** New flagship page
  `docs/confidence/overview.md` ("Confidence beyond Shannon: hesitation-gated decisions"); the old IDC page moves to
  `docs/history/` (docs-only) with a site redirect. README, glossary, primer, landing page, Studio Concepts tab and
  public explainers updated; option-order numbers in the regression matrix now come from the v0.2.0 reference.

## v0.2.1 (2026-10-03)

Gateway, MCP, CLI and Studio release; serving images rebuilt for one additive diagnostics field (no behaviour change).

- **Prompt layout on every surface.** `dgem decide --layout`, gateway `/api/decide` (`layout`, `X-DGem-Layout`,
  `?layout=`), MCP `decide_policy` / `decide_custom_questions` (`layout`) and a Studio **Prompt layout** menu (also used
  by the batch runner). Empty keeps the template's layout or the server default. The serving image reports the layout it
  used in `diagnostics.layout`. New guide: `docs/policies/prompt-layout.md`.
- Regression matrix: coverage gate and refusal reasons (`context`, `capacity`, `na`, `error`) in reports and
  `summary.json` (#60).
- **Context length 8,192 tokens** by default on RTX PRO 6000 in `deploy_vertex_endpoint.sh` and
  `deploy_cloudrun_vllm.sh` (EXP-20: same short-prompt accuracy and latency as 4,096; longer prompts answered instead
  of refused). L4 stays at 4,096.
- Vertex deploys turn container logging on (`enableContainerLogging`); it was off, which hid an engine crash.

## v0.2.0 (2026-10-02)

Serving images rebuilt. **Behaviour change: the default prompt layout is now `document_first`.**

- **Prompt layout.** The server puts the state first and the questions after it, both in the user turn. Before,
  the questions were the system prompt. Requests can choose with `"layout": "schema_first" | "document_first"`, and a
  deployment can restore the old default with `DEFAULT_LAYOUT=schema_first`. `dgem systemone serve --prompt-layout`
  (default `document_first`) sets it on every upstream request. Measured (EXP-19; release gate T2, v0.2.0 vs v0.1.3):
  typed-decisions 0.674 → 0.728, XNLI 0.662 → 0.699, MASSIVE 0.820 → 0.824, CLINC150-validation in the Decision
  Index format 0.740 → 0.863 (out-of-scope recall 1.00 → 0.825), RAGTruth train 0.712 → 0.771; JevBench unchanged.
  ECE improves on typed-decisions and XNLI and is slightly worse on MASSIVE (0.082 → 0.104).
- `scripts/deploy_vertex_endpoint.sh` passes `MAX_MODEL_LEN` (default 4096).
- Regression matrix: dev suites `di_catchall` (CLINC150 validation) and `rag_dev` (RAGTruth train) in T1/T2;
  per-target request options (`--target name=<URL>#layout=schema_first`); fixed `summary.json` `baseline` on runs
  with `rag_dev`.
- Apache-2.0 headers, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`; receipts scrubbed of internal identifiers.
- `dgem systemone serve` options for two Decision Index weaknesses, **off by default** (measured on dev data; each
  helps one benchmark family and costs another, so they are opt-in):
  - `--catch-all off|final|both|verify`: handling of "none of the listed" / "out of scope" options in >26-option
    bracket tournaments. On CLINC150 validation, `final` raised macro-F1 0.734 → 0.775 and in-scope accuracy
    0.691 → 0.757, but lowered out-of-scope recall 0.96 → 0.87.
  - `--noul-mode noul|choice`: read yes/no questions as a 2-option choice with the true/false criteria as
    descriptions. On RAGTruth train (held-out confirm split), hallucination F1 0.441 → 0.752; slightly lower accuracy
    on other yes/no sets (JevBench −3.6, calibration suite −6, typed-decisions −1 points; none significant alone).

- **`dgem systemone serve` accepts non-string option descriptions.** The Decision Index suite sends objects (POP909
  chords, ChessBench moves) and arrays (cfcolor swatches) as `criteria` values; the adapter rejected them with HTTP 400
  before reaching the model. Strings pass through unchanged; other values are rendered as compact JSON text.

## v0.1.5 (2026-10-02)

Gateway, MCP and CLI only. **Serving images are unchanged**: still `v0.1.3` (`dgem@sha256:ceb17887…`).

- **`dgem systemone serve` wide-option fixes (Decision Index adapter):**
  - Brackets are now balanced. With fixed chunks of 20, K % 20 == 1 (41, 61, 101 … options) left a 1-option
    bracket that the server rejected; the adapter returned HTTP 500, which the Decision Index kit retries as an error.
  - Probabilities for >26 options are fused from the model's own readouts. The final round weights the brackets and
    Round-1 shares the mass inside each one. Previously a fixed 92% / 8% finalist split capped every wide-option
    confidence at about 0.92, which set ECE/Brier on BANKING77, CLINC150, API-Bank and similar benchmarks.
  - `--bracket-size` is now passed to the engine (it was only reported in `/health`).
  - Validated on production with real labels (59-option MASSIVE): accuracy unchanged; AUROC 0.826 → 0.851, Brier
    0.248 → 0.240; 65% of items now reach ≥ 0.95 confidence, 97.9% correct there.
- **Regression matrix:**
  - `summary.json` is a self-contained, schema-checked result (`benchmarks/matrix/summary.schema.json`,
    `dgem.matrix.summary/v2`) with per-run coverage, macro-F1, ECE, Brier, NLL, AUROC, held-out calibration and
    reliability bins (#47).
  - Decision Index adapter track (#48): `di_probes` (awkward option counts, complete probabilities, batching,
    422 refusals with kit markers), `di_wide` (59-option MASSIVE through bracket routing), and `di_kit_compat` (the
    kit's compatibility pass, when configured).
  - `scripts/matrix_trends.py` turns scheduled runs into a time series per suite and gate (#49).

## v0.1.4 (2026-10-02)

Gateway, MCP and CLI only. **Serving images are unchanged**: the serving image is still `v0.1.3`
(`dgem@sha256:ceb17887…`), and production Vertex and Cloud Run keep running it.

- **Regression matrix (#45):** `scripts/bench_matrix.py` runs a versioned set of benchmarks (tiers T0 smoke, T1
  gate, T2 full with `--confirm`) against one or more serving targets (Vertex, Cloud Run, or a self-hosted
  `http://<GPU_HOST>:8080`). It judges accuracy against the noise floor it measures in the same session, and writes
  `benchmarks/runs/<run>/` with redacted receipts, `report.md` and `summary.json`. Datasets are pinned by Hugging Face
  commit and SHA-256 (`benchmarks/matrix/datasets.lock.json`); v0.1.3 reference ranges are in
  `benchmarks/matrix/matrix_v1.json`. Guide: `docs/operate/regression-matrix.md`. The evaluate-on-your-own-GPU guide,
  the runbook and the release process use it.
- Scheduled matrix runs (`scripts/deploy_bench_matrix_job.sh`): T0 daily and T1 weekly as Cloud Run jobs, with
  reports in Cloud Storage and verdicts in Cloud Logging.
- **Alerts (#46):** `scripts/setup_alerts.py` adds a scheduled-matrix FAIL alert and missing-run alerts for T0 (25 h)
  and T1 (7 d + 8 h).
- **CLI (#43):** `bench-permutation`, `bench-decision-index` and `bench-rerank` no longer write to committed
  reference receipts by default; pass `-o/--output`. `--out` remains as a deprecated alias.
- `scripts/receipt_agreement.py` reads every receipt format.

## v0.1.3 (2026-10-01)

Serving images rebuilt (structured server fix). Gateway, MCP and CLI from the same commit.

- **Fix (#42):** `/v1/systemone` escaped every non-ASCII character in a JSON-object `state` to `\uXXXX`
  (`json.dumps` defaults, inherited from vLLM's example), so Russian, Thai or Hindi input reached the model as hex
  codes. Now `ensure_ascii=False`; ASCII-only states produce byte-identical prompts. MASSIVE (51 languages, 20 options)
  macro accuracy 0.444 → 0.821 on Vertex G4. `chat/completions` (`dgem decide`, bench commands), the gateway's default
  systemone adapter and string states were not affected.
- `/v1/systemone` noul answers include `probabilities` (`true`/`false`) and `confidence`, like choice and score.
- Decision responses send `Server-Timing: decide;dur=<ms>` and `X-Inference-Time-Ms`.
- `scripts/contract_diff.py`: six multilingual systemone cases with known answers, reported as `WRONG` on any target.
- `bench-intents`: more than 26 candidate intents run as a 2-stage bracket (groups of ≤ 20, top 5 to a final round)
  instead of being rejected by the server (`--dataset banking77` sends 30).
- Docs: self-hosted evaluation guide; EXP-18 rerun on v0.1.0, throughput and cost per 1k; CI check for docs/ and
  docs-site/ drift.

## v0.1.2 (2026-09-28)

Gateway and monitoring; serving images unchanged (v0.1.0).

- Monitoring: scheduled health check (`scripts/probe.py`, `scripts/deploy_probe.sh`: Cloud Run jobs + Scheduler),
  alert policies and log-based metrics (`scripts/setup_alerts.py`), guide `docs/operate/monitoring.md`.
- Gateway decision logs record the requested backend (`dgem_backend_requested`) next to the one that answered, and
  failed requests are logged with backend `none` instead of `cloudrun`, so failover can be measured.
- `scripts/template_sweep.py`: run every policy template through a gateway with its sample variables.

## v0.1.1 (2026-09-28)

Gateway, MCP and CLI only. **Serving images are unchanged**: the latest serving image is still `v0.1.0`
(`dgem@sha256:5fa4a866…`), and production Vertex and Cloud Run keep running it.

- Security (#18): `POST /api/backend-config` and `/api/vertex/deploy|teardown` require `--enable-admin-api`
  (off by default); request-supplied `vertex_url` limited to the configured endpoint or
  `--allowed-vertex-endpoints`. Previously any signed-in user could change routing for everyone or deploy/tear down
  the Vertex endpoint.
- Backend allow-list (#18): `--backends` / `DGEM_BACKENDS`, validated at startup; unknown or disabled backends
  return 400 on every surface. Studio backend choice is per browser.
- MCP: `dgem mcp` over stdio honours `DGEM_VERTEX_URL` for the allow-list (#19), health and routing (#22); tool
  schemas list only the configured backends (#23).

## v0.1.0 (2026-09-28)

First versioned release (tag on `b8e1af1`; images were built from `f241b77`, which has the same `deploy/` tree
and differs only in docs and MCP help text). Serving is built on upstream vLLM nightly `a9eafde` (PRs #57250, #58216) plus
`deploy/cloudrun/server/dgem.patch`.

- Serving: upstream vLLM base (replaces the fork base), `/health` readiness and version fields, HTTP/1.1
  keep-alive with backlog 256 (fixes Vertex 503s at 16+ concurrent requests), `MAX_INFLIGHT` queueing,
  `DEFAULT_SAMPLES=1`, digit-string `samples`, `/predict` and `/rawPredict` aliases, atomic warm-up state,
  baked weights used when present.
- Validated side by side with the previous production image: API contract identical, JevBench and calibration
  within run-to-run noise with slightly better Brier, 6–10% faster GPU time, 0 errors at 32 concurrent clients
  ([image parity run](benchmarks/runs/20260927-image-parity/README.md)).
- Gateway and MCP: derived `available_backends`, port 8090, Gemini-safe MCP tool schemas, shared decide summary
  fields, `--vertex-url` needs only the project number.
- Images carry OCI version labels; `/health` (serving and gateway) and `dgem --version` report the version.
