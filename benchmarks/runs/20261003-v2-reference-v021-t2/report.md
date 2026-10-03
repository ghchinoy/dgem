# Regression matrix report: 20261003-v2-reference-v021-t2

- Matrix **v2**, tier **T2**, started 2026-10-03T20:10:57+00:00, finished 2026-10-03T20:26:00+00:00
- Repository commit `bd2f19651a`
- Mode: single target, compared with the reference ranges

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod | vertex | v0.2.1 | dec8ff8 | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| contract | prod | **PASS** | all cases as expected |
| calibration | prod | **PASS** | acc 0.867; reference 0.860–0.900 (v0.2.1) |
| massive_spot | prod | **PASS** | es 19/20, hi 20/20, ja 19/20, ru 20/20, th 19/20 |
| jev_native | prod | **PASS** | acc 0.811; reference 0.800–0.817 (v0.2.1) |
| jev_systemone | prod | **PASS** | acc 0.844; reference 0.827–0.857 (v0.2.1) |
| intents_banking77 | prod | **PASS** | acc 0.767; reference 0.767–0.833 (v0.2.1) |
| intents_clinc150 | prod | **PASS** | acc 0.967; reference 0.967–1.000 (v0.2.1) |
| massive | prod | **PASS** | acc 0.829; reference 0.813–0.839 (v0.2.1) |
| xnli | prod | **PASS** | acc 0.699; reference 0.683–0.713 (v0.2.1) |
| typed | prod | **PASS** | acc 0.720; reference 0.700–0.748 (v0.2.1) |
| di_wide | prod | **PASS** | acc 0.877; reference 0.850–0.890 (v0.2.1) |
| di_catchall | prod | **PASS** | acc 0.873; reference 0.840–0.890 (v0.2.1) |
| rag_dev | prod | **PASS** | acc 0.764; reference 0.758–0.773 (v0.2.1) |
| calib_systemone | prod | **PASS** | acc 0.873; reference 0.779–0.967 (v0.2.1) |
| intents_systemone | prod | **PASS** | acc 0.872; reference 0.786–0.958 (v0.2.1) |
| gate_mixed_noul | prod | **PASS** | acc 0.595; reference 0.498–0.692 (v0.2.1) |
| vision_spot | prod | **PASS** | acc 0.773; reference 0.703–0.844 (v0.2.1) |
| vision | prod | **PASS** | acc 0.753; reference 0.748–0.761 (v0.2.1) |
| coverage | prod | **PASS** | no coverage loss |
| order | prod | **INFO** | order_emotion net +0.075, order_jev_choice net +0.086, order_massive_en net +0.075, order_xnli_en net +0.040 |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 2.1%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.665 (always-flag 0.601), recall 54.5%, predicted yes 27.4% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.913–0.995 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=90.909, mean_expectation_iou=0.636 |
| decision_index | prod | **INFO** | headline_decision_index=96.667, ece_10bin=0.034 |
| latency | prod | **PASS** | within tolerance |

