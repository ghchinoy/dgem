---
title: Experiment Authoring Guide & Backend Target Selection
description: Step-by-step guide for designing .json.tmpl decision policies, selecting between Vertex AI Dedicated Endpoints (vertex_first / vertex) and Serverless Cloud Run GPU (cloudrun), and configuring the Stage 2 Gemini Cascade (gemini-3.8-flash default).
---

# Experiment Authoring Guide & Backend Target Selection

> **Target Audience**: Applied AI Engineers, Researchers & Domain Experts (`group:aaie-decision-model@google.com`)  
> **Studio & API Gateway**: run `./bin/dgem serve --port 8090` locally (`http://localhost:8090`) or deploy your own gateway (`./scripts/deploy_cloudrun_gateway.sh`). Examples below use `https://<your-dgem-gateway>` as a placeholder for your gateway URL.

This guide covers how to author declarative **Policy-as-Template (`.json.tmpl`)** decision schemas, select the optimal GPU serving backend (**`vertex_first` vs. `vertex` vs. `cloudrun`**), and configure **Stage 2 Gemini Cascades (`gemini-3.8-flash` default)** across the Web Studio, HTTP Gateway API, MCP Server, and `dgem` CLI.

---

## 1. Choosing an Inference Backend Target & Recommended Configuration (`vertex_first` vs. `vertex` vs. `cloudrun`)

`dgem` and `dgemma-gateway` (`https://<your-dgem-gateway>`) support routing any policy template, ad-hoc `/v1/systemone` query, or batch evaluation across **Vertex AI Dedicated Endpoints (`/invoke/*`)** and **Serverless Cloud Run GPU (`dgemma`)**. For a deep architectural breakdown and empirical benchmark receipts, see **[Vertex AI Dedicated Endpoints (`/invoke/*`) vs. Cloud Run GPU](vertex-ai-vs-cloudrun.md)**.

### Backend Target Decision Matrix

| Backend Mode (`backend` / `X-DGem-Backend`) | Target Infrastructure | Cold-Start / Wakeup | Warm GPU Denoise / Wall Time | Cost Profile | Recommended Use Case |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`vertex_first`** *(Recommended Default — High-Availability Hybrid)* | Primary: **Vertex AI Dedicated Endpoint (`4217256562927861760`, `g2-standard-16` `1× NVIDIA L4`, `64 GB` RAM)**<br/>Auto-Failover: **Serverless Cloud Run GPU (`dgemma`)** | **`0.0 s`** cold-start wakeup (`auto-failover` if Vertex is updating or scaled to zero) | **`~490 ms` GPU denoise** (`~536 ms` wall time for `N=4`; `~195 ms` single-pass) | Dedicated L4 (`~$1.12/hr`) while active; `$0.00/hr` Cloud Run standby | **Recommended default for Web Studio, MCP agents, and production APIs.** Routes to the warm Vertex AI Dedicated Endpoint for `0.0 s` wakeup and `~490 ms` GPU denoise, and automatically fails over to Serverless Cloud Run GPU if Vertex is ever updating or scaled to zero. |
| **`vertex`** *(Strict Production SLA / Zero-Downtime Priority)* | **Vertex AI Dedicated Endpoint (`4217256562927861760`, `/invoke/v1/*`)** (`g2-standard-16`, `1× NVIDIA L4`, `64 GB` RAM) | **`0.0 s`** (`minReplicaCount >= 1`, permanently warm) | **`~490 ms` GPU denoise** (`~536 ms` wall time) | **`~$1.12/hr`** (`1× L4` on `g2-standard-16`) until undeployed via `make vertex-teardown` | **Recommended for production pipelines, synchronous CI/CD gates, interactive agents, and shared internal platform services** where zero cold-start latency (`0.0 s`) and `64 GB` host RAM headroom (for multimodal `SigLIP` workloads) take priority over idle GPU reservation cost (`~$1.12/hr` for `1× L4`). |
| **`cloudrun`** *(Strict Scale-to-Zero / Cost-Sensitive & Ad-Hoc Batch)* | **Serverless Cloud Run GPU (`dgemma`)** (`1× NVIDIA RTX Pro 6000` `48GB` or `1× NVIDIA L4` `24GB`, `min-instances=0`) | **`6–8 min`** first-request cold start from `0 → 1` (`0.0 s` while warm) | **`~427 ms` GPU denoise** (`~459 ms` wall time once warm) | **`$0.00/hr` idle cost** (`min-instances=0`); billed per-second only during active traffic | **Recommended for episodic batch jobs, research experiments, and dev/test sandboxes** where **`$0.00/hr` idle cost** (`min-instances=0`) is the primary requirement and a `6–8 minute` first-request cold start is acceptable. |

