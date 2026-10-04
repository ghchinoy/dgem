# Regression matrix report: 20261004-v030b-release-t2

- Matrix **v2**, tier **T2**, started 2026-10-04T03:20:22+00:00, finished 2026-10-04T05:27:31+00:00
- Repository commit `d8484cb485`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | v0.2.1 | dec8ff8 | a9eafde59c |
| new | vertex | v0.2.2-12-gd8484cb | d8484cb | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| health | new | **PASS** | ready |
| contract | new | **PASS** | all cases as expected |
| calibration | new | **PASS** | acc 0.873 vs 0.867; agreement 0.976 (floor 0.967, allowance 0.071); McNemar p=1 |
| massive_spot | new | **PASS** | es 19/20, hi 20/20, ja 18/20, ru 19/20, th 19/20 |
| jev_native | new | **PASS** | acc 0.811 vs 0.807; agreement 0.970 (floor 0.963, allowance 0.045); McNemar p=0.58 |
| jev_systemone | new | **PASS** | acc 0.844 vs 0.847; agreement 0.945 (floor 0.944, allowance 0.050); McNemar p=0.86 |
| intents_banking77 | new | **PASS** | acc 0.767 vs 0.756; agreement 0.922 (floor 0.900, allowance 0.130); McNemar p=1 |
| intents_clinc150 | new | **PASS** | acc 0.978 vs 0.978; agreement 0.985 (floor 0.978, allowance 0.074); McNemar p=1 |
| massive | new | **PASS** | acc 0.826 vs 0.826; agreement 0.944 (floor 0.948, allowance 0.026); McNemar p=1 [frozen set] |
| xnli | new | **PASS** | acc 0.697 vs 0.696; agreement 0.948 (floor 0.948, allowance 0.027); McNemar p=0.84 [frozen set] |
| typed | new | **PASS** | acc 0.725 vs 0.719; agreement 0.924 (floor 0.948, allowance 0.030); McNemar p=0.38 [frozen set] |
| di_wide | new | **PASS** | acc 0.860 vs 0.880; agreement 0.962 (floor 0.963, allowance 0.058); McNemar p=0.07 |
| di_catchall | new | **PASS** | acc 0.850 vs 0.883; agreement 0.930 (floor 0.935, allowance 0.069); McNemar p=0.031 |
| rag_dev | new | **PASS** | acc 0.779 vs 0.756; agreement 0.929 (floor 0.946, allowance 0.052); McNemar p=0.044 |
| calib_systemone | new | **PASS** | acc 0.867 vs 0.860; agreement 0.967 (floor 0.967, allowance 0.071); McNemar p=1 |
| intents_systemone | new | **PASS** | acc 0.850 vs 0.856; agreement 0.948 (floor 0.944, allowance 0.079); McNemar p=1 |
| gate_mixed_noul | new | **FAIL** | acc 0.967 vs 0.588; agreement 0.606 (floor 0.922, allowance 0.073); McNemar p=1.1e-32 — agreement 0.606 < noise floor 0.922 - 0.073; McNemar p=1.1e-32 (118 vs 2) |
| vision_spot | new | **PASS** | acc 0.787 vs 0.787; agreement 0.957 (floor 0.948, allowance 0.057); McNemar p=1 |
| vision | new | **PASS** | acc 0.754 vs 0.758; agreement 0.953 (floor 0.952, allowance 0.035); McNemar p=0.56 |
| coverage | new | **PASS** | no coverage loss |
| order | new | **REVIEW** | net flip vs baseline: order_jev_choice +0.065 |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 1.2%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.656 (always-flag 0.601), recall 54.1%, predicted yes 27.9% vs gold 42.9% |
| di_catchall_oos_rate | new | **INFO** | in-scope answered as the catch-all: 2.5%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | new | **INFO** | hallucinated-class F1 0.697 (always-flag 0.601), recall 59.2%, predicted yes 30.0% vs gold 42.9% |
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
| bbox | prod | **INFO** | acc_at_50_expectation_pct=90.909, mean_expectation_iou=0.631 |
| bbox | new | **INFO** | acc_at_50_expectation_pct=72.727, mean_expectation_iou=0.593 |
| decision_index | prod | **INFO** | headline_decision_index=96.667, ece_10bin=0.039 |
| decision_index | new | **INFO** | headline_decision_index=96.667, ece_10bin=0.045 |
| latency | new | **FAIL** | 1126 errors |

