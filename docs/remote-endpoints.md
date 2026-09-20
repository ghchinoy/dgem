# Remote Endpoints & Cloud Deployment Guide

This guide explains how to point `dgem` to hosted instances of **DiffusionGemma** (such as Google Cloud Vertex AI, remote GPU clusters, or private enterprise endpoints), details the mechanics of **discrete diffusion slot readout**, and compares cloud serving architectures.

---

## 1. Connecting `dgem` to Remote Hosts

`dgem` is built as an independent client layer that connects to any OpenAI-compatible API endpoint via HTTP.

### Method A: Command-Line Flags
Specify the target URL and model name for an individual command:

```bash
# Execute a structured decision against a remote endpoint
./bin/dgem decide \
  -u "https://diffusiongemma.internal.net/v1" \
  -m "google/diffusiongemma-26B-A4B-it" \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Emergency: cluster outage' \
  --stats

# Execute a generative prompt against a remote endpoint
./bin/dgem ask \
  -u "https://diffusiongemma.internal.net/v1" \
  -m "google/diffusiongemma-26B-A4B-it" \
  "Explain discrete block diffusion in two sentences."
```

### Method B: Environment Variables
Configure target settings using the `DGEM_` prefix:

```bash
export DGEM_URL="https://diffusiongemma.internal.net/v1"
export DGEM_MODEL="google/diffusiongemma-26B-A4B-it"
export DGEM_TIMEOUT="60s"
export DGEM_STATS="true"

# Now dgem commands automatically route to the remote host:
./bin/dgem decide -t templates/support_triage.json.tmpl -v 'ticket=Database corrupted'
```

### Method C: Configuration File (`.dgem.yaml`)
Place `.dgem.yaml` in the root of your project or in `~/.config/dgem/config.yaml`:

```yaml
url: "https://diffusiongemma.internal.net/v1"
model: "google/diffusiongemma-26B-A4B-it"
timeout: 60s
stats: true
```

---

## 2. Understanding "Discrete Diffusion Slot Readout"

A common question when seeing sub-second Jev-style decisions is: *Is this a custom fine-tuned model or adapter?*

**No. It runs on the 100% stock, official weights from Google DeepMind (`google/diffusiongemma-26B-A4B-it`).**

### How It Works Mechanically

In traditional autoregressive LLMs (e.g. Gemma 2, Llama 3), generation is strictly causal:
* Token $t$ can only attend to previous tokens $x_{<t}$.
* To generate `"urgent: yes"`, the model must predict `"urgent"`, then `":"`, then `" "`, then `"yes"`, taking multiple sequential passes.

DiffusionGemma operates on a **256-token canvas** with **bidirectional cross-attention**:

```
Seeded Canvas:
[<|channel>thought\n<channel|>urgent: @ \nteam: @ \nsentiment: @ \n<eos>]
                                      ▲         ▲          ▲
                                   Slot 1    Slot 2     Slot 3
                               [yes, no]   [A, B, C]  [1, 2, 3]
```

1. **Canvas Seeding**: The known question template is pre-seeded into fixed positions on the 256-token canvas.
2. **Noise Masking**: Target answer slots (`@`) are initialized with random noise tokens drawn from the vocabulary.
3. **Single Forward Pass (~850–900 ms)**: A single forward pass is executed through the 30 transformer layers. Because attention is bidirectional, the noise tokens at all answer slots are evaluated simultaneously in the context of the input document and the surrounding template.
4. **Restricted Logit Readout**: The model produces logits over the entire 262,144-token vocabulary for every position. The engine looks **only** at the candidate slot positions and applies a restricted softmax over the allowed single-token labels:
   - **Slot 1 (urgent)**: Isolates logits for token `yes` and token `no`.
   - **Slot 2 (team)**: Isolates logits for labels `A`, `B`, `C` (billing, support, engineering).
   - **Slot 3 (sentiment)**: Isolates logits for `1`, `2`, `3` (calm, frustrated, furious).
5. **Noise Averaging & Uncertainty Quantification**: By drawing 1 to 4 independent noise vectors, the engine calculates:
   - **Confidence**: Mean softmax probability of the winning label.
   - **Shannon Entropy ($H$)**: Degree of spread over candidate logprobs.
   - **Standard Error (`stderr`)**: Variation of probability across noise draws.
   - **Agreement**: Percentage of noise draws that converged on the winning label.

---

## 3. Deploying DiffusionGemma on Google Cloud Run with GPU

While Google Cloud Vertex AI Model Garden hosts DiffusionGemma, its standard prebuilt containers do not yet expose the intermediate discrete diffusion slot-readout hooks.

**Google Cloud Run with GPUs** provides the ideal serverless platform on Google Cloud: you retain full control over the container image, benefit from zero-to-N autoscaling, and have access to enterprise NVIDIA GPUs (`nvidia-rtx-pro-6000` with 48 GB VRAM, or `nvidia-l4` with 24 GB VRAM).

### Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Google Cloud Run (Gen2)                         │
│                                                                        │
│   ┌────────────────────────────────────────────────────────────────┐   │
│   │  Container: vllm-openai (Overlay with PR #57250)               │   │
│   │  Entrypoint: vllm serve nvidia/diffusiongemma-26B-A4B-it-NVFP4  │   │
│   │              --diffusion-config '{"canvas_length": 32}'        │   │
│   └───────────────────────────────┬────────────────────────────────┘   │
│                                   │                                    │
│   ┌───────────────────────────────┴────────────────────────────────┐   │
│   │  Hardware: 1x NVIDIA RTX Pro 6000 (48GB VRAM)                  │   │
│   │  Storage: Optional GCS FUSE volume mount for <15s cold starts  │   │
│   └────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────▲────────────────────────────────────┘
                                    │ HTTPS (IAM OIDC Token)
                                    │
                       Local CLI: dgem (on your Mac)
```

### Staging & Deploying Option A (vLLM with PR #57250)

Because Matt Mastracci's PR #57250 changes are purely in Python (`vllm/model_executor/models/diffusion_gemma.py`, `vllm/v1/core/sched/diffusion_scheduler.py`), we can build directly on top of the official `vllm/vllm-openai:latest` base image without compiling CUDA C++ from source:

1. **Dockerfile**: Available at `deploy/cloudrun/Dockerfile.vllm`.
2. **Deploy with One Command**:
   ```bash
   # Deploy using the automated script:
   ./scripts/deploy_cloudrun_vllm.sh
   # Or via Makefile:
   make cloudrun-deploy
   ```

> [!NOTE]
> For details on C++ CUDA ABI compatibility, why pure Python overlays on prebuilt images encounter PyTorch dispatcher mismatches, and blueprints for full native CUDA compilations on Cloud Run vs. GCE, see **[Cloud Run Lessons Learned & Native CUDA Build Guide](cloudrun-lessons-learned.md)**.

### Connecting to Cloud Run with `dgem`

Cloud Run services require IAM authentication by default. `dgem` supports two frictionless connection methods:

#### Method 1: Automatic GCP Authentication (`--gcp-auth`)
`dgem` can automatically invoke `gcloud auth print-identity-token` to obtain and refresh your IAM token:

```bash
./bin/dgem decide \
  -u "https://diffusiongemma-vllm-xyz.a.run.app/v1" \
  --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Outage: production database connection refused' \
  --stats
```

#### Method 2: Explicit Bearer Token (`--token` or `DGEM_TOKEN`)
```bash
export DGEM_URL="https://diffusiongemma-vllm-xyz.a.run.app/v1"
export DGEM_TOKEN="$(gcloud auth print-identity-token)"

./bin/dgem decide -t templates/support_triage.json.tmpl -v 'ticket=Outage' --stats
```

#### Method 3: Cloud Run Developer Proxy
```bash
# In terminal 1:
gcloud run services proxy diffusiongemma-vllm --region us-central1 --port 8080

# In terminal 2 (queries localhost:8080 directly without auth headers):
./bin/dgem decide -t templates/support_triage.json.tmpl -v 'ticket=Outage' --stats
```

---

## 4. Targeting Google Cloud Vertex AI

Google Cloud hosts DiffusionGemma in [Vertex AI Model Garden](https://console.cloud.google.com/vertex-ai/publishers/google/model-garden/diffusiongemma).

### Deployment Architecture on Vertex AI
When deploying from Model Garden:
* DiffusionGemma is deployed to a **Vertex AI Prediction Endpoint** backed by NVIDIA L4, A100, or H100 GPUs.
* By default, Vertex AI exposes the prediction endpoint at:
  ```
  POST https://{REGION}-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/{REGION}/endpoints/{ENDPOINT_ID}:rawPredict
  ```

### Authentication
Requests to Vertex AI require a Google Cloud IAM OAuth2 access token:

```bash
# Obtain access token
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Query Vertex AI rawPredict route directly
curl -s -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  "https://us-central1-aiplatform.googleapis.com/v1/projects/MY_PROJECT/locations/us-central1/endpoints/MY_ENDPOINT:rawPredict" \
  -d '{
    "model": "google/diffusiongemma-26B-A4B-it",
    "messages": [{"role": "user", "content": "Explain discrete diffusion."}]
  }'
