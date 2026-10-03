# Regression matrix report: 20261003-compare-strands-v19

- Matrix **v2**, tier **TC**, started 2026-10-03T05:00:00+00:00, finished 2026-10-03T14:10:00+00:00
- Repository commit ``
- Mode: baseline `dgem` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| dgem (baseline) | vertex | v0.2.0 | 027b68e | a9eafde59c |
| strands (competitor: strands-decider-2b) | cloudrun | — | — | — |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|

Overall: no dgem candidate in this run (comparison run: the baseline is measured, competitors are informational)

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| jev_systemone | dgem | 194, 194, 195 / 231 | 1.000 | — | 0.841 | 0.798 | 0.070 | 0.226 | 0.875 | 233 |
| jev_systemone | strands | 169, 169, 169 / 231 | 1.000 | — | 0.732 | 0.693 | 0.048 | 0.348 | 0.806 | 160 |
| massive | dgem | 4221, 4195 / 5100 | 1.000 | — | 0.825 | 0.793 | 0.104 | 0.276 | 0.914 | 220 |
| massive | strands | 3426 / 5100 | 1.000 | — | 0.672 | 0.666 | 0.142 | 0.472 | 0.871 | 157 |
| xnli | dgem | 3149, 3144 / 4500 | 1.000 | — | 0.699 | 0.703 | 0.228 | 0.524 | 0.615 | 205 |
| xnli | strands | 2866 / 4500 | 1.000 | — | 0.637 | 0.640 | 0.035 | 0.477 | 0.685 | 143 |
| typed | dgem | 1436, 1446 / 2000 | 1.000 | — | 0.720 | 0.600 | 0.181 | 0.446 | 0.766 | 265 |
| typed | strands | 1203 / 2000 | 1.000 | — | 0.602 | 0.446 | 0.096 | 0.541 | 0.626 | 306 |
| calib_systemone | dgem | 47, 45, 46 / 50 | 1.000 | — | 0.920 | 0.822 | 0.058 | 0.181 | 0.731 | 253 |
| calib_systemone | strands | 38, 38, 38 / 50 | 1.000 | — | 0.760 | 0.642 | 0.192 | 0.425 | 0.634 | 157 |
| intents_systemone | dgem | 48, 49, 47 / 60 | 1.000 | — | 0.800 | 0.743 | 0.131 | 0.301 | 0.856 | 260 |
| intents_systemone | strands | 52, 52, 52 / 60 | 1.000 | — | 0.867 | 0.827 | 0.053 | 0.197 | 0.892 | 159 |
| gate_mixed_noul | dgem | 59, 57, 63 / 102 | 1.000 | — | 0.585 | 0.533 | 0.319 | 0.656 | 0.708 | 232 |
| gate_mixed_noul | strands | 88, 88, 88 / 102 | 1.000 | — | 0.863 | 0.856 | 0.209 | 0.291 | 0.849 | 238 |
| gate_mixed_noul_single | dgem | 100 / 102 | 1.000 | — | 0.980 | 0.980 | 0.033 | 0.044 | 0.935 | 231 |
| gate_mixed_noul_single | strands | 88 / 102 | 1.000 | — | 0.863 | 0.856 | 0.209 | 0.290 | 0.851 | 156 |

## Competitors (informational, no verdicts; paired against `dgem`, first run of each)

Accuracy is the mean over runs; the paired columns use run 1 of each side. ≥0.9 = share of answers with confidence ≥ 0.9 and the accuracy of those answers (an "act automatically" threshold).

