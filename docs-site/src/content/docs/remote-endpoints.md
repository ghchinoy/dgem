---
title: Remote Endpoints & Cloud Deployment
description: Guide for pointing dgem to hosted instances of DiffusionGemma on Google Cloud Vertex AI, remote GPU clusters, and custom containers.
---

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

## 3. Targeting Google Cloud Vertex AI

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
   * **`dgem decide`** requires the serving engine in the container to support canvas pre-seeding (see Section 4).

---

## 4. Cloud Deployment Targets for Discrete Slot Readout

To achieve sub-second single-pass slot readout on cloud infrastructure, the serving container must support canvas seeding. Three primary architectures enable this:

### Target A: Containerized `diffgemma` (GCE / GKE / Cloud Run with GPU)
Run `diffgemma serve` inside an NVIDIA GPU container:
* Uses the exact HTTP schema API that `dgem` talks to natively.
* Supports both standard generative chat and sub-second structured decisions out of the box.

### Target B: vLLM with PR #57250
Matt Mastracci’s PR #57250 (`[Core] structured generation mode for DiffusionGemma model`) adds discrete reading parameters to vLLM's `vllm_xargs`:
* `diffusion_seed_canvas`: Array of fixed tokens replacing random noise after prefill.
* `diffusion_read_only: true`: Emits the argmax canvas on the converging step without committing full text generation.
* `diffusion_max_steps: 1`: Limits the run to 1 forward pass.

Deploying a vLLM container built from this branch enables high-throughput batching of structured decisions on NVIDIA H100/A100 GPUs.

### Target C: Python Microservice (`open-jev` style on Modal / GKE)
Deploy a lightweight FastAPI container using Hugging Face `transformers` (`DiffusionGemmaForBlockDiffusion`) with a custom `LogitsProcessor` that locks template positions and reads slot logits in 1 step.

---

## 5. Latency & Quality: Slot Readout vs. Generative JSON

A comparison of single-pass slot readout (Option B) vs. generative JSON prompting (Option D):

| Evaluation Metric | Slot Readout (Option B) | Generative JSON (Option D) | Advantage |
| :--- | :--- | :--- | :--- |
| **Model Forward Compute** | **~850 – 900 ms** (1 forward pass) | ~11,200 – 16,000 ms (32–50 passes) | **~12.6× faster compute** |
| **End-to-End Latency** | **~1.9 – 2.9 s** | ~12 – 22 s | **~6× faster wall time** |
| **Token / Cloud Cost** | **Zero completion tokens** | 32–80 completion tokens per ticket | **~79% cheaper GPU cost** |
| **Syntactic Reliability** | **100% Schema-Guaranteed** | Vulnerable to syntax drift | Mathematically bounded |
| **Uncertainty Calibration** | Calibrated entropy & empirical `stderr` | Uncalibrated (hallucinatory) | Native error bars |
