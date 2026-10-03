# Regression matrix report: 20261002-v020-canary-t2

- Matrix **v1**, tier **T2**, started 2026-10-02T21:19:25+00:00, finished 2026-10-02T21:43:39+00:00
- Repository commit `7d36527583`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | v0.1.3 | e685ef9 | a9eafde59c |
| new | vertex | v0.1.5-22-ge37012a | e37012a | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| health | new | **PASS** | ready |
| contract | new | **PASS** | all cases as expected |
| calibration | new | **REVIEW** | acc 0.860 vs 0.873; agreement 0.860 (floor 0.980, allowance 0.060); McNemar p=0.8 — agreement 0.860 < noise floor 0.980 - 0.060 |
| jev_native | new | **REVIEW** | acc 0.812 vs 0.807; agreement 0.869 (floor 0.968, allowance 0.043); McNemar p=1 — agreement 0.869 < noise floor 0.968 - 0.043 |
| jev_systemone | new | **REVIEW** | acc 0.848 vs 0.840; agreement 0.890 (floor 0.954, allowance 0.048); McNemar p=0.53 — agreement 0.890 < noise floor 0.954 - 0.048 |
| intents_banking77 | new | **PASS** | acc 0.778 vs 0.800; agreement 0.900 (floor 0.889, allowance 0.135); McNemar p=0.69 |
| intents_clinc150 | new | **PASS** | acc 0.967 vs 0.989; agreement 0.978 (floor 0.989, allowance 0.058); McNemar p=0.5 |
| massive_spot | new | **PASS** | es 19/20, hi 19/20, ja 19/20, ru 20/20, th 19/20 |
| di_wide | new | **PASS** | acc 0.867 vs 0.847; agreement 0.926 (floor 0.967, allowance 0.056); McNemar p=0.29 |
| di_catchall | new | **FAIL** | acc 0.863 vs 0.740; agreement 0.776 (floor 0.882, allowance 0.085); McNemar p=4.3e-07 — agreement 0.776 < noise floor 0.882 - 0.085; McNemar p=4.3e-07 (46 vs 9) |
| rag_dev | new | **FAIL** | acc 0.771 vs 0.712; agreement 0.858 (floor 0.929, allowance 0.056); McNemar p=0.00015 — agreement 0.858 < noise floor 0.929 - 0.056; McNemar p=0.00015 (59 vs 24) |
| massive | new | **REVIEW** | acc 0.824 vs 0.820; agreement 0.845 (floor 0.945, allowance 0.026); McNemar p=0.44 [frozen set] — agreement 0.845 < noise floor 0.945 - 0.026 |
| xnli | new | **FAIL** | acc 0.699 vs 0.662; agreement 0.890 (floor 0.945, allowance 0.027); McNemar p=9.4e-15 [frozen set] — agreement 0.890 < noise floor 0.945 - 0.027; McNemar p=9.4e-15 (318 vs 151) |
| typed | new | **FAIL** | acc 0.728 vs 0.674; agreement 0.739 (floor 0.945, allowance 0.030); McNemar p=6.6e-07 [frozen set] — agreement 0.739 < noise floor 0.945 - 0.030; McNemar p=6.6e-07 (278 vs 172) |
| coverage | new | **REVIEW** | fewer answered items: jev_native 230 vs 231 answered (context 3) |
| order | new | **PASS** | net flip vs baseline: within tolerance |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 23.3%; out-of-scope recall 96.7% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.544 (always-flag 0.601), recall 40.0%, predicted yes 20.2% vs gold 42.9% |
| di_catchall_oos_rate | new | **INFO** | in-scope answered as the catch-all: 1.7%; out-of-scope recall 86.7% (pooled runs) |
| rag_dev_yes_bias | new | **INFO** | hallucinated-class F1 0.681 (always-flag 0.601), recall 56.9%, predicted yes 28.8% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.816–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | new | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | new | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | new | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | new | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| di_kit_compat | new | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=36.364, mean_expectation_iou=0.412 |
| bbox | new | **INFO** | acc_at_50_expectation_pct=81.818, mean_expectation_iou=0.611 |
| decision_index | prod | **INFO** | headline_decision_index=99.444, ece_10bin=0.018 |
| decision_index | new | **INFO** | headline_decision_index=96.667, ece_10bin=0.049 |
| latency | new | **PASS** | within tolerance |

