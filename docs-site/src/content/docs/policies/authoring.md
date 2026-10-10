---
title: "Policy Authoring and Stage 2 Cascades"
description: Step-by-step guide for designing .json.tmpl decision policies, selecting between Vertex AI Dedicated Endpoints (vertex_first / vertex) and Serverless Cloud Run GPU (cloudrun), and configuring the Stage 2 Gemini Cascade (gemini-3.8-flash default).
---

For policy authors: how to design `.json.tmpl` policies that are fast and reliable, and how to escalate hesitant
answers to a larger model (Stage 2 cascade) on every surface. New to policies? Start with
[Your first decision policy](/dgem/policies/first-policy/). Choosing a GPU backend (`vertex_first`, `vertex`, `cloudrun`) is
covered in [Gateway and routing](/dgem/deploy/gateway/).

---

## 1. Stage 2 Gemini Cascade (`gemini-3.8-flash` Default) in Batch Eval, `/api/decide`, and MCP

When Stage 1 (`DiffusionGemma`) encounters an ambiguous input (where restricted-softmax Shannon entropy $H \ge 0.35\text{ nats}$, e.g. `ChaosNLI` human-disagreement items or complex multi-hop `ANLI-R3` contradictions), `dgem` can automatically escalate that item to a **Stage 2 Gemini Cascade (`EXP-05`)**.

> [!IMPORTANT]
> **Supported Stage 2 Gemini Models**: Always use **`gemini-3.8-flash`** (default), **`gemini-3.7-flash`**, or **`gemini-3.5-flash-lite`** via standard Vertex AI `generateContent` routes. Never reference legacy Gemini 2.x models.

| Parameter | Values / Default | Surface Support | Description |
| :--- | :--- | :--- | :--- |
| **`cascade_mode`** | `"off"` (default) \| `"entropy"` \| `"on_miss"` | Web Studio Batch Eval, `POST /api/decide`, MCP (`decide_policy`, `decide_custom_questions`) | **`"off"`**: Stage 1 `dgemma` only (`57.5 ms` single-read GPU denoise).<br/>**`"entropy"`**: Production uncertainty gate — early-exits confident decisions at Stage 1 ($H < \tau$, ~72% of traffic) and escalates only high-entropy items ($H \ge \tau$) to Stage 2 Gemini.<br/>**`"on_miss"`**: Batch Eval ground-truth audit mode — escalates items where Stage 1 disagrees with the dataset's `expected` label. |
| **`cascade_threshold`** | `0.35` *(default, in nats)*; `0.16` with `cascade_threshold_mode: "normalized"` | Web Studio Batch Eval, `POST /api/decide`, MCP, `dgem bench-calibration` | Entropy gate $\tau$, computed from the answer's label probabilities: raw nats by default, or normalized $\tilde{H} = H / \ln|\mathcal{V}_m|$. |
| **`cascade_threshold_mode`** | `"nats"` *(default)* \| `"normalized"` | Web Studio Batch Eval ("Escalation gate"), `POST /api/decide` (or header `X-DGem-Cascade-Threshold-Mode`), MCP | What `cascade_threshold` is compared with. Raw entropy grows with the option count (max $\ln K$), so at 0.35 nats a 26-option slot escalates far more readily than a yes/no slot. `"normalized"` uses the Hesitation scale shown in the Studio (EXP-05b's gate). Cascade telemetry reports `threshold_mode` and each slot's `stage1_normalized_entropy`. **The two defaults are not equivalent:** on JevBench + calibration suite + intents (341 decisions), 0.16 normalized escalated 31% of decisions vs 22% at 0.35 nats (about 40% more Stage-2 calls), the extra mostly on yes/no questions. At matched escalation rates the two gates reach about the same accuracy, and normalized sends fewer wide-option (> 10) slots to Stage 2. Fit τ on your own traffic before switching mode. |
| **`cascade_model`** | `"gemini-3.8-flash"` *(default)* | Web Studio Batch Eval, `POST /api/decide`, MCP, `dgem serve`, `dgem bench-calibration` | Target Vertex AI Gemini model (`"gemini-3.8-flash"`, `"gemini-3.7-flash"`, or `"gemini-3.5-flash-lite"`). Configurable globally via `--cascade-model` (`DGEM_CASCADE_MODEL`) and `--cascade-models` (`DGEM_CASCADE_MODELS`). |
| **`stage2_prior`** | `"soft"` *(default)* \| `"full"` \| `"none"` | Web Studio Batch Eval ("Stage-1 hint to Gemini"), `POST /api/decide` (or header `X-DGem-Stage2-Prior`), MCP, `dgem bench-vision --cascade-prior`, env `DGEM_CASCADE_PRIOR` | How much of Stage 1's answer Gemini sees. **`soft`**: only that a fast first stage leaned toward an answer and may be wrong. **`full`**: answer, confidence, entropy and distribution as "the candidate" (the behaviour before v0.3.1), which anchored Gemini on low-threshold image cascades (escalated answers 0.86 vs 0.95 for Gemini alone). **`none`**: the question alone. Measured (issue #75): image cascade at 0.10 nats soft 0.906 vs full 0.880 (+2.6, 2 SE 1.4); text (JevBench) no difference. |

### Example: Calling `/api/decide` and MCP with `cascade_mode: "entropy"`

```bash
curl -sS "https://<your-dgem-gateway>/api/decide/calibration/chaos_nli" \
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
    "template": "calibration/chaos_nli",
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

## 2. Designing `.json.tmpl` Policy Schemas & Prefix-Cache Optimization

1. **Keep `"schema"` Static, Put Row Variables in `"state"`**:
   - With the default `document_first` layout (serving v0.2.0+), the prompt is the state followed by the question list, so the shared prefix across rows is only the chat preamble. If prefix-cache reuse matters more than the accuracy gain from `document_first` (EXP-19), set `"layout": "schema_first"`: the questions then form the system prompt ahead of the state, and vLLM's automatic prefix cache reuses that part across every row in a batch. Either way, keep the schema identical across rows. See [Prompt layout](/dgem/policies/prompt-layout/).
2. **Single-Pass Readout (`reads=1`)**:
   - Omit `depends_on` and `ask_if` unless you explicitly want a 2-stage conditional policy DAG (`reads=2`). All `level 0` slots (`boolean`, `choice [A–Z]`, `score`) are resolved simultaneously in a single forward pass.
3. **Registering New Experiments (`EXP-XX`)**:
   - Whenever you add a new benchmark harness, `.jsonl` dataset, or cascade study, register it in [`docs/experiments/`](/dgem/experiments/) with links to its `.json.tmpl` template, CLI command, and JSON receipt (`benchmarks/results_*.json`). For the full step-by-step Python batch runner (`run_dgem_dataset.py`), see the **[Custom Dataset & Experiment Cookbook](/dgem/policies/datasets/)**.