```

### Using Vertex AI with `dgem`
To use `dgem` with Vertex AI:
1. **With an OpenAI-Compatible Reverse Proxy (e.g. LiteLLM / Cloud Run)**:
   Point `dgem` to the proxy URL:
   ```bash
   export DGEM_URL="https://vertex-proxy.internal.net/v1"
   ./bin/dgem ask "Explain discrete block diffusion."
   ```
2. **Generative vs. Structured on Stock Vertex**:
   * **`dgem ask`** works immediately with any standard Vertex AI endpoint.
   * **`dgem decide`** requires the serving engine in the container to support canvas pre-seeding (see Section 5).

---

## 5. Cloud Deployment Targets for Discrete Slot Readout

To achieve sub-second single-pass slot readout on cloud infrastructure, the serving container must support canvas seeding. Four primary architectures enable this:

### Target A: Serverless Google Cloud Run (1× NVIDIA L4, 24 GB)
Deploy our self-contained container image (`deploy/cloudrun/Dockerfile`) with GCS FUSE volume mounting:
* **Hardware**: 1× NVIDIA L4 GPU, 8 vCPUs, 32 GB RAM.
* **Weights**: 17.57 GB NVFP4 safetensors shards streamed from regional GCS bucket via Cloud Storage FUSE (`--safetensors-load-strategy prefetch`).
* **Performance**: **458.9 ms** average wall latency (**427.3 ms** model denoise compute).
* **IAM Authentication**: Protected by Google Cloud IAM; query directly with `./bin/dgem decide --gcp-auth -u https://...`.
* **Zero Idle Cost**: Spin up via `make cloudrun-deploy`, execute evaluations, and immediately destroy via `make cloudrun-teardown`.

### Target B: Google Compute Engine (NVIDIA L4 / A100 VM with vLLM PR #57250)
Provisions automated, production-grade GCE instances (`g2-standard-8` or `a2-highgpu-2g`) with the nightly vLLM wheel (`wheels.vllm.ai`, commit `133b71e0be`) and Triton attention:
* Supports continuous batching and high concurrent throughput.
* Deploy via `make gce-deploy` (options for 4-bit, 8-bit, or 16-bit precision) and tear down via `make gce-teardown`.

### Target C: Containerized `diffgemma` (GCE / GKE / Metal)
Run `diffgemma serve` inside an Apple Silicon or NVIDIA container:
* Uses the native Rust discrete slot engine.
* Native `StructuredDecisionResponse` with restricted-softmax probabilities and Shannon entropy telemetry.

### Target D: Python Microservice (`open-jev` style on Modal / GKE)
Deploy a lightweight FastAPI container using Hugging Face `transformers` (`DiffusionGemmaForBlockDiffusion`) with a custom `LogitsProcessor` that locks template positions and reads slot logits in 1 step.

---

## 6. Latency & Quality: Slot Readout vs. Generative JSON

A comparison of single-pass slot readout (Option B) vs. generative JSON prompting (Option D):

| Evaluation Metric | Slot Readout (Option B) | Generative JSON (Option D) | Advantage |
| :--- | :--- | :--- | :--- |
| **Model Forward Compute** | **~850 – 900 ms** (1 forward pass) | ~11,200 – 16,000 ms (32–50 passes) | **~12.6× faster compute** |
| **End-to-End Latency** | **~1.9 – 2.9 s** | ~12 – 22 s | **~6× faster wall time** |
| **Token / Cloud Cost** | **Zero completion tokens** | 32–80 completion tokens per ticket | **~79% cheaper GPU cost** |
| **Syntactic Reliability** | **100% Schema-Guaranteed** | Vulnerable to syntax drift | Mathematically bounded |
| **Uncertainty Calibration** | Calibrated entropy & empirical `stderr` | Uncalibrated (hallucinatory) | Native error bars |

---

## 7. Custom Logits Processors in Rust and Go

### Rust (Inside the Model Engine)
`diffgemma` implements slot scoring directly in Rust (`src/structured.rs`):

```rust
pub struct Slot {
    pub pos: usize,          // Canvas position of the candidate token
    pub label_ids: Vec<u32>, // Allowed token IDs (e.g. "yes", "no")
}

pub fn score_slot(logits: &[f32], slot: &Slot, vocab_size: usize) -> Vec<f32> {
    // 1. Slice logits at the candidate slot position
    let row = &logits[slot.pos * vocab_size .. (slot.pos + 1) * vocab_size];
    
    // 2. Isolate candidate token logits
    let candidate_logits: Vec<f32> = slot.label_ids.iter().map(|&id| row[id as usize]).collect();
    
    // 3. Normalize via softmax over allowed tokens only
    softmax(&candidate_logits)
}
```

### Go (Proxy / Interposer Layer)
In Go, an interposer or proxy can inspect raw logprobs returned by remote servers, re-normalize probabilities over allowed choices, and compute Shannon entropy:

```go
type LogitsFilter struct {
    AllowedLabels map[string]struct{}
}

func (f *LogitsFilter) FilterAndNormalize(rawLogprobs map[string]float64) map[string]float64 {
    // Filter only allowed labels and re-normalize softmax sum to 1.0
    filtered := make(map[string]float64)
    var sum float64
    for label, prob := range rawLogprobs {
        if _, ok := f.AllowedLabels[label]; ok {
            filtered[label] = prob
            sum += prob
        }
    }
    for k := range filtered {
        filtered[k] /= sum
    }
    return filtered
}
```
