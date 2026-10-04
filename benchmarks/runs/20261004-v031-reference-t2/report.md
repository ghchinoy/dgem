# Regression matrix report: 20261004-v031-reference-t2

- Matrix **v2**, tier **T2**, started 2026-10-04T18:32:03+00:00, finished 2026-10-04T18:49:37+00:00
- Repository commit `708af1963c`
- Mode: single target, compared with the reference ranges

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod | vertex | v0.3.1 | c13cd20 | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| contract | prod | **PASS** | all cases as expected |
| calibration | prod | **PASS** | acc 0.860; reference 0.860–0.900 (v0.3.0) |
| massive_spot | prod | **PASS** | es 19/20, hi 20/20, ja 19/20, ru 20/20, th 19/20 |
| jev_native | prod | **PASS** | acc 0.814; reference 0.800–0.817 (v0.3.0) |
| jev_systemone | prod | **PASS** | acc 0.841; reference 0.827–0.857 (v0.3.0) |
| intents_banking77 | prod | **PASS** | acc 0.800; reference 0.733–0.833 (v0.3.0) |
| intents_clinc150 | prod | **PASS** | acc 0.989; reference 0.967–1.000 (v0.3.0) |
| massive | prod | **PASS** | acc 0.825; reference 0.813–0.839 (v0.3.0) |
| xnli | prod | **PASS** | acc 0.697; reference 0.683–0.713 (v0.3.0) |
| typed | prod | **PASS** | acc 0.721; reference 0.700–0.748 (v0.3.0) |
| di_wide | prod | **PASS** | acc 0.870; reference 0.850–0.890 (v0.3.0) |
| di_catchall | prod | **PASS** | acc 0.867; reference 0.840–0.890 (v0.3.0) |
| rag_dev | prod | **PASS** | acc 0.758; reference 0.758–0.773 (v0.3.0) |
| calib_systemone | prod | **PASS** | acc 0.867; reference 0.779–0.967 (v0.3.0) |
| intents_systemone | prod | **PASS** | acc 0.872; reference 0.786–0.958 (v0.3.0) |
| gate_mixed_noul | prod | **PASS** | acc 0.964; reference 0.935–1.000 (v0.3.0) |
| vision_spot | prod | **PASS** | acc 0.787; reference 0.703–0.844 (v0.3.0) |
| vision | prod | **PASS** | acc 0.762; reference 0.748–0.761 (v0.3.0) |
| coverage | prod | **PASS** | no coverage loss |
| order | prod | **INFO** | order_emotion net +0.035, order_jev_choice net +0.058, order_massive_en net +0.065, order_xnli_en net +0.040 |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 2.1%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.656 (always-flag 0.601), recall 53.7%, predicted yes 27.4% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=81.818, mean_expectation_iou=0.619 |
| decision_index | prod | **INFO** | headline_decision_index=96.667, ece_10bin=0.044 |
| latency | prod | **PASS** | within tolerance |