Overall: **new: FAIL**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 43, 43 / 50 | 1.000 | — | 0.867 | 0.685 | 0.096 | 0.258 | 0.805 | 225 |
| calibration | new | 43, 45, 43 / 50 | 1.000 | — | 0.873 | 0.688 | 0.071 | 0.227 | 0.829 | 220 |
| massive_spot | prod | 96 / 100 | 1.000 | — | 0.960 | 0.961 | 0.032 | 0.075 | 0.771 | 337 |
| massive_spot | new | 95 / 100 | 1.000 | — | 0.950 | 0.955 | 0.030 | 0.093 | 0.882 | 333 |
| jev_native | prod | 184, 188, 187 / 231 | 1.000 | — | 0.807 | 0.799 | 0.096 | 0.294 | 0.793 | 150 |
| jev_native | new | 186, 190, 186 / 231 | 1.000 | — | 0.811 | 0.798 | 0.090 | 0.289 | 0.790 | 141 |
| jev_systemone | prod | 194, 196, 197 / 231 | 1.000 | — | 0.847 | 0.810 | 0.073 | 0.227 | 0.875 | 364 |
| jev_systemone | new | 193, 196, 196 / 231 | 1.000 | — | 0.844 | 0.814 | 0.075 | 0.229 | 0.864 | 353 |
| intents_banking77 | prod | 24, 22, 22 / 30 | 1.000 | — | 0.756 | 0.685 | 0.203 | 0.343 | 0.729 | 396 |
| intents_banking77 | new | 22, 23, 24 / 30 | 1.000 | — | 0.767 | 0.700 | 0.218 | 0.365 | 0.715 | 418 |
| intents_clinc150 | prod | 29, 29, 30 / 30 | 1.000 | — | 0.978 | 0.966 | 0.038 | 0.028 | 0.966 | 193 |
| intents_clinc150 | new | 29, 29, 30 / 30 | 1.000 | — | 0.978 | 0.966 | 0.056 | 0.043 | 0.931 | 160 |
| massive | prod | 4211 / 5100 | 1.000 | — | 0.826 | 0.795 | 0.104 | 0.275 | 0.910 | 336 |
| massive | new | 4211 / 5100 | 1.000 | — | 0.826 | 0.794 | 0.103 | 0.278 | 0.914 | 378 |
| xnli | prod | 3134 / 4500 | 1.000 | — | 0.696 | 0.700 | 0.229 | 0.529 | 0.614 | 295 |
| xnli | new | 3138 / 4500 | 1.000 | — | 0.697 | 0.701 | 0.230 | 0.528 | 0.614 | 300 |
| typed | prod | 1438 / 2000 | 1.000 | — | 0.719 | 0.599 | 0.181 | 0.441 | 0.771 | 344 |
| typed | new | 1449 / 2000 | 1.000 | — | 0.725 | 0.602 | 0.179 | 0.442 | 0.762 | 380 |
| di_wide | prod | 88, 88, 88 / 100 | 1.000 | — | 0.880 | 0.852 | 0.071 | 0.193 | 0.907 | 465 |
| di_wide | new | 86, 86, 86 / 100 | 1.000 | — | 0.860 | 0.832 | 0.085 | 0.213 | 0.919 | 473 |
| di_catchall | prod | 88, 86, 91 / 100 | 1.000 | — | 0.883 | 0.846 | 0.066 | 0.213 | 0.659 | 844 |
| di_catchall | new | 84, 85, 86 / 100 | 1.000 | — | 0.850 | 0.802 | 0.076 | 0.252 | 0.691 | 850 |
| rag_dev | prod | 150, 150, 149 / 198 | 1.000 | — | 0.756 | 0.733 | 0.195 | 0.425 | 0.676 | 296 |
| rag_dev | new | 156, 153, 154 / 198 | 1.000 | — | 0.779 | 0.762 | 0.172 | 0.397 | 0.620 | 288 |
| calib_systemone | prod | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.814 | 0.091 | 0.196 | 0.803 | 370 |
| calib_systemone | new | 44, 43, 43 / 50 | 1.000 | — | 0.867 | 0.825 | 0.071 | 0.195 | 0.806 | 374 |
| intents_systemone | prod | 52, 50, 52 / 60 | 1.000 | — | 0.856 | 0.805 | 0.096 | 0.221 | 0.885 | 399 |
| intents_systemone | new | 50, 51, 52 / 60 | 1.000 | — | 0.850 | 0.801 | 0.104 | 0.221 | 0.883 | 348 |
| gate_mixed_noul | prod | 63, 61, 56 / 102 | 1.000 | — | 0.588 | 0.537 | 0.323 | 0.661 | 0.723 | 379 |
| gate_mixed_noul | new | 100, 98, 98 / 102 | 1.000 | — | 0.967 | 0.967 | 0.024 | 0.049 | 0.919 | 444 |
| vision_spot | prod | 111 / 141 | 1.000 | — | 0.787 | 0.758 | 0.180 | 0.380 | 0.825 | 366 |
| vision_spot | new | 111 / 141 | 1.000 | — | 0.787 | 0.780 | 0.169 | 0.377 | 0.806 | 363 |
| vision | prod | 652, 641 / 853 | 1.000 | — | 0.758 | 0.665 | 0.185 | 0.414 | 0.802 | 365 |
| vision | new | 644, 643 / 853 | 1.000 | — | 0.754 | 0.666 | 0.189 | 0.414 | 0.805 | 355 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | calib_systemone | intents_systemone | gate_mixed_noul | vision | mean |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| prod | 0.973 | 0.964 | 0.949 | 0.889 | 0.978 | 0.973 | 0.947 | 0.953 | 1.000 | 0.933 | 0.869 | 0.939 | 0.947 |
| new | 0.960 | 0.962 | 0.938 | 0.911 | 0.978 | 0.953 | 0.923 | 0.939 | 0.933 | 0.956 | 0.974 | 0.965 | 0.949 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.090 | 0.069 | 0.459 | 0.521 | 1.50, 0.90, 1.50, 1.45, 1.55 |
| calibration | new | 50 | 0.101 | 0.080 | 0.423 | 0.422 | 1.40, 1.10, 1.40, 1.35, 1.45 |
| massive_spot | prod | 100 | 0.032 | 0.051 | 0.282 | 0.301 | 1.35, 1.00, 1.30, 1.40, 1.45 |
| massive_spot | new | 100 | 0.030 | 0.035 | 0.267 | 0.252 | 1.40, 1.20, 1.30, 1.40, 1.40 |
| jev_native | prod | 231 | 0.103 | 0.061 | 0.668 | 0.539 | 1.95, 1.85, 1.95, 1.85, 1.90 |
| jev_native | new | 231 | 0.094 | 0.039 | 0.626 | 0.523 | 1.80, 1.80, 1.85, 1.75, 1.80 |
| jev_systemone | prod | 231 | 0.070 | 0.033 | 0.476 | 0.431 | 1.60, 1.45, 1.65, 1.50, 1.60 |
| jev_systemone | new | 231 | 0.077 | 0.052 | 0.472 | 0.426 | 1.55, 1.50, 1.65, 1.45, 1.60 |
| intents_banking77 | prod | 29 | 0.171 | 0.086 | 0.720 | 0.639 | 1.65, 1.50, 1.35, 1.70, 1.65 |
| intents_banking77 | new | 29 | 0.125 | 0.078 | 0.753 | 0.656 | 1.60, 1.65, 1.40, 1.75, 1.70 |
| intents_clinc150 | prod | 30 | 0.014 | 0.042 | 0.092 | 0.103 | 0.75, 0.80, 0.50, 0.80, 0.75 |
| intents_clinc150 | new | 30 | 0.069 | 0.065 | 0.131 | 0.149 | 0.80, 0.85, 0.50, 0.75, 0.75 |
| massive | prod | 5093 | 0.103 | 0.048 | 0.894 | 0.693 | 1.75, 1.75, 1.75, 1.75, 1.75 |
| massive | new | 5094 | 0.102 | 0.046 | 0.894 | 0.695 | 1.75, 1.75, 1.75, 1.75, 1.75 |
| xnli | prod | 4497 | 0.229 | 0.036 | 1.467 | 0.776 | 3.60, 3.65, 3.65, 3.60, 3.65 |
| xnli | new | 4495 | 0.229 | 0.035 | 1.452 | 0.773 | 3.55, 3.65, 3.65, 3.55, 3.55 |
| typed | prod | 1998 | 0.180 | 0.023 | 0.999 | 0.666 | 2.70, 2.70, 2.80, 2.75, 2.70 |
| typed | new | 1998 | 0.178 | 0.021 | 1.008 | 0.667 | 2.70, 2.70, 2.80, 2.80, 2.70 |
| di_wide | prod | 100 | 0.070 | 0.047 | 0.646 | 0.587 | 1.30, 1.30, 1.45, 1.40, 1.45 |
| di_wide | new | 100 | 0.081 | 0.047 | 0.638 | 0.590 | 1.30, 1.25, 1.40, 1.40, 1.40 |
| di_catchall | prod | 100 | 0.066 | 0.069 | 0.740 | 0.722 | 1.15, 1.20, 1.20, 1.15, 1.15 |
| di_catchall | new | 99 | 0.080 | 0.088 | 0.854 | 0.818 | 1.25, 1.20, 1.25, 1.20, 1.20 |
| rag_dev | prod | 198 | 0.199 | 0.042 | 0.876 | 0.525 | 3.40, 3.70, 3.40, 3.30, 3.40 |
| rag_dev | new | 198 | 0.171 | 0.044 | 0.857 | 0.511 | 3.45, 3.55, 3.25, 3.50, 3.30 |
| calib_systemone | prod | 50 | 0.090 | 0.075 | 0.407 | 0.401 | 1.40, 1.15, 1.45, 1.35, 1.25 |
| calib_systemone | new | 50 | 0.057 | 0.058 | 0.358 | 0.400 | 1.30, 0.85, 1.25, 1.25, 1.25 |
| intents_systemone | prod | 60 | 0.095 | 0.098 | 0.656 | 0.528 | 1.70, 1.65, 1.55, 1.60, 1.75 |
| intents_systemone | new | 60 | 0.097 | 0.098 | 0.585 | 0.514 | 1.55, 1.55, 1.40, 1.45, 1.60 |
| gate_mixed_noul | prod | 102 | 0.288 | 0.088 | 1.106 | 0.630 | 5.00, 5.00, 4.65, 4.40, 4.45 |
| gate_mixed_noul | new | 102 | 0.016 | 0.042 | 0.095 | 0.163 | 1.15, 1.20, 0.50, 1.35, 1.25 |
| vision_spot | prod | 141 | 0.180 | 0.058 | 1.038 | 0.602 | 2.75, 2.75, 2.55, 2.90, 2.70 |
| vision_spot | new | 141 | 0.169 | 0.067 | 1.055 | 0.606 | 2.80, 2.75, 2.55, 2.95, 2.80 |
| vision | prod | 853 | 0.179 | 0.048 | 1.076 | 0.631 | 2.75, 2.80, 2.70, 2.80, 2.80 |
| vision | new | 853 | 0.188 | 0.046 | 1.060 | 0.624 | 2.70, 2.80, 2.70, 2.80, 2.80 |

