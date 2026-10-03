# Regression matrix report: 20261002-v020-reference-t2

- Matrix **v1**, tier **T2**, started 2026-10-02T23:02:23+00:00, finished 2026-10-02T23:15:18+00:00
- Repository commit `027b68ef88`
- Mode: single target, compared with the reference ranges

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod | vertex | v0.2.0 | 027b68e | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| contract | prod | **PASS** | all cases as expected |
| calibration | prod | **PASS** | acc 0.887; reference 0.860–0.900 (v0.2.0) |
| jev_native | prod | **PASS** | acc 0.803; reference 0.800–0.817 (v0.2.0) |
| jev_systemone | prod | **PASS** | acc 0.843; reference 0.827–0.857 (v0.2.0) |
| intents_banking77 | prod | **PASS** | acc 0.800; reference 0.767–0.833 (v0.2.0) |
| intents_clinc150 | prod | **PASS** | acc 0.989; reference 0.967–1.000 (v0.2.0) |
| massive_spot | prod | **PASS** | es 19/20, hi 20/20, ja 19/20, ru 19/20, th 19/20 |
| di_wide | prod | **PASS** | acc 0.877; reference 0.850–0.890 (v0.2.0) |
| di_catchall | prod | **PASS** | acc 0.877; reference 0.850–0.880 (v0.2.0) |
| rag_dev | prod | **PASS** | acc 0.763; reference 0.758–0.773 (v0.2.0) |
| massive | prod | **PASS** | acc 0.826; reference 0.813–0.837 (v0.2.0) |
| xnli | prod | **PASS** | acc 0.697; reference 0.683–0.713 (v0.2.0) |
| typed | prod | **PASS** | acc 0.723; reference 0.703–0.748 (v0.2.0) |
| coverage | prod | **REVIEW** | fewer answered items: jev_native coverage 0.9957 < reference 1.0000 (context 3) |
| order | prod | **INFO** | order_emotion net +0.090, order_jev_choice net +0.036, order_massive_en net +0.040, order_xnli_en net +0.035 |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 1.2%; out-of-scope recall 88.3% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.667 (always-flag 0.601), recall 55.3%, predicted yes 28.3% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=81.818, mean_expectation_iou=0.611 |
| decision_index | prod | **INFO** | headline_decision_index=96.667, ece_10bin=0.048 |
| latency | prod | **PASS** | within tolerance |

Overall: **prod: REVIEW**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 45, 44 / 50 | 1.000 | — | 0.887 | 0.681 | 0.085 | 0.230 | 0.810 | 204 |
| jev_native | prod | 184, 185, 185 / 230 | 0.996 | context 3 | 0.803 | 0.801 | 0.099 | 0.292 | 0.805 | 134 |
| jev_systemone | prod | 191, 195, 198 / 231 | 1.000 | — | 0.843 | 0.810 | 0.077 | 0.231 | 0.846 | 227 |
| intents_banking77 | prod | 23, 24, 25 / 30 | 1.000 | — | 0.800 | 0.744 | 0.161 | 0.344 | 0.719 | 405 |
| intents_clinc150 | prod | 30, 29, 30 / 30 | 1.000 | — | 0.989 | 0.983 | 0.056 | 0.041 | 0.931 | 151 |
| massive_spot | prod | 96 / 100 | 1.000 | — | 0.960 | 0.969 | 0.036 | 0.067 | 0.766 | 243 |
| di_wide | prod | 88, 88, 87 / 100 | 1.000 | — | 0.877 | 0.859 | 0.069 | 0.197 | 0.906 | 437 |
| di_catchall | prod | 87, 88, 88 / 100 | 1.000 | — | 0.877 | 0.826 | 0.103 | 0.232 | 0.637 | 810 |
| rag_dev | prod | 151, 152, 150 / 198 | 1.000 | — | 0.763 | 0.741 | 0.195 | 0.426 | 0.651 | 295 |
| massive | prod | 4212 / 5100 | 1.000 | — | 0.826 | 0.791 | 0.104 | 0.275 | 0.913 | 244 |
| xnli | prod | 3137 / 4500 | 1.000 | — | 0.697 | 0.701 | 0.229 | 0.529 | 0.607 | 207 |
| typed | prod | 1446 / 2000 | 1.000 | — | 0.723 | 0.602 | 0.178 | 0.438 | 0.773 | 321 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | mean |
|---|---|---|---|---|---|---|---|---|---|
| prod | 0.987 | 0.965 | 0.948 | 0.956 | 0.978 | 0.947 | 0.940 | 0.953 | 0.959 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.106 | 0.104 | 0.337 | 0.375 | 1.35, 0.90, 1.35, 1.25, 1.35 |
| jev_native | prod | 230 | 0.102 | 0.040 | 0.655 | 0.534 | 1.90, 1.85, 1.90, 1.85, 1.85 |
| jev_systemone | prod | 231 | 0.085 | 0.038 | 0.476 | 0.425 | 1.60, 1.55, 1.65, 1.55, 1.55 |
| intents_banking77 | prod | 29 | 0.129 | 0.105 | 0.814 | 0.679 | 1.55, 1.70, 1.55, 1.80, 1.75 |
| intents_clinc150 | prod | 30 | 0.053 | 0.015 | 0.064 | 0.018 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| massive_spot | prod | 100 | 0.036 | 0.031 | 0.302 | 0.307 | 1.40, 1.10, 1.30, 1.45, 1.50 |
| di_wide | prod | 100 | 0.070 | 0.035 | 0.581 | 0.550 | 1.25, 1.20, 1.40, 1.35, 1.35 |
| di_catchall | prod | 99 | 0.125 | 0.085 | 0.791 | 0.779 | 1.20, 1.15, 1.25, 1.20, 1.10 |
| rag_dev | prod | 198 | 0.198 | 0.039 | 0.880 | 0.518 | 3.50, 3.65, 3.30, 3.45, 3.45 |
| massive | prod | 5095 | 0.103 | 0.046 | 0.894 | 0.699 | 1.75, 1.70, 1.75, 1.70, 1.70 |
| xnli | prod | 4493 | 0.227 | 0.036 | 1.467 | 0.777 | 3.60, 3.65, 3.65, 3.60, 3.60 |
| typed | prod | 1997 | 0.177 | 0.035 | 0.960 | 0.654 | 2.60, 2.60, 2.70, 2.65, 2.60 |

