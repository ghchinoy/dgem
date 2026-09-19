# dgem (DiffusionGemma CLI & Assistant)

High-performance CLI assistant and automation harness for Google DeepMind's DiffusionGemma on Apple Silicon, powered by native Metal kernels and single-pass discrete block diffusion decisions.

---

## Installation

```bash
# Clone the repository
git clone https://github.com/ghchinoy/dgem.git
cd dgem

# Build the dgem binary into bin/
make build
```

*Note: `dgem` interacts with a running `diffgemma` inference server on macOS. See [docs/setup.md](docs/setup.md) for instructions on compiling the Rust Metal backend and downloading model weights.*

---

## Usage

### 1. Discrete Diffusion Slot Readout (Single-Pass Decisions)

Evaluate customer tickets, code changes, or security alerts in a single ~880 ms forward pass without conversational text overhead:

```bash
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=I was billed $500 twice for my annual renewal this morning!' \
  --stats
```

Output:

```
QUESTION         | TYPE       | VALUE / CHOICE       | CONFIDENCE | STDERR     | AGREEMENT 
----------------------------------------------------------------------------------------
sentiment        | score      | frustrated           | 80.1%      | ±0.0317    | 1.00      
team             | choice     | billing              | 100.0%     | ±0.0000    | 1.00      
urgent           | boolean    | yes                  | 97.0%      | ±0.0081    | 1.00      

──────────────────────────────── STATS ────────────────────────────────
  Model:             diffgemma-26b-a4b-it-q4
  Endpoint:          http://127.0.0.1:8080/v1/chat/completions

  Timing:
    • Total Wall Time:     4.10s
    • Server Prefill:      3,207 ms
    • Server Denoise:      856 ms (1 forward pass)

  Token Breakdown:
    • Prompt Tokens:       204 tokens
    • KV Cache Reused:     169 tokens (82.8% cache hit rate)
    • Completion Tokens:   13 tokens
    • Total Tokens:        217 tokens

  Inference Mechanics:
    • Denoise Steps:       1 step(s)
    • Noise Samples (N):   1 sample(s) (policy: auto, threshold: 0.10)
    • Multi-Read Extended: false (unambiguous, stopped at sample 1)

  Question Diagnostics:
    • sentiment   : argmax='1' (entropy=0.0009 nats, label_mass=100.0%)
    • team        : argmax='▁B' (entropy=0.0008 nats, label_mass=100.0%)
    • urgent      : argmax='▁no' (entropy=0.0003 nats, label_mass=100.0%)
───────────────────────────────────────────────────────────────────────
```

### 2. Generative Prompt Completion

Execute standard chat completions with optional thinking mode:

```bash
./bin/dgem ask "Explain the core mechanics of discrete block diffusion in two sentences."
```

### 3. Querying Hosted Google Cloud Run Endpoints

`dgem` seamlessly routes to remote Cloud Run GPU endpoints using automatic IAM authentication:

```bash
./bin/dgem decide \
  -u "https://diffusiongemma-vllm-xyz.a.run.app/v1" \
  --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Outage: production database cluster unreachable' \
  --stats
```

### 4. Multimodal Visual Assessment (`--image` / `-I`)

Attach local image paths (automatically base64 encoded) or remote URLs:

```bash
./bin/dgem decide -t templates/multimodal/ui_design_review.json.tmpl \
  -I fixtures/ui_component.svg \
  -v 'component=CheckoutCard' \
  --stats
```

---

## Architecture: Discrete Diffusion Slot Readout vs. Autoregression

DiffusionGemma operates on a **256-token canvas** with bidirectional attention. Instead of sequentially generating JSON token-by-token across hundreds of forward passes, `dgem` seeds the template structure and evaluates logits directly at token slots:

| Metric | Discrete Diffusion Slot Readout | Autoregressive Generative JSON | Advantage |
| :--- | :--- | :--- | :--- |
| **Model Forward Pass** | **~885 ms** (1 step) | ~11,200 ms (32 tokens) | **~12.6× faster model compute** |
| **End-to-End Latency** | **~1.97 s** | ~11.5 – 12.6 s | **~6× faster wall time** |
| **Output Reliability** | **100% Schema-Guaranteed** | Vulnerable to syntax drift | Mathematically bounded |
| **Uncertainty Quantification** | Calibrated entropy & empirical `stderr` | Uncalibrated generation | Native confidence metrics |

> **Terminology & Context**: In community benchmarks (such as `open-jev` and vLLM PR #57250), this single-forward evaluation pattern was popularized under the moniker "Jev-style" following TypeSafe AI's commercial evaluations. The underlying technique is **discrete diffusion slot readout**—pre-seeding a bidirectional diffusion canvas with fixed syntax and evaluating intermediate logits at candidate slot coordinates.

### Question Types
* **`boolean`** (formerly known as "noul"): Binary proposition (`yes` / `no`).
* **`choice`**: Single-token categorical selection (`A`–`Z` corresponding to named options).
* **`score`**: Ordered qualitative scale (`["low", "medium", "critical"]`).

---

## Development & Makefile Targets

The included `Makefile` automates building, testing, model setup, and server management:

```bash
make help        # Show all available make targets
make build       # Compile dgem into bin/dgem
make test        # Run Go unit tests
make fmt         # Format Go source code
make setup       # Install diffgemma engine via cargo
make download    # Download 4-bit model pack (~18.84 GiB)
make serve       # Launch background diffgemma server (port 8080)
make stop        # Stop background diffgemma server
make bench       # Run the Jev vs autoregression benchmark
make cloudrun-deploy # Deploy vLLM with PR #57250 to Google Cloud Run (RTX Pro 6000 GPU)
make docs-build  # Build the Catppuccin Latte Starlight documentation site
```

### Inspecting Templates
```bash
./bin/dgem template list
./bin/dgem template render -t templates/code_review.json.tmpl -v diff="sample diff"
```

---

## Documentation

* **[Setup & Metal Engine Guide](docs/setup.md)**: Hardware requirements, memory budgeting (`--ctx 32768`), Metal shader pipeline compilation, and serving.
* **[User Guide](docs/user-guide.md)**: Full CLI reference, template authoring, interpreting `--stats`, and CI/CD integration.
* **[Architecture: Discrete Diffusion vs. Autoregression](docs/architecture.md)**: Mechanical breakdown of 256-token canvas denoising, bidirectional slot readout, and terminology history.
* **[Real-World Applications & Production Patterns](docs/applications.md)**: Production architectures for agentic dispatch, DevSecOps git hooks, SIEM alert triage, and high-scale evaluations.
* **[Remote Endpoints & Cloud Deployment](docs/remote-endpoints.md)**: Pointing `dgem` to Google Cloud Vertex AI, hosted vLLM clusters, and understanding discrete slot readout mechanics.
* **[Benchmark Evaluation Report](docs/benchmarks-report.md)**: Empirical metrics comparing slot readout against generative autoregression on Apple Silicon Metal and Cloud Run L4.

---

## Contributing

Pull requests are welcome! For major changes, please open an issue first to discuss what you would like to change. Please ensure unit tests pass:

```bash
go test ./pkg/...
```

---

## License

This project is licensed under the [Apache-2.0 License](LICENSE).
