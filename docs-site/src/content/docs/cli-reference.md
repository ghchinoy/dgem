---
title: "CLI, HTTP Gateway & MCP Reference"
description: "Complete command-line flag reference for dgem (--vertex-url, dgem serve --default-backend vertex_first --cascade-model gemini-3.8-flash), HTTP Gateway proxy routes (/api/decide, /v1/systemone), and Model Context Protocol (MCP) tool parameters."
---

# `dgem` CLI, HTTP Gateway & MCP Reference

This reference documents the global `dgem` CLI flags, the `dgem serve` Decision Studio & Gateway server options, the `/v1/systemone` and `/api/decide` HTTP routes, and the Model Context Protocol (`MCP`) tool schemas.

---

## 1. Global CLI Flags (`dgem`)

All `dgem` subcommands (`decide`, `ask`, `serve`, `mcp`, `bench`, `bench-calibration`, `bench-rerank`, `bench-bbox`, `bench-intents`, `bench-ecotone`, `bench-jev`) share the following global flags and `DGEM_*` environment variables:

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| **`--vertex-url`** | `DGEM_VERTEX_URL` | **`4217256562927861760`** | Vertex AI Dedicated Endpoint ID (`4217256562927861760`) or full `/invoke/v1` URL (`https://4217256562927861760.us-central1-882920967572.prediction.vertexai.goog/v1/projects/882920967572/locations/us-central1/endpoints/4217256562927861760/invoke/v1`). When specified on CLI commands (`dgem decide --vertex-url 4217256562927861760 --gcp-auth`), `dgem` mints an OAuth2 `cloud-platform` access token (`gcloud auth print-access-token`) and routes directly to `/invoke/v1/*`. |
| **`-u, --url`** | `DGEM_URL` | `http://127.0.0.1:8080/v1` | Base URL of the upstream server (Local Apple Silicon `diffgemma`, Serverless Cloud Run `dgemma`, or `dgemma-gateway`). |
| **`-m, --model`** | `DGEM_MODEL` | `diffgemma-26b-a4b-it-q4` | Model identifier (`/model` or `/mnt/gcs/dgemma` on Vertex AI / Cloud Run). |
| **`--gcp-auth`** | `DGEM_GCP_AUTH` | `false` | Automatically mint Google Cloud authentication tokens (`gcloud auth print-access-token` for Vertex AI `/invoke/*` and Stage 2 Gemini `generateContent`; `gcloud auth print-identity-token` for Cloud Run). |
| **`-k, --token`** | `DGEM_TOKEN` | `""` | Explicit Bearer token override for secured endpoints. |
| **`-s, --stats`** | `DGEM_STATS` | `false` | Print comprehensive GPU timing, KV cache hit rate, restricted-softmax probabilities, and Shannon entropy ($H$ & $\tilde{H}$). |
| **`--timeout`** | `DGEM_TIMEOUT` | `120s` | HTTP client timeout duration. |

### CLI Examples Against Vertex AI Dedicated Endpoint (`--vertex-url`)

```bash
# 1. Single-pass decision on Vertex AI Dedicated Endpoint (0.0s cold start, ~490ms GPU denoise):
./bin/dgem decide --vertex-url 4217256562927861760 --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed twice for my annual renewal this morning!' \
  --stats

# 2. Full 30-case multi-domain benchmark on Vertex AI Dedicated Endpoint:
./bin/dgem bench --vertex-url 4217256562927861760 --gcp-auth \
  -d benchmarks/eval_dataset.jsonl \
  -o benchmarks/results_vertex_l4_invoke.json
```

---

## 2. `dgem serve` Flags (Decision Studio, HTTP Gateway & `/mcp`)

`dgem serve` starts the unified Web Studio UI, HTTP Gateway REST API, `/v1/systemone` proxy, and Streamable HTTP MCP server:

