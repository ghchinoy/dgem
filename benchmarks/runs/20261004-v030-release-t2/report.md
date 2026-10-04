# Regression matrix report: 20261003-v030-release-t2

- Matrix **v2**, tier **T2**, started 2026-10-03T23:04:48+00:00, finished 2026-10-03T23:49:03+00:00
- Repository commit `c88cb0be42`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | v0.2.1 | dec8ff8 | a9eafde59c |
| new | vertex | v0.2.2-3-gc88cb0b | c88cb0b | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| health | new | **PASS** | ready |
| contract | new | **PASS** | all cases as expected |
| calibration | new | **PASS** | acc 0.867 vs 0.860; agreement 0.973 (floor 0.973, allowance 0.066); McNemar p=1 |
| massive_spot | new | **PASS** | es 19/20, hi 20/20, ja 19/20, ru 20/20, th 19/20 |
| jev_native | new | **PASS** | acc 0.808 vs 0.802; agreement 0.955 (floor 0.961, allowance 0.045); McNemar p=0.56 |
| jev_systemone | new | **PASS** | acc 0.847 vs 0.841; agreement 0.940 (floor 0.943, allowance 0.051); McNemar p=0.61 |
| intents_banking77 | new | **PASS** | acc 0.800 vs 0.822; agreement 0.930 (floor 0.922, allowance 0.118); McNemar p=0.69 |
| intents_clinc150 | new | **PASS** | acc 0.989 vs 0.967; agreement 0.978 (floor 0.989, allowance 0.058); McNemar p=0.5 |
| massive | new | **PASS** | acc 0.826 vs 0.825; agreement 0.950 (floor 0.948, allowance 0.026); McNemar p=0.87 [frozen set] |
| xnli | new | **PASS** | acc 0.697 vs 0.698; agreement 0.945 (floor 0.948, allowance 0.027); McNemar p=0.7 [frozen set] |
| typed | new | **REVIEW** | acc 0.707 vs 0.721; agreement 0.875 (floor 0.948, allowance 0.030); McNemar p=0.054 [frozen set] — agreement 0.875 < noise floor 0.948 - 0.030 |
| di_wide | new | **PASS** | acc 0.867 vs 0.860; agreement 0.947 (floor 0.943, allowance 0.066); McNemar p=0.73 |
| di_catchall | new | **PASS** | acc 0.877 vs 0.847; agreement 0.933 (floor 0.938, allowance 0.068); McNemar p=0.035 |
| rag_dev | new | **PASS** | acc 0.761 vs 0.751; agreement 0.936 (floor 0.939, allowance 0.054); McNemar p=0.42 |
| calib_systemone | new | **PASS** | acc 0.880 vs 0.873; agreement 0.967 (floor 0.960, allowance 0.075); McNemar p=1 |
| intents_systemone | new | **PASS** | acc 0.856 vs 0.850; agreement 0.952 (floor 0.944, allowance 0.079); McNemar p=1 |
| gate_mixed_noul | new | **FAIL** | acc 0.977 vs 0.601; agreement 0.600 (floor 0.925, allowance 0.072); McNemar p=1.8e-30 — agreement 0.600 < noise floor 0.925 - 0.072; McNemar p=1.8e-30 (119 vs 4) |
| vision_spot | new | **PASS** | acc 0.787 vs 0.787; agreement 0.943 (floor 0.948, allowance 0.058); McNemar p=1 |
| vision | new | **REVIEW** | acc 0.746 vs 0.762; agreement 0.887 (floor 0.932, allowance 0.037); McNemar p=0.038 — agreement 0.887 < noise floor 0.932 - 0.037 |
| coverage | new | **PASS** | no coverage loss |
| order | new | **PASS** | net flip vs baseline: within tolerance |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 4.2%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.648 (always-flag 0.601), recall 53.3%, predicted yes 27.8% vs gold 42.9% |
| di_catchall_oos_rate | new | **INFO** | in-scope answered as the catch-all: 2.1%; out-of-scope recall 88.3% (pooled runs) |
| rag_dev_yes_bias | new | **INFO** | hallucinated-class F1 0.665 (always-flag 0.601), recall 55.3%, predicted yes 28.5% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | new | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | new | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | new | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | new | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| di_kit_compat | new | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=90.909, mean_expectation_iou=0.642 |
| bbox | new | **INFO** | acc_at_50_expectation_pct=72.727, mean_expectation_iou=0.567 |
| decision_index | prod | **INFO** | headline_decision_index=96.667, ece_10bin=0.052 |
| decision_index | new | **INFO** | headline_decision_index=96.667, ece_10bin=0.053 |
| latency | new | **FAIL** | q5 p50 x1.28; q5_s4 p50 x1.23; q1/w16 throughput x0.82; q5/w16 throughput x0.80; q5/w32 throughput x0.79 |