Overall: **prod: PASS**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.680 | 0.097 | 0.271 | 0.805 | 217 |
| massive_spot | prod | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.024 | 0.067 | 0.742 | 315 |
| jev_native | prod | 188, 190, 186 / 231 | 1.000 | — | 0.814 | 0.798 | 0.092 | 0.285 | 0.788 | 136 |
| jev_systemone | prod | 197, 190, 196 / 231 | 1.000 | — | 0.841 | 0.803 | 0.078 | 0.232 | 0.855 | 389 |
| intents_banking77 | prod | 24, 24, 24 / 30 | 1.000 | — | 0.800 | 0.744 | 0.166 | 0.379 | 0.667 | 497 |
| intents_clinc150 | prod | 29, 30, 30 / 30 | 1.000 | — | 0.989 | 0.983 | 0.056 | 0.028 | 0.931 | 159 |
| massive | prod | 4210 / 5100 | 1.000 | — | 0.825 | 0.792 | 0.105 | 0.275 | 0.914 | 334 |
| xnli | prod | 3137 / 4500 | 1.000 | — | 0.697 | 0.701 | 0.231 | 0.530 | 0.610 | 309 |
| typed | prod | 1441 / 2000 | 1.000 | — | 0.721 | 0.595 | 0.180 | 0.442 | 0.768 | 387 |
| di_wide | prod | 87, 88, 86 / 100 | 1.000 | — | 0.870 | 0.858 | 0.079 | 0.208 | 0.901 | 450 |
| di_catchall | prod | 89, 85, 86 / 100 | 1.000 | — | 0.867 | 0.822 | 0.072 | 0.234 | 0.683 | 824 |
| rag_dev | prod | 150, 150, 150 / 198 | 1.000 | — | 0.758 | 0.734 | 0.198 | 0.429 | 0.663 | 291 |
| calib_systemone | prod | 43, 43, 44 / 50 | 1.000 | — | 0.867 | 0.830 | 0.076 | 0.202 | 0.777 | 350 |
| intents_systemone | prod | 52, 52, 53 / 60 | 1.000 | — | 0.872 | 0.827 | 0.078 | 0.193 | 0.872 | 328 |
| gate_mixed_noul | prod | 97, 99, 99 / 102 | 1.000 | — | 0.964 | 0.964 | 0.028 | 0.046 | 0.979 | 431 |
| vision_spot | prod | 111 / 141 | 1.000 | — | 0.787 | 0.781 | 0.165 | 0.371 | 0.821 | 356 |
| vision | prod | 654, 646 / 853 | 1.000 | — | 0.762 | 0.670 | 0.180 | 0.406 | 0.799 | 352 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | calib_systemone | intents_systemone | gate_mixed_noul | vision | mean |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| prod | 0.987 | 0.957 | 0.941 | 0.956 | 0.978 | 0.967 | 0.937 | 0.970 | 0.973 | 0.956 | 0.974 | 0.954 | 0.962 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.164 | 0.116 | 0.528 | 0.518 | 1.60, 1.15, 1.60, 1.60, 1.70 |
| massive_spot | prod | 100 | 0.024 | 0.044 | 0.257 | 0.270 | 1.35, 1.05, 1.20, 1.40, 1.45 |
| jev_native | prod | 231 | 0.095 | 0.065 | 0.643 | 0.529 | 1.90, 1.85, 1.95, 1.75, 1.90 |
| jev_systemone | prod | 231 | 0.090 | 0.052 | 0.468 | 0.420 | 1.55, 1.50, 1.65, 1.55, 1.60 |
| intents_banking77 | prod | 29 | 0.218 | 0.193 | 0.555 | 0.507 | 1.30, 1.50, 1.45, 1.55, 1.50 |
| intents_clinc150 | prod | 30 | 0.040 | 0.037 | 0.097 | 0.102 | 0.75, 0.75, 0.50, 0.75, 0.75 |
| massive | prod | 5086 | 0.103 | 0.050 | 0.876 | 0.683 | 1.75, 1.75, 1.75, 1.75, 1.70 |
| xnli | prod | 4492 | 0.230 | 0.035 | 1.455 | 0.773 | 3.60, 3.60, 3.65, 3.55, 3.60 |
| typed | prod | 1997 | 0.179 | 0.032 | 1.006 | 0.668 | 2.80, 2.65, 2.80, 2.75, 2.70 |
| di_wide | prod | 100 | 0.072 | 0.053 | 0.644 | 0.595 | 1.30, 1.25, 1.40, 1.40, 1.45 |
| di_catchall | prod | 98 | 0.058 | 0.067 | 0.682 | 0.671 | 1.10, 1.15, 1.20, 1.20, 1.20 |
| rag_dev | prod | 198 | 0.203 | 0.039 | 0.914 | 0.527 | 3.55, 3.85, 3.40, 3.45, 3.75 |
| calib_systemone | prod | 50 | 0.073 | 0.035 | 0.413 | 0.430 | 1.40, 1.00, 1.35, 1.35, 1.35 |
| intents_systemone | prod | 60 | 0.070 | 0.074 | 0.561 | 0.492 | 1.60, 1.55, 1.35, 1.50, 1.60 |
| gate_mixed_noul | prod | 102 | 0.035 | 0.047 | 0.093 | 0.094 | 1.20, 1.25, 1.10, 1.25, 1.05 |
| vision_spot | prod | 141 | 0.165 | 0.049 | 1.025 | 0.603 | 2.75, 2.70, 2.50, 2.90, 2.75 |
| vision | prod | 853 | 0.179 | 0.048 | 1.030 | 0.618 | 2.65, 2.75, 2.65, 2.75, 2.75 |

