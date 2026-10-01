# Regression matrix report: 20261001-v013-reference-t2

- Matrix **v1**, tier **T2**, started 2026-10-01T22:20:51+00:00, finished 2026-10-01T22:35:02+00:00
- Repository commit `8608361d9b`
- Mode: single target, compared with the reference ranges

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod | vertex | v0.1.3 | e685ef9 | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| contract | prod | **PASS** | all cases as expected |
| calibration | prod | **PASS** | acc 0.867; reference 0.860–0.880 (v0.1.3) |
| jev_native | prod | **PASS** | acc 0.804; reference 0.792–0.823 (v0.1.3) |
| jev_systemone | prod | **PASS** | acc 0.853; reference 0.823–0.862 (v0.1.3) |
| intents_banking77 | prod | **PASS** | acc 0.833; reference 0.733–0.833 (v0.1.3) |
| intents_clinc150 | prod | **PASS** | acc 0.967; reference 0.967–1.000 (v0.1.3) |
| massive_spot | prod | **PASS** | es 16/20, hi 18/20, ja 18/20, ru 17/20, th 16/20 |
| massive | prod | **PASS** | acc 0.822; reference 0.811–0.832 (v0.1.3) |
| xnli | prod | **PASS** | acc 0.678; reference 0.664–0.692 (v0.1.3) |
| typed | prod | **PASS** | acc 0.667; reference 0.646–0.688 (v0.1.3) |
| order | prod | **INFO** | order_emotion net +0.050, order_jev_choice net +0.029, order_massive_en net +0.065, order_xnli_en net +0.025 |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=36.364, mean_expectation_iou=0.412 |
| decision_index | prod | **INFO** | headline_decision_index=98.889, ece_10bin=0.052 |
| latency | prod | **PASS** | within tolerance |

Overall: **prod: PASS**

## Suites

| suite | target | runs: correct / n | mean accuracy | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 43, 43 / 50 | 0.867 | 0.100 | 0.239 | 0.858 | 207 |
| jev_native | prod | 188, 184, 185 / 231 | 0.804 | 0.078 | 0.267 | 0.859 | 129 |
| jev_systemone | prod | 199, 197, 195 / 231 | 0.853 | 0.076 | 0.239 | 0.737 | 220 |
| intents_banking77 | prod | 25 / 30 | 0.833 | 0.093 | 0.331 | 0.896 | 401 |
| intents_clinc150 | prod | 29 / 30 | 0.967 | 0.066 | 0.045 | 0.931 | 141 |
| massive_spot | prod | 85 / 100 | 0.850 | 0.092 | 0.227 | 0.891 | 219 |
| massive | prod | 4190 / 5100 | 0.822 | 0.080 | 0.269 | 0.911 | 212 |
| xnli | prod | 3049 / 4500 | 0.678 | 0.246 | 0.551 | 0.648 | 200 |
| typed | prod | 1334 / 2000 | 0.667 | 0.234 | 0.539 | 0.751 | 257 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | mean |
|---|---|---|---|---|
| prod | 0.947 | 0.968 | 0.947 | 0.954 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.123 | 0.119 | 0.439 | 0.432 | 1.45, 1.15, 1.40, 1.30, 1.40 |
| jev_native | prod | 231 | 0.070 | 0.042 | 0.517 | 0.470 | 1.55, 1.55, 1.60, 1.50, 1.50 |
| jev_systemone | prod | 231 | 0.082 | 0.065 | 0.588 | 0.503 | 1.75, 1.55, 1.80, 1.75, 1.75 |
| intents_banking77 | prod | 30 | 0.127 | 0.091 | 1.097 | 0.827 | 1.80, 2.25, 1.95, 1.90, 2.05 |
| intents_clinc150 | prod | 30 | 0.066 | 0.021 | 0.092 | 0.087 | 0.75, 0.50, 0.75, 0.65, 0.65 |
| massive_spot | prod | 100 | 0.092 | 0.093 | 0.559 | 0.500 | 1.45, 1.40, 1.45, 1.45, 1.35 |
| massive | prod | 5099 | 0.080 | 0.036 | 0.879 | 0.759 | 1.55, 1.50, 1.55, 1.50, 1.50 |
| xnli | prod | 4497 | 0.245 | 0.022 | 1.602 | 0.803 | 4.05, 4.10, 4.10, 4.05, 4.00 |
| typed | prod | 1995 | 0.232 | 0.017 | 1.315 | 0.768 | 3.35, 3.40, 3.40, 3.55, 3.40 |