Overall: **new: FAIL**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.669 | 0.091 | 0.259 | 0.837 | 211 |
| calibration | new | 43, 43, 44 / 50 | 1.000 | — | 0.867 | 0.669 | 0.090 | 0.255 | 0.828 | 210 |
| massive_spot | prod | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.031 | 0.070 | 0.821 | 293 |
| massive_spot | new | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.022 | 0.066 | 0.787 | 271 |
| jev_native | prod | 187, 185, 184 / 231 | 1.000 | — | 0.802 | 0.789 | 0.100 | 0.299 | 0.807 | 164 |
| jev_native | new | 189, 184, 187 / 231 | 1.000 | — | 0.808 | 0.794 | 0.093 | 0.285 | 0.802 | 134 |
| jev_systemone | prod | 197, 191, 195 / 231 | 1.000 | — | 0.841 | 0.807 | 0.071 | 0.230 | 0.867 | 357 |
| jev_systemone | new | 194, 196, 197 / 231 | 1.000 | — | 0.847 | 0.806 | 0.070 | 0.227 | 0.857 | 367 |
| intents_banking77 | prod | 24, 26, 24 / 30 | 1.000 | — | 0.822 | 0.772 | 0.164 | 0.366 | 0.607 | 483 |
| intents_banking77 | new | 26, 23, 23 / 30 | 1.000 | — | 0.800 | 0.741 | 0.188 | 0.362 | 0.597 | 464 |
| intents_clinc150 | prod | 29, 29, 29 / 30 | 1.000 | — | 0.967 | 0.949 | 0.052 | 0.048 | 0.943 | 213 |
| intents_clinc150 | new | 30, 30, 29 / 30 | 1.000 | — | 0.989 | 0.983 | 0.052 | 0.035 | 0.931 | 157 |
| massive | prod | 4208 / 5100 | 1.000 | — | 0.825 | 0.792 | 0.105 | 0.277 | 0.913 | 285 |
| massive | new | 4211 / 5100 | 1.000 | — | 0.826 | 0.794 | 0.104 | 0.276 | 0.914 | 317 |
| xnli | prod | 3143 / 4500 | 1.000 | — | 0.698 | 0.702 | 0.230 | 0.528 | 0.608 | 308 |
| xnli | new | 3136 / 4500 | 1.000 | — | 0.697 | 0.701 | 0.231 | 0.528 | 0.614 | 298 |
| typed | prod | 1442 / 2000 | 1.000 | — | 0.721 | 0.603 | 0.181 | 0.441 | 0.772 | 376 |
| typed | new | 1413 / 2000 | 1.000 | — | 0.707 | 0.583 | 0.185 | 0.477 | 0.713 | 474 |
| di_wide | prod | 88, 85, 85 / 100 | 1.000 | — | 0.860 | 0.813 | 0.089 | 0.203 | 0.933 | 475 |
| di_wide | new | 88, 85, 87 / 100 | 1.000 | — | 0.867 | 0.825 | 0.074 | 0.206 | 0.927 | 460 |
| di_catchall | prod | 86, 84, 84 / 100 | 1.000 | — | 0.847 | 0.795 | 0.094 | 0.276 | 0.659 | 829 |
| di_catchall | new | 88, 87, 88 / 100 | 1.000 | — | 0.877 | 0.830 | 0.088 | 0.238 | 0.612 | 839 |
| rag_dev | prod | 149, 151, 146 / 198 | 1.000 | — | 0.751 | 0.727 | 0.194 | 0.430 | 0.664 | 293 |
| rag_dev | new | 151, 152, 149 / 198 | 1.000 | — | 0.761 | 0.740 | 0.181 | 0.405 | 0.670 | 297 |
| calib_systemone | prod | 43, 43, 45 / 50 | 1.000 | — | 0.873 | 0.839 | 0.058 | 0.188 | 0.775 | 379 |
| calib_systemone | new | 45, 43, 44 / 50 | 1.000 | — | 0.880 | 0.858 | 0.074 | 0.196 | 0.742 | 388 |
| intents_systemone | prod | 51, 52, 50 / 60 | 1.000 | — | 0.850 | 0.801 | 0.097 | 0.204 | 0.943 | 373 |
| intents_systemone | new | 52, 48, 54 / 60 | 1.000 | — | 0.856 | 0.808 | 0.098 | 0.215 | 0.875 | 341 |
| gate_mixed_noul | prod | 65, 60, 59 / 102 | 1.000 | — | 0.601 | 0.555 | 0.309 | 0.633 | 0.710 | 418 |
| gate_mixed_noul | new | 99, 100, 100 / 102 | 1.000 | — | 0.977 | 0.977 | 0.015 | 0.039 | 0.960 | 347 |
| vision_spot | prod | 111 / 141 | 1.000 | — | 0.787 | 0.768 | 0.172 | 0.376 | 0.816 | 350 |
| vision_spot | new | 111 / 141 | 1.000 | — | 0.787 | 0.742 | 0.158 | 0.381 | 0.655 | 411 |
| vision | prod | 651, 649 / 853 | 1.000 | — | 0.762 | 0.669 | 0.185 | 0.407 | 0.800 | 357 |
| vision | new | 639, 633 / 853 | 1.000 | — | 0.746 | 0.642 | 0.165 | 0.412 | 0.730 | 379 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | calib_systemone | intents_systemone | gate_mixed_noul | vision | mean |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| prod | 0.960 | 0.957 | 0.935 | 0.933 | 1.000 | 0.967 | 0.927 | 0.939 | 0.960 | 0.967 | 0.863 | 0.946 | 0.946 |
| new | 0.987 | 0.965 | 0.951 | 0.911 | 0.978 | 0.920 | 0.950 | 0.939 | 0.960 | 0.922 | 0.987 | 0.918 | 0.949 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.121 | 0.091 | 0.455 | 0.503 | 1.55, 0.95, 1.50, 1.45, 1.50 |
| calibration | new | 50 | 0.104 | 0.085 | 0.440 | 0.439 | 1.45, 1.10, 1.45, 1.40, 1.50 |
| massive_spot | prod | 100 | 0.031 | 0.036 | 0.240 | 0.259 | 1.30, 1.00, 1.25, 1.35, 1.45 |
| massive_spot | new | 100 | 0.022 | 0.036 | 0.228 | 0.246 | 1.30, 1.00, 1.25, 1.35, 1.45 |
| jev_native | prod | 231 | 0.091 | 0.033 | 0.645 | 0.530 | 1.90, 1.85, 1.90, 1.80, 1.85 |
| jev_native | new | 231 | 0.090 | 0.060 | 0.637 | 0.525 | 1.85, 1.80, 1.90, 1.80, 1.85 |
| jev_systemone | prod | 231 | 0.065 | 0.059 | 0.457 | 0.413 | 1.50, 1.50, 1.65, 1.50, 1.55 |
| jev_systemone | new | 231 | 0.072 | 0.041 | 0.492 | 0.439 | 1.55, 1.55, 1.75, 1.50, 1.65 |
| intents_banking77 | prod | 29 | 0.199 | 0.185 | 0.932 | 0.748 | 1.70, 1.60, 1.75, 1.95, 1.80 |
| intents_banking77 | new | 29 | 0.163 | 0.094 | 0.855 | 0.684 | 1.70, 1.75, 1.60, 1.90, 1.80 |
| intents_clinc150 | prod | 30 | 0.074 | 0.031 | 0.090 | 0.036 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| intents_clinc150 | new | 30 | 0.032 | 0.056 | 0.118 | 0.121 | 0.75, 0.75, 0.50, 0.70, 0.70 |
| massive | prod | 5092 | 0.104 | 0.049 | 0.891 | 0.691 | 1.75, 1.75, 1.75, 1.75, 1.75 |
| massive | new | 5093 | 0.103 | 0.048 | 0.900 | 0.696 | 1.75, 1.75, 1.75, 1.75, 1.75 |
| xnli | prod | 4498 | 0.229 | 0.039 | 1.482 | 0.779 | 3.65, 3.70, 3.70, 3.65, 3.65 |
| xnli | new | 4498 | 0.230 | 0.038 | 1.466 | 0.776 | 3.60, 3.65, 3.65, 3.60, 3.60 |
| typed | prod | 1998 | 0.180 | 0.019 | 1.005 | 0.666 | 2.70, 2.70, 2.80, 2.75, 2.75 |
| typed | new | 2000 | 0.185 | 0.017 | 1.085 | 0.735 | 2.65, 2.60, 2.70, 2.65, 2.65 |
| di_wide | prod | 100 | 0.073 | 0.054 | 0.606 | 0.568 | 1.30, 1.20, 1.40, 1.40, 1.40 |
| di_wide | new | 100 | 0.067 | 0.079 | 0.625 | 0.567 | 1.30, 1.30, 1.40, 1.40, 1.45 |
| di_catchall | prod | 100 | 0.097 | 0.062 | 0.862 | 0.821 | 1.20, 1.20, 1.30, 1.25, 1.25 |
| di_catchall | new | 98 | 0.078 | 0.101 | 0.813 | 0.777 | 1.20, 1.25, 1.30, 1.20, 1.25 |
| rag_dev | prod | 198 | 0.197 | 0.027 | 0.959 | 0.535 | 3.75, 3.90, 3.60, 3.75, 4.00 |
| rag_dev | new | 198 | 0.181 | 0.036 | 0.918 | 0.530 | 3.65, 3.90, 3.45, 3.60, 3.60 |
| calib_systemone | prod | 50 | 0.088 | 0.076 | 0.416 | 0.417 | 1.45, 1.10, 1.45, 1.40, 1.30 |
| calib_systemone | new | 50 | 0.073 | 0.059 | 0.363 | 0.425 | 1.30, 0.80, 1.25, 1.25, 1.30 |
| intents_systemone | prod | 59 | 0.099 | 0.055 | 0.488 | 0.439 | 1.50, 1.45, 1.30, 1.55, 1.55 |
| intents_systemone | new | 60 | 0.103 | 0.061 | 0.647 | 0.535 | 1.70, 1.60, 1.50, 1.60, 1.70 |
| gate_mixed_noul | prod | 102 | 0.283 | 0.117 | 1.040 | 0.623 | 5.00, 4.60, 4.65, 3.85, 4.50 |
| gate_mixed_noul | new | 102 | 0.009 | 0.032 | 0.082 | 0.140 | 1.20, 1.25, 0.50, 1.25, 1.25 |
| vision_spot | prod | 141 | 0.172 | 0.042 | 1.030 | 0.600 | 2.75, 2.70, 2.50, 2.85, 2.75 |
| vision_spot | new | 141 | 0.158 | 0.040 | 0.975 | 0.667 | 2.35, 2.25, 2.20, 2.50, 2.45 |
| vision | prod | 853 | 0.187 | 0.047 | 1.045 | 0.619 | 2.70, 2.75, 2.70, 2.75, 2.80 |
| vision | new | 853 | 0.159 | 0.039 | 0.892 | 0.651 | 2.10, 2.15, 2.20, 2.20, 2.20 |

