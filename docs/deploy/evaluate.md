---
title: "Evaluate dgem on Your Own GPU"
description: "A step-by-step evaluation guide for teams hosting DiffusionGemma themselves: GPU requirements, verifying the install, a baseline against public benchmarks with pass/fail ranges, then a custom evaluation on your own labelled data."
---

# Evaluate dgem on Your Own GPU

This guide is for teams that host DiffusionGemma themselves and want to know two things: **is the install
working correctly**, and **how well does it work on our decisions**. Run the steps in order; each has a pass
criterion, so a problem found in step 3 is not mistaken for a model limitation in step 5.

| Step | What you check | Time | Pass when |
| :--- | :--- | :--- | :--- |
| 0 | Hardware | — | GPU and host RAM meet the table below |
| 1 | The container starts | 5–15 min (weight download) | `/health` shows `vllm_ready` and `warmed` |
| 2 | A first decision | 1 min | Valid answers with probabilities; GPU time in the expected range |
| 3 | Baseline on public benchmarks | ~10 min | JevBench and calibration scores inside our measured range, 0 errors |
| 4 | Concurrency (optional) | 5 min | 0 errors at your expected concurrency |
| 5 | Write a policy and evaluate it on your own labelled data | hours to days | Accuracy, calibration and escalation rate meet your bar |

Identifiers such as `<GPU_HOST>` are placeholders.

## 0. Hardware

The public images serve `nvidia/diffusiongemma-26B-A4B-it-NVFP4`, a 4-bit (NVFP4) checkpoint of about 18 GiB.

| GPU | Status | Host RAM | Notes |
| :--- | :--- | :--- | :--- |
| **NVIDIA RTX PRO 6000 (Blackwell, 48 GB)** | **Recommended; what we test and run in production** | 80 GiB | Runs FP4 natively; vision tower on (`DISABLE_MM=0`) |
| NVIDIA L4 (Ada, 24 GB) | Works, about 3× slower per request | 32 GiB with `DISABLE_MM=1`; 64 GB with the vision tower on | Out-of-memory restarts ~10 min into boot if host RAM is too small |
| Other NVIDIA GPUs (Hopper, Ampere, other Blackwell cards) | Not tested with the public images | — | May work; run steps 1–4 and compare with the ranges below before relying on it |

Latency on each: [From laptop to production](index.md) (RTX PRO 6000) and
[the L4 comparison](../../benchmarks/runs/20260925-serving-speed/README.md). The unquantized 16-bit weights
(`google/diffusiongemma-26B-A4B-it`, ~50 GiB) need about twice the GPU memory; we ran them on 2× A100-40GB
([benchmark report](../benchmarks-report.md)), not with the public images' defaults.

## 1. Start the container and wait for readiness

```bash
docker run --gpus all -p 8080:8080 \
  us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem:v0.1.0@sha256:5fa4a866163169aaf91c3bdb727020ff3c86e26e84ffadad86bc859053d45b26
```

- The lean image downloads the weights from Hugging Face at boot. To avoid downloading on every start, download
  them once (`huggingface-cli download nvidia/diffusiongemma-26B-A4B-it-NVFP4 --local-dir /data/dgemma`) and
  mount them: `-v /data/dgemma:/mnt/gcs/dgemma:ro`. Hosts without internet access can use the `dgem-weights`
  image instead ([public images](public-images.md)).
- On an L4 add `-e DISABLE_MM=1` (and see the host RAM note above).
- Leave `API_KEY` unset for the evaluation, or reach the container by host name rather than `localhost` when you
  use the gateway in step 5 (the gateway does not forward keys to loopback addresses).

Poll health until it is ready:

```bash
curl -s http://<GPU_HOST>:8080/health
# booting: {"phase": "...", "vllm_ready": false, ...}
# ready:   {"status": "ok", "vllm_ready": true, "phase": "ready", "warmed": true, "version": "v0.1.0", ...}
```

**Pass:** `vllm_ready` and `warmed` are both `true`. Record `version`, `revision` and `vllm_commit` from this
response; they identify the build you evaluated.

## 2. Make a first decision

Build the CLI (`make build`, needs Go), then:

```bash
./bin/dgem decide -u http://<GPU_HOST>:8080/v1 \
  -t templates/support_triage.json.tmpl \
  -v 'ticket=Urgent: production database connection pool exhausted after deploy' \
  -v samples=1 --stats
```

Run it three or four times and read the last runs.

**Pass:**

- Three answers (`sentiment`, `team`, `urgent`), each with a confidence, and no errors.
- **Server Denoise** (GPU time) of roughly 55–65 ms on an RTX PRO 6000 (we measured 55–61 ms p50 for a
  3-question, 1-sample request); expect about 3× that on an L4. End-to-end time adds your network round trip.

If the request fails with connection refused or `429`, the engine is still loading: go back to step 1.

## 3. Baseline against public benchmarks

These suites ship with the repository and have known results on the same image, so they separate an install
problem from a model limitation.

```bash
./bin/dgem bench-jev -u http://<GPU_HOST>:8080/v1 -w 4 -o jevbench_run1.json
./bin/dgem bench-calibration -u http://<GPU_HOST>:8080/v1 -w 4 -o calibration_run1.json
```

**Pass:**

| Suite | Items | Our runs on RTX PRO 6000 (`samples=1`, 4 workers) | Investigate if |
| :--- | ---: | :--- | :--- |
| JevBench | 231 | 183–192 correct (16 runs; mean ~187) | below ~180, or any request errors |
| Calibration suite | 50 | 43–45 correct (16 runs) | below ~42, or any request errors |

