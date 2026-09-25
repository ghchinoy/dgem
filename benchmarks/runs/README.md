# Versioned benchmark runs

Each subdirectory is one **run**: a set of receipts produced in one session against one backend, so results
inside a run are directly comparable. Receipts are immutable; a new session gets a new run id
(`<YYYYMMDD>-<label>`).

```
benchmarks/runs/<run_id>/manifest.json            git commit, notes, and one entry per receipt
benchmarks/runs/<run_id>/<suite>__<config>.json   receipt written by `dgem bench-*`
```

Each manifest entry records the suite, config, exact command, SHA-256 of the receipt, and headline
numbers recomputed at T=1 (accuracy, Brier, 10-bin ECE, correct answers above 0.9 confidence, mean Mirror TVD).

| Run | What it is |
| :--- | :--- |
| `20260920-legacy` | Receipts produced before versioning (Sep 20–24, 2026), registered in place. Mixed endpoints and service revisions. |
| `20260925-offline-prop01` | PROP-01 held-out temperature scaling, computed offline from legacy receipts. |
| `20260925-serving-speed` | Serving speed review: Vertex L4 vs Vertex G4 (RTX PRO 6000) vs Cloud Run; per-mode latency and concurrency sweeps (`scripts/serving_speed.py`). |
| `20260925-g4-idc` | IDC configurations re-run on Vertex G4 with the `__rev` mirror suffix (replicates EXP-14). |
| `20260925-vertex-idc` | Same-session IDC re-run on the Vertex endpoint (calibration suite ×7 configs incl. 3 baselines, EXP-13 permutation suite, JevBench ×4 configs, Gemini Stage-2 for every item). Dual-mirror receipts here use the old `__mirror_rev` slot id. |
| `20260925-vertex-idc-mirrorfix` | Same session, after renaming the mirror slot to `__rev` (commit `2f731b0`), with fresh baselines. |

Write-up: [`docs/experiments/exp-14-idc-rerun.md`](../../docs/experiments/exp-14-idc-rerun.md).

Tools:

```bash
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