---

### Selecting the Backend Across All 4 Surfaces

#### 1. Web Studio (`https://<your-dgem-gateway>`)
- Click the topbar **Backend Target** selector to choose between:
  - **`Vertex First (Auto)`** *(Recommended Default)*
  - **`Cloud Run GPU (Strict)`**
  - **`Vertex AI Strict (/invoke/*)`**
- The Backend Target popover displays live replica state for Vertex AI Dedicated Endpoint `4217256562927861760` (`g2-standard-16` `1× NVIDIA L4`) alongside 1-click **Provision Vertex GPU (`1× L4`)** and **Teardown Replica (`$0/hr`)** actions.

#### 2. HTTP Gateway API (`/api/decide`, `/v1/systemone`, `/v1/chat/completions`)
- Specify the backend via any of three mechanisms:
  - **HTTP Header**: `X-DGem-Backend: vertex_first | vertex | cloudrun`
  - **Query Parameter**: `?backend=vertex_first` (or `?backend=vertex` / `?backend=cloudrun`)
  - **JSON Body Field**: `"backend": "vertex_first"` (and optional `"vertex_url"` override)
- **Response Telemetry**: Every response returns the `X-DGem-Backend-Used: vertex | cloudrun` HTTP header and `"backend_used"` JSON field indicating which GPU backend executed the forward pass.

```bash
# Route to Vertex First (Auto-Failover) via /api/decide:
curl -sS "https://<your-dgem-gateway>/api/decide/support_triage" \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  -H "Content-Type: application/json" \
  -H "X-DGem-Backend: vertex_first" \
  -d '{
    "backend": "vertex_first",
    "variables": {
      "ticket": "Production database cluster unreachable after certificate rotation."
    }
  }' | jq .
```

#### 3. MCP Server (`https://<your-dgem-gateway>/mcp` or `dgem mcp`)
- Pass `"backend": "vertex_first" | "vertex" | "cloudrun"` (and optional `"vertex_url"`) in the arguments to **`decide_policy`**, **`decide_custom_questions`**, and **`locate_bounding_boxes`**:

```json
{
  "name": "decide_custom_questions",
  "arguments": {
    "backend": "vertex_first",
    "vertex_url": "4217256562927861760",
    "context": "Customer reports duplicate $500 annual renewal charge on enterprise account.",
    "questions": [
      { "id": "urgent", "type": "boolean", "question": "Does this ticket require urgent billing escalation?" },
      { "id": "team", "type": "choice", "question": "Which team owns this?", "options": ["billing", "engineering", "support"] }
    ]
  }
}
```

#### 4. CLI (`dgem`)
- Pass `--vertex-url 4217256562927861760 --gcp-auth` to point any `dgem` subcommand (`decide`, `bench`, `bench-calibration`, `bench-rerank`, `bench-bbox`, `bench-jev`) directly at the Vertex AI Dedicated Endpoint `/invoke/v1` route, or pass `-u https://<your-dgem-gateway>/v1 --gcp-auth` to route via the Cloud Run Gateway:

```bash
# Direct Vertex AI Dedicated Endpoint (/invoke/v1):
./bin/dgem decide --vertex-url 4217256562927861760 --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Charged twice on invoice #9481' --stats

# Via Cloud Run Gateway (uses vertex_first by default):
./bin/dgem decide -u "https://<your-dgem-gateway>/v1" --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Charged twice on invoice #9481' --stats
```

---

## 2. Stage 2 Gemini Cascade (`gemini-3.8-flash` Default) in Batch Eval, `/api/decide`, and MCP

When Stage 1 (`DiffusionGemma`) encounters an ambiguous input (where restricted-softmax Shannon entropy $H \ge 0.35\text{ nats}$, e.g. `ChaosNLI` human-disagreement items or complex multi-hop `ANLI-R3` contradictions), `dgem` can automatically escalate that item to a **Stage 2 Gemini Cascade (`EXP-05`)**.

