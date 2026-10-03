---
title: "CLI, HTTP Gateway & MCP Reference"
description: "Complete command-line flag reference for dgem (--vertex-url, dgem serve --default-backend vertex_first --cascade-model gemini-3.8-flash, dgem systemone serve), HTTP Gateway proxy routes (/api/decide, /v1/systemone), and Model Context Protocol (MCP) tool parameters."
---

# `dgem` CLI, HTTP Gateway & MCP Reference

This reference documents the global `dgem` CLI flags, the `dgem serve` Decision Studio & Gateway server options, the `dgem systemone serve` tournament adapter, the `/v1/systemone` and `/api/decide` HTTP routes, and the Model Context Protocol (`MCP`) tool schemas.

---

## 1. Global CLI Flags (`dgem`)

All `dgem` subcommands (`decide`, `ask`, `serve`, `systemone`, `mcp`, `bench`, `bench-calibration`, `bench-rerank`, `bench-bbox`, `bench-intents`, `bench-ecotone`, `bench-jev`) share the following global flags and `DGEM_*` environment variables:

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| **`--vertex-url`** | `DGEM_VERTEX_URL` | `""` | Vertex AI dedicated endpoint ID or full invoke URL. A bare ID expands to `https://<endpoint-id>.<region>-<project-number>.prediction.vertexai.goog/v1/projects/<project-number>/locations/<region>/endpoints/<endpoint-id>/invoke/v1` and needs `GCP_PROJECT_NUMBER` (region from `GCP_REGION`, default `us-central1`). When specified on CLI commands (`dgem decide --vertex-url <endpoint-id> --gcp-auth`), `dgem` mints an OAuth2 `cloud-platform` access token (`gcloud auth print-access-token`) and routes directly to `/invoke/v1/*`. |
| **`-u, --url`** | `DGEM_URL` | `http://127.0.0.1:8080/v1` | Base URL of the upstream server (Local Apple Silicon `diffgemma`, Serverless Cloud Run `dgemma`, or `dgemma-gateway`). |
| **`-m, --model`** | `DGEM_MODEL` | `diffgemma-26b-a4b-it-q4` | Model identifier (`/model` or `/mnt/gcs/dgemma` on Vertex AI / Cloud Run). |
| **`--gcp-auth`** | `DGEM_GCP_AUTH` | `false` | Automatically mint Google Cloud authentication tokens (`gcloud auth print-access-token` for Vertex AI `/invoke/*` and Stage 2 Gemini `generateContent`; `gcloud auth print-identity-token` for Cloud Run). |
| **`-k, --token`** | `DGEM_TOKEN` | `""` | Explicit Bearer token override for secured endpoints. |
| **`-s, --stats`** | `DGEM_STATS` | `false` | Print comprehensive GPU timing, KV cache hit rate, restricted-softmax probabilities, and Shannon entropy ($H$ & $\tilde{H}$). |
| **`--timeout`** | `DGEM_TIMEOUT` | `2m` | HTTP client timeout duration. |
| **`--http-retries`** | — | `0` (`serve`, `mcp`: 3) | Retry HTTP 429/503 with backoff. |

### CLI Examples Against Vertex AI Dedicated Endpoint (`--vertex-url`)

```bash
# 1. One decision on a Vertex AI dedicated endpoint (bare IDs need GCP_PROJECT_NUMBER):
./bin/dgem decide --vertex-url <endpoint-id> --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed twice for my annual renewal this morning!' \
  --stats

# 2. Full 30-case multi-domain benchmark on Vertex AI Dedicated Endpoint:
./bin/dgem bench --vertex-url <endpoint-id> --gcp-auth \
  -d benchmarks/eval_dataset.jsonl \
  -o results_vertex.json
```

---

## 2. `dgem serve` Flags (Decision Studio, HTTP Gateway & `/mcp`)

`dgem serve` starts the unified Web Studio UI, HTTP Gateway REST API, `/v1/systemone` proxy, and Streamable HTTP MCP server:

```bash
./bin/dgem serve \
  --default-backend vertex_first \
  --vertex-url <endpoint-id> \
  --cascade-model gemini-3.8-flash \
  -u "https://<CLOUD_RUN_URL>/v1" \
  --gcp-auth
```

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| **`--default-backend`** | `DGEM_DEFAULT_BACKEND` | **`vertex_first`** | Default upstream GPU routing policy when a request does not specify `X-DGem-Backend` or `backend`:<br/>• **`vertex_first`** *(Recommended)*: Routes to the warm Vertex AI Dedicated Endpoint (`--vertex-url`) and automatically falls back to Serverless Cloud Run GPU (`-u`) if Vertex is updating or scaled to zero.<br/>• **`vertex`**: Strictly pins requests to the Vertex AI Dedicated Endpoint (`/invoke/*`).<br/>• **`cloudrun`**: Strictly pins requests to Serverless Cloud Run GPU (`dgemma`). |
| **`--vertex-url`** | `DGEM_VERTEX_URL` | `""` | Target Vertex AI Dedicated Endpoint ID or `/invoke/v1` URL used by `vertex_first` and `vertex` routing modes. |
| **`--cascade-model`** | `DGEM_CASCADE_MODEL` | **`gemini-3.8-flash`** | Default Vertex AI Gemini model for Stage 2 Escalation Cascades (`gemini-3.8-flash`, `gemini-3.7-flash`, or `gemini-3.5-flash-lite`). |
| **`--cascade-models`** | `DGEM_CASCADE_MODELS` | **`gemini-3.8-flash,gemini-3.7-flash,gemini-3.5-flash-lite`** | Comma-separated list of selectable Stage 2 Vertex AI Gemini 3.x models exposed in `GET /api/backend-config` and the Web Studio. |
| **`--systemone-mode`** | `DGEM_SYSTEMONE_MODE` | `adapter` | Mode for the `/v1/systemone` route: `adapter` (bracket tournaments and multi-slot batching, default) or `passthrough` (direct proxy). |
| **`--systemone-temperature`** | `DGEM_SYSTEMONE_TEMP` | `1.0` | Post-hoc slot logit temperature scaling $T^*$ for `/v1/systemone` decisions. |
| **`-u, --url`** | `DGEM_URL` | `http://127.0.0.1:8080/v1` | Upstream Serverless Cloud Run GPU `/v1` URL used by `cloudrun` routing and `vertex_first` failover. |
| **`-p, --port`** | `PORT` | `8090` | HTTP listener port (the flag overrides `PORT`). |
| **`--local`, `--local-url`** | `DGEM_SERVE_LOCAL`, `DGEM_LOCAL_URL` | off | Serve against a local `diffgemma` engine only (backend `local`, no cloud failover). |
| **`--backends`** | `DGEM_BACKENDS` | derived | Backend allow-list (`vertex_first`, `vertex`, `cloudrun`, `local`); validated at startup. |
| **`--enable-admin-api`** | `DGEM_ADMIN_API` | off | Allow `POST /api/backend-config` and `/api/vertex/deploy`, `/api/vertex/teardown`. |
| **`--allowed-vertex-endpoints`** | `DGEM_ALLOWED_VERTEX_ENDPOINTS` | none | Extra Vertex endpoints selectable per request; others are rejected. |
| **`--wakeup-timeout`** | — | `10m` | How long `cloudrun` requests are held and retried while the service wakes from zero. |

---

## 3. `dgem systemone serve` (Decision Index Adapter)

`dgem systemone serve` runs a standalone HTTP adapter that implements `POST /v1/systemone` (the protocol used by `apolinario/decision-index`). It splits choices with more than 26 options into 2-stage bracket tournaments and batches more than 8 questions across forward passes, sending sub-requests to an upstream DiffusionGemma server over `/v1/chat/completions`.

```bash
# In front of a local structured server / vLLM:
dgem systemone serve --port 8080 --upstream http://127.0.0.1:8081/v1

# In front of a Vertex AI Dedicated Endpoint (a GCP access token is added automatically):
dgem systemone serve --port 8095 \
  --upstream "https://<endpoint-id>.<region>-<project-number>.prediction.vertexai.goog/v1/projects/<project-id>/locations/<region>/endpoints/<endpoint-id>/invoke/v1"

# Require a Bearer key on incoming requests:
dgem systemone serve --port 8080 --api-key "my-secret-key"
```

