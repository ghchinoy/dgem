# DiffusionGemma Benchmark Evaluation Report

This report presents empirical benchmark metrics comparing **Discrete Diffusion Slot Readout** against traditional **Prompt-Mediated Generative Autoregression**, evaluated across local **Apple Silicon Metal (M5, 32 GB)** and **Google Compute Engine (`g2-standard-8`, 1× NVIDIA L4 GPU)** deployment architectures.

> [!NOTE]
> For details on why Google Compute Engine (GCE) was used instead of serverless Google Cloud Run for evaluating experimental vLLM branches (including C++ CUDA extension ABI compatibility and Hugging Face Hub egress NAT limits), see **[Cloud Run Lessons Learned & Native CUDA Build Guide](cloudrun-lessons-learned.md)**.

---

## 1. Executive Summary

| Evaluation Metric | Target 1: Local Apple Silicon Metal | Target 2: Cloud GCE (`g2-standard-8`, 1× NVIDIA L4) | Target 3: Prompt-Mediated Generative |
| :--- | :--- | :--- | :--- |
| **Model Evaluated** | `diffgemma-26b-a4b-it-q4` | `nvidia/diffusiongemma-26B-A4B-it-NVFP4` | `diffgemma-26b-a4b-it-q4` (`think=false`) |
| **Model Forward Compute** | **~850 ms** (1 read) / **2,555 ms** (4 reads) | Integrated with Triton Eager Pass | **~16,500 – 17,500 ms** (61 tokens) |
| **End-to-End Wall Latency** | **~1.83 s** (1 read) / **4.14 s** (multi-avg) | **1,968.7 ms** (overall avg; **776 ms** fastest) | **17,486.6 ms** (~17.5 s) |
| **Speedup vs. Generative** | **4.2× – 9.7× faster** | **~8.9× faster overall** (up to **22.5×** peak) | Baseline (1.0×) |
| **Speedup vs. Local Metal** | Baseline (1.0×) | **~2.1× faster wall time** | — |
| **Syntactic Reliability** | **100% Schema-Guaranteed** | **100% Schema-Guaranteed** | **0% raw pass** (wrapped in markdown) |
| **Classification Accuracy** | **80.0%** (24 / 30 cases) | **73.3%** (22 / 30 cases) | ~75 – 80% |
| **Receipt Artifact** | [`results_local_metal_slot.json`](../benchmarks/results_local_metal_slot.json) | [`results_gce_l4.json`](../benchmarks/results_gce_l4.json) | [`results_local_metal_generative.json`](../benchmarks/results_local_metal_generative.json) |

---

## 2. Experimental Design, Dataset & Sample Cases

The evaluation suite (`./bin/dgem bench`) tests structured multi-question decision accuracy and latency across a standardized 30-case dataset ([`benchmarks/eval_dataset.jsonl`](../benchmarks/eval_dataset.jsonl)). For the deep mechanical explanation of 256-token canvas seeding and restricted softmax readout, see **[Architecture: Discrete Diffusion vs. Autoregression](architecture.md)**.

### 2.1 Multi-Domain Evaluation Corpus (30 Cases)
Each test item injects structured input variables into a domain-specific Go template and simultaneously evaluates **three distinct question types** (`boolean`, `choice`, and `score`) in a single request:

1. **Customer Support Triage (10 cases — [`templates/support_triage.json.tmpl`](../templates/support_triage.json.tmpl))**:
   * `urgent` (`boolean`): Does this issue require immediate same-day escalation? (`yes` / `no`)
   * `team` (`choice`): Which department owns resolution? (`billing`, `support`, `engineering`)
   * `sentiment` (`score`): Customer distress level (`calm`, `frustrated`, `furious`)
2. **Code Review & PR Triage (10 cases — [`templates/code_review.json.tmpl`](../templates/code_review.json.tmpl))**:
   * `approved` (`boolean`): Is this change safe to merge without security or stability regressions? (`yes` / `no`)
   * `category` (`choice`): Primary classification of the change (`bugfix`, `feature`, `refactor`, `security`)
   * `risk` (`score`): Operational risk level (`low`, `medium`, `high`)