| suite | target | runs: correct / n | mean accuracy | ECE10 | Brier | AUROC | ≥0.9: share / accuracy | only dgem right | only competitor right | McNemar p |
|---|---|---|---|---|---|---|---|---|---|---|
| jev_systemone | dgem | 194, 194, 195 / 231 | 0.841 | 0.070 | 0.226 | 0.875 | 0.77 / 0.944 | — | — | — |
| jev_systemone | strands | 169, 169, 169 / 231 | 0.732 | 0.048 | 0.348 | 0.806 | 0.24 / 0.982 | 36 | 11 | 0.00035 |
| massive | dgem | 4221, 4195 / 5100 | 0.825 | 0.104 | 0.276 | 0.914 | 0.81 / 0.928 | — | — | — |
| massive | strands | 3426 / 5100 | 0.672 | 0.142 | 0.472 | 0.871 | 0.55 / 0.903 | 1010 | 215 | 1.8e-123 |
| xnli | dgem | 3149, 3144 / 4500 | 0.699 | 0.228 | 0.524 | 0.615 | 0.78 / 0.742 | — | — | — |
| xnli | strands | 2866 / 4500 | 0.637 | 0.035 | 0.477 | 0.685 | 0.00 / 1.000 | 763 | 480 | 9.5e-16 |
| typed | dgem | 1436, 1446 / 2000 | 0.720 | 0.181 | 0.446 | 0.766 | 0.69 / 0.826 | — | — | — |
| typed | strands | 1203 / 2000 | 0.602 | 0.096 | 0.541 | 0.626 | 0.03 / 0.941 | 437 | 204 | 1.9e-20 |
| calib_systemone | dgem | 47, 45, 46 / 50 | 0.920 | 0.058 | 0.181 | 0.731 | 0.74 / 0.973 | — | — | — |
| calib_systemone | strands | 38, 38, 38 / 50 | 0.760 | 0.192 | 0.425 | 0.634 | 0.12 / 0.833 | 10 | 1 | 0.012 |
| intents_systemone | dgem | 48, 49, 47 / 60 | 0.800 | 0.131 | 0.301 | 0.856 | 0.77 / 0.891 | — | — | — |
| intents_systemone | strands | 52, 52, 52 / 60 | 0.867 | 0.053 | 0.197 | 0.892 | 0.72 / 0.953 | 3 | 7 | 0.34 |
| gate_mixed_noul | dgem | 59, 57, 63 / 102 | 0.585 | 0.319 | 0.656 | 0.708 | 0.67 / 0.662 | — | — | — |
| gate_mixed_noul | strands | 88, 88, 88 / 102 | 0.863 | 0.209 | 0.291 | 0.849 | 0.00 / — | 10 | 39 | 3.8e-05 |
| gate_mixed_noul_single | dgem | 100 / 102 | 0.980 | 0.033 | 0.044 | 0.935 | 0.87 / 1.000 | — | — | — |
| gate_mixed_noul_single | strands | 88 / 102 | 0.863 | 0.209 | 0.290 | 0.851 | 0.00 / — | 13 | 1 | 0.0018 |

Cross-slot coupling on `gate_mixed_noul` (cases whose two yes/no answers are the same label; gold 7 of 51):

- strands: 11, 11, 11 of 51
- dgem: 36, 40, 34 of 51

## Measured noise floor (answer agreement between repeated identical runs)

| target | jev_systemone | massive | xnli | typed | calib_systemone | intents_systemone | gate_mixed_noul | mean |
|---|---|---|---|---|---|---|---|---|
| dgem | 0.942 | 0.944 | 0.947 | 0.921 | 0.960 | 0.967 | 0.882 | 0.938 |
| strands | 1.000 | — | — | — | 1.000 | 1.000 | 1.000 | 1.000 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| jev_systemone | dgem | 231 | 0.071 | 0.036 | 0.456 | 0.410 | 1.50, 1.50, 1.60, 1.50, 1.60 |
| jev_systemone | strands | 231 | 0.048 | 0.045 | 0.609 | 0.612 | 1.00, 0.90, 1.00, 0.95, 0.95 |
| massive | dgem | 5067 | 0.096 | 0.038 | 0.838 | 0.655 | 1.75, 1.75, 1.80, 1.75, 1.75 |
| massive | strands | 5090 | 0.141 | 0.021 | 1.291 | 1.108 | 1.75, 1.75, 1.75, 1.75, 1.80 |
| xnli | dgem | 4454 | 0.219 | 0.028 | 1.353 | 0.743 | 3.45, 3.50, 3.45, 3.45, 3.45 |
| xnli | strands | 4500 | 0.035 | 0.028 | 0.799 | 0.796 | 0.90, 0.90, 0.90, 0.90, 0.85 |
| typed | dgem | 1994 | 0.177 | 0.020 | 0.970 | 0.655 | 2.75, 2.80, 2.75, 2.65, 2.80 |
| typed | strands | 2000 | 0.096 | 0.104 | 0.944 | 0.940 | 0.85, 0.85, 0.85, 0.80, 0.80 |
| calib_systemone | dgem | 50 | 0.031 | 0.039 | 0.410 | 0.447 | 1.45, 0.95, 1.35, 1.35, 1.45 |
| calib_systemone | strands | 50 | 0.152 | 0.144 | 0.879 | 0.907 | 1.10, 0.90, 1.35, 1.25, 1.15 |
| intents_systemone | dgem | 59 | 0.124 | 0.097 | 0.519 | 0.488 | 1.30, 1.25, 1.40, 1.30, 1.40 |
| intents_systemone | strands | 60 | 0.049 | 0.051 | 0.348 | 0.340 | 1.35, 1.25, 1.25, 1.15, 1.30 |
| gate_mixed_noul | dgem | 102 | 0.328 | 0.153 | 1.177 | 0.647 | 5.00, 5.00, 5.00, 5.00, 5.00 |
| gate_mixed_noul | strands | 102 | 0.209 | 0.106 | 0.468 | 0.362 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| gate_mixed_noul_single | dgem | 102 | 0.033 | 0.027 | 0.075 | 0.077 | 0.80, 0.60, 0.75, 0.75, 0.55 |
| gate_mixed_noul_single | strands | 102 | 0.209 | 0.106 | 0.468 | 0.361 | 0.50, 0.50, 0.50, 0.50, 0.50 |