## massive per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| af | 0.77 / 0.88 | 0.73 / 0.90 |
| am | 0.76 / 0.91 | 0.76 / 0.90 |
| ar | 0.89 / 0.93 | 0.87 / 0.93 |
| az | 0.80 / 0.94 | 0.82 / 0.94 |
| bn | 0.81 / 0.93 | 0.81 / 0.93 |
| cy | 0.42 / 0.71 | 0.43 / 0.70 |
| da | 0.86 / 0.95 | 0.86 / 0.95 |
| de | 0.91 / 0.96 | 0.90 / 0.96 |
| el | 0.85 / 0.95 | 0.86 / 0.95 |
| en | 0.91 / 0.97 | 0.91 / 0.97 |
| es | 0.85 / 0.96 | 0.85 / 0.96 |
| fa | 0.94 / 0.96 | 0.95 / 0.97 |
| fi | 0.85 / 0.93 | 0.85 / 0.94 |
| fr | 0.88 / 0.94 | 0.87 / 0.95 |
| he | 0.88 / 0.94 | 0.87 / 0.94 |
| hi | 0.95 / 0.95 | 0.94 / 0.96 |
| hu | 0.70 / 0.89 | 0.72 / 0.89 |
| hy | 0.78 / 0.90 | 0.78 / 0.91 |
| id | 0.90 / 0.96 | 0.89 / 0.96 |
| is | 0.73 / 0.91 | 0.76 / 0.90 |
| it | 0.91 / 0.97 | 0.92 / 0.97 |
| ja | 0.96 / 0.95 | 0.92 / 0.96 |
| jv | 0.67 / 0.88 | 0.68 / 0.88 |
| ka | 0.73 / 0.91 | 0.72 / 0.91 |
| km | 0.71 / 0.91 | 0.76 / 0.94 |
| kn | 0.81 / 0.92 | 0.81 / 0.92 |
| ko | 0.92 / 0.96 | 0.90 / 0.96 |
| lv | 0.65 / 0.84 | 0.62 / 0.85 |
| ml | 0.87 / 0.95 | 0.91 / 0.95 |
| mn | 0.67 / 0.91 | 0.70 / 0.90 |
| ms | 0.87 / 0.95 | 0.86 / 0.94 |
| my | 0.91 / 0.95 | 0.91 / 0.94 |
| nb | 0.83 / 0.93 | 0.83 / 0.93 |
| nl | 0.87 / 0.94 | 0.86 / 0.94 |
| pl | 0.90 / 0.97 | 0.89 / 0.96 |
| pt | 0.86 / 0.96 | 0.85 / 0.96 |
| ro | 0.80 / 0.91 | 0.82 / 0.92 |
| ru | 0.94 / 0.97 | 0.94 / 0.97 |
| sl | 0.72 / 0.90 | 0.71 / 0.89 |
| sq | 0.62 / 0.86 | 0.62 / 0.87 |
| sv | 0.84 / 0.93 | 0.83 / 0.93 |
| sw | 0.75 / 0.89 | 0.75 / 0.90 |
| ta | 0.86 / 0.93 | 0.88 / 0.94 |
| te | 0.85 / 0.94 | 0.87 / 0.93 |
| th | 0.92 / 0.97 | 0.92 / 0.97 |
| tl | 0.85 / 0.95 | 0.84 / 0.95 |
| tr | 0.89 / 0.94 | 0.87 / 0.94 |
| ur | 0.85 / 0.93 | 0.82 / 0.92 |
| vi | 0.85 / 0.95 | 0.85 / 0.94 |
| zh | 0.90 / 0.96 | 0.91 / 0.96 |
| **macro** | **0.824** | **0.824** |