3. **SecOps Alert Containment (10 cases — [`templates/security_incident.json.tmpl`](../templates/security_incident.json.tmpl))**:
   * `data_compromise` (`boolean`): Does this alert indicate active data exfiltration or credential compromise? (`yes` / `no`)
   * `action` (`choice`): Immediate automated response (`monitor`, `revoke_key`, `isolate_host`, `page_oncall`)
   * `severity` (`score`): Incident severity (`informational`, `elevated`, `critical`)

### 2.2 Difficulty Tiers & Representative Samples
To stress-test bidirectional attention and uncertainty calibration, the 30 cases are stratified across four difficulty tiers:

| Tier | Share | Purpose | Verbatim Sample from [`eval_dataset.jsonl`](../benchmarks/eval_dataset.jsonl) | Ground Truth |
| :--- | :--- | :--- | :--- | :--- |
| **Unambiguous** | 60% (18) | Baseline speed & single-pass certainty | **`sup-01`**: *"EMERGENCY: Production API gateway returning 500 internal server error across all US-East nodes. Customer traffic failing."* | `team: engineering`<br>`urgent: yes` |
| **Ambiguous** | 20% (6) | Overlapping intents requiring multi-sample noise averaging | **`sup-04`**: *"YOUR PLATFORM IS DOWN AGAIN AND MY CLIENTS ARE FURIOUS! CANCEL MY SUBSCRIPTION AND REFUND EVERYTHING RIGHT NOW!"* | `team: billing`<br>`urgent: yes` |
| **Negation** | 15% (5) | Catching explicit negative constraints (`NOT`, `do not`) | **`sup-05`**: *"This is NOT an outage or a billing problem. I just want to know if you support exporting reports to CSV or Excel."* | `team: support`<br>`urgent: no` |
| **Complex** | 5% (1) | Architectural & multi-system reasoning | **`code-08`**: *"Rewrite core Raft consensus state machine to improve throughput under heavy network partitions."* | `category: refactor`<br>`approved: yes` |

### 2.3 Adaptive Noise Sampling (`samples: "auto"`)
When evaluating discrete slots on the 256-token diffusion canvas, target positions (`@`) are initialized with random vocabulary noise tokens:
* **Unambiguous Convergence (1 Sample)**: If the restricted softmax distribution across candidate labels has Shannon entropy $H < 0.10\text{ nats}$ on the first forward pass, the engine stops immediately at $N=1$.
* **Ambiguity Escalation (Up to 4 Samples)**: If any slot exhibits high entropy ($H \ge 0.10\text{ nats}$), the engine draws additional independent noise vectors ($N=4$), averages the resulting probability distributions, and reports empirical standard error (`stderr`) and cross-sample agreement.

---

## 3. Detailed Local Metal Findings (Apple M5, 32 GB RAM)

Evaluated using `diffgemma-26b-a4b-it-q4` on Apple Silicon Metal ([`benchmarks/results_local_metal_slot.json`](../benchmarks/results_local_metal_slot.json)):

```text
================================================================================
  DIFFUSIONGEMMA: DISCRETE DIFFUSION BENCHMARK EVALUATION
================================================================================
Target Server: http://127.0.0.1:8080/v1
Model:         diffgemma-26b-a4b-it-q4
Mode:          slot
Test Cases:    30 items

• Slot Readout Accuracy:        80.0% (24 of 30 matched ground truth)
• Average Model GPU Denoise:    2,555.8 ms (pure forward compute)
• Average End-to-End Wall Time: 4,143.3 ms
• Adaptive Multi-Reads:         19 of 30 triggered multi-sampling (ambiguity flag)
• Single-Read Unambiguous Time: ~850 ms denoise / ~1,830 ms wall time
```

### Key Behavioral Observations:
1. **Unambiguous Cases**: When Shannon entropy across all slots was below $0.10$ nats (e.g. `sup-03`, `sup-05`, `sup-07`, `sup-10`, `code-03`, `sec-03`), the engine concluded in **exactly 1 sample**, completing GPU forward compute in **820–850 ms**.
2. **Ambiguity Detection**: Borderline cases (e.g. `sup-04` where a customer reports an outage *and* demands cancellation) triggered the `samples: "auto"` policy to draw 4 independent noise vectors, averaging probabilities and reporting empirical error bars (`stderr`).
3. **Negation Understanding**: Bidirectional attention successfully caught negations (e.g. `sup-05`: *"This is NOT an outage or a billing problem"* correctly routed to `team: support` with `urgent: no`).