Overall: **new: FAIL**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 45, 43, 43 / 50 | 1.000 | — | 0.873 | 0.717 | 0.084 | 0.218 | 0.895 | 219 |
| calibration | new | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.664 | 0.091 | 0.269 | 0.811 | 213 |
| jev_native | prod | 185, 186, 188 / 231 | 1.000 | — | 0.807 | 0.780 | 0.077 | 0.261 | 0.871 | 129 |
| jev_native | new | 186, 186, 188 / 230 | 0.996 | context 3 | 0.812 | 0.808 | 0.088 | 0.283 | 0.801 | 131 |
| jev_systemone | prod | 198, 192, 192 / 231 | 1.000 | — | 0.840 | 0.797 | 0.086 | 0.252 | 0.761 | 228 |
| jev_systemone | new | 196, 195, 197 / 231 | 1.000 | — | 0.848 | 0.807 | 0.066 | 0.229 | 0.843 | 230 |
| intents_banking77 | prod | 24, 24, 24 / 30 | 1.000 | — | 0.800 | 0.741 | 0.174 | 0.377 | 0.819 | 425 |
| intents_banking77 | new | 24, 23, 23 / 30 | 1.000 | — | 0.778 | 0.713 | 0.154 | 0.333 | 0.722 | 411 |
| intents_clinc150 | prod | 29, 30, 30 / 30 | 1.000 | — | 0.989 | 0.983 | 0.043 | 0.039 | 0.966 | 138 |
| intents_clinc150 | new | 29, 29, 29 / 30 | 1.000 | — | 0.967 | 0.949 | 0.044 | 0.052 | 0.931 | 152 |
| massive_spot | prod | 83 / 100 | 1.000 | — | 0.830 | 0.796 | 0.109 | 0.243 | 0.906 | 223 |
| massive_spot | new | 96 / 100 | 1.000 | — | 0.960 | 0.961 | 0.026 | 0.071 | 0.826 | 254 |
| di_wide | prod | 85, 85, 84 / 100 | 1.000 | — | 0.847 | 0.820 | 0.091 | 0.248 | 0.846 | 310 |
| di_wide | new | 86, 85, 89 / 100 | 1.000 | — | 0.867 | 0.826 | 0.085 | 0.199 | 0.917 | 440 |
| di_catchall | prod | 70, 76, 76 / 100 | 1.000 | — | 0.740 | 0.646 | 0.117 | 0.413 | 0.720 | 798 |
| di_catchall | new | 85, 87, 87 / 100 | 1.000 | — | 0.863 | 0.808 | 0.085 | 0.244 | 0.650 | 803 |
| rag_dev | prod | 138, 142, 143 / 198 | 1.000 | — | 0.712 | 0.666 | 0.199 | 0.479 | 0.616 | 257 |
| rag_dev | new | 153, 152, 153 / 198 | 1.000 | — | 0.771 | 0.751 | 0.188 | 0.410 | 0.634 | 289 |
| massive | prod | 4184 / 5100 | 1.000 | — | 0.820 | 0.786 | 0.082 | 0.274 | 0.903 | 210 |
| massive | new | 4202 / 5100 | 1.000 | — | 0.824 | 0.792 | 0.104 | 0.275 | 0.918 | 243 |
| xnli | prod | 2979 / 4500 | 1.000 | — | 0.662 | 0.664 | 0.263 | 0.587 | 0.625 | 199 |
| xnli | new | 3146 / 4500 | 1.000 | — | 0.699 | 0.703 | 0.229 | 0.527 | 0.611 | 206 |
| typed | prod | 1349 / 2000 | 1.000 | — | 0.674 | 0.545 | 0.226 | 0.537 | 0.740 | 265 |
| typed | new | 1455 / 2000 | 1.000 | — | 0.728 | 0.607 | 0.176 | 0.437 | 0.760 | 310 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | mean |
|---|---|---|---|---|---|---|---|---|---|
| prod | 0.960 | 0.954 | 0.951 | 0.867 | 0.978 | 0.980 | 0.817 | 0.923 | 0.929 |
| new | 1.000 | 0.983 | 0.957 | 0.911 | 1.000 | 0.953 | 0.947 | 0.936 | 0.961 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.072 | 0.070 | 0.347 | 0.345 | 1.25, 1.10, 1.15, 1.20, 1.15 |
| calibration | new | 50 | 0.109 | 0.061 | 0.479 | 0.469 | 1.50, 1.15, 1.50, 1.55, 1.60 |
| jev_native | prod | 231 | 0.082 | 0.036 | 0.508 | 0.467 | 1.50, 1.45, 1.60, 1.50, 1.50 |
| jev_native | new | 230 | 0.092 | 0.040 | 0.603 | 0.509 | 1.80, 1.75, 1.80, 1.80, 1.75 |
| jev_systemone | prod | 231 | 0.082 | 0.055 | 0.579 | 0.491 | 1.75, 1.60, 1.80, 1.70, 1.75 |
| jev_systemone | new | 231 | 0.073 | 0.033 | 0.489 | 0.433 | 1.60, 1.55, 1.70, 1.60, 1.60 |
| intents_banking77 | prod | 29 | 0.183 | 0.139 | 0.904 | 0.723 | 1.75, 1.70, 1.75, 2.00, 1.85 |
| intents_banking77 | new | 29 | 0.153 | 0.162 | 0.529 | 0.527 | 1.35, 1.10, 1.40, 1.50, 1.40 |
| intents_clinc150 | prod | 30 | 0.020 | 0.020 | 0.086 | 0.070 | 0.70, 0.50, 0.70, 0.60, 0.65 |
| intents_clinc150 | new | 30 | 0.058 | 0.021 | 0.070 | 0.024 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| massive_spot | prod | 100 | 0.109 | 0.086 | 0.582 | 0.516 | 1.50, 1.40, 1.45, 1.45, 1.40 |
| massive_spot | new | 100 | 0.026 | 0.042 | 0.261 | 0.273 | 1.35, 1.05, 1.25, 1.40, 1.45 |
| di_wide | prod | 100 | 0.093 | 0.078 | 0.688 | 0.657 | 1.20, 1.20, 1.30, 1.25, 1.30 |
| di_wide | new | 100 | 0.102 | 0.058 | 0.670 | 0.612 | 1.30, 1.30, 1.45, 1.40, 1.45 |
| di_catchall | prod | 99 | 0.131 | 0.115 | 1.298 | 1.258 | 1.20, 1.20, 1.25, 1.25, 1.15 |
| di_catchall | new | 100 | 0.086 | 0.093 | 1.016 | 0.951 | 1.25, 1.30, 1.35, 1.30, 1.30 |
| rag_dev | prod | 198 | 0.192 | 0.030 | 0.883 | 0.590 | 3.55, 3.65, 3.15, 3.60, 3.50 |
| rag_dev | new | 198 | 0.192 | 0.032 | 0.891 | 0.515 | 3.65, 3.70, 3.55, 3.55, 3.55 |
| massive | prod | 5098 | 0.082 | 0.033 | 0.885 | 0.764 | 1.55, 1.50, 1.55, 1.50, 1.50 |
| massive | new | 5098 | 0.103 | 0.050 | 0.898 | 0.701 | 1.75, 1.75, 1.75, 1.70, 1.75 |
| xnli | prod | 4469 | 0.258 | 0.023 | 1.661 | 0.812 | 4.25, 4.30, 4.25, 4.20, 4.15 |
| xnli | new | 4494 | 0.228 | 0.029 | 1.445 | 0.772 | 3.55, 3.60, 3.60, 3.55, 3.55 |
| typed | prod | 1980 | 0.220 | 0.011 | 1.280 | 0.750 | 3.40, 3.40, 3.50, 3.55, 3.40 |
| typed | new | 1998 | 0.175 | 0.024 | 0.988 | 0.663 | 2.70, 2.65, 2.75, 2.75, 2.65 |

