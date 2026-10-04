# Versioned benchmark runs

Each subdirectory is one **run**: a set of receipts produced in one session against one backend, so results
inside a run are directly comparable. Receipts are immutable; a new session gets a new run id
(`<YYYYMMDD>-<label>`).

```
benchmarks/runs/<run_id>/manifest.json            git commit, notes, and one entry per receipt
benchmarks/runs/<run_id>/<suite>__<config>.json   receipt written by `dgem bench-*`
```

Each manifest entry records the suite, config, exact command, SHA-256 of the receipt, the Vertex endpoint state at record
time for Vertex runs (`backend_state`: machine, accelerator, min/max/available replicas, image), and headline
numbers recomputed at T=1 (accuracy, Brier, 10-bin ECE, correct answers above 0.9 confidence, mean Mirror TVD).

**What gets committed (from 2026-10-03):** `manifest.json`, `report.md`, `summary.json` and receipts up to about
2 MB each. Larger raw per-item outputs go to `gs://<PROJECT>-dgem-matrix/receipts/<run_id>/`, with a committed
`receipts.lock.json` listing each file's path, size and SHA-256, so the run stays reproducible and verifiable. Runs
committed before that date keep their raw receipts in place. `*.log` files are never committed.
`scripts/bench_matrix.py fetch-receipts benchmarks/runs/<run_id>` restores the bucket copies (size and SHA-256
checked against the lock file; `DGEM_MATRIX_BUCKET=<PROJECT>-dgem-matrix`); add a run-level `.gitignore` for them so
they are not re-committed. `report` names the command when a receipt is missing, and judges a run against the matrix
version recorded in its manifest unless `--matrix` is given.