## massive per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| af | 0.71 / 0.86 |
| am | 0.82 / 0.91 |
| ar | 0.87 / 0.93 |
| az | 0.80 / 0.90 |
| bn | 0.81 / 0.90 |
| cy | 0.42 / 0.69 |
| da | 0.86 / 0.92 |
| de | 0.89 / 0.93 |
| el | 0.83 / 0.92 |
| en | 0.89 / 0.94 |
| es | 0.88 / 0.93 |
| fa | 0.90 / 0.93 |
| fi | 0.84 / 0.91 |
| fr | 0.85 / 0.93 |
| he | 0.85 / 0.91 |
| hi | 0.89 / 0.94 |
| hu | 0.67 / 0.81 |
| hy | 0.75 / 0.85 |
| id | 0.89 / 0.96 |
| is | 0.75 / 0.87 |
| it | 0.92 / 0.96 |
| ja | 0.94 / 0.95 |
| jv | 0.74 / 0.85 |
| ka | 0.76 / 0.88 |
| km | 0.76 / 0.87 |
| kn | 0.86 / 0.90 |
| ko | 0.89 / 0.94 |
| lv | 0.67 / 0.83 |
| ml | 0.86 / 0.91 |
| mn | 0.77 / 0.84 |
| ms | 0.85 / 0.93 |
| my | 0.87 / 0.91 |
| nb | 0.84 / 0.91 |
| nl | 0.83 / 0.92 |
| pl | 0.87 / 0.93 |
| pt | 0.85 / 0.92 |
| ro | 0.81 / 0.89 |
| ru | 0.88 / 0.95 |
| sl | 0.71 / 0.84 |
| sq | 0.68 / 0.81 |
| sv | 0.82 / 0.92 |
| sw | 0.77 / 0.86 |
| ta | 0.83 / 0.90 |
| te | 0.84 / 0.92 |
| th | 0.89 / 0.95 |
| tl | 0.85 / 0.93 |
| tr | 0.87 / 0.91 |
| ur | 0.85 / 0.91 |
| vi | 0.87 / 0.94 |
| zh | 0.89 / 0.93 |
| **macro** | **0.820** |

## xnli per language (accuracy / mean confidence)

| lang | prod |
|---|---|
| ar | 0.66 / 0.92 |
| bg | 0.70 / 0.91 |
| de | 0.73 / 0.92 |
| el | 0.70 / 0.93 |
| en | 0.82 / 0.94 |
| es | 0.68 / 0.92 |
| fr | 0.71 / 0.93 |
| hi | 0.61 / 0.92 |
| ru | 0.67 / 0.94 |
| sw | 0.63 / 0.91 |
| th | 0.65 / 0.93 |
| tr | 0.66 / 0.92 |
| ur | 0.64 / 0.92 |
| vi | 0.64 / 0.92 |
| zh | 0.66 / 0.92 |
| **macro** | **0.678** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.667 | 0.554 | 0.303 | 0.414 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.060 | 0.110 (+0.050) | 0.100 (+0.040) | 0.425 / 0.305 |
| prod | order_jev_choice | 139 | 0.079 | 0.108 (+0.029) | 0.094 (+0.014) | 0.230 / 0.216 |
| prod | order_massive_en | 200 | 0.030 | 0.095 (+0.065) | 0.085 (+0.055) | 0.055 / 0.045 |
| prod | order_xnli_en | 200 | 0.040 | 0.065 (+0.025) | 0.085 (+0.045) | 0.275 / 0.335 |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 90.9 | 96.5 | 102.3 | 53.3 | — |
| prod | q5 | 100/100 | 94.4 | 98.3 | 111.7 | 56.4 | — |
| prod | q10 | 100/100 | 100.4 | 106.3 | 113.0 | 59.7 | — |
| prod | long_q1 | 100/100 | 108.2 | 112.5 | 118.4 | 60.5 | — |
| prod | q5_s4 | 100/100 | 133.7 | 138.0 | 201.1 | 94.7 | — |
| prod | sweep q1/w16 | 256/256 | 216.4 | 230.2 | 515.5 | 107.9 | 67.7 |
| prod | sweep q1/w32 | 256/256 | 442.0 | 695.9 | 958.7 | 109.4 | 61.8 |
| prod | sweep q5/w16 | 256/256 | 225.8 | 238.8 | 671.4 | 111.7 | 62.4 |
| prod | sweep q5/w32 | 256/256 | 467.4 | 831.0 | 1211.1 | 115.5 | 57.0 |