## massive per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| af | 0.72 / 0.86 | 0.76 / 0.90 |
| am | 0.82 / 0.90 | 0.77 / 0.92 |
| ar | 0.88 / 0.92 | 0.86 / 0.93 |
| az | 0.80 / 0.88 | 0.80 / 0.93 |
| bn | 0.81 / 0.91 | 0.78 / 0.92 |
| cy | 0.42 / 0.69 | 0.41 / 0.72 |
| da | 0.86 / 0.92 | 0.86 / 0.94 |
| de | 0.88 / 0.94 | 0.90 / 0.96 |
| el | 0.85 / 0.92 | 0.86 / 0.94 |
| en | 0.87 / 0.95 | 0.91 / 0.98 |
| es | 0.87 / 0.94 | 0.86 / 0.95 |
| fa | 0.88 / 0.92 | 0.94 / 0.96 |
| fi | 0.84 / 0.90 | 0.83 / 0.94 |
| fr | 0.85 / 0.92 | 0.85 / 0.94 |
| he | 0.85 / 0.91 | 0.87 / 0.94 |
| hi | 0.90 / 0.94 | 0.94 / 0.96 |
| hu | 0.70 / 0.81 | 0.74 / 0.89 |
| hy | 0.76 / 0.87 | 0.78 / 0.90 |
| id | 0.88 / 0.96 | 0.89 / 0.96 |
| is | 0.74 / 0.87 | 0.72 / 0.91 |
| it | 0.93 / 0.96 | 0.92 / 0.96 |
| ja | 0.94 / 0.95 | 0.96 / 0.96 |
| jv | 0.72 / 0.84 | 0.66 / 0.88 |
| ka | 0.76 / 0.87 | 0.74 / 0.91 |
| km | 0.78 / 0.89 | 0.75 / 0.93 |
| kn | 0.80 / 0.90 | 0.83 / 0.93 |
| ko | 0.89 / 0.94 | 0.89 / 0.96 |
| lv | 0.69 / 0.84 | 0.63 / 0.82 |
| ml | 0.88 / 0.93 | 0.89 / 0.94 |
| mn | 0.77 / 0.83 | 0.70 / 0.88 |
| ms | 0.83 / 0.93 | 0.87 / 0.93 |
| my | 0.87 / 0.91 | 0.89 / 0.94 |
| nb | 0.84 / 0.92 | 0.83 / 0.93 |
| nl | 0.83 / 0.91 | 0.86 / 0.95 |
| pl | 0.88 / 0.93 | 0.88 / 0.96 |
| pt | 0.83 / 0.94 | 0.85 / 0.96 |
| ro | 0.83 / 0.89 | 0.81 / 0.90 |
| ru | 0.87 / 0.95 | 0.92 / 0.96 |
| sl | 0.71 / 0.83 | 0.72 / 0.91 |
| sq | 0.65 / 0.83 | 0.65 / 0.85 |
| sv | 0.81 / 0.91 | 0.84 / 0.93 |
| sw | 0.77 / 0.87 | 0.73 / 0.88 |
| ta | 0.84 / 0.90 | 0.87 / 0.93 |
| te | 0.84 / 0.92 | 0.86 / 0.94 |
| th | 0.88 / 0.94 | 0.93 / 0.96 |
| tl | 0.85 / 0.93 | 0.83 / 0.94 |
| tr | 0.87 / 0.92 | 0.87 / 0.94 |
| ur | 0.84 / 0.91 | 0.84 / 0.94 |
| vi | 0.86 / 0.92 | 0.87 / 0.95 |
| zh | 0.90 / 0.93 | 0.90 / 0.97 |
| **macro** | **0.819** | **0.822** |