Source: [image parity run](../../benchmarks/runs/20260927-image-parity/README.md#accuracy-and-calibration-jevbench__json-calibration__json).
Individual items can change between runs (roughly 5% of items flip between repeated runs), so compare totals, not
item by item, and run twice if a result lands at the edge of the range. We have not measured these suites on an
L4 with the current image; results should be close, but treat the ranges as RTX PRO 6000 figures.

A score well below the range with no errors usually means the wrong weights or a modified serving image;
errors usually mean the engine is overloaded or still loading.

## 4. Concurrency (optional)

Repeat step 3 with the concurrency you expect in production, for example `-w 16`.

**Pass:** 0 errors and the same score range. The server processes up to `MAX_INFLIGHT` (default 8) decisions at a
time and queues the rest; it returns `503` if a request waits in the queue for more than 30 s. Throughput and
latency under load, and how to tune them: [Latency and capacity](../operate/latency-capacity.md).

## 5. Evaluate your own decisions

With the install verified, measure the model on your task.

### 5.1 Design your policy

A policy (a `.json.tmpl` file) lists the questions to answer about each input and the allowed answers; it is
what you are really evaluating. The [Policies & Decisions](../policies/first-policy.md) docs cover it in depth:

| If you want to… | Read |
| :--- | :--- |
| Check that your use case fits a decision model | [Real-world applications](../policies/applications.md), especially the [decision matrix](../policies/applications.md#7-architecture-decision-matrix): bounded answers fit; free-form text, conversation and code generation do not |
| Start from an existing policy instead of a blank file | [Template catalog](../policies/templates.md): a [creator's guide](../policies/templates.md#0-gentle-template-intro--template-creators-guide), then ready-made policies for triage, routing and intent, vision, guardrails and reranking |
| Write your first policy step by step | [Your first decision policy](../policies/first-policy.md): write it, run it, read hesitation, add samples and conditional questions |
| Make it fast and reliable | [Authoring guide](../policies/authoring.md#2-designing-jsontmpl-policy-schemas--prefix-cache-optimization): prefix caching, single-pass policies, option design |
| Co-design it with an AI assistant on sample rows | [Run a dataset, step 2](../policies/datasets.md#step-2-co-design-your-template-using-mcp-decide_custom_questions) (MCP `decide_custom_questions`) |

Rules that matter most for an evaluation:

- Keep the instructions and options fixed across all rows and put per-row data in `state`
  ([why](../policies/datasets.md#step-1-map-your-dataset-columns-to-a-jsontmpl-policy-schema)).
- Use `"samples": 1` unless you need an error bar per decision.
- A `choice` question has at most 26 options; for larger label sets, split into two questions (for example
  domain, then intent).
- `depends_on` / `ask_if` (ask a question only if another answer applies) costs a second forward pass; leave them
  out unless a question makes no sense otherwise.
- Evaluate the policy you will ship: changing wording, options or `samples` later means re-running 5.3–5.4.

### 5.2 Label a sample and hold part of it out

- Collect real inputs with the answer you consider correct, one JSON object per line: template variables plus
  `expected_<question id>` columns ([format](../confidence/calibrate-your-policy.md#1-label-a-sample)).
- Aim for **100+ labelled answers per question**, including the hard and ambiguous cases you actually see.
- Split it: a **development set** for rewriting the policy, and a **test set** you run only once at the end.
  Rewording options until the test set looks good overstates accuracy.

### 5.3 Run the dataset

Start a local gateway in front of your GPU host; it serves Decision Studio and the batch API:

```bash
./bin/dgem serve --port 8090 -u http://<GPU_HOST>:8080/v1
```

- **Up to ~250 rows:** open <http://localhost:8090>, **Batch Eval → Custom Template & Dataset**, paste the policy
  and upload the `.jsonl`. It shows accuracy per question live and exports results
  ([details](../policies/datasets.md#step-3-run-your-dataset-in-the-web-studio-httpsyour-dgem-gateway)).
- **Larger datasets:** use the Python runner in
  [Run a dataset, step 4](../policies/datasets.md#step-4-run-large-datasets-10010000-rows-from-python) with
  `GATEWAY_URL = "http://localhost:8090/api/decide"`. A local gateway needs no token: make
  `get_gcp_identity_token()` return `""`. It writes `results_receipt.jsonl`.

### 5.4 Read accuracy and calibration

```bash
python3 scripts/policy_calibration.py results_receipt.jsonl
```

This reports, per question, accuracy, how closely confidence matches accuracy (Brier score, ECE, a reliability
table), and what happens if every answer above a hesitation threshold goes to a person or a larger model. How to
read it and what to do about hesitant answers: [Calibrate your policy](../confidence/calibrate-your-policy.md).

On the development set, most errors are policy wording rather than the model. Match what you see to a fix:

| What you see | What to do |
| :--- | :--- |
| Errors or hesitation cluster on one question | Rewrite that question and its option descriptions so they don't overlap ([authoring guide](../policies/authoring.md#2-designing-jsontmpl-policy-schemas--prefix-cache-optimization), [first policy §3](../policies/first-policy.md#3-read-the-uncertainty)) |
| Inputs that fit none of the options | Add a catch-all and let the model propose the missing options ([taxonomy discovery](../policies/taxonomy-discovery.md)) |
| Too many hesitant answers overall | Escalate only those to a larger model with a Stage 2 cascade ([authoring guide §1](../policies/authoring.md)); this calls Gemini on Vertex AI, so it needs a Google Cloud project |
| You need an error bar per decision | `"samples": 4` ([first policy §4](../policies/first-policy.md#4-take-more-samples-when-you-need-agreement)) |

Fix those on the development set, then run the test set once, with the final policy and any cascade enabled, so
the reported accuracy and escalation rate are what you would ship.

### 5.5 Record the result

A result is reproducible when it records:

- the image digest and `/health` `version` / `revision`, and the GPU type;
- the policy file and `samples` value;
- test-set size, accuracy per question, ECE, and the escalation rate at the hesitation threshold you chose;
- p50 and p95 latency at your production concurrency.

Re-run steps 3 and 5.4 after changing the image, the policy or `samples`: a new serving image can change a few
percent of individual answers even when overall accuracy is unchanged.

## Next

- Go deeper on policies: [your first decision policy](../policies/first-policy.md),
  [authoring and Stage 2 cascades](../policies/authoring.md), [run a dataset](../policies/datasets.md),
  [template catalog](../policies/templates.md), [real-world applications](../policies/applications.md) and
  [taxonomy discovery](../policies/taxonomy-discovery.md).
- Run it in production: [Cloud Run](cloud-run.md) (scale to zero) or [Vertex AI](vertex.md) (always warm), with
  a [gateway](gateway.md) in front.
- Operate it: [runbook](../operate/runbook.md), including how to validate a new serving image before switching
  traffic.
