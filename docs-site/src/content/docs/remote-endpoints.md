---
title: "Connecting dgem to Remote Endpoints"
description: "How to configure dgem CLI, SDK, and MCP clients to connect to remote DiffusionGemma endpoints: Vertex AI Dedicated Endpoints, Serverless Cloud Run GPU, and Decision Studio Gateways."
---

# Connecting `dgem` to Remote Endpoints

`dgem` is designed as a lightweight client and gateway layer that connects to any OpenAI-compatible or dedicated DiffusionGemma serving instance over HTTP/HTTPS.

---

## 1. Connection Methods

### Method A: Command-Line Flags
Specify the target backend directly on any command:

```bash
# 1. Target a Vertex AI Dedicated Endpoint (/invoke/*) with GCP OAuth2 auth:
./bin/dgem decide --vertex-url <endpoint-id> --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Emergency: database connection pool exhausted' --stats

# 2. Target a Serverless Cloud Run GPU service with IAM OIDC auth:
./bin/dgem decide -u "https://dgemma-<hash>-uc.a.run.app/v1" --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Billing dispute' --stats

# 3. Target an authenticated Decision Studio Gateway:
./bin/dgem decide -u "https://<your-dgem-gateway>/v1" --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=General inquiry'
```

### Method B: Environment Variables
Configure persistent defaults using `DGEM_*` environment variables:

```bash
export DGEM_VERTEX_URL="<endpoint-id>"
export DGEM_GCP_AUTH="true"
export DGEM_TIMEOUT="120s"
export DGEM_STATS="true"

# Now commands automatically use these settings:
./bin/dgem decide -t templates/support_triage.json.tmpl -v 'ticket=Outage'
```

### Method C: Configuration File (`.dgem.yaml`)
Place `.dgem.yaml` in your project root or `~/.config/dgem/config.yaml`:

```yaml
url: "https://<your-dgem-gateway>/v1"
vertex_url: "<endpoint-id>"
gcp_auth: true
timeout: 120s
stats: true
```

---

## 2. Authentication & Security

### Automatic Google Cloud Authentication (`--gcp-auth`)
When `--gcp-auth` (or `DGEM_GCP_AUTH=1`) is enabled, `dgem` automatically mints and refreshes the appropriate Google Cloud credentials:
* **For Vertex AI Dedicated Endpoints (`/invoke/*`)**: Mints an OAuth2 `cloud-platform` access token (`gcloud auth print-access-token` or pure-Go Application Default Credentials `~/.config/gcloud/application_default_credentials.json`).
* **For Serverless Cloud Run (`*.run.app`)**: Mints an OIDC identity token (`gcloud auth print-identity-token` or GCP metadata server) targeted to the Cloud Run service URL.

### Static Bearer Token / API Key (`-k, --token`)
For self-hosted gateways or endpoints requiring an API key:

```bash
./bin/dgem decide -u "https://my-endpoint.example.com/v1" \
  -k "secret-api-key" \
  -t templates/support_triage.json.tmpl -v 'ticket=test'
```

### Cloud Run Developer Proxy
For local debugging without transmitting tokens in command history:

```bash
# Terminal 1: Proxy Cloud Run to localhost:8080
gcloud run services proxy dgemma --region us-central1 --port 8080

# Terminal 2: Connect directly to loopback proxy
./bin/dgem decide -u "http://127.0.0.1:8080/v1" \
  -t templates/support_triage.json.tmpl -v 'ticket=test'
```

---

## 3. Supported Serving Backends

| Backend | Primary Protocol | Cold-Start | Typical GPU Denoise | How to Connect |
| :--- | :--- | :---: | :---: | :--- |
| **Vertex AI Dedicated Endpoint** | `/invoke/v1/chat/completions` | **`0.0 s`** | **`57.5 ms`** (`143 ms` wall, p50) | `--vertex-url <endpoint-id> --gcp-auth` |
| **Serverless Cloud Run GPU** | `/v1/chat/completions` | `~90–120 s` (`0 → 1`) | **`107.7 ms`** (`187 ms` wall, p50) | `-u "https://...run.app/v1" --gcp-auth` |
| **Decision Studio Gateway** | `/api/decide`, `/v1/systemone` | `0.0 s` (gateway) | Delegated to backend | `-u "https://<your-gateway>/v1" --gcp-auth` |
| **Local Metal Engine** | `/v1/chat/completions` | **`0.0 s`** | **`210 ms`** (`N=1`) | `-u "http://127.0.0.1:8080/v1"` |

For instructions on deploying your own cloud endpoints, see:
* **[Deploy on Your Own Cloud GPU](/dgem/deploy-your-own-gpu/)** — Deploy using official public images to Cloud Run or Vertex AI.
* **[Vertex AI Dedicated Endpoints vs. Cloud Run GPU](/dgem/vertex-ai-vs-cloudrun/)** — Architectural and performance comparison.
* **[Path to Production](/dgem/path-to-production/)** — Recommendations for scaling, concurrency, and cost optimization.