## xnli per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| ar | 0.67 / 0.93 | 0.69 / 0.92 |
| bg | 0.66 / 0.91 | 0.72 / 0.90 |
| de | 0.70 / 0.92 | 0.78 / 0.93 |
| el | 0.67 / 0.92 | 0.71 / 0.93 |
| en | 0.82 / 0.93 | 0.84 / 0.94 |
| es | 0.69 / 0.93 | 0.73 / 0.94 |
| fr | 0.71 / 0.93 | 0.71 / 0.94 |
| hi | 0.60 / 0.92 | 0.66 / 0.93 |
| ru | 0.64 / 0.93 | 0.67 / 0.93 |
| sw | 0.60 / 0.92 | 0.63 / 0.92 |
| th | 0.64 / 0.93 | 0.66 / 0.94 |
| tr | 0.63 / 0.92 | 0.69 / 0.92 |
| ur | 0.63 / 0.92 | 0.65 / 0.92 |
| vi | 0.63 / 0.92 | 0.67 / 0.93 |
| zh | 0.63 / 0.92 | 0.69 / 0.92 |
| **macro** | **0.662** | **0.699** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.674 | 0.553 | 0.305 | 0.416 |
| new | 0.728 | 0.575 | 0.264 | 0.426 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.075 | 0.110 (+0.035) | 0.135 (+0.060) | 0.440 / 0.305 |
| prod | order_jev_choice | 139 | 0.043 | 0.108 (+0.065) | 0.094 (+0.050) | 0.223 / 0.216 |
| prod | order_massive_en | 200 | 0.015 | 0.070 (+0.055) | 0.065 (+0.050) | 0.050 / 0.045 |
| prod | order_xnli_en | 200 | 0.055 | 0.080 (+0.025) | 0.065 (+0.010) | 0.295 / 0.335 |
| new | order_emotion | 200 | 0.025 | 0.075 (+0.050) | 0.075 (+0.050) | 0.450 / 0.305 |
| new | order_jev_choice | 139 | 0.050 | 0.122 (+0.072) | 0.101 (+0.050) | 0.237 / 0.216 |
| new | order_massive_en | 200 | 0.020 | 0.075 (+0.055) | 0.070 (+0.050) | 0.045 / 0.045 |
| new | order_xnli_en | 200 | 0.050 | 0.065 (+0.015) | 0.045 (-0.005) | 0.280 / 0.335 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.983 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.948 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.816 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.986 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.988 |
| prod | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.972 |
| prod | batch_12q | 200 | yes |  |
| prod | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| prod | noul_criteria | 200 | yes |  |
| prod | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.994 |
| prod | long_9k | 422 | yes | marker=maximum context length |
| prod | context_refusal | 422 | yes | marker=maximum context length |
| new | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| new | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| new | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| new | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| new | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| new | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| new | batch_12q | 200 | yes |  |
| new | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| new | noul_criteria | 200 | yes |  |
| new | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
| new | long_9k | 422 | yes | marker=maximum context length |
| new | context_refusal | 422 | yes | marker=maximum context length |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 91.9 | 95.6 | 100.5 | 54.5 | — |
| prod | q5 | 100/100 | 98.3 | 174.9 | 227.3 | 57.1 | — |
| prod | q10 | 100/100 | 101.2 | 105.7 | 116.9 | 59.8 | — |
| prod | long_q1 | 100/100 | 109.8 | 118.7 | 189.6 | 60.5 | — |
| prod | q5_s4 | 100/100 | 135.1 | 140.9 | 155.2 | 95.4 | — |
| prod | sweep q1/w16 | 256/256 | 219.5 | 228.3 | 520.6 | 109.1 | 66.4 |
| prod | sweep q1/w32 | 256/256 | 462.0 | 662.1 | 956.1 | 114.5 | 59.4 |
| prod | sweep q5/w16 | 256/256 | 228.6 | 245.6 | 801.2 | 113.3 | 60.0 |
| prod | sweep q5/w32 | 256/256 | 449.8 | 644.0 | 998.0 | 111.3 | 60.9 |
| new | q1 | 100/100 | 91.4 | 100.0 | 115.4 | 53.8 | — |
| new | q5 | 100/100 | 95.3 | 105.5 | 123.2 | 57.4 | — |
| new | q10 | 100/100 | 102.4 | 110.8 | 123.8 | 61.2 | — |
| new | long_q1 | 100/100 | 111.3 | 117.5 | 128.3 | 62.9 | — |
| new | q5_s4 | 100/100 | 134.6 | 138.8 | 143.7 | 95.6 | — |
| new | sweep q1/w16 | 256/256 | 218.7 | 261.1 | 554.9 | 108.5 | 65.7 |
| new | sweep q1/w32 | 256/256 | 449.8 | 606.5 | 962.0 | 111.6 | 61.4 |
| new | sweep q5/w16 | 256/256 | 220.9 | 255.9 | 563.5 | 109.9 | 65.1 |
| new | sweep q5/w32 | 256/256 | 443.8 | 693.7 | 932.0 | 110.3 | 62.0 |

