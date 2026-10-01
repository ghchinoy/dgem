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
| **T1 gate** | Serving-image, prompt-path or server changes; release validation; weekly on production | T0 + calibration ×3, JevBench ×3 through `bench-jev` (chat/completions) and ×3 through `/v1/systemone`, intents slices ×3 (banking77, clinc150), latency (5 modes × 100, sweep at 16 and 32 workers) | ~3,000 requests, ~10 minutes |
| **T2 full** | Release candidates, vLLM or model changes; never automatic; needs `--confirm` | T1 + MASSIVE test (51 languages × 100), XNLI test (15 × 300), typed-decisions test (2,000 decisions), option order (739 cases × 4 orderings), bounding boxes, Decision Index | ~16,500 requests, ~25 minutes |
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
- **Resuming:** an interrupted run continues with `--resume` and the same `--label`.
- **Exit code:** 1 if any target's overall verdict is FAIL, so the command can gate a script.

### Output

Each run is a directory, `benchmarks/runs/<YYYYMMDD>-<label>/` by default (`--out-dir` to change it):

| File | Contents |
| :--- | :--- |
| `manifest.json` | matrix version, tier, targets with the `version` / `revision` / `vllm_commit` each reported at `/health`, git commit, one entry per receipt with its SHA-256 (compatible with `scripts/bench_runs.py list`) |
| `<suite>__<target>__r<N>.json` | receipts: the `dgem bench-*` receipt as written, or per-item `/v1/systemone` rows |
| `report.md` | verdicts, per-suite tables, the measured noise floor, raw vs held-out calibration, per-language tables, latency |
| `summary.json` | verdicts and headline numbers, for automation |

Receipts and logs are redacted as they are written: endpoint URLs become `<kind:name>` and tokens `<TOKEN>`, so a
run directory can be committed. Run `make check-public` before committing one anyway.

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
| Option order | net flip rate (permuted minus identical-repeat) no more than 0.05 above the baseline | REVIEW |
| Latency | every mode's p50 within ×1.10 of the baseline, throughput at least ×0.90, 0 errors | REVIEW up to ×1.25, FAIL beyond or on errors |
| Bounding boxes, Decision Index | reported, not gated (INFO) | — |

Thresholds live in `matrix_v1.json` (`thresholds`). Look at a REVIEW before promoting; a FAIL blocks.

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

## Reference ranges (v0.1.3, Vertex G4)

Measured on 2026-10-01 against v0.1.3 on Vertex G4 (`g4-standard-48`, 1× RTX PRO 6000), `samples=1`:
[T1, v0.1.3 against itself](../../benchmarks/runs/20261001-v013-reference-t1/report.md) (the self-comparison passes
every gate, which also checks the verdict rules) and [T2](../../benchmarks/runs/20261001-v013-reference-t2/report.md).
The ranges are stored in `matrix_v1.json` (`reference`) and used in single-target mode.

| Suite | Items | Runs | Accuracy range | ECE raw → held-out |
| :--- | ---: | ---: | :--- | :--- |
| Calibration suite (`bench-calibration`) | 50 | 9 | 0.860–0.880 (43–44 correct) | 0.123 → 0.119 |
| JevBench (`bench-jev`, chat/completions) | 231 | 9 | 0.792–0.823 (183–190) | 0.070 → 0.042 |
| JevBench (`/v1/systemone`, native types) | 231 | 9 | 0.823–0.862 (190–199) | 0.082 → 0.065 |
| Intents, banking77 slice (30 options, bracket) | 30 | 7 | 0.733–0.833 | — |
| Intents, clinc150 slice | 30 | 7 | 0.967–1.000 | — |
| Multilingual spot check (MASSIVE validation, 5 languages) | 100 | 3 | 0.830–0.850 (every language ≥ 0.75) | — |
| MASSIVE test, 51 languages, 20 options *(frozen)* | 5,100 | 1 | 0.822 (± 2 SE: 0.811–0.832) | 0.080 → 0.036 |
| XNLI test, 15 languages *(frozen)* | 4,500 | 1 | 0.678 (0.664–0.692) | 0.245 → 0.022 |
| typed-decisions test, 2,000 decisions *(frozen)* | 2,000 | 1 | 0.667 (0.646–0.688); soft accuracy 0.554 | 0.232 → 0.017 |

**Noise floor** (answer agreement between identical repeated runs, same image): calibration 0.99–1.00, JevBench
0.93–0.96, clinc150 0.98, **banking77 0.82–0.86** (the two-stage bracket for 30 options is the least stable suite).
Accuracy is judged against these measured values in every run, not against a fixed band.

**Calibration needs a per-domain temperature.** The held-out temperature is about 1.5 on JevBench, MASSIVE and the
calibration suite, but about 3.4 on typed-decisions and 4.0 on XNLI, where raw confidence is much too high. A single
global temperature does not fit every domain; fit one on held-out data from your own decisions.

**Option order** (T2, net flip = shuffled minus identical repeat): MASSIVE-en +0.065, emotion +0.050, XNLI-en
+0.025, JevBench choice +0.029. On 6-option emotion the first-shown option is picked 42.5% of the time against a
gold rate of 30.5%.

**Latency** (T1, `/v1/systemone`, client in the same region, keep-alive): p50 wall / server time
92 / 54 ms (1 question), 95 / 57 ms (5 questions), 101 / 60 ms (10), 111 / 62 ms (long policy state),
134 / 95 ms (5 questions, `samples: 4`); about 62–66 requests/s at 16 and 32 concurrent requests with 0 errors.
Latency depends on hardware and network, so single-target runs report it but gate only on errors.

Bounding boxes (12 cases): acc@IoU 0.5 = 36.4%, mean IoU 0.41. Decision Index panel (22 requests): 98.9.

## Scheduled runs (our deployment)

Our production endpoints run T0 daily and T1 weekly as Cloud Run jobs
([`scripts/deploy_bench_matrix_job.sh`](../../scripts/deploy_bench_matrix_job.sh)). Each run writes its directory to
a Cloud Storage bucket and one JSON log line per event (`matrix_event`: `dgem.matrix.start`, `dgem.matrix.suite`,
`dgem.matrix.gate`, `dgem.matrix.done`), so log-based alerts can fire on a FAIL. Scheduled runs are single-target
(compared with the reference ranges) and are not committed. T2 is never scheduled.

## Related

- [Operations runbook: promote a new serving image](runbook.md#promote-a-new-serving-image) (uses T1)
- [Evaluate dgem on your own GPU](../deploy/evaluate.md) (uses T0 and T1 in single-target mode)
- [Versioned benchmark runs](../../benchmarks/runs/README.md)
- [Monitoring and alerts](monitoring.md) (the hourly synthetic probe is separate and lighter than T0)
