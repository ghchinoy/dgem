# `dgem` Examples

Standalone, runnable Go examples showing how to embed **DiffusionGemma (`dgem`)** and its executable `.json.tmpl` decision policies (`templates/`) into applications and agent frameworks.

Each example lives in its own subdirectory with an isolated `go.mod` (using `replace github.com/ghchinoy/dgem => ../..`) so extra framework dependencies are not added to the core `dgem` CLI binary.

## Available Examples

| Example | Framework | Templates Used | What It Shows |
| :--- | :--- | :--- | :--- |
| **[`genkit-tool-approval/`](genkit-tool-approval/)** | **[Genkit Go `v1.14.0`](https://github.com/genkit-ai/genkit/releases/tag/go%2Fv1.14.0)** (`plugins/systemone/exp`, `plugins/middleware`, `ai/tool`) | `templates/support_triage.json.tmpl`<br/>`templates/calibration/agent_step_drift.json.tmpl` | 1. Executing existing `dgem` `.json.tmpl` policies through Genkit's `systemonex.SystemOne` plugin (`POST /v1/systemone`) with **Hesitation %** ($H / \ln K$) gating.<br/>2. Handling Genkit `v1.14.0` typed tool interrupts (`middleware.ToolApprovalInterrupted` and `genkit.DefineResumableTool`) where `confirm(name, input)` runs `agent_step_drift.json.tmpl` to block prompt-injected tool calls and resume approved ones via `ai.WithResume(...)`. |

## Quick Start

### 1. Run Offline Unit Tests (No GPU or Server Needed)

```bash
cd examples/genkit-tool-approval
go test -v ./...
```

### 2. Run Live Against Local Apple Silicon (`diffgemma`)

From the repository root:

```bash
# Start local diffgemma on 127.0.0.1:8080 (4096 context keeps memory footprint ~19.5 GiB on 32 GB Macs)
DIFFGEMMA_CTX=4096 ./scripts/serve.sh

# Run the Genkit v1.14.0 example
cd examples/genkit-tool-approval
go run .

# Stop the local diffgemma server when finished
cd ../..
make stop
```