## massive per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| af | 0.75 / 0.89 | 0.77 / 0.90 |
| am | 0.75 / 0.91 | 0.76 / 0.91 |
| ar | 0.86 / 0.92 | 0.88 / 0.93 |
| az | 0.80 / 0.92 | 0.80 / 0.92 |
| bn | 0.79 / 0.93 | 0.81 / 0.91 |
| cy | 0.47 / 0.74 | 0.47 / 0.72 |
| da | 0.86 / 0.95 | 0.86 / 0.94 |
| de | 0.90 / 0.96 | 0.89 / 0.97 |
| el | 0.87 / 0.95 | 0.86 / 0.94 |
| en | 0.90 / 0.97 | 0.91 / 0.97 |
| es | 0.85 / 0.96 | 0.85 / 0.96 |
| fa | 0.95 / 0.96 | 0.93 / 0.97 |
| fi | 0.85 / 0.94 | 0.83 / 0.94 |
| fr | 0.89 / 0.94 | 0.89 / 0.94 |
| he | 0.88 / 0.94 | 0.88 / 0.94 |
| hi | 0.94 / 0.96 | 0.94 / 0.96 |
| hu | 0.72 / 0.89 | 0.72 / 0.88 |
| hy | 0.77 / 0.91 | 0.79 / 0.91 |
| id | 0.89 / 0.96 | 0.88 / 0.95 |
| is | 0.75 / 0.91 | 0.72 / 0.92 |
| it | 0.91 / 0.96 | 0.92 / 0.97 |
| ja | 0.96 / 0.96 | 0.96 / 0.96 |
| jv | 0.66 / 0.88 | 0.67 / 0.87 |
| ka | 0.71 / 0.91 | 0.73 / 0.90 |
| km | 0.75 / 0.92 | 0.75 / 0.93 |
| kn | 0.78 / 0.92 | 0.82 / 0.93 |
| ko | 0.90 / 0.98 | 0.89 / 0.96 |
| lv | 0.63 / 0.86 | 0.64 / 0.85 |
| ml | 0.90 / 0.95 | 0.90 / 0.95 |
| mn | 0.70 / 0.89 | 0.68 / 0.89 |
| ms | 0.87 / 0.95 | 0.83 / 0.95 |
| my | 0.88 / 0.94 | 0.90 / 0.95 |
| nb | 0.85 / 0.94 | 0.85 / 0.94 |
| nl | 0.88 / 0.94 | 0.86 / 0.95 |
| pl | 0.88 / 0.96 | 0.91 / 0.96 |
| pt | 0.85 / 0.96 | 0.85 / 0.97 |
| ro | 0.83 / 0.92 | 0.82 / 0.91 |
| ru | 0.93 / 0.97 | 0.94 / 0.96 |
| sl | 0.73 / 0.89 | 0.72 / 0.90 |
| sq | 0.62 / 0.85 | 0.60 / 0.86 |
| sv | 0.83 / 0.93 | 0.83 / 0.94 |
| sw | 0.73 / 0.89 | 0.73 / 0.88 |
| ta | 0.86 / 0.94 | 0.88 / 0.94 |
| te | 0.86 / 0.93 | 0.85 / 0.94 |
| th | 0.93 / 0.97 | 0.92 / 0.97 |
| tl | 0.83 / 0.95 | 0.85 / 0.94 |
| tr | 0.89 / 0.95 | 0.86 / 0.95 |
| ur | 0.85 / 0.93 | 0.85 / 0.93 |
| vi | 0.85 / 0.95 | 0.87 / 0.95 |
| zh | 0.90 / 0.97 | 0.90 / 0.96 |
| **macro** | **0.824** | **0.824** |