The serving container can run the adapter itself: set `ROLE=decision-index` (optionally `TEMPERATURE`, `CANVAS`)
and the container serves the adapter on its port, in front of its own structured server.

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| **`-p, --port`** | `PORT`, then `SYSTEMONE_PORT` | `8080` | Port to listen on. |
| **`--host`** | — | `0.0.0.0` | Interface to bind. |
| **`--upstream`** | `SYSTEMONE_UPSTREAM_URL`, then `UPSTREAM_DGEMMA_URL` | `-u` / `--url` if set, else `http://127.0.0.1:8081/v1` | Upstream `/v1` base URL (local server, Cloud Run, or a Vertex `.../invoke/v1` URL). |
| **`--temperature`** | — | `1.0` | Post-hoc slot temperature scaling $T^*$ (`1.0` = unscaled). |
| **`--max-slots`** | — | `8` | Maximum questions per forward pass before batching. |
| **`--bracket-size`** | — | `20` | Maximum options per round-1 tournament bracket. |
| **`--api-key`** | `SYSTEMONE_API_KEY`, then `API_KEY` | `""` (open) | If set, requests must send `Authorization: Bearer <key>`. |
| **`--null-prior-debias`** | — | `false` | Divide out the positional option-`A` prior (validate on your data first). |
| **`--prior-alpha`** | — | `0.50` | Exponent for null-prior de-biasing. |
| **`--dual-mirror`** | — | `false` | Also read a reversed option ordering (research diagnostic). |
| **`--naive-limits`** | — | `false` | Reproduce the naive 26-option / 10-question rejections (benchmark ablation only). |
| **`--prompt-layout`** | — | `document_first` | How the upstream prompt is laid out: `document_first` (the state, then the questions, both in the user turn) or `schema_first` (the questions as the system prompt, the layout before v0.2.0). Serving images from v0.2.0 default to `document_first` themselves (`DEFAULT_LAYOUT`); older ones ignore the field. On dev suites `document_first` raised CLINC150-validation accuracy 0.79 → 0.87 and RAGTruth-train accuracy 0.705 → 0.77, with JevBench unchanged; out-of-scope recall fell 1.00 → 0.825. |
| **`--catch-all`** | — | `off` | Wide-option catch-all handling (`off`, `final`, `both`, `verify`). `final` keeps "none of the listed"-style options out of round-1 brackets and adds them to the final. Opt-in: it trades out-of-scope recall for in-scope accuracy. |
| **`--noul-mode`** | — | `noul` | How yes/no questions are read: `noul`, or `choice` (a 2-option yes/no choice with the true/false criteria as descriptions). Opt-in: it helps hallucination-style questions and costs a little on other yes/no families. |

---

## 4. HTTP Gateway Endpoints (`/api/decide`, `/v1/systemone`, `/v1/chat/completions`)

`/api/decide` also accepts a prompt layout as JSON `"layout": "document_first" | "schema_first"`, header `X-DGem-Layout` or `?layout=`; the server reports the layout it used in `diagnostics.layout` ([Prompt layout](../policies/prompt-layout.md)). `/v1/systemone` and `/v1/chat/completions` pass a `layout` key in the body or schema through unchanged.

Every inference endpoint on `dgem serve` (`https://<your-dgem-gateway>`) accepts backend selection via HTTP header `X-DGem-Backend: vertex_first | vertex | cloudrun | local`, query parameter `?backend=vertex_first`, or JSON body field `"backend": "vertex_first"`, and returns the **`X-DGem-Backend-Used: vertex | cloudrun | local`** response header (`"backend_target"` in JSON bodies).

| Route | Method | Description |
| :--- | :---: | :--- |
| **`/api/decide` & `/api/decide/{template}`** | `POST` | Renders a named or inline (`custom_template`) `.json.tmpl` policy with `variables`, executes Stage 1 `DiffusionGemma` readout on `vertex_first` / `vertex` / `cloudrun`, and optionally runs the **Stage 2 Gemini Cascade** (`cascade_mode`: `"off" \| "entropy" \| "on_miss"`, `cascade_threshold`: `0.35`, `cascade_model`: `"gemini-3.8-flash"`). |
| **`/v1/systemone`** | `POST` | Direct pass-through proxy to `structured_server.py`'s `/v1/systemone` (`SystemOne` / `JevBench` schema evaluation). Supports both `application/json` (`{"state": ..., "questions": ...}`) and `multipart/form-data` (`image` file + JSON fields), routing to `/invoke/v1/systemone` on Vertex AI or `/v1/systemone` on Cloud Run GPU. |
| **`/v1/chat/completions`** | `POST` | OpenAI-compatible structured diffusion decision envelope proxy with `vertex_first` auto-failover and automatic GCP token injection. |
| **`/v1/raw/chat/completions`** | `POST` | Direct pass-through proxy to `vLLM`'s raw `/v1/chat/completions` endpoint. |
| **`/api/backend-config`** (live `vertex_status`, `available_backends`, `admin_api`), **`/api/vertex/deploy`**, **`/api/vertex/teardown`** | `GET` / `POST` | Live Vertex endpoint status (`GET`, always allowed). `POST` changes gateway-wide settings or deploys/tears down the endpoint and requires `--enable-admin-api` (otherwise `403`). |