> [!IMPORTANT]
> **Supported Stage 2 Gemini Models**: Always use **`gemini-3.8-flash`** (default), **`gemini-3.7-flash`**, or **`gemini-3.5-flash-lite`** via standard Vertex AI `generateContent` routes. Never reference legacy Gemini 2.x models.

| Parameter | Values / Default | Surface Support | Description |
| :--- | :--- | :--- | :--- |
| **`cascade_mode`** | `"off"` (default) \| `"entropy"` \| `"on_miss"` | Web Studio Batch Eval, `POST /api/decide`, MCP (`decide_policy`, `decide_custom_questions`) | **`"off"`**: Stage 1 `dgemma` only (`~490 ms` GPU denoise).<br/>**`"entropy"`**: Production uncertainty gate — early-exits confident decisions at Stage 1 ($H < \tau$, ~72% of traffic) and escalates only high-entropy items ($H \ge \tau$) to Stage 2 Gemini.<br/>**`"on_miss"`**: Batch Eval ground-truth audit mode — escalates items where Stage 1 disagrees with the dataset's `expected` label. |
| **`cascade_threshold`** | `0.35` *(default, in nats)* | Web Studio Batch Eval, `POST /api/decide`, MCP, `dgem bench-calibration` | Shannon entropy gate $\tau$ in nats (`0.35` raw nats, or `0.16` normalized $\tilde{H} = H / \ln|\mathcal{V}_m|$). |
| **`cascade_model`** | `"gemini-3.8-flash"` *(default)* | Web Studio Batch Eval, `POST /api/decide`, MCP, `dgem serve`, `dgem bench-calibration` | Target Vertex AI Gemini model (`"gemini-3.8-flash"`, `"gemini-3.7-flash"`, or `"gemini-3.5-flash-lite"`). Configurable globally via `--cascade-model` (`DGEM_CASCADE_MODEL`) and `--cascade-models` (`DGEM_CASCADE_MODELS`). |

### Example: Calling `/api/decide` and MCP with `cascade_mode: "entropy"`

```bash
curl -sS "https://<your-dgem-gateway>/api/decide/calibration/nli_calibration" \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  -H "Content-Type: application/json" \
  -H "X-DGem-Backend: vertex_first" \
  -d '{
    "backend": "vertex_first",
    "cascade_mode": "entropy",
    "cascade_threshold": 0.35,
    "cascade_model": "gemini-3.8-flash",
    "variables": {
      "premise": "All four categories reached 50 to 75 percent of the cap.",
      "hypothesis": "Every category met the cap."
    }
  }' | jq .
```

```json
{
  "name": "decide_policy",
  "arguments": {
    "template": "calibration/nli_calibration",
    "backend": "vertex_first",
    "cascade_mode": "entropy",
    "cascade_threshold": 0.35,
    "cascade_model": "gemini-3.8-flash",
    "variables": {
      "premise": "All four categories reached 50 to 75 percent of the cap.",
      "hypothesis": "Every category met the cap."
    }
  }
}
```

---

## 3. Designing `.json.tmpl` Policy Schemas & Prefix-Cache Optimization

1. **Keep `"schema"` Static, Put Row Variables in `"state"`**:
   - `structured_server.py` serializes `"schema"` before `"state"` so `vLLM`'s automatic prefix cache reuses **70–85% of the KV cache** across every row in a batch evaluation (`~60–195 ms` per pass).
2. **Single-Pass Readout (`reads=1`)**:
   - Omit `depends_on` and `ask_if` unless you explicitly want a 2-stage conditional policy DAG (`reads=2`). All `level 0` slots (`boolean`, `choice [A–Z]`, `score`) are resolved simultaneously in a single forward pass.
3. **Registering New Experiments (`EXP-XX`)**:
   - Whenever you add a new benchmark harness, `.jsonl` dataset, or cascade study, register it in [`docs/experiments/README.md`](experiments/README.md) with links to its `.json.tmpl` template, CLI command, and JSON receipt (`benchmarks/results_*.json`). For the full step-by-step Python batch runner (`run_dgem_dataset.py`), see the **[Custom Dataset & Experiment Cookbook](custom-dataset-guide.md)**.