| Run | What it is |
| :--- | :--- |
| `20260920-legacy` | Receipts produced before versioning (Sep 20–24, 2026), registered in place. Mixed endpoints and service revisions. |
| `20260925-offline-prop01` | PROP-01 held-out temperature scaling, computed offline from legacy receipts. |
| `20260925-serving-speed` | Serving speed review: Vertex L4 vs Vertex G4 (RTX PRO 6000) vs Cloud Run; per-mode latency and concurrency sweeps (`scripts/serving_speed.py`). |
| `20260925-g4-idc` | IDC configurations re-run on Vertex G4 with the `__rev` mirror suffix (replicates EXP-14). |
| `20260925-vertex-idc` | Same-session IDC re-run on the Vertex endpoint (calibration suite ×7 configs incl. 3 baselines, EXP-13 permutation suite, JevBench ×4 configs, Gemini Stage-2 for every item). Dual-mirror receipts here use the old `__mirror_rev` slot id. |
| `20260925-parity-g4-vs-cloudrun` | Parity probe: Vertex G4 vs Cloud Run (same image, RTX PRO 6000), JevBench ×3 and 50-item suite ×2. Vertex ≥ Cloud Run; JevBench noise ±3 items. |
| `20260926-prop12-separate-pass` | PROP-12 / EXP-17: forward ×2 and reversed (`--flip-options`) ×2 on JevBench, Vertex G4. |
| `20260926-prop16-slot-names` | PROP-16 / EXP-16: slot-id variants (single and two-slot copy) on JevBench, Vertex G4. |
| `20260925-prop11-letter-collision` | PROP-11 / EXP-15: 3 baselines + 4 `--mirror-mode` conditions on JevBench, Vertex G4. |
| `20260925-vertex-idc-mirrorfix` | Same session, after renaming the mirror slot to `__rev` (commit `2f731b0`), with fresh baselines. |
| `20260927-image-parity` | Serving-image parity: old production vs two candidate images on Vertex G4 and Cloud Run, same session (JevBench ×3, calibration ×3, latency, sweep, cold start). The method the regression matrix automates. |
| `20260928-v010-verification` | Every suite once against v0.1.0 on Vertex G4, plus a rollback drill (README only, no manifest). |
| `20261001-v013-reference-t1` | **Regression matrix v1, T1**, v0.1.3 against itself on Vertex G4: the measured noise floor and reference ranges. |
| `20261001-v013-reference-t2` | **Regression matrix v1, T2**, v0.1.3 on Vertex G4: full matrix incl. the frozen multilingual, typed-decisions and option-order sets. |
| `20261003-exp09-rebaseline` | EXP-09 re-baseline (PROP-18 phase 1): the 12-case bbox suite on Vertex G4 v0.2.0 with every probe variant ×3, the 2026-09-21 Cloud Run receipt re-analyzed, and a simulated self-check (`scripts/run_exp09_rebaseline.sh`). Write-up: [`exp-09-spatial-grounding.md`](../../docs/experiments/exp-09-spatial-grounding.md). |
| `20261003-prop18-gemini` | PROP-18 phase 2 (EXP-22): Gemini 3.8/3.7-flash reference boxes on the EXP-09 fixtures (7 variants ×3) and Gemini judge validation (`bench-bbox-judge`; `_rescored` = box-level metric fixed). |
| `20261003-prop18-sweep` | PROP-18 phase 3 (EXP-22): generated 230-item sweep (`scripts/generate_bbox_sweep.py`), dgem ×2 and Gemini 3.8. |
| `20261003-prop18-real` | Superseded by `-real-v2` (duplicate ScreenSpot ids). |
| `20261003-prop18-real-v2` | PROP-18 phase 3 (EXP-22): RefCOCO + ScreenSpot sample (`scripts/fetch_bbox_real.py`), dgem ×2 and Gemini 3.8. |
| `20261003-prop18-detectors` | PROP-18 phase 3 (EXP-22): OWLv2, Grounding DINO, Grounding DINO + SAM on a deleted-after-use L4 VM (`scripts/run_detectors_gce.sh`); raw predictions + scored receipts. |
| `20261003-prop18-vision` | PROP-18 phase 4 pilot (EXP-22): `bench-vision` multi-aspect questions on the sweep: dgem ×2, dgem blank-image prior, Gemini 3.8. |
| `20261003-prop18-vision-real` | PROP-18 (EXP-22 §5): `bench-vision` on RefCOCO + ScreenSpot (`benchmarks/bbox_real_aspects.jsonl`): dgem ×2, blank prior, Gemini 3.8. |
| `20261003-prop18-cascade` | PROP-18 (EXP-22 §6): live Stage-2 cascade with the image, thresholds 0.35 / 0.10 nats, six vs scored questions (`scripts/run_prop18_cascade.sh`). |
| `20261003-vision-gate-ref` | Regression matrix T1 `health` + `vision_spot` + `vision` on production v0.2.0: the first image-gate run (part of the vision reference ranges). |
| `20261003-compare-strands-v19` | **EXP-23** model comparison: dgem (Vertex G4, v0.2.0) vs Strands Decider 2B v19 (Cloud Run, RTX PRO 6000) in matrix v2 format (tier TC suites plus frozen sets, option order, latency, post-hoc `gate_diag_*`), exported from the study harness. Write-up: [`exp-23-strands-decider.md`](../../docs/experiments/exp-23-strands-decider.md). |
| `20261003-exp24-guided` | EXP-24: dgem-guided Gemini 3.8 boxes (full/hint/crop × thinking) and masks (Gemini polygons; SAM from Gemini/Grounding DINO/ground-truth boxes on a deleted-after-use VM). |
| `20261004-exp26-domains` | EXP-26 (#76): guided boxes (Gemini 3.8 and 3.7) and `bench-vision` on mobile UI, web UI, DocLayNet pages and synthetic PCBs; `decision.json` holds the pre-registered rule outcomes. `vision__vertex_g4_x2.json` (2.8 MB) is in the matrix bucket (`receipts.lock.json`). |
| `20261004-locate-acceptance` | #74 acceptance of guided locate: the EXP-24 items (432) through `POST /api/locate` against production. Recorded with the **pre-EXP-26 defaults** (hint on, skip at 0.16) under concurrent EXP-26 load; mIoU 0.626 on 228 positives, 139 requests skipped Gemini. The shipped defaults (hint and skip off) came later. |

Write-up: [`docs/experiments/exp-14-idc-rerun.md`](../../docs/experiments/exp-14-idc-rerun.md).

Runs made with `scripts/bench_matrix.py` (the [regression matrix](../../docs/operate/regression-matrix.md)) add
`report.md` and `summary.json` with verdicts. Model comparisons are named `<YYYYMMDD>-compare-<model>` and follow
[Comparing dgem with Another Decision Model](../../docs/operate/model-comparison.md); re-runs of a published experiment
are named `<YYYYMMDD>-rerun-<exp>`.

Tools:

```bash
scripts/bench_matrix.py report benchmarks/runs/<run_id>   # rebuild a matrix report
python3 scripts/bench_runs.py list                      # runs and receipts
python3 scripts/bench_runs.py compare --suite jevbench  # comparison table across runs
python3 scripts/analyze_idc.py cv-temperature <receipt...>
python3 scripts/analyze_idc.py gates --stage1 <receipt> --stage2 <receipt>
python3 scripts/analyze_idc.py merge-rules <dual-mirror receipt>
RUN_ID=<id> ./scripts/run_idc_rerun.sh                  # reproduce the IDC re-run
```

Receipts written with `--null-prior-debias` / `--dual-mirror` include a per-item `idc` block (raw and
post-processed forward/reversed distributions, Mirror TVD/JSD) so merge rules and gates can be re-scored
offline without re-running the model.
