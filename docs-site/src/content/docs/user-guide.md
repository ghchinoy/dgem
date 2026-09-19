---
title: dgem CLI User Guide
description: Complete command reference, template authoring guide, and telemetry interpretation for dgem.
---

`dgem` is the command-line companion and automation tool for **DiffusionGemma** on Apple Silicon. It connects to a local or remote `diffgemma` server and enables:
1. **Jev-Style Structured Decisions**: Bounded, schema-governed classification and evaluation executed in a single forward pass (~880 ms) without generative text overhead.
2. **Generative Prompt Execution**: Natural-language chat and reasoning with thinking mode control.
3. **Template Management**: Authoring and rendering dynamic Go template definitions with variable substitution.
4. **Execution Telemetry (`--stats`)**: Detailed inspection of Metal GPU prefill, denoise times, KV cache reuse %, entropy, and error bars.

---

## 1. Configuration & Environment

`dgem` reads configuration hierarchically: command-line flags override environment variables, which override YAML configuration files.

### Configuration File (`.dgem.yaml`)
By default, `dgem` looks for `.dgem.yaml` in the current directory or `~/.config/dgem/config.yaml`:

```yaml
url: "http://127.0.0.1:8080/v1"
model: "diffgemma-26b-a4b-it-q4"
timeout: 120s
stats: false
```

### Environment Variables
Any flag can be configured using the `DGEM_` prefix:
```bash
export DGEM_URL="http://127.0.0.1:8080/v1"
export DGEM_MODEL="diffgemma-26b-a4b-it-q4"
export DGEM_STATS="true"
export DGEM_TOKEN="Bearer <your-token>"
export DGEM_GCP_AUTH="true"
```

### Global CLI Flags
* `--url`, `-u`: Base URL of the running server (default: `http://127.0.0.1:8080/v1`).
* `--model`, `-m`: Model identifier (default: `diffgemma-26b-a4b-it-q4`).
* `--timeout`: HTTP timeout duration (default: `120s`).
* `--stats`, `-s`: Print comprehensive timing, token reuse, and entropy stats.
* `-k`, `--token`: Authorization Bearer token or API key for secured endpoints.
* `--gcp-auth`: Automatically obtain and inject a Google Cloud IAM identity token via `gcloud auth print-identity-token`.
* `--config`: Path to custom config file.

> **Remote & Cloud Endpoints**: To connect `dgem` to remote hosts such as Google Cloud Run with GPU, Vertex AI, or private clusters, see the [Remote Endpoints & Cloud Deployment Guide](remote-endpoints.md).

---

## 2. Command Reference

### `dgem decide`
Executes single-forward structured decision reading.

```bash
dgem decide [flags]
```

#### Flags
* `-t`, `--template string`: Path to a Go template file (`.json.tmpl`).
* `-v`, `--var stringArray`: Template variables in `key=value` format (can be specified multiple times).
* `-d`, `--data string`: Path to a JSON file containing variables.
* `-f`, `--format string`: Output format: `table` (default) or `json`.
* `-I`, `--image stringArray`: Attach local image file path or remote image URL (can be specified multiple times for frame sequences).
* `--schema string`: Path to a raw JSON schema file (skips template engine).
* `--state string`: Raw JSON state string or file path.

#### Example: Table Output
```bash
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=How do I update my profile picture? No rush at all.'
```

```
QUESTION         | TYPE       | VALUE / CHOICE       | CONFIDENCE | STDERR     | AGREEMENT 
----------------------------------------------------------------------------------------
sentiment        | score      | calm                 | 100.0%     | ±0.0000    | 1.00      
team             | choice     | support              | 100.0%     | ±0.0000    | 1.00      
urgent           | boolean    | no                   | 100.0%     | ±0.0000    | 1.00      
```

#### Example: Querying a Hosted Cloud Run Endpoint with IAM Auth
```bash
./bin/dgem decide \
  -u "https://diffusiongemma-vllm-xyz.a.run.app/v1" \
  --gcp-auth \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Emergency: production database cluster down!' \
  --stats
```

#### Example: Multimodal Visual Inspection (`--image` / `-I`)
```bash
# Attach local image file (auto-encoded to base64 data URI)
./bin/dgem decide -t templates/multimodal/ui_design_review.json.tmpl \
  -I fixtures/ui_component.svg \
  -v 'component=CheckoutCard' \
  --stats

# Attach remote image URL or sequential video frames
./bin/dgem decide -t templates/multimodal/workplace_hazard_video.json.tmpl \
  -I "https://example.com/dock_frame_01.jpg" \
  -I "https://example.com/dock_frame_02.jpg" \
  -v 'zone=Dock-Door-12' \
  --stats
```