## massive per language (accuracy / mean confidence)

| lang | strands | dgem |
|---|---|---|
| af | 0.70 / 0.84 | 0.76 / 0.89 |
| am | 0.34 / 0.68 | 0.76 / 0.91 |
| ar | 0.62 / 0.84 | 0.88 / 0.94 |
| az | 0.74 / 0.84 | 0.81 / 0.92 |
| bn | 0.60 / 0.80 | 0.80 / 0.92 |
| cy | 0.34 / 0.64 | 0.46 / 0.70 |
| da | 0.76 / 0.85 | 0.86 / 0.94 |
| de | 0.80 / 0.88 | 0.89 / 0.97 |
| el | 0.71 / 0.84 | 0.88 / 0.96 |
| en | 0.89 / 0.88 | 0.89 / 0.96 |
| es | 0.84 / 0.85 | 0.86 / 0.96 |
| fa | 0.79 / 0.89 | 0.94 / 0.97 |
| fi | 0.59 / 0.77 | 0.85 / 0.93 |
| fr | 0.82 / 0.86 | 0.85 / 0.94 |
| he | 0.77 / 0.85 | 0.87 / 0.94 |
| hi | 0.68 / 0.80 | 0.94 / 0.96 |
| hu | 0.68 / 0.83 | 0.75 / 0.88 |
| hy | 0.69 / 0.82 | 0.82 / 0.91 |
| id | 0.77 / 0.85 | 0.90 / 0.97 |
| is | 0.53 / 0.75 | 0.74 / 0.92 |
| it | 0.83 / 0.86 | 0.93 / 0.96 |
| ja | 0.91 / 0.90 | 0.95 / 0.97 |
| jv | 0.51 / 0.73 | 0.71 / 0.89 |
| ka | 0.52 / 0.77 | 0.72 / 0.91 |
| km | 0.41 / 0.74 | 0.73 / 0.91 |
| kn | 0.65 / 0.81 | 0.77 / 0.92 |
| ko | 0.85 / 0.92 | 0.90 / 0.97 |
| lv | 0.50 / 0.72 | 0.63 / 0.82 |
| ml | 0.59 / 0.74 | 0.86 / 0.95 |
| mn | 0.46 / 0.73 | 0.71 / 0.90 |
| ms | 0.72 / 0.82 | 0.87 / 0.95 |
| my | 0.27 / 0.63 | 0.90 / 0.95 |
| nb | 0.72 / 0.87 | 0.85 / 0.93 |
| nl | 0.79 / 0.86 | 0.88 / 0.95 |
| pl | 0.83 / 0.87 | 0.89 / 0.97 |
| pt | 0.79 / 0.84 | 0.86 / 0.96 |
| ro | 0.72 / 0.82 | 0.81 / 0.91 |
| ru | 0.89 / 0.91 | 0.93 / 0.97 |
| sl | 0.66 / 0.82 | 0.71 / 0.90 |
| sq | 0.49 / 0.73 | 0.63 / 0.85 |
| sv | 0.76 / 0.88 | 0.84 / 0.93 |
| sw | 0.34 / 0.67 | 0.74 / 0.90 |
| ta | 0.54 / 0.74 | 0.89 / 0.94 |
| te | 0.64 / 0.83 | 0.86 / 0.94 |
| th | 0.56 / 0.77 | 0.92 / 0.97 |
| tl | 0.58 / 0.76 | 0.84 / 0.94 |
| tr | 0.81 / 0.88 | 0.87 / 0.94 |
| ur | 0.71 / 0.83 | 0.84 / 0.95 |
| vi | 0.81 / 0.88 | 0.86 / 0.95 |
| zh | 0.87 / 0.91 | 0.90 / 0.96 |
| **macro** | **0.668** | **0.826** |