## xnli per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| ar | 0.69 / 0.93 | 0.69 / 0.93 |
| bg | 0.72 / 0.90 | 0.73 / 0.91 |
| de | 0.79 / 0.93 | 0.76 / 0.92 |
| el | 0.71 / 0.93 | 0.73 / 0.93 |
| en | 0.83 / 0.95 | 0.81 / 0.95 |
| es | 0.71 / 0.93 | 0.72 / 0.93 |
| fr | 0.74 / 0.94 | 0.73 / 0.95 |
| hi | 0.67 / 0.93 | 0.65 / 0.93 |
| ru | 0.64 / 0.93 | 0.64 / 0.92 |
| sw | 0.61 / 0.92 | 0.65 / 0.92 |
| th | 0.65 / 0.92 | 0.66 / 0.92 |
| tr | 0.69 / 0.92 | 0.68 / 0.93 |
| ur | 0.66 / 0.92 | 0.63 / 0.92 |
| vi | 0.69 / 0.93 | 0.67 / 0.93 |
| zh | 0.69 / 0.92 | 0.68 / 0.93 |
| **macro** | **0.698** | **0.697** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.721 | 0.574 | 0.265 | 0.429 |
| new | 0.707 | 0.561 | 0.275 | 0.504 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.030 | 0.095 (+0.065) | 0.070 (+0.040) | 0.445 / 0.305 |
| prod | order_jev_choice | 139 | 0.065 | 0.122 (+0.058) | 0.115 (+0.050) | 0.237 / 0.216 |
| prod | order_massive_en | 200 | 0.005 | 0.060 (+0.055) | 0.075 (+0.070) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.040 | 0.075 (+0.035) | 0.080 (+0.040) | 0.265 / 0.335 |
| new | order_emotion | 200 | 0.045 | 0.095 (+0.050) | 0.090 (+0.045) | 0.455 / 0.305 |
| new | order_jev_choice | 139 | 0.050 | 0.115 (+0.065) | 0.151 (+0.101) | 0.223 / 0.216 |
| new | order_massive_en | 200 | 0.015 | 0.080 (+0.065) | 0.075 (+0.060) | 0.045 / 0.045 |
| new | order_xnli_en | 200 | 0.020 | 0.055 (+0.035) | 0.060 (+0.040) | 0.285 / 0.335 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| prod | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| prod | batch_12q | 200 | yes |  |
| prod | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| prod | noul_criteria | 200 | yes |  |
| prod | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
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
| prod | q1 | 100/100 | 96.5 | 122.2 | 128.8 | 54.6 | — |
| prod | q5 | 100/100 | 100.7 | 143.2 | 184.5 | 58.6 | — |
| prod | q10 | 100/100 | 105.1 | 111.5 | 145.8 | 62.0 | — |
| prod | long_q1 | 100/100 | 114.5 | 119.3 | 123.4 | 64.0 | — |
| prod | q5_s4 | 100/100 | 138.6 | 144.7 | 225.6 | 96.3 | — |
| prod | sweep q1/w16 | 256/256 | 186.1 | 197.2 | 546.6 | 92.2 | 76.4 |
| prod | sweep q1/w32 | 256/256 | 443.9 | 601.3 | 993.8 | 110.5 | 62.8 |
| prod | sweep q5/w16 | 256/256 | 230.7 | 273.1 | 869.9 | 112.6 | 59.7 |
| prod | sweep q5/w32 | 256/256 | 453.2 | 740.2 | 1039.3 | 111.7 | 59.8 |
| new | q1 | 100/100 | 93.4 | 98.7 | 159.7 | 54.0 | — |
| new | q5 | 100/100 | 129.4 | 135.8 | 142.0 | 89.6 | — |
| new | q10 | 100/100 | 104.2 | 111.9 | 120.3 | 60.9 | — |
| new | long_q1 | 100/100 | 112.6 | 177.2 | 213.9 | 62.6 | — |
| new | q5_s4 | 100/100 | 169.9 | 177.8 | 354.6 | 120.5 | — |
| new | sweep q1/w16 | 256/256 | 229.1 | 283.5 | 502.1 | 113.5 | 62.9 |
| new | sweep q1/w32 | 256/256 | 461.8 | 612.5 | 884.5 | 114.6 | 60.8 |
| new | sweep q5/w16 | 256/256 | 312.1 | 330.2 | 681.7 | 154.4 | 47.7 |
| new | sweep q5/w32 | 256/256 | 585.3 | 834.3 | 1012.6 | 145.5 | 47.4 |