## massive per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| af | 0.76 / 0.89 |
| am | 0.76 / 0.91 |
| ar | 0.88 / 0.93 |
| az | 0.81 / 0.93 |
| bn | 0.79 / 0.92 |
| cy | 0.45 / 0.70 |
| da | 0.85 / 0.94 |
| de | 0.88 / 0.97 |
| el | 0.86 / 0.94 |
| en | 0.90 / 0.97 |
| es | 0.86 / 0.96 |
| fa | 0.95 / 0.96 |
| fi | 0.84 / 0.94 |
| fr | 0.90 / 0.95 |
| he | 0.88 / 0.95 |
| hi | 0.95 / 0.96 |
| hu | 0.66 / 0.88 |
| hy | 0.78 / 0.92 |
| id | 0.90 / 0.97 |
| is | 0.74 / 0.91 |
| it | 0.92 / 0.97 |
| ja | 0.95 / 0.96 |
| jv | 0.68 / 0.88 |
| ka | 0.72 / 0.91 |
| km | 0.74 / 0.93 |
| kn | 0.82 / 0.92 |
| ko | 0.90 / 0.96 |
| lv | 0.63 / 0.87 |
| ml | 0.88 / 0.95 |
| mn | 0.71 / 0.89 |
| ms | 0.88 / 0.95 |
| my | 0.90 / 0.95 |
| nb | 0.84 / 0.94 |
| nl | 0.89 / 0.95 |
| pl | 0.90 / 0.96 |
| pt | 0.86 / 0.97 |
| ro | 0.82 / 0.93 |
| ru | 0.92 / 0.96 |
| sl | 0.72 / 0.90 |
| sq | 0.61 / 0.86 |
| sv | 0.85 / 0.94 |
| sw | 0.73 / 0.90 |
| ta | 0.86 / 0.94 |
| te | 0.85 / 0.94 |
| th | 0.92 / 0.96 |
| tl | 0.81 / 0.93 |
| tr | 0.89 / 0.94 |
| ur | 0.85 / 0.93 |
| vi | 0.85 / 0.95 |
| zh | 0.90 / 0.96 |
| **macro** | **0.824** |

## xnli per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| ar | 0.70 / 0.93 |
| bg | 0.72 / 0.91 |
| de | 0.78 / 0.93 |
| el | 0.71 / 0.93 |
| en | 0.82 / 0.95 |
| es | 0.73 / 0.93 |
| fr | 0.72 / 0.94 |
| hi | 0.65 / 0.93 |
| ru | 0.64 / 0.93 |
| sw | 0.62 / 0.92 |
| th | 0.65 / 0.93 |
| tr | 0.68 / 0.92 |
| ur | 0.66 / 0.92 |
| vi | 0.68 / 0.93 |
| zh | 0.70 / 0.93 |
| **macro** | **0.697** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.721 | 0.573 | 0.265 | 0.432 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.050 | 0.085 (+0.035) | 0.080 (+0.030) | 0.440 / 0.305 |
| prod | order_jev_choice | 139 | 0.065 | 0.122 (+0.058) | 0.165 (+0.101) | 0.223 / 0.216 |
| prod | order_massive_en | 200 | 0.025 | 0.090 (+0.065) | 0.065 (+0.040) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.020 | 0.060 (+0.040) | 0.055 (+0.035) | 0.275 / 0.335 |

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

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 93.6 | 98.7 | 110.7 | 53.6 | — |
| prod | q5 | 100/100 | 98.0 | 106.0 | 169.7 | 56.6 | — |
| prod | q10 | 100/100 | 107.0 | 149.1 | 173.6 | 60.0 | — |
| prod | long_q1 | 100/100 | 113.1 | 118.1 | 137.5 | 61.7 | — |
| prod | q5_s4 | 100/100 | 139.1 | 177.7 | 220.3 | 96.4 | — |
| prod | sweep q1/w16 | 256/256 | 186.4 | 227.9 | 551.9 | 92.1 | 74.2 |
| prod | sweep q1/w32 | 256/256 | 451.7 | 725.7 | 1065.8 | 111.7 | 60.3 |
| prod | sweep q5/w16 | 256/256 | 223.8 | 231.3 | 661.2 | 111.8 | 62.7 |
| prod | sweep q5/w32 | 256/256 | 449.1 | 800.3 | 1082.0 | 111.0 | 60.0 |