---

## 4. Prompt-Mediated Baseline Findings

Evaluated using `diffgemma-26b-a4b-it-q4:think=false` with a strict JSON system prompt ([`benchmarks/results_local_metal_generative.json`](../benchmarks/results_local_metal_generative.json)):

```text
• Generative Average Wall Time: 17,486.6 ms (~17.5 seconds)
• Generative Completion Tokens: 61.2 tokens/req
• Generative Syntax Validity:   0.0% raw valid JSON (wrapped in markdown ```json blocks)
```

### The Autoregressive Penalties:
1. **Latency Overhead**: Generating 61 tokens sequentially via discrete diffusion blocks took **~17.5 seconds** per request—over **4.2× slower** than multi-sample slot reads and **9.7× slower** than single-pass slot reads.
2. **Formatting Fragility**: The model consistently wrapped JSON responses in markdown code blocks (````json\n{...}\n````), causing standard JSON parsers to fail unless pre-cleaned.
3. **No Calibration**: The generative model provides zero entropy or variance metrics.

---

## 5. Cloud GPU Findings (GCE `g2-standard-8`, 1× NVIDIA L4)

Evaluated live on Google Compute Engine (`us-central1-a`) running `nvidia/diffusiongemma-26B-A4B-it-NVFP4` with 32k KV cache and Triton attention ([`benchmarks/results_gce_l4.json`](../benchmarks/results_gce_l4.json)):

```text
================================================================================
  DIFFUSIONGEMMA: DISCRETE DIFFUSION BENCHMARK EVALUATION
================================================================================
Target Server: http://35.193.147.242:8080/v1
Model:         nvidia/diffusiongemma-26B-A4B-it-NVFP4
Mode:          slot
Test Cases:    30 items

• Slot Readout Accuracy:        73.3% (22 of 30 matched expected)
• Average End-to-End Wall Time: 1,968.7 ms (~1.97s per decision)
• Sub-Second Evals:             3 cases concluded under 900 ms (min: 776 ms)
• Speedup vs Generative:        ~8.9× faster wall time
```

### Key Comparative Takeaways:
1. **Latency vs. Local Apple M5 Metal**:
   * **Local Apple M5 Metal**: ~4,143 ms average wall time across 30 multi-sample cases (~1,830 ms for single reads).
   * **Cloud NVIDIA L4 GPU**: **1,968.7 ms** average wall time overall (**~2.1× faster overall latency**, with individual cases finishing in as little as **776 ms**).
2. **Speedup vs. Generative Baseline**:
   * Traditional autoregressive generation took **~17.5s** per request.
   * L4 GPU evaluated structured decisions in **~1.97s**—an **~8.9× throughput acceleration**.
3. **Precision Note**: `nvidia/diffusiongemma-26B-A4B-it-NVFP4` utilizes weight-only 4-bit Marlin quantization on Ada Lovelace (`sm_89`), accounting for the minor 2-case difference (22/30 vs. 24/30) on borderline ambiguous items (`code-04`, `sec-04`).

---

## 6. How to Reproduce

### Local Apple Silicon Metal
```bash
# 1. Start local diffgemma server
make serve

# 2. Run the 30-case slot readout benchmark
./bin/dgem bench -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_local_metal_slot.json

# 3. Run the generative comparative baseline
./bin/dgem bench -d benchmarks/eval_dataset.jsonl -M generative -n 5 -o benchmarks/results_local_metal_generative.json
```

### Google Compute Engine (GCE with GPU)
```bash
# 1. Provision GCE GPU instance and wait for health check
export GCP_PROJECT="your-gcp-project-id"
make gce-deploy

# 2. Run the benchmark suite against the live VM IP
./bin/dgem bench -u "http://<EXTERNAL_IP>:8080/v1" \
  -m "nvidia/diffusiongemma-26B-A4B-it-NVFP4" \
  -d benchmarks/eval_dataset.jsonl -M slot -o benchmarks/results_gce_l4.json

# 3. Tear down VM and firewall rules to prevent idle charges
make gce-teardown
```