---

## 5. Model Context Protocol (`MCP`) Tool Parameters (`POST /mcp` & `dgem mcp`)

The MCP inference tools (`decide_policy`, `decide_custom_questions`, and `locate_bounding_boxes`) accept the following backend routing parameters. The two decide tools also accept the Stage 2 Gemini Cascade and taxonomy expansion parameters:

| MCP Argument | Type | Allowed Values / Default | Description |
| :--- | :--- | :--- | :--- |
| **`backend`** | `string` | `"vertex_first"` *(default)* \| `"vertex"` \| `"cloudrun"` \| `"local"` | Selects the execution target (`vertex_first` routes to warm Vertex AI Dedicated Endpoint with automatic Cloud Run failover; `local` is a `diffgemma` engine on your machine and is the default under `dgem mcp --local`). Only configured backends are accepted; any other value returns `backend "…" is not enabled` with the available list. The tool schema lists exactly the backends this server has (`enum`) and names its default, so agents see the right choices. Omit it to use the default. `locate_bounding_boxes` does not support `local`. |
| **`vertex_url`** | `string` | `""` *(optional)* | Vertex AI Dedicated Endpoint ID or `/invoke/v1` URL override. Must be the configured endpoint or one listed in `--allowed-vertex-endpoints` on `dgem serve`; any other returns `vertex_url … is not allowed`. |
| **`layout`** | `string` | `""` *(template or server default)* \| `"document_first"` \| `"schema_first"` | Decide tools only. Prompt layout; see [Prompt layout](../policies/prompt-layout.md). |
| **`cascade_mode`** | `string` | `"off"` *(default)* \| `"entropy"` \| `"on_miss"` | Stage 2 Gemini Cascade trigger policy (`"entropy"` escalates when Stage 1 Shannon entropy $H \ge$ `cascade_threshold`; `"on_miss"` escalates slots that disagree with `expected_answers`). |
| **`cascade_threshold`** | `number` | `0.35` *(default, in nats)* | Shannon entropy threshold $\tau$ in nats for `"entropy"` escalation. |
| **`cascade_model`** | `string` | `"gemini-3.8-flash"` *(default)* | Stage 2 Vertex AI Gemini model (`"gemini-3.8-flash"`, `"gemini-3.7-flash"`, or `"gemini-3.5-flash-lite"`). |
| **`expected_answers`** | `object` | `{}` *(optional)* | Map of slot id to expected value, used by `cascade_mode: "on_miss"`. |
| **`suggest_expansions`** | `boolean` | `false` | Adds an `other_unclassified` option to `choice` slots and proposes new options when a slot is unclassified or hesitant ([taxonomy discovery](../policies/taxonomy-discovery.md)). |
| **`expansion_entropy`** | `number` | `0.35` *(nats)* | Entropy threshold on `choice` slots that triggers an expansion proposal. |

