---
title: "Connecting dgem to Remote Endpoints"
description: "How to configure dgem CLI, SDK, and MCP clients to connect to remote DiffusionGemma endpoints: Vertex AI Dedicated Endpoints, Serverless Cloud Run GPU, and Decision Studio Gateways."
---

`dgem` is a thin client: every command works the same against your laptop, a Cloud Run GPU service, a Vertex AI
endpoint or a `dgem serve` gateway. If your team already runs an endpoint, this page is all you need; to run your
own, continue with [Deploy on Cloud Run](/dgem/deploy/cloud-run/).

## Choose a target

| Target | How to connect | Auth | Cold start | Warm latency, 1 sample (GPU / end to end) |
| :--- | :--- | :--- | :--- | :--- |
| Vertex AI dedicated endpoint | `--vertex-url <ENDPOINT_ID>` | OAuth access token (`--gcp-auth`) | None | 55 / 150 ms |
| Cloud Run GPU service | `-u https://<CLOUD_RUN_URL>/v1` | ID token (`--gcp-auth`) | ~2.5 min after idle | 61 / 153 ms |
| `dgem serve` gateway | `-u https://<your-dgem-gateway>/v1` | ID token / IAP (`--gcp-auth`) | Depends on routed backend | backend + a few ms |
| Local engine | `-u http://127.0.0.1:8080/v1` | none | none | ~0.9 s on Metal |

Latencies from [the current serving image](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260927-image-parity/README.md).

## Connect

```bash
# Vertex AI dedicated endpoint. A bare ID is expanded to the dedicated /invoke/* URL, which needs
# GCP_PROJECT and GCP_PROJECT_NUMBER (or run on GCP); otherwise pass the full invoke URL.
export GCP_PROJECT=<PROJECT> GCP_PROJECT_NUMBER=<PROJECT_NUMBER>
./bin/dgem decide --vertex-url <ENDPOINT_ID> --gcp-auth \
  -t templates/support_triage.json.tmpl -v 'ticket=Database connection pool exhausted' --stats

# Cloud Run GPU service
./bin/dgem decide -u "https://<CLOUD_RUN_URL>/v1" --gcp-auth \
  -t templates/support_triage.json.tmpl -v 'ticket=Billing dispute'

# A gateway (routes to Vertex first, falls back to Cloud Run)
./bin/dgem decide -u "https://<your-dgem-gateway>/v1" --gcp-auth \
  -t templates/support_triage.json.tmpl -v 'ticket=General inquiry'
```

Persistent defaults go in environment variables or `.dgem.yaml` (project root or `~/.config/dgem/config.yaml`):

```bash
export DGEM_VERTEX_URL=<ENDPOINT_ID>   # or DGEM_URL=https://<CLOUD_RUN_URL>/v1
export DGEM_GCP_AUTH=true
```

```yaml
url: "https://<your-dgem-gateway>/v1"
gcp_auth: true
timeout: 120s
```

## Authentication

- `--gcp-auth` uses Application Default Credentials (`gcloud auth application-default login`) and mints the right
  token per target: an OAuth access token for Vertex AI, an ID token for Cloud Run and IAP-protected gateways.
- Static keys: `-k <token>` for self-hosted endpoints started with an API key.
- Debugging without tokens: `gcloud run services proxy <SERVICE> --region <REGION> --port 8080`, then
  `-u http://127.0.0.1:8080/v1`.
- Clients should retry `429`/`503` with backoff (`--http-retries N`): Vertex returns `429` while a replica is
  starting and the server returns `503` when its in-flight queue is full.

## Studio and MCP against a remote target

```bash
# Local Studio in front of your endpoints
./bin/dgem serve --port 8090 --gcp-auth --vertex-url <ENDPOINT_ID> -u https://<CLOUD_RUN_URL>/v1

# MCP (stdio) bridged to a hosted gateway, with automatic ADC auth
./bin/dgem mcp --remote https://<your-dgem-gateway>/mcp
```

More MCP and HTTP options: [Studio, MCP and HTTP API](/dgem/reference/studio-mcp-api/). Routing between backends:
[Gateway and routing](/dgem/deploy/gateway/).

## Other hosts

Any host running the `dgem` container (a GCE VM, Kubernetes, a workstation) serves the same API on port 8080:
point `-u` at `http://<host>:8080/v1`. The GCE helper scripts (`make gce-deploy`, `make gce-teardown`) create and
delete a single-GPU VM; delete it as soon as you're done, since it bills while running.