## xnli per language (accuracy / mean confidence)

| lang | strands | dgem |
|---|---|---|
| ar | 0.62 / 0.62 | 0.70 / 0.93 |
| bg | 0.68 / 0.63 | 0.72 / 0.91 |
| de | 0.68 / 0.64 | 0.78 / 0.93 |
| el | 0.70 / 0.64 | 0.70 / 0.93 |
| en | 0.73 / 0.67 | 0.83 / 0.94 |
| es | 0.68 / 0.63 | 0.70 / 0.93 |
| fr | 0.69 / 0.63 | 0.73 / 0.94 |
| hi | 0.55 / 0.56 | 0.67 / 0.92 |
| ru | 0.65 / 0.63 | 0.66 / 0.93 |
| sw | 0.48 / 0.59 | 0.64 / 0.92 |
| th | 0.53 / 0.56 | 0.65 / 0.93 |
| tr | 0.66 / 0.61 | 0.69 / 0.92 |
| ur | 0.55 / 0.61 | 0.64 / 0.92 |
| vi | 0.67 / 0.63 | 0.70 / 0.93 |
| zh | 0.69 / 0.65 | 0.69 / 0.92 |
| **macro** | **0.637** | **0.700** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| strands | 0.602 | 0.414 | 0.170 | 0.532 |
| dgem | 0.718 | 0.571 | 0.266 | 0.432 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| strands | order_emotion | 200 | 0.000 | 0.085 (+0.085) | 0.130 (+0.130) | 0.470 / 0.305 |
| strands | order_jev_choice | 139 | 0.000 | 0.108 (+0.108) | 0.137 (+0.137) | 0.259 / 0.216 |
| strands | order_massive_en | 200 | 0.000 | 0.075 (+0.075) | 0.060 (+0.060) | 0.065 / 0.045 |
| strands | order_xnli_en | 200 | 0.000 | 0.025 (+0.025) | 0.050 (+0.050) | 0.440 / 0.335 |
| dgem | order_emotion | 200 | 0.030 | 0.090 (+0.060) | 0.070 (+0.040) | 0.450 / 0.305 |
| dgem | order_jev_choice | 139 | 0.079 | 0.115 (+0.036) | 0.144 (+0.065) | 0.230 / 0.216 |
| dgem | order_massive_en | 200 | 0.010 | 0.090 (+0.080) | 0.075 (+0.065) | 0.045 / 0.045 |
| dgem | order_xnli_en | 200 | 0.030 | 0.060 (+0.030) | 0.045 (+0.015) | 0.275 / 0.335 |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| strands | len100 | 100/100 | 44.7 | 48.3 | 50.4 | 33.9 | — |
| strands | len1000 | 100/100 | 56.6 | 59.2 | 65.5 | 42.2 | — |
| strands | len2000 | 100/100 | 80.2 | 83.0 | 85.4 | 65.1 | — |
| strands | len250 | 100/100 | 48.6 | 50.4 | 56.3 | 34.1 | — |
| strands | len3000 | 100/100 | 101.9 | 105.1 | 129.7 | 87.1 | — |
| strands | len3800 | 100/100 | 131.9 | 135.4 | 138.4 | 117.9 | — |
| strands | len500 | 100/100 | 45.2 | 49.5 | 51.9 | 35.0 | — |
| strands | long_q1 | 100/100 | 131.6 | 136.6 | 140.0 | 120.7 | — |
| strands | nq1 | 100/100 | 47.1 | 51.8 | 63.4 | 35.3 | — |
| strands | nq10 | 100/100 | 85.9 | 90.3 | 93.8 | 73.0 | — |
| strands | nq20 | 100/100 | 101.0 | 104.6 | 106.7 | 87.8 | — |
| strands | nq5 | 100/100 | 86.5 | 90.2 | 112.4 | 72.1 | — |
| strands | q1 | 100/100 | 43.9 | 48.0 | 50.2 | 33.9 | — |
| strands | q10 | 100/100 | 89.3 | 94.2 | 97.5 | 78.3 | — |
| strands | q5 | 100/100 | 82.3 | 87.5 | 97.4 | 72.0 | — |
| strands | sweep q1/w1 | 256/256 | 43.3 | 47.2 | 50.1 | 33.6 | 22.6 |
| strands | sweep q1/w16 | 256/256 | 581.4 | 587.2 | 590.9 | 571.4 | 27.3 |
| strands | sweep q1/w32 | 256/256 | 1161.1 | 1165.7 | 1172.7 | 1150.9 | 27.4 |
| strands | sweep q1/w4 | 256/256 | 147.7 | 149.6 | 152.5 | 137.2 | 27.0 |
| strands | sweep q1/w8 | 256/256 | 291.5 | 294.8 | 298.7 | 281.3 | 27.3 |
| strands | sweep q5/w1 | 256/256 | 82.7 | 85.6 | 90.6 | 72.0 | 12.0 |
| strands | sweep q5/w16 | 256/256 | 1202.7 | 1212.3 | 1215.9 | 1192.4 | 13.3 |
| strands | sweep q5/w32 | 256/256 | 2411.1 | 2438.6 | 2444.6 | 2400.8 | 13.2 |
| strands | sweep q5/w4 | 256/256 | 299.2 | 305.0 | 309.0 | 288.9 | 13.3 |
| strands | sweep q5/w8 | 256/256 | 598.1 | 603.1 | 687.5 | 588.4 | 13.2 |
| dgem | len100 | 100/100 | 61.3 | 65.0 | 70.9 | 53.7 | — |
| dgem | len1000 | 100/100 | 67.9 | 70.2 | 71.9 | 57.0 | — |
| dgem | len2000 | 100/100 | 68.8 | 71.5 | 74.7 | 58.1 | — |
| dgem | len250 | 100/100 | 65.2 | 67.2 | 70.3 | 54.6 | — |
| dgem | len3000 | 100/100 | 71.8 | 75.1 | 79.0 | 59.4 | — |
| dgem | len3800 | 100/100 | 73.7 | 76.7 | 79.6 | 61.8 | — |
| dgem | len500 | 100/100 | 62.6 | 64.7 | 69.1 | 54.8 | — |
| dgem | long_q1 | 100/100 | 71.2 | 77.9 | 82.2 | 62.5 | — |
| dgem | nq1 | 100/100 | 65.5 | 67.7 | 71.8 | 55.3 | — |
| dgem | nq10 | 100/100 | 69.0 | 72.5 | 75.7 | 58.2 | — |
| dgem | nq20 | 100/100 | 69.8 | 72.6 | 77.2 | 58.8 | — |
| dgem | nq5 | 100/100 | 66.9 | 69.2 | 73.8 | 56.4 | — |
| dgem | q1 | 100/100 | 62.0 | 64.1 | 66.4 | 54.2 | — |
| dgem | q10 | 100/100 | 72.2 | 76.6 | 84.0 | 64.1 | — |
| dgem | q5 | 100/100 | 65.6 | 67.1 | 74.3 | 57.3 | — |
| dgem | sweep q1/w1 | 256/256 | 61.3 | 64.8 | 71.2 | 53.9 | 16.1 |
| dgem | sweep q1/w16 | 256/256 | 184.1 | 195.5 | 230.0 | 91.6 | 83.9 |
| dgem | sweep q1/w32 | 256/256 | 367.7 | 400.1 | 405.9 | 91.1 | 83.5 |
| dgem | sweep q1/w4 | 256/256 | 94.3 | 98.4 | 114.0 | 85.1 | 42.0 |
| dgem | sweep q1/w8 | 256/256 | 111.0 | 113.7 | 126.7 | 102.1 | 71.4 |
| dgem | sweep q5/w1 | 256/256 | 65.1 | 68.3 | 76.0 | 57.7 | 15.1 |
| dgem | sweep q5/w16 | 256/256 | 245.9 | 252.5 | 255.9 | 122.2 | 64.2 |
| dgem | sweep q5/w32 | 256/256 | 495.5 | 529.3 | 536.0 | 122.3 | 63.1 |
| dgem | sweep q5/w4 | 256/256 | 107.3 | 111.6 | 122.8 | 97.4 | 37.0 |
| dgem | sweep q5/w8 | 256/256 | 123.7 | 130.2 | 147.6 | 113.4 | 63.8 |