## xnli per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| ar | 0.71 / 0.93 | 0.70 / 0.93 |
| bg | 0.72 / 0.90 | 0.71 / 0.90 |
| de | 0.79 / 0.93 | 0.78 / 0.93 |
| el | 0.71 / 0.93 | 0.71 / 0.93 |
| en | 0.83 / 0.94 | 0.82 / 0.95 |
| es | 0.72 / 0.94 | 0.72 / 0.94 |
| fr | 0.72 / 0.95 | 0.72 / 0.94 |
| hi | 0.64 / 0.92 | 0.66 / 0.93 |
| ru | 0.66 / 0.92 | 0.65 / 0.92 |
| sw | 0.62 / 0.91 | 0.63 / 0.92 |
| th | 0.66 / 0.93 | 0.64 / 0.93 |
| tr | 0.67 / 0.92 | 0.67 / 0.92 |
| ur | 0.63 / 0.92 | 0.66 / 0.92 |
| vi | 0.67 / 0.93 | 0.68 / 0.93 |
| zh | 0.69 / 0.93 | 0.70 / 0.92 |
| **macro** | **0.696** | **0.697** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.719 | 0.574 | 0.263 | 0.431 |
| new | 0.725 | 0.574 | 0.268 | 0.429 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.030 | 0.090 (+0.060) | 0.075 (+0.045) | 0.445 / 0.305 |
| prod | order_jev_choice | 139 | 0.079 | 0.101 (+0.022) | 0.101 (+0.022) | 0.230 / 0.216 |
| prod | order_massive_en | 200 | 0.015 | 0.075 (+0.060) | 0.065 (+0.050) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.025 | 0.065 (+0.040) | 0.085 (+0.060) | 0.260 / 0.335 |
| new | order_emotion | 200 | 0.035 | 0.105 (+0.070) | 0.090 (+0.055) | 0.450 / 0.305 |
| new | order_jev_choice | 139 | 0.036 | 0.122 (+0.086) | 0.158 (+0.122) | 0.223 / 0.216 |
| new | order_massive_en | 200 | 0.015 | 0.080 (+0.065) | 0.075 (+0.060) | 0.045 / 0.045 |
| new | order_xnli_en | 200 | 0.030 | 0.055 (+0.025) | 0.070 (+0.040) | 0.275 / 0.335 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.974 |
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
| prod | q1 | 100/100 | 93.8 | 104.3 | 172.9 | 53.5 | — |
| prod | q5 | 100/100 | 98.6 | 104.8 | 169.2 | 57.4 | — |
| prod | q10 | 100/100 | 103.9 | 124.9 | 236.2 | 60.3 | — |
| prod | long_q1 | 100/100 | 112.6 | 180.5 | 249.9 | 61.2 | — |
| prod | q5_s4 | 100/100 | 139.6 | 150.2 | 197.5 | 96.3 | — |
| prod | sweep q1/w16 | 256/256 | 190.6 | 331.7 | 677.5 | 92.4 | 66.0 |
| prod | sweep q1/w32 | 256/256 | 406.3 | 1009.5 | 1265.5 | 92.1 | 60.7 |
| prod | sweep q5/w16 | 256/256 | 233.2 | 320.9 | 596.3 | 114.7 | 57.7 |
| prod | sweep q5/w32 | 256/256 | 515.9 | 1063.5 | 1357.7 | 116.1 | 49.4 |
| new | q1 | 100/100 | 93.8 | 176.7 | 304.7 | 53.2 | — |
| new | q5 | 100/100 | 97.8 | 106.5 | 119.7 | 56.3 | — |
| new | q10 | 100/100 | 109.8 | 177.2 | 310.3 | 60.9 | — |
| new | long_q1 | 98/100 | 112.9 | 121.2 | 202.2 | 61.2 | — |
| new | q5_s4 | 0/100 | — | — | — | — | — |
| new | sweep q1/w16 | 0/256 | — | — | — | — | 0.0 |
| new | sweep q1/w32 | 0/256 | — | — | — | — | 0.0 |
| new | sweep q5/w16 | 0/256 | — | — | — | — | 0.0 |
| new | sweep q5/w32 | 0/256 | — | — | — | — | 0.0 |