The response fields of the decide tools are described in [Studio, MCP and HTTP API](studio-mcp-api.md#33-reading-decide-tool-results).

The backend allow-list and endpoint restrictions (`--backends` / `DGEM_BACKENDS`, `--allowed-vertex-endpoints` /
`DGEM_ALLOWED_VERTEX_ENDPOINTS`, `--enable-admin-api`) are settings of `dgem serve`, so they apply to a gateway's
`POST /mcp` and to `dgem mcp --remote`. Stdio `dgem mcp` runs with your own credentials and offers every backend it
has an endpoint for: Vertex from `--vertex-url` or `DGEM_VERTEX_URL` (a bare endpoint ID also needs
`GCP_PROJECT_NUMBER`; a full `/invoke` URL doesn't), Cloud Run from `-u` or `DGEM_REMOTE_URL`, and `local` with
`--local`.

---

## 6. Environment Variables Reference

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| **`DGEM_VERTEX_URL`** | `""` | Target Vertex AI Dedicated Endpoint ID or `/invoke/*` URL. |
| **`DGEM_VERTEX_ENDPOINT_ID`** | `""` | Dedicated Endpoint ID override. |
| **`DGEM_VERTEX_MODEL_ID`** | `""` | Target Vertex AI Model ID deployed by `/api/vertex/deploy`. |
| **`DGEM_VERTEX_SA`** | `""` | Dedicated Service Account for Vertex AI model deployment. |
| **`DGEM_DEFAULT_BACKEND`** | `vertex_first` | Default backend selector (`vertex_first`, `vertex`, `cloudrun`, `local`). |
| **`DGEM_CASCADE_MODEL`** | `gemini-3.8-flash` | Default Gemini model for Stage 2 escalation. |
| **`DGEM_CASCADE_MODELS`** | `gemini-3.8-flash,gemini-3.7-flash,gemini-3.5-flash-lite` | List of enabled Gemini cascade models. |
| **`DGEM_GATEWAY_HOSTS`** | `""` | Comma-separated extra hostnames recognized as remote GCP endpoints for token injection. |
| **`DGEM_REMOTE_URL`** | `""` | Upstream `/v1` URL used by plain `dgem mcp` (no `--local` / `--remote`); overrides `-u`. |
| **`DGEM_MCP_LOCAL`** | `""` | Set to `1` or `true` to behave like `dgem mcp --local`. |
| **`DGEM_BACKENDS`**, **`DGEM_ALLOWED_VERTEX_ENDPOINTS`**, **`DGEM_ADMIN_API`** | `""` | `dgem serve` only: backend allow-list, extra selectable Vertex endpoints, and admin API ([Gateway and routing](../deploy/gateway.md#restrict-backends-and-admin-actions)). Not read by stdio `dgem mcp`. |
| **`GCP_PROJECT`** | `""` | Google Cloud project ID (falls back to GCP metadata server). |
| **`GCP_PROJECT_NUMBER`** | `""` | Google Cloud numeric project ID (detected automatically if on GCP). |
| **`GCP_REGION`** | `us-central1` | Default Google Cloud region for services and endpoints. |

---

## 7. Commands

### `dgem decide`
Executes single-pass discrete diffusion slot readout (or multi-stage conditional DAG execution when a template declares `depends_on` and `ask_if`).

```bash
dgem decide [flags]
```

#### Flags
* `-t`, `--template string`: Path to a Go policy template file (`.json.tmpl`).
* `-v`, `--var stringArray`: Template variables in `key=value` format (can be specified multiple times).
* `-d`, `--data string`: Path to a JSON file containing variables.
* `-f`, `--format string`: Output format: `table` (default) or `json`.
* `-I`, `--image stringArray`: Attach local image file path or remote image URL (can be specified multiple times for video frame sequences).
* `--layout document_first|schema_first`: Prompt layout for this decision. Empty uses the template's `layout`, or else the server default (`document_first` from serving v0.2.0). See [Prompt layout](../policies/prompt-layout.md).
* `--suggest-expansions`: Dynamically inject an `other_unclassified` catch-all option into `choice` slots (if absent) and propose new `{"name", "description"}` options when an item resolves to `other*` or exceeds `--expansion-entropy` (see [Unclassified Grouping & Taxonomy Discovery](../policies/taxonomy-discovery.md)).
* `--expansion-entropy float`: Shannon entropy threshold in nats on `choice` slots to trigger taxonomy expansion proposals (default: `0.35`).
* `--dual-mirror`: Evaluate forward and reversed option orderings simultaneously on the same $O(1)$ diffusion canvas (`EXP-13C`).
* `--null-prior-debias`: Divide out content-free positional `'A'`-bias in logit space (`EXP-13B`).
* `--schema string`: Path to a raw JSON schema file (skips template engine).
* `--state string`: Raw JSON state string or file path.

#### Example 1: Multi-question triage against a Cloud Run service
```bash
./bin/dgem decide \
  -u "https://<CLOUD_RUN_URL>/v1" \
  --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Emergency: production database cluster down!' \
  --stats
```

```text
QUESTION         | TYPE       | VALUE / CHOICE       | CONFIDENCE | STDERR     | AGREEMENT 
----------------------------------------------------------------------------------------
sentiment        | score      | furious              | 99.8%      | ±0.0000    | 1.00      
team             | choice     | engineering          | 99.9%      | ±0.0000    | 1.00      
urgent           | boolean    | yes                  | 100.0%     | ±0.0000    | 1.00      
```

#### Example 2: Conditional questions (`depends_on` / `ask_if`)
Questions with dependencies run in stages; a question whose `ask_if` condition is not met is skipped.

```bash
./bin/dgem decide -t templates/secops_conditional_dag.json.tmpl \
  -v 'alert_payload=Unusual IAM key creation followed by snapshot sharing to an external account.'
```

#### Example 3: Multimodal Visual Inspection (`--image` / `-I`)
```bash
./bin/dgem decide -t templates/multimodal/ui_design_review.json.tmpl \
  -I fixtures/ui_component.svg \
  -v 'component=CheckoutCard' \
  --stats
```


### `dgem bench-calibration`
Runs the **50-case Public Dataset Calibration, Guardrail & Entropy-Gated Cascade Suite (`EXP-04` & `EXP-05b`)** across 11 public datasets (`ChaosNLI`, `ANLI-R3`, `AgentDrift`, `deepset/prompt-injections`, `LLM-AggreFact`, `MS MARCO`, `Banking77`, `CLINC150`, `GoEmotions`, `CivilComments`, `Financial-PhraseBank`).

```bash
dgem bench-calibration [flags]
```

#### Key Flags
* `-c`, `--corpus string`: Path to calibration JSONL dataset (default: `benchmarks/calibration_suite.jsonl`).
* `-w`, `--workers int`: Concurrent request workers (default: `4`).
* `-o`, `--output string`: Path to write detailed JSON telemetry receipt (including `entropy`, `vocab_cardinality`, `normalized_entropy`, and `top_probabilities`).
* `--normalize-entropy`: Gate escalation using **Cardinality-Normalized Epistemic Entropy** ($\tilde{H}_m = H_m / \ln|\mathcal{V}_m| \in [0, 1]$) instead of raw Shannon entropy $H_m$ in nats.
* `--cascade`: Enable Stage-2 escalation for items whose entropy meets or exceeds `--cascade-threshold`.
* `--cascade-from string`: Hydrate Stage-1 results offline from an existing JSON receipt (e.g., `benchmarks/results_calibration_cloudrun.json`) and execute Stage-2 escalation only on items exceeding the threshold.
* `--cascade-threshold float`: Entropy threshold for escalation (default: `0.35` nats raw, or `0.16` when paired with `--normalize-entropy`).
* `--cascade-model string`: Vertex AI model for Cross-Model Stage-2 escalation (default: `gemini-3.8-flash`).
* `--cascade-self-think int`: **Intra-Model Self-Cascade**: instead of calling an external model, re-invoke the **same `DiffusionGemma` endpoint** with `"think": <tokens>` (e.g., `256`) and the Pass-1 prior distribution block when $\tilde{H}_m \ge \tau$.

#### Examples
```bash
# 1. Run the 50-case calibration suite
./bin/dgem bench-calibration \
  -u "<CLOUD_RUN_URL>/v1" --gcp-auth -w 4 \
  -o benchmarks/results_calibration_cloudrun.json

# 2. EXP-05b: escalate hesitant items from an existing receipt to Gemini
./bin/dgem bench-calibration \
  --cascade-from benchmarks/results_calibration_cloudrun.json \
  --normalize-entropy \
  --cascade-threshold 0.16 \
  --cascade-project <PROJECT> \
  -o benchmarks/results_calibration_cascade_normalized.json

# 3. Self-cascade: re-ask hesitant items on the same model with a reasoning trace
./bin/dgem bench-calibration \
  -u "<CLOUD_RUN_URL>/v1" --gcp-auth \
  --normalize-entropy \
  --cascade-threshold 0.16 \
  --cascade-self-think 256 \
  -o benchmarks/results_calibration_self_cascade.json
```


### `dgem bench`
Runs the **30-case Multi-Domain Decision Suite (`EXP-01`)** evaluating joint 3-slot decisions across `support_triage` (10 cases), `code_review` (10 cases), and `security_incident` (10 cases) (`benchmarks/eval_dataset.jsonl`).

```bash
dgem bench [flags]
```

#### Key Flags
* `-d`, `--dataset string`: Path to evaluation dataset (default: `benchmarks/eval_dataset.jsonl`).
* `-w`, `--workers int`: Number of parallel evaluation workers (default: `1`, use `4` on Cloud Run GPU).
* `-M, --mode both`: also run full autoregressive text generation for comparison (much slower).
* `-o`, `--output string`: Save structured JSON receipt (e.g., `benchmarks/results_cloudrun.json`).

```bash
./bin/dgem bench -u "<CLOUD_RUN_URL>/v1" --gcp-auth -w 4 -o benchmarks/results_cloudrun.json
```


### `dgem bench-intents`
Runs high-cardinality intent classification and Out-of-Scope (`oos`) detection (`EXP-03` & `EXP-08`) on **`PolyAI/banking77`** (3,080 test items) and **`DeepPavlov/clinc150`** (5,500 test items).

```bash
dgem bench-intents [flags]
```

#### Key Flags
* `--dataset string`: Target benchmark (`banking77`, `clinc150`, or `both`).
* `--full`: Download and evaluate the complete upstream Hugging Face test splits (`3,080` / `5,500` items) instead of the 30-item curated smoke subsets.
* `-w`, `--workers int`: Concurrent worker pool size (default: `4`, recommended: `16` on Cloud GPU).
* `-o`, `--output string`: Output JSON receipt path.

```bash
./bin/dgem bench-intents --dataset banking77 --full --workers 16 \
  -u "<CLOUD_RUN_URL>/v1" --gcp-auth \
  -o benchmarks/results_intents_banking77_full.json
```


### `dgem bench-ecotone`
Runs the 49-case Text Normalization evaluation (`EXP-02` & `EXP-07`) comparing DiffusionGemma slot readout against the C++ `Ecotone` Sparrowhawk/NeMo WFST sidecar (`unix:///tmp/ecotone.sock`) across `tn_semiotics.jsonl` (30 context-dependent polysemy traps) and `tn_challenge_en.jsonl` (19 deterministic NSWs).

```bash
./bin/dgem bench-ecotone -c benchmarks/ecotone/tn_semiotics.jsonl -o benchmarks/results_ecotone_semiotics.json
```


### `dgem bench-bbox`
Runs the **single-pass bounding-box suite (`EXP-09`)** (`benchmarks/bbox_suite.jsonl`, `fixtures/bbox/`) or any custom image directory (`--dir`). For each image it reads `[ymin, xmin, ymax, xmax]` in one pass and reports:
- argmax and softmax-expectation boxes, with mIoU and Acc@0.5/0.75 and case-level bootstrap 95% intervals;
- image-free baselines (a fixed centre box, the leave-one-out mean box and, with `--variants blank`, the model's own box on a blank image) and the model's paired lift over each;
- the edge error split into rounding (quantization) and localization, plus signed bias per edge;
- whether per-edge entropy predicts wrong edges (AUROC);
- paired occlusion deltas and run-to-run stability.

A slot the model does not return counts as a failure. See [EXP-09](../experiments/exp-09-spatial-grounding.md).

```bash
dgem bench-bbox [flags]
```

#### Key Flags
* `-d`, `--dataset string`: Path to the bounding-box JSONL suite (default: `benchmarks/bbox_suite.jsonl`).
* `--dir string`: Custom directory containing `.png`/`.jpg`/`.webp` images (and optional `manifest.jsonl` or `index.txt`) for ad-hoc runs.
* `--target string`: Default target object description when running `--dir` without a manifest.
* `--repeat int`: Run every case N times to measure run-to-run stability (default 1).
* `--variants string`: Probe variants, comma-separated or `all`:
  * `reversed`: the coordinate options in reverse order.
  * `digits9`, `digits9_reversed`: 9-level digit labels, forward or descending.
  * `hflip`, `vflip`: the image mirrored.
  * `pad_right`, `pad_left`, `pad_bottom`: the canvas extended by 50%.
  * `blank`: a blank image of the same size.

  Image transforms report consistency with the original box mapped through the same transform.
* `-w`, `--workers int`: Concurrent requests (default 4). `--samples int`: noise draws per request (default 1).
* `--from-receipt string`: Re-analyze an existing receipt (adds baselines, intervals and the error split) without calling a server.
* `--engine dgem|gemini|predictions`: who produces the boxes. `gemini` asks a Gemini 3.x model (`--gemini-model`, default `gemini-3.8-flash`). `predictions` scores an offline JSONL of detector boxes (`--predictions <file> --pred-model <model> [--pred-threshold <score>]`), for example from `scripts/detectors/run_detectors.py`. All engines use the same scoring, baselines and variants.
* `--annotate`: Write `annotated_<name>.svg` overlays with the argmax (dashed) and expectation (solid) boxes.
* `--simulate`: Offline harness check with synthetic slot distributions (no GPU).
* `-o`, `--output string`: Output JSON receipt path.

```bash
# 1. Re-baseline: every probe variant, 3 repeats (or ./scripts/run_exp09_rebaseline.sh)
./bin/dgem bench-bbox --vertex-url <ENDPOINT_ID> --gcp-auth --variants all --repeat 3 -o receipt.json

# 2. Run a custom directory of images + index.txt prompts
./bin/dgem bench-bbox -u "<CLOUD_RUN_URL>/v1" --gcp-auth \
  --dir ./tmp/dgem-bounding-boxes --annotate \
  -o ./tmp/dgem-bounding-boxes/results_cloudrun.json
```


### `dgem bench-jev`
Runs the 231-item JevBench suite ([EXP-11](../experiments/exp-11-jevbench-parity.md)): `--scoring v1.3.1|v1.4|both`,
`--compare-leaderboard`, `--from-receipt <file> --auto-temperature` (offline temperature calibration),
`--sync --ref <tag>` / `--check-upstream` (dataset provenance).

```bash
./bin/dgem bench-jev --vertex-url <ENDPOINT_ID> --gcp-auth --http-retries 3 -w 4 -o jevbench.json
```

### `dgem bench-bbox-judge`
Validates a Gemini 3.x model as a **visual judge** of bounding boxes ([EXP-22](../experiments/exp-22-image-readouts.md)). It draws each proposed box on the image and asks the judge to grade every edge (`correct`, `too_far_in` or `too_far_out`) and the box overall. Verdicts are scored against ground truth:
- `--calibrate`: ground-truth boxes with one edge shifted in or out by known amounts (`--shifts 5,10,20`), plus the exact boxes.
- `--receipt <bench-bbox receipt>`: the boxes a localizer actually produced.

It reports three-class edge accuracy and κ, recall and specificity for wrong edges (`--tol`, `--off`), how often each error size is flagged (the judge's resolution), and box-level agreement. `--rescore <judge receipt>` recomputes the summaries without new Gemini calls.

```bash
./bin/dgem bench-bbox-judge --calibrate --receipt receipt.json --judge-model gemini-3.8-flash -o judge.json
```

### `dgem bench-vision`
Scores **multi-aspect image decisions** ([EXP-22](../experiments/exp-22-image-readouts.md)). One template (default `templates/multimodal/vision_aspects.json.tmpl`) asks several questions about each image in one pass: presence, 3×3 grid cell, element count, spatial relation, image quality and occlusion. The manifest (default `benchmarks/bbox_sweep.jsonl`) carries ground-truth `aspects` per item. Per aspect it reports:
- accuracy with a bootstrap interval;
- the majority-class baseline and the paired lift over it;
- the predicted-label shares;
- hesitation→error AUROC.

`--engine gemini` puts the same questions to Gemini 3.x. `--variant blank` measures the prompt prior. `--repeat N` measures run-to-run stability. `--only-scored` asks only the questions each item has ground truth for. `--cascade-threshold <nats>` (with `--cascade-model`) runs the live Stage-2 cascade: slots at or above the threshold go to Gemini together with the image. It reports dgem-only vs cascaded accuracy, the share of requests that called Gemini, and latency.

```bash
./bin/dgem bench-vision --vertex-url <ENDPOINT_ID> --gcp-auth --repeat 2 -o vision.json
./bin/dgem bench-vision --engine gemini --gemini-model gemini-3.8-flash -o vision_gemini.json
```

### `dgem bench-rerank`
Runs the 30-query listwise reranking and RAG-poisoning suite ([EXP-10](../experiments/exp-10-listwise-diffusion-reranking.md)).

```bash
./bin/dgem bench-rerank -u "https://<CLOUD_RUN_URL>/v1" --gcp-auth -o rerank.json
```

### `dgem ask`
Executes standard generative completions with optional `<|think|>` mode.

```bash
./bin/dgem ask "Explain discrete block diffusion in two sentences."
./bin/dgem ask --think "Verify whether 2015 + 4 precedes 2018."
```


### `dgem template`
Manages and inspects Go `.json.tmpl` policy templates locally without calling a GPU endpoint.

```bash
# List available templates across templates/ and templates/calibration/
./bin/dgem template list

# Render and inspect compiled JSON schema locally
./bin/dgem template render -t templates/calibration/chaos_nli.json.tmpl \
  -v premise="All four categories reached 50 to 75 percent of the cap." \
  -v hypothesis="Every category met the cap."
```
