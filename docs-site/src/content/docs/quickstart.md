---
title: "5-Minute Quickstart"
description: "Get up and running with DiffusionGemma (dgem) in 5 minutes on Apple Silicon Metal, Docker with NVIDIA GPU, or remote cloud endpoints."
---

# 5-Minute Quickstart

Get DiffusionGemma running and execute your first sub-second structured decision in under five minutes.

---

## Step 1: Choose Your Inference Backend

`dgem` requires a running backend to execute neural network forward passes. Choose the option matching your hardware:

### Option A: Local Mac (Apple Silicon Metal) — No Docker or Cloud Needed
Runs 100% locally on your Mac using native Unified Memory:
```bash
# Verify prerequisites, download 4-bit model pack (~18.8 GB), and start engine on :8080
make setup
make download
make serve
```

### Option B: Any Linux/Windows Workstation with NVIDIA GPU (Docker)
Pulls our official, pre-verified public container image:
```bash
# Option B1: Self-contained image with pre-baked NVFP4 weights (instant start):
docker run --gpus all -p 8080:8080 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem-weights:56baadf@sha256:893f45a29e774bcda67ec66574f6b084c878795f95ecd9301a9d424cd726d36a

# Option B2: Lean base image (downloads public weights on first boot):
docker run --gpus all -p 8080:8080 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:56baadf@sha256:a7ace753973b6c3521dbc4c62ea4dfbea5384c582884e98a5ccae9f81b1f6dd9
```

### Option C: Remote Google Cloud Endpoint (Cloud Run or Vertex AI)
If your team has an existing endpoint or you deployed via [Deploy on Your Own Cloud GPU](/dgem/deploy-your-own-gpu/):
```bash
export DGEM_VERTEX_URL="<endpoint-id>" # Or export DGEM_URL="https://<service>.run.app/v1"
export DGEM_GCP_AUTH="true"
```

---

## Step 2: Build the `dgem` CLI

In your local repository clone:
```bash
make build
```
This compiles the fast Go binary into `./bin/dgem`.

---

## Step 3: Run Your First Structured Decision

Run an operational ticket triage decision using a built-in policy template:

```bash
./bin/dgem decide \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Urgent: production database connection pool exhausted after deploy' \
  --stats
```

### Expected Output:
```text
SLOT             | TYPE       | VALUE                | PROB       | ENTROPY     | MARGIN    
-----------------------------------------------------------------------------------------
sentiment        | score      | frustrated           | 99.8%      | 0.002 nats  | 1.00      
team             | choice     | engineering          | 100.0%     | 0.000 nats  | 1.00      
urgent           | boolean    | yes                  | 99.9%      | 0.001 nats  | 1.00      

──────────────────────────────── STATS ────────────────────────────────
  Model:             nvidia/diffusiongemma-26B-A4B-it-NVFP4
  Endpoint:          http://127.0.0.1:8080/v1/chat/completions
  Total Wall Time:   185 ms
  KV Cache Reused:   169 tokens (82.8% hit rate)
  Denoise Steps:     1 step (policy: samples=1)
───────────────────────────────────────────────────────────────────────
```

All three question slots (`sentiment`, `team`, `urgent`) were resolved simultaneously in a **single forward pass** ($O(1)$) with calibrated probability and epistemic Shannon entropy ($H$).

---

## Step 4: Open Decision Studio

Launch the local interactive UI and gateway:
```bash
./bin/dgem serve --port 8090
```

Open [http://localhost:8090](http://localhost:8090) in your browser:
* Explore all **26+ pre-built `.json.tmpl` policies**.
* Test **Stage-2 Gemini 3.8 Flash Cascades** on ambiguous inputs.
* Inspect live **OpenTelemetry trace waterfalls** and denoise timings.
* Test multimodal **SigLIP Bounding Box** detection on images.

---

## Step 5: Connect AI Assistants via MCP

`dgem` includes a built-in Model Context Protocol server. Connect it to Gemini CLI, Claude Desktop, Cursor, or Cline:

```json
{
  "mcpServers": {
    "dgem": {
      "command": "/path/to/diffusiongemma/bin/dgem",
      "args": ["mcp", "--local"]
    }
  }
}
```

Now your AI assistant can execute zero-shot discrete policies and query calibrated confidence scores natively.

---

## Next Steps
* **[dgem User Guide](/dgem/user-guide/)** — Detailed guide to commands, flags, and scripting.
* **[Template Catalog (Policy-as-Code)](/dgem/templates/)** — Browse and author custom `.json.tmpl` decision schemas.
* **[Deploy on Your Own Cloud GPU](/dgem/deploy-your-own-gpu/)** — Scale out to serverless Cloud Run or Vertex AI.
