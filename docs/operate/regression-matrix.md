---
title: "Regression Matrix"
description: "The standard benchmark matrix we run against dgem serving endpoints: tiers by change type, how to run and read it, pass rules based on measured noise, frozen held-out sets, and the published reference ranges."
---

# Regression Matrix

One command runs a fixed, versioned set of benchmarks against one or more dgem serving endpoints and says whether
anything changed: `scripts/bench_matrix.py`. Use it for every serving-image change, release, configuration change
and experiment that could move answers, and to check a self-hosted install against our numbers
([Evaluate dgem on your own GPU](../deploy/evaluate.md)).

The matrix is defined in [`benchmarks/matrix/matrix_v1.json`](../../benchmarks/matrix/matrix_v1.json). Changing
what a tier contains means a new matrix version, never an edit to a past run.

## Which tier to run

| Tier | When | Suites | Size (per target, RTX PRO 6000) |
| :--- | :--- | :--- | :--- |
| **T0 smoke** | Every deploy or configuration change; daily on production | `/health`, API contract (incl. multilingual cases with known answers), calibration suite ×1, multilingual spot check (MASSIVE **validation**, ru/th/hi/ja/es × 20) | ~200 requests, under 1 minute |
| **T1 gate** | Serving-image, prompt-path or server changes; release validation; weekly on production | T0 + calibration ×3, JevBench ×3 through `bench-jev` (chat/completions) and ×3 through `/v1/systemone`, intents slices ×3 (banking77, clinc150), the [Decision Index adapter track](#decision-index-adapter-track) (probes + wide-option MASSIVE ×3), latency (5 modes × 100, sweep at 16 and 32 workers) | ~3,500 requests, ~12 minutes |
| **T2 full** | Release candidates, vLLM or model changes; never automatic; needs `--confirm` | T1 + MASSIVE test (51 languages × 100), XNLI test (15 × 300), typed-decisions test (2,000 decisions), option order (739 cases × 4 orderings), bounding boxes, Decision Index panel, and the Decision Index kit's compatibility pass when configured | ~17,000 requests, ~25 minutes |
| **T-cal** | Calibration or threshold changes | Every report computes ECE and NLL with a 5-fold **held-out** temperature from the saved receipts; no extra requests | — |

Excluded from v1 (and why) is listed under `excluded` in the matrix file: `bench-rerank` cannot target a Vertex
endpoint yet, `bench-ecotone` needs its sidecar, and `bench-permutation` is covered by the option-order suite.

## Run it

```bash
make build                                   # bin/dgem: the dgem bench-* suites shell out to it
scripts/bench_matrix.py list                 # tiers and suites

# Compare a candidate with production in the same session
scripts/bench_matrix.py run --tier T1 \
  --target prod=<PRODUCTION_URL> --target new=<CANDIDATE_URL> --baseline prod --label image-<TAG>

# One target against the published reference ranges (e.g. your own GPU)
scripts/bench_matrix.py run --tier T0 --target gpu=http://<GPU_HOST>:8080

# Full matrix (reads parquet for some sets)
pip install -r scripts/requirements-matrix.txt
scripts/bench_matrix.py fetch                # download and verify the pinned datasets once
scripts/bench_matrix.py run --tier T2 --confirm --target prod=<PRODUCTION_URL>
```

- **Target URLs:** a Vertex dedicated endpoint invoke base
  (`https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke`),
  a Cloud Run URL (tagged revision URLs work, so a `--no-traffic` canary can be tested), or `http://<GPU_HOST>:8080`.
- **Auth:** Vertex gets an access token and Cloud Run an ID token, from the metadata server, Application Default
  Credentials or gcloud. Self-hosted servers get none, unless you set `DGEM_MATRIX_TOKEN` (for example the
  server's `API_KEY`).
- **Interleaving:** with two targets, every repeated suite alternates targets run by run, so both see the same
  conditions. Latency runs last, so it is not measured under the matrix's own load.
- **Request options (A/B on one deployment):** append `#option=value` to a target URL to add a request option to
  every `/v1/systemone` and chat body that target gets, and the matching flag to its local adapter. Today:
  `layout=document_first` or `layout=schema_first` (`--prompt-layout` on the adapter; needs a serving image with
  the `layout` field, default `document_first` from v0.2.0). For example `--target base=<URL> --target old=<URL>#layout=schema_first
  --baseline base`. Template suites (`dgem decide`) don't get the option.
- **Resuming:** an interrupted run continues with `--resume` and the same `--label`.
- **Exit code:** 1 if any target's overall verdict is FAIL, so the command can gate a script.

### Output

Each run is a directory, `benchmarks/runs/<YYYYMMDD>-<label>/` by default (`--out-dir` to change it):

| File | Contents |
| :--- | :--- |
| `manifest.json` | matrix version, tier, targets with the `version` / `revision` / `vllm_commit` each reported at `/health`, git commit, one entry per receipt with its SHA-256 (compatible with `scripts/bench_runs.py list`) |
| `<suite>__<target>__r<N>.json` | receipts: the `dgem bench-*` receipt as written, or per-item `/v1/systemone` rows |
| `report.md` | verdicts, per-suite tables, the measured noise floor, raw vs held-out calibration, per-language tables, latency |
| `summary.json` | the machine-readable result ([schema](../../benchmarks/matrix/summary.schema.json), `dgem.matrix.summary/v2`): verdicts, the noise floor, and per suite and target every run's accuracy, coverage, macro-F1, ECE, Brier, NLL, AUROC and latency, plus held-out calibration and reliability bins; option-order flip rates; latency modes and sweep. Dashboards and automation read this file, not `report.md` |

Receipts and logs are redacted as they are written: endpoint URLs become `<kind:name>` and tokens `<TOKEN>`, so a
run directory can be committed. Run `make check-public` before committing one anyway.

### Metrics in the report

- **Case-exact:** for suites with several questions per case (typed-decisions), the share of cases with every field
  right. Field accuracy can hide this: on one Decision Index benchmark, 77% of fields were right but only 1% of cases.

- **Coverage:** answered / attempted items. Refusals (for example HTTP 422 when a request exceeds a capacity limit) and
  errors count as unanswered, the way the Decision Index scores them. Accuracy and the calibration metrics are computed
  over answered items, so read them together with coverage.
- **Macro-F1:** F1 averaged over gold labels, with labels kept separate per question in multi-question suites. Intent
  benchmarks such as BANKING77 and CLINC150 are usually reported this way, so use it when comparing with published
  numbers; accuracy alone favours frequent labels.
- **ECE, Brier, NLL, AUROC:** confidence quality: how well stated confidence matches accuracy (ECE), proper scores for
  the whole distribution (Brier, NLL), and how well confidence separates right from wrong answers (AUROC). Held-out ECE
  uses a temperature fitted on other folds.

## How verdicts are decided

The model is not deterministic: about 5–7% of answers change between identical requests on the same image. A
single run's score therefore says little on its own. The matrix measures that noise in every run and judges
changes against it.

| Gate | PASS when | Otherwise |
| :--- | :--- | :--- |
| Health | `/health` is `ok` and vLLM is ready | FAIL |
| Contract | every case returns the expected status; no multilingual case answers wrong; status codes match the baseline | FAIL |
| Accuracy suites (baseline mode) | answer agreement between candidate and baseline is at least the measured within-target agreement minus 2 points, **and** a paired McNemar test is not significant (p ≥ 0.01) | REVIEW if one fails, FAIL if both |
| Accuracy suites (single target) | mean accuracy at or above the reference range | REVIEW within 2% below it, FAIL further below |
| Multilingual spot check | every language at least 70% | FAIL |
| Coverage | every accuracy suite answers as many items as the baseline (baseline mode) or at least the reference coverage (single target). Unanswered items are classified as `context` (prompt longer than the served context), `capacity` (server shape limits), `na` (skipped by the matrix) or `error`, in the report and in `summary.json` (`refusals`) | REVIEW |
| Option order | net flip rate (permuted minus identical-repeat) no more than 0.05 above the baseline | REVIEW |
| Latency | every mode's p50 within ×1.10 of the baseline, throughput at least ×0.90, 0 errors | REVIEW up to ×1.25, FAIL beyond or on errors |
| Bounding boxes, Decision Index | reported, not gated (INFO) | — |

Thresholds live in `matrix_v1.json` (`thresholds`). Look at a REVIEW before promoting; a FAIL blocks.

Accuracy is computed over answered items, so the coverage gate is what catches an item that starts being refused, for
example a prompt that a new prompt layout pushes past the served context length. Under v0.2.0 at a 4,096-token context
one JevBench `long_policy` item is refused (`context`); deployments at 8,192 tokens (the default from EXP-20) answer it.

## Decision Index adapter track

Decision Index runs go through `dgem systemone serve`, the adapter that splits questions with more than 26 options
into brackets, batches more than 8 questions per request, and turns capacity limits into HTTP 422 responses the kit
records as *unsupported*. The matrix starts that adapter locally, from this checkout's `bin/dgem`, in front of each
target. These suites therefore test the adapter code you are about to ship against the deployed model:

| Suite | Checks |
| :--- | :--- |
| `di_probes` | Wide options at K = 27, 41, 61, 101, 151 and 255 (right answer; probabilities for every option, summing to 1); a 12-question request (slot batching); option descriptions given as JSON objects and arrays (as the Decision Index sends for POP909, ChessBench and cfcolor); a ~9k-token and a ~31k-token input that must be answered, or refused with 422 plus a kit marker (`maximum context length`, `options per choice`, …). Any miss is a FAIL. Two informational probes are reported but not gated: a yes/no question whose meaning is defined only in its true/false descriptions, and a 151-option question with an "out of scope" catch-all. It also reports, as INFO, the top probability on unambiguous wide-option items: a flat ceiling across K would mean bracket fusion is capping confidence (before #50 a fixed 92/8 split capped it at about 0.92) |
| `di_wide` | MASSIVE validation (English) with the full ~60-label set as options, 100 items ×3, judged like the other accuracy suites (reference 0.83–0.89; accuracy does not depend on the probability fusion, which keeps the final-round winner) |
| `di_catchall` | CLINC150 **validation** in the Decision Index request format: 151 options including an "out of scope" catch-all, 80 in-scope + 20 out-of-scope items ×2. Reports (INFO) how often in-scope requests are answered "out of scope", the bracket-routing weakness behind CLINC150 in the Decision Index run. `dgem systemone serve --catch-all final` trades out-of-scope recall for in-scope accuracy |
| `rag_dev` | RAGTruth **train**, 198 items ×2, with the Decision Index hallucination question and its true/false criteria. Reports (INFO) F1 and recall on the hallucinated class against the always-flag baseline, the yes/no "no" bias. `--noul-mode choice` raises it on this family |
| `di_kit_compat` (T2) | The Decision Index kit's own compatibility pass (86 requests) through the adapter. Runs only when `DGEM_DI_KIT_DIR` (a kit checkout with its `.venv`) and `DGEM_DI_COMPAT_ROWS` (compatibility rows built from a rebuilt suite) are set; otherwise reported as skipped. FAIL on any `error` row |

## Frozen held-out sets

MASSIVE test, XNLI test, typed-decisions test and the option-order suites run only in T2. They are **scored only
after a change ships**, never used to tune prompts, wording or thresholds. Development and wording work uses
JevBench, the calibration and intents suites, and validation splits (the multilingual spot check reads MASSIVE
*validation*). This keeps the T2 numbers honest: a change that only helps because it was tuned on the test items
cannot show up there.

The runtime datasets are listed in [`benchmarks/matrix/datasets.lock.json`](../../benchmarks/matrix/datasets.lock.json),
each file pinned to a Hugging Face dataset commit and checked against its SHA-256 before use. The files are cached in
`~/.cache/dgem-matrix` (`DGEM_MATRIX_CACHE`) and never committed. Licences: MASSIVE CC BY 4.0, typed-decisions
Apache-2.0, XNLI CC BY-NC 4.0, dair-ai/emotion per its dataset card.

## Reference ranges (v0.2.0, Vertex G4)

Measured on 2026-10-02 with the v0.2.0 server (prompt layout `document_first`) on Vertex G4 (`g4-standard-48`,
1× RTX PRO 6000), `samples=1`, in two T2 sessions:
[production](../../benchmarks/runs/20261002-v020-reference-t2/report.md) and the
[release-gate canary](../../benchmarks/runs/20261002-v020-release-t2/report.md), which also compares v0.2.0 with
v0.1.3 item by item. The ranges are stored in `matrix_v1.json` (`reference`) and used in single-target mode. The v0.1.3
reference is in [T1](../../benchmarks/runs/20261001-v013-reference-t1/report.md) and
[T2](../../benchmarks/runs/20261001-v013-reference-t2/report.md).

| Suite | Items | Runs | Accuracy range (v0.1.3) | ECE raw → held-out |
| :--- | ---: | ---: | :--- | :--- |
| Calibration suite (`bench-calibration`) | 50 | 6 | 0.860–0.900 (0.860–0.880) | 0.106 → 0.104 |
| JevBench (`bench-jev`, chat/completions) | 231 | 6 | 0.800–0.817 (0.792–0.823) | 0.102 → 0.040 |
| JevBench (`/v1/systemone`, native types) | 231 | 6 | 0.827–0.857 (0.823–0.862) | 0.085 → 0.038 |
| Intents, banking77 slice (30 options, bracket) | 30 | 6 | 0.767–0.833 (0.733–0.833) | — |
| Intents, clinc150 slice | 30 | 6 | 0.967–1.000 (same) | — |
| Multilingual spot check (MASSIVE validation, 5 languages) | 100 | 2 | 0.96 (0.83–0.85) | 0.036 → 0.031 |
| Decision Index format, wide options (`di_wide`) | 100 | 6 | 0.850–0.890 (0.830–0.890) | 0.070 → 0.035 |
| CLINC150 validation, 151 options (`di_catchall`) | 100 | 6 | 0.850–0.880 (0.691–0.859) | 0.125 → 0.085 |
| RAGTruth train (`rag_dev`) | 198 | 6 | 0.758–0.773 (0.674–0.800) | 0.198 → 0.039 |
| MASSIVE test, 51 languages, 20 options *(frozen)* | 5,100 | 2 | 0.824–0.826 (0.822) | 0.103 → 0.046 |
| XNLI test, 15 languages *(frozen)* | 4,500 | 2 | 0.697–0.699 (0.678) | 0.227 → 0.036 |
| typed-decisions test, 2,000 decisions *(frozen)* | 2,000 | 2 | 0.723–0.728 (0.667) | 0.177 → 0.035 |

Frozen sets store min..max of the two sessions ± 2 SE as their range. One JevBench chat item (`long_policy`) now
exceeds a 4,096-token context by a few tokens with the new layout, so `jev_native` answers 230 of 231 there (231 at
the 8,192-token default from EXP-20).

**Noise floor** (answer agreement between identical repeated runs, same image; v0.2.0 production): calibration 0.99,
JevBench 0.95–0.97, clinc150 0.98, banking77 0.96 (0.82–0.86 on v0.1.3), Decision Index suites 0.94–0.95.
Accuracy is judged against these measured values in every run, not against a fixed band.

**Calibration needs a per-domain temperature.** With v0.2.0 the held-out temperature is about 1.2–1.9 on JevBench,
MASSIVE, the wide-option suites and the calibration suite, but 2.6 on typed-decisions, 3.5 on RAGTruth and 3.6 on XNLI,
where raw confidence is much too high. A single
global temperature does not fit every domain; fit one on held-out data from your own decisions.

**Option order** (T2, net flip = shuffled minus identical repeat): MASSIVE-en +0.065, emotion +0.050, XNLI-en
+0.025, JevBench choice +0.029. On 6-option emotion the first-shown option is picked 42.5% of the time against a
gold rate of 30.5%.

**Latency** (v0.2.0, `/v1/systemone`, client in the same region, keep-alive): p50 wall / server time
92 / 54 ms (1 question), 99 / 58 ms (5 questions), 103 / 62 ms (10), 111 / 63 ms (long policy state),
136 / 96 ms (5 questions, `samples: 4`); about 60–64 requests/s at 16 and 32 concurrent requests with 0 errors
(unchanged from v0.1.3).
Latency depends on hardware and network, so single-target runs report it but gate only on errors.

Bounding boxes (12 cases): acc@IoU 0.5 = 81.8%, mean IoU 0.61 (v0.1.3: 36.4%, 0.41). Decision Index panel (22
requests): 96.7 (v0.1.3: 99.4 in the same session).

## Scheduled runs (our deployment)

Our production endpoints run T0 daily and T1 weekly as Cloud Run jobs
([`scripts/deploy_bench_matrix_job.sh`](../../scripts/deploy_bench_matrix_job.sh)). Each run writes its directory to
a Cloud Storage bucket and one JSON log line per event (`matrix_event`: `dgem.matrix.start`, `dgem.matrix.suite`,
`dgem.matrix.gate`, `dgem.matrix.done`), so log-based alerts can fire on a FAIL. Scheduled runs are single-target
(compared with the reference ranges) and are not committed. T2 is never scheduled.

To see how production has moved over time, collect the scheduled runs into a time series:

```bash
scripts/matrix_trends.py --source gs://<bucket> --out trends/     # trends.json (series per suite and gate) + trends.md
```

`trends.json` holds, per tier, suite and target, every run's accuracy, coverage, macro-F1, ECE, Brier, AUROC, verdict
and the image version the target reported; and per gate the verdict history. Dashboards read that file. Runs written
before summary schema v2 contribute verdicts only. The job image includes the verified dataset files T0 and T1 read,
because anonymous Hugging Face downloads from cloud egress are rate-limited (`HF_TOKEN`, if set, is used for any
other download). `scripts/setup_alerts.py` creates the alerts: a FAIL verdict on any gate, no completed T0 run for 25 h, and no T1
execution for 7 days + 8 h ([Monitoring and alerts](monitoring.md#alerts-what-they-mean-and-what-to-do)).

## Related

- [Operations runbook: promote a new serving image](runbook.md#promote-a-new-serving-image) (uses T1)
- [Evaluate dgem on your own GPU](../deploy/evaluate.md) (uses T0 and T1 in single-target mode)
- [Versioned benchmark runs](../../benchmarks/runs/README.md)
- [Monitoring and alerts](monitoring.md) (the hourly synthetic probe is separate and lighter than T0)