Overall: **prod: PASS**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 43, 43 / 50 | 1.000 | — | 0.867 | 0.685 | 0.088 | 0.246 | 0.861 | 190 |
| massive_spot | prod | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.019 | 0.066 | 0.845 | 309 |
| jev_native | prod | 187, 187, 188 / 231 | 1.000 | — | 0.811 | 0.798 | 0.097 | 0.284 | 0.798 | 142 |
| jev_systemone | prod | 192, 196, 197 / 231 | 1.000 | — | 0.844 | 0.812 | 0.072 | 0.229 | 0.854 | 389 |
| intents_banking77 | prod | 23, 23, 23 / 30 | 1.000 | — | 0.767 | 0.696 | 0.190 | 0.317 | 0.716 | 443 |
| intents_clinc150 | prod | 29, 29, 29 / 30 | 1.000 | — | 0.967 | 0.949 | 0.052 | 0.021 | 0.954 | 162 |
| massive | prod | 4227 / 5100 | 1.000 | — | 0.829 | 0.796 | 0.102 | 0.273 | 0.912 | 314 |
| xnli | prod | 3144 / 4500 | 1.000 | — | 0.699 | 0.703 | 0.228 | 0.524 | 0.611 | 318 |
| typed | prod | 1440 / 2000 | 1.000 | — | 0.720 | 0.603 | 0.181 | 0.448 | 0.766 | 380 |
| di_wide | prod | 87, 88, 88 / 100 | 1.000 | — | 0.877 | 0.835 | 0.070 | 0.192 | 0.908 | 468 |
| di_catchall | prod | 89, 84, 89 / 100 | 1.000 | — | 0.873 | 0.832 | 0.071 | 0.231 | 0.663 | 830 |
| rag_dev | prod | 150, 151, 153 / 198 | 1.000 | — | 0.764 | 0.742 | 0.188 | 0.416 | 0.644 | 294 |
| calib_systemone | prod | 44, 44, 43 / 50 | 1.000 | — | 0.873 | 0.839 | 0.086 | 0.205 | 0.752 | 336 |
| intents_systemone | prod | 52, 52, 53 / 60 | 1.000 | — | 0.872 | 0.828 | 0.094 | 0.200 | 0.875 | 339 |
| gate_mixed_noul | prod | 62, 58, 62 / 102 | 1.000 | — | 0.595 | 0.539 | 0.319 | 0.666 | 0.695 | 368 |
| vision_spot | prod | 109 / 141 | 1.000 | — | 0.773 | 0.759 | 0.186 | 0.390 | 0.827 | 369 |
| vision | prod | 647, 638 / 853 | 1.000 | — | 0.753 | 0.674 | 0.190 | 0.415 | 0.813 | 349 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | calib_systemone | intents_systemone | gate_mixed_noul | vision | mean |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| prod | 0.973 | 0.957 | 0.960 | 0.822 | 1.000 | 0.940 | 0.930 | 0.943 | 0.920 | 0.933 | 0.882 | 0.926 | 0.932 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.070 | 0.113 | 0.396 | 0.484 | 1.45, 0.80, 1.45, 1.35, 1.45 |
| massive_spot | prod | 100 | 0.019 | 0.044 | 0.224 | 0.232 | 1.30, 1.05, 1.20, 1.35, 1.40 |
| jev_native | prod | 231 | 0.094 | 0.068 | 0.649 | 0.531 | 1.90, 1.85, 1.90, 1.80, 1.90 |
| jev_systemone | prod | 231 | 0.070 | 0.050 | 0.443 | 0.409 | 1.45, 1.45, 1.55, 1.40, 1.50 |
| intents_banking77 | prod | 29 | 0.182 | 0.124 | 0.865 | 0.687 | 1.65, 1.70, 1.70, 1.90, 1.80 |
| intents_clinc150 | prod | 30 | 0.058 | 0.021 | 0.071 | 0.024 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| massive | prod | 5092 | 0.100 | 0.047 | 0.883 | 0.687 | 1.75, 1.75, 1.75, 1.75, 1.75 |
| xnli | prod | 4496 | 0.228 | 0.039 | 1.455 | 0.773 | 3.60, 3.65, 3.65, 3.60, 3.60 |
| typed | prod | 1999 | 0.181 | 0.019 | 1.011 | 0.672 | 2.70, 2.70, 2.80, 2.80, 2.70 |
| di_wide | prod | 100 | 0.083 | 0.054 | 0.584 | 0.549 | 1.30, 1.20, 1.40, 1.35, 1.40 |
| di_catchall | prod | 100 | 0.066 | 0.069 | 0.739 | 0.714 | 1.20, 1.20, 1.25, 1.20, 1.15 |
| rag_dev | prod | 198 | 0.187 | 0.055 | 0.889 | 0.525 | 3.50, 3.85, 3.40, 3.30, 3.45 |
| calib_systemone | prod | 50 | 0.062 | 0.073 | 0.444 | 0.436 | 1.50, 1.15, 1.45, 1.35, 1.40 |
| intents_systemone | prod | 60 | 0.119 | 0.073 | 0.618 | 0.522 | 1.65, 1.55, 1.50, 1.55, 1.65 |
| gate_mixed_noul | prod | 102 | 0.314 | 0.135 | 1.159 | 0.641 | 5.00, 5.00, 5.00, 5.00, 5.00 |
| vision_spot | prod | 141 | 0.186 | 0.048 | 1.080 | 0.625 | 2.80, 2.70, 2.50, 2.95, 2.85 |
| vision | prod | 853 | 0.186 | 0.051 | 1.054 | 0.621 | 2.70, 2.75, 2.70, 2.80, 2.80 |