## massive per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| af | 0.75 / 0.90 |
| am | 0.75 / 0.90 |
| ar | 0.87 / 0.94 |
| az | 0.80 / 0.93 |
| bn | 0.80 / 0.93 |
| cy | 0.41 / 0.72 |
| da | 0.86 / 0.94 |
| de | 0.89 / 0.96 |
| el | 0.86 / 0.94 |
| en | 0.90 / 0.97 |
| es | 0.86 / 0.97 |
| fa | 0.94 / 0.96 |
| fi | 0.85 / 0.93 |
| fr | 0.86 / 0.93 |
| he | 0.89 / 0.95 |
| hi | 0.95 / 0.96 |
| hu | 0.74 / 0.88 |
| hy | 0.75 / 0.91 |
| id | 0.91 / 0.97 |
| is | 0.72 / 0.91 |
| it | 0.91 / 0.96 |
| ja | 0.96 / 0.96 |
| jv | 0.69 / 0.88 |
| ka | 0.74 / 0.91 |
| km | 0.75 / 0.93 |
| kn | 0.84 / 0.92 |
| ko | 0.90 / 0.96 |
| lv | 0.62 / 0.85 |
| ml | 0.87 / 0.95 |
| mn | 0.70 / 0.89 |
| ms | 0.86 / 0.95 |
| my | 0.92 / 0.96 |
| nb | 0.84 / 0.95 |
| nl | 0.86 / 0.94 |
| pl | 0.89 / 0.96 |
| pt | 0.86 / 0.96 |
| ro | 0.85 / 0.92 |
| ru | 0.92 / 0.96 |
| sl | 0.73 / 0.90 |
| sq | 0.64 / 0.86 |
| sv | 0.84 / 0.93 |
| sw | 0.74 / 0.88 |
| ta | 0.89 / 0.93 |
| te | 0.83 / 0.94 |
| th | 0.92 / 0.96 |
| tl | 0.85 / 0.96 |
| tr | 0.86 / 0.94 |
| ur | 0.83 / 0.93 |
| vi | 0.85 / 0.94 |
| zh | 0.90 / 0.96 |
| **macro** | **0.824** |

## xnli per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| ar | 0.70 / 0.92 |
| bg | 0.74 / 0.90 |
| de | 0.76 / 0.93 |
| el | 0.69 / 0.93 |
| en | 0.83 / 0.94 |
| es | 0.72 / 0.94 |
| fr | 0.72 / 0.94 |
| hi | 0.67 / 0.92 |
| ru | 0.66 / 0.93 |
| sw | 0.63 / 0.92 |
| th | 0.64 / 0.93 |
| tr | 0.68 / 0.92 |
| ur | 0.65 / 0.92 |
| vi | 0.69 / 0.92 |
| zh | 0.68 / 0.92 |
| **macro** | **0.697** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.723 | 0.575 | 0.261 | 0.423 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.010 | 0.100 (+0.090) | 0.085 (+0.075) | 0.455 / 0.305 |
| prod | order_jev_choice | 139 | 0.058 | 0.094 (+0.036) | 0.144 (+0.086) | 0.237 / 0.216 |
| prod | order_massive_en | 200 | 0.025 | 0.065 (+0.040) | 0.065 (+0.040) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.035 | 0.070 (+0.035) | 0.060 (+0.025) | 0.285 / 0.335 |

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
| prod | q1 | 100/100 | 92.0 | 100.2 | 118.9 | 54.3 | — |
| prod | q5 | 100/100 | 99.1 | 106.4 | 115.7 | 58.1 | — |
| prod | q10 | 100/100 | 103.4 | 108.9 | 150.4 | 61.7 | — |
| prod | long_q1 | 100/100 | 111.1 | 119.2 | 124.4 | 63.0 | — |
| prod | q5_s4 | 100/100 | 136.3 | 143.2 | 162.6 | 96.4 | — |
| prod | sweep q1/w16 | 256/256 | 218.0 | 260.7 | 652.8 | 108.3 | 64.4 |
| prod | sweep q1/w32 | 256/256 | 435.2 | 839.9 | 1153.5 | 107.8 | 59.7 |
| prod | sweep q5/w16 | 256/256 | 219.7 | 229.5 | 596.8 | 109.2 | 65.1 |
| prod | sweep q5/w32 | 256/256 | 444.3 | 855.6 | 1194.5 | 110.4 | 58.8 |