#### Example: Raw JSON Output
```bash
./bin/dgem decide -t templates/code_review.json.tmpl \
  -v 'diff=Fix critical SQL injection using parameterized query' \
  -f json
```

```json
{
  "answers": {
    "approved": {
      "type": "boolean",
      "label": "yes",
      "confidence": 0.9957,
      "stderr": 0.0018,
      "agreement": 1.0,
      "probabilities": { "no": 0.0043, "yes": 0.9957 }
    },
    "category": {
      "type": "choice",
      "choice": "security",
      "label": "D",
      "confidence": 0.9228,
      "stderr": 0.0739,
      "agreement": 1.0
    },
    "risk_level": {
      "type": "score",
      "level": "low",
      "score": 1.937,
      "confidence": 0.4132,
      "stderr": 0.1606,
      "agreement": 0.75
    }
  }
}
```

---

### `dgem ask`
Executes standard generative completions.

```bash
dgem ask [PROMPT] [flags]
```

#### Flags
* `-t`, `--template string`: Path to a text prompt template file (`.txt.tmpl`).
* `-v`, `--var stringArray`: Template variables in `key=value` format.
* `-I`, `--image stringArray`: Attach local image file path or remote image URL.
* `--max-tokens int`: Maximum tokens to generate (default: `256`).
* `--think`: Enable thinking mode (`<|think|>`) to output internal deliberative reasoning.

#### Examples
```bash
# Direct prompt
./bin/dgem ask "Explain discrete block diffusion in two sentences."

# Using a prompt template
./bin/dgem ask -t templates/ask_summary.txt.tmpl \
  -v 'topic=discrete block diffusion' \
  -v 'length=one short sentence'
```

---

### `dgem bench`
Runs an empirical benchmark comparing single-pass structured decisions against traditional autoregressive text generation.

```bash
dgem bench [flags]
```

#### Flags
* `--with-generative`: Include standard autoregressive text generation comparison on initial test cases (default: `true`).

#### Example
```bash
./bin/dgem bench
```

---

### `dgem template`
Manages and inspects Go template definition files.

```bash
# List available templates in templates/
./bin/dgem template list

# Render and inspect template variables locally without executing
./bin/dgem template render -t templates/support_triage.json.tmpl -v ticket="System down"
```

---

## 3. Understanding the Question Types

DiffusionGemma's structured reader replaces arbitrary prose with mathematically bounded question types:

| Type | Semantic Purpose | Allowed Values | Output Fields |
| :--- | :--- | :--- | :--- |
| **`boolean`** *(or `bool`)* | Predicate proposition | `yes` / `no` (`true` / `false`) | `confidence`, `probabilities.yes`, `probabilities.no` |
| **`choice`** | Categorical routing | Up to 26 named options (`A`–`Z`) | `choice`, `label`, `confidence`, `probabilities` |
| **`score`** | Ordered qualitative scale | Ordered list of levels (e.g. `["low", "medium", "high"]`) | `level`, `score` (expected float), `confidence` |

---

## 4. Interpreting `--stats` Telemetry

When `--stats` (or `-s`) is provided, `dgem` outputs an operational metrics panel:

```
──────────────────────────────── STATS ────────────────────────────────
  Model:             diffgemma-26b-a4b-it-q4
  Endpoint:          http://127.0.0.1:8080/v1/chat/completions

  Timing:
    • Total Wall Time:     4.099s
    • Server Prefill:      3207 ms
    • Server Denoise:      856 ms

  Token Breakdown:
    • Prompt Tokens:       204 tokens
    • KV Cache Reused:     169 tokens (82.8% cache hit rate)
    • Completion Tokens:   13 tokens
    • Total Tokens:        217 tokens

  Inference Mechanics:
    • Denoise Steps:       1 step(s)
    • Noise Samples (N):   1 sample(s) (policy: auto, threshold: 0.10)
    • Multi-Read Extended: false (first-read max entropy: 0.0009 nats)

  Question Diagnostics:
    • sentiment   : argmax='1' (entropy=0.0009 nats, label_mass=100.0%)
    • team        : argmax='▁B' (entropy=0.0008 nats, label_mass=100.0%)
    • urgent      : argmax='▁no' (entropy=0.0003 nats, label_mass=100.0%)
───────────────────────────────────────────────────────────────────────
```