## massive per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| af | 0.76 / 0.90 |
| am | 0.78 / 0.91 |
| ar | 0.89 / 0.93 |
| az | 0.81 / 0.94 |
| bn | 0.83 / 0.92 |
| cy | 0.48 / 0.72 |
| da | 0.85 / 0.95 |
| de | 0.88 / 0.96 |
| el | 0.87 / 0.95 |
| en | 0.91 / 0.97 |
| es | 0.86 / 0.96 |
| fa | 0.95 / 0.97 |
| fi | 0.85 / 0.93 |
| fr | 0.89 / 0.94 |
| he | 0.88 / 0.94 |
| hi | 0.93 / 0.96 |
| hu | 0.72 / 0.89 |
| hy | 0.75 / 0.92 |
| id | 0.90 / 0.97 |
| is | 0.71 / 0.91 |
| it | 0.92 / 0.96 |
| ja | 0.94 / 0.96 |
| jv | 0.67 / 0.88 |
| ka | 0.75 / 0.91 |
| km | 0.77 / 0.93 |
| kn | 0.79 / 0.92 |
| ko | 0.90 / 0.97 |
| lv | 0.62 / 0.85 |
| ml | 0.90 / 0.95 |
| mn | 0.71 / 0.88 |
| ms | 0.88 / 0.95 |
| my | 0.90 / 0.95 |
| nb | 0.86 / 0.94 |
| nl | 0.87 / 0.94 |
| pl | 0.89 / 0.97 |
| pt | 0.86 / 0.96 |
| ro | 0.83 / 0.91 |
| ru | 0.93 / 0.96 |
| sl | 0.74 / 0.90 |
| sq | 0.66 / 0.86 |
| sv | 0.85 / 0.92 |
| sw | 0.73 / 0.89 |
| ta | 0.87 / 0.94 |
| te | 0.84 / 0.93 |
| th | 0.92 / 0.98 |
| tl | 0.83 / 0.94 |
| tr | 0.86 / 0.95 |
| ur | 0.84 / 0.94 |
| vi | 0.85 / 0.95 |
| zh | 0.90 / 0.95 |
| **macro** | **0.828** |

## xnli per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| ar | 0.71 / 0.94 |
| bg | 0.71 / 0.91 |
| de | 0.78 / 0.93 |
| el | 0.69 / 0.93 |
| en | 0.83 / 0.94 |
| es | 0.72 / 0.94 |
| fr | 0.73 / 0.94 |
| hi | 0.65 / 0.92 |
| ru | 0.67 / 0.92 |
| sw | 0.65 / 0.91 |
| th | 0.65 / 0.93 |
| tr | 0.66 / 0.92 |
| ur | 0.65 / 0.92 |
| vi | 0.68 / 0.93 |
| zh | 0.71 / 0.93 |
| **macro** | **0.699** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.720 | 0.572 | 0.268 | 0.436 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.030 | 0.105 (+0.075) | 0.075 (+0.045) | 0.445 / 0.305 |
| prod | order_jev_choice | 139 | 0.043 | 0.129 (+0.086) | 0.144 (+0.101) | 0.216 / 0.216 |
| prod | order_massive_en | 200 | 0.015 | 0.090 (+0.075) | 0.065 (+0.050) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.025 | 0.065 (+0.040) | 0.065 (+0.040) | 0.265 / 0.335 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.948 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.987 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.981 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.985 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.913 |
| prod | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.964 |
| prod | batch_12q | 200 | yes |  |
| prod | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.995 |
| prod | noul_criteria | 200 | yes |  |
| prod | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.963 |
| prod | long_9k | 422 | yes | marker=maximum context length |
| prod | context_refusal | 422 | yes | marker=maximum context length |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 111.5 | 133.0 | 151.1 | 68.3 | — |
| prod | q5 | 100/100 | 101.5 | 136.7 | 155.6 | 57.4 | — |
| prod | q10 | 100/100 | 103.6 | 110.1 | 121.8 | 61.2 | — |
| prod | long_q1 | 100/100 | 113.2 | 117.7 | 124.8 | 61.9 | — |
| prod | q5_s4 | 100/100 | 139.0 | 143.5 | 229.3 | 96.6 | — |
| prod | sweep q1/w16 | 256/256 | 190.3 | 261.1 | 657.5 | 94.9 | 69.0 |
| prod | sweep q1/w32 | 256/256 | 464.4 | 851.0 | 1150.0 | 115.0 | 57.1 |
| prod | sweep q5/w16 | 256/256 | 232.0 | 239.9 | 603.1 | 115.8 | 61.7 |
| prod | sweep q5/w32 | 256/256 | 450.0 | 733.8 | 1053.6 | 111.5 | 60.7 |

