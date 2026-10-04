# PROP-20: served temperature per question type (2026-10-04)

Leave-one-suite-out on 8 development suites from the matrix v2 reference run (production v0.2.1): fit on 7 suites,
score the 8th (10-bin ECE, AUROC). Candidates: raw (T = 1), one global T, one T per question type (yes/no, choice ≤ 5,
choice 6–26, choice > 26, score). `fit.py` reproduces `loso_results.json` from the run's per-item receipts.

| suite | raw ECE | global T | per-type T |
|---|---|---|---|
| jev_systemone | 0.070 | 0.050 | 0.102 |
| calib_systemone | 0.062 | 0.076 | 0.110 |
| intents_systemone | 0.119 | 0.059 | 0.078 |
| di_wide | 0.083 | 0.119 | 0.070 |
| di_catchall | 0.066 | 0.288 | 0.140 |
| rag_dev | 0.187 | 0.155 | 0.071 |
| gate_mixed_noul | 0.314 | 0.277 | 0.177 |
| massive_spot | 0.019 | 0.072 | 0.034 |

Both candidates are worse than raw on 4 of 8 suites, so neither meets the pre-registered rule (≥ 30% ECE cut on every
suite). Result: no served default; a per-policy `temperature` option instead.