```bash
./bin/dgem serve \
  --default-backend vertex_first \
  --vertex-url 4217256562927861760 \
  --cascade-model gemini-3.8-flash \
  -u "https://dgemma-882920967572.us-central1.run.app/v1" \
  --gcp-auth \
  --port 8080
```

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| **`--default-backend`** | `DGEM_DEFAULT_BACKEND` | **`vertex_first`** | Default upstream GPU routing policy when a request does not specify `X-DGem-Backend` or `backend`:<br/>• **`vertex_first`** *(Recommended)*: Routes to the warm Vertex AI Dedicated Endpoint (`--vertex-url`) for `0.0 s` wakeup and `~490 ms` GPU denoise, and automatically fails over to Serverless Cloud Run GPU (`-u`) if Vertex is updating or scaled to zero.<br/>• **`vertex`**: Strictly pins requests to the Vertex AI Dedicated Endpoint (`/invoke/*`).<br/>• **`cloudrun`**: Strictly pins requests to Serverless Cloud Run GPU (`dgemma`). |
| **`--vertex-url`** | `DGEM_VERTEX_URL` | **`4217256562927861760`** | Target Vertex AI Dedicated Endpoint ID or `/invoke/v1` URL used by `vertex_first` and `vertex` routing modes. |
| **`--cascade-model`** | `DGEM_CASCADE_MODEL` | **`gemini-3.8-flash`** | Default Vertex AI Gemini model for Stage 2 Escalation Cascades (`gemini-3.8-flash`, `gemini-3.7-flash`, or `gemini-3.5-flash-lite`). |
| **`--cascade-models`** | `DGEM_CASCADE_MODELS` | **`gemini-3.8-flash,gemini-3.7-flash,gemini-3.5-flash-lite`** | Comma-separated list of selectable Stage 2 Vertex AI Gemini 3.x models exposed in `GET /api/backend-config` and the Web Studio. |
| **`-u, --url`** | `DGEM_URL` | `http://127.0.0.1:8080/v1` | Upstream Serverless Cloud Run GPU `/v1` URL used by `cloudrun` routing and `vertex_first` failover. |
| **`--port`** | `PORT` | `8080` | HTTP listener port. |

---

## 3. HTTP Gateway Endpoints (`/api/decide`, `/v1/systemone`, `/v1/chat/completions`)

Every inference endpoint on `dgem serve` (`https://<your-dgem-gateway>`) accepts backend selection via HTTP header `X-DGem-Backend: vertex_first | vertex | cloudrun`, query parameter `?backend=vertex_first`, or JSON body field `"backend": "vertex_first"`, and returns the **`X-DGem-Backend-Used: vertex | cloudrun`** response header.

| Route | Method | Description |
| :--- | :---: | :--- |
| **`/api/decide` & `/api/decide/{template}`** | `POST` | Renders a named or inline (`custom_template`) `.json.tmpl` policy with `variables`, executes Stage 1 `DiffusionGemma` readout on `vertex_first` / `vertex` / `cloudrun`, and optionally runs the **Stage 2 Gemini Cascade** (`cascade_mode`: `"off" \| "entropy" \| "on_miss"`, `cascade_threshold`: `0.35`, `cascade_model`: `"gemini-3.8-flash"`). |
| **`/v1/systemone`** | `POST` | Direct pass-through proxy to `structured_server.py`'s `/v1/systemone` (`SystemOne` / `JevBench` schema evaluation). Supports both `application/json` (`{"state": ..., "questions": ...}`) and `multipart/form-data` (`image` file + JSON fields), routing to `/invoke/v1/systemone` on Vertex AI or `/v1/systemone` on Cloud Run GPU. |
| **`/v1/chat/completions`** | `POST` | OpenAI-compatible structured diffusion decision envelope proxy with `vertex_first` auto-failover and automatic GCP token injection. |
| **`/v1/raw/chat/completions`** | `POST` | Direct pass-through proxy to `vLLM`'s raw `/v1/chat/completions` endpoint. |
| **`/api/vertex/status`**, **`/api/vertex/deploy`**, **`/api/vertex/teardown`** | `GET` / `POST` | Live Vertex AI Dedicated Endpoint (`4217256562927861760`) replica telemetry and 1-click provisioning/teardown (`g2-standard-16` `1× NVIDIA L4`). |

---

## 4. Model Context Protocol (`MCP`) Tool Parameters (`POST /mcp` & `dgem mcp`)

The MCP inference tools (`decide_policy`, `decide_custom_questions`, and `locate_bounding_boxes`) accept the following backend routing and Stage 2 Gemini Cascade parameters:

| MCP Argument | Type | Allowed Values / Default | Description |
| :--- | :--- | :--- | :--- |
| **`backend`** | `string` | `"vertex_first"` *(default)* \| `"vertex"` \| `"cloudrun"` | Selects the GPU execution target (`vertex_first` routes to warm Vertex AI `4217256562927861760` with automatic Cloud Run failover). |
| **`vertex_url`** | `string` | `"4217256562927861760"` *(optional)* | Custom Vertex AI Dedicated Endpoint ID or `/invoke/v1` URL override. |
| **`cascade_mode`** | `string` | `"off"` *(default)* \| `"entropy"` \| `"on_miss"` | Stage 2 Gemini Cascade trigger policy (`"entropy"` escalates when Stage 1 Shannon entropy $H \ge$ `cascade_threshold`). |
| **`cascade_threshold`** | `number` | `0.35` *(default, in nats)* | Shannon entropy threshold $\tau$ in nats for `"entropy"` escalation. |
| **`cascade_model`** | `string` | `"gemini-3.8-flash"` *(default)* | Stage 2 Vertex AI Gemini model (`"gemini-3.8-flash"`, `"gemini-3.7-flash"`, or `"gemini-3.5-flash-lite"`). |
