# Regression matrix report: 20261002-w2-image-a2d5b0b-t2

- Matrix **v1**, tier **T2**, started 2026-10-02T19:13:04+00:00, finished 2026-10-02T19:35:42+00:00
- Repository commit `e5e92ef8eb`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | v0.1.3 | e685ef9 | a9eafde59c |
| new | vertex | v0.1.5-11-ga2d5b0b | a2d5b0b | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| health | new | **PASS** | ready |
| contract | new | **PASS** | all cases as expected |
| calibration | new | **PASS** | acc 0.867 vs 0.880; agreement 0.940 (floor 0.973, allowance 0.066); McNemar p=0.73 |
| jev_native | new | **PASS** | acc 0.815 vs 0.802; agreement 0.921 (floor 0.962, allowance 0.045); McNemar p=0.22 |
| jev_systemone | new | **PASS** | acc 0.841 vs 0.846; agreement 0.953 (floor 0.955, allowance 0.047); McNemar p=0.7 |
| intents_banking77 | new | **PASS** | acc 0.822 vs 0.778; agreement 0.915 (floor 0.917, allowance 0.121); McNemar p=0.22 |
| intents_clinc150 | new | **PASS** | acc 0.978 vs 0.989; agreement 0.981 (floor 0.978, allowance 0.074); McNemar p=1 |
| massive_spot | new | **PASS** | es 14/20, hi 17/20, ja 17/20, ru 18/20, th 16/20 |
| di_wide | new | **PASS** | acc 0.850 vs 0.860; agreement 0.972 (floor 0.977, allowance 0.050); McNemar p=0.51 |
| di_catchall | new | **PASS** | acc 0.730 vs 0.760; agreement 0.841 (floor 0.843, allowance 0.093); McNemar p=0.2 |
| rag_dev | new | **PASS** | acc 0.724 vs 0.699; agreement 0.913 (floor 0.921, allowance 0.058); McNemar p=0.053 |
| massive | new | **PASS** | acc 0.831 vs 0.825; agreement 0.937 (floor 0.941, allowance 0.027); McNemar p=0.053 [frozen set] |
| xnli | new | **REVIEW** | acc 0.672 vs 0.662; agreement 0.938 (floor 0.941, allowance 0.027); McNemar p=0.0057 [frozen set] — McNemar p=0.0057 (156 vs 110) |
| typed | new | **PASS** | acc 0.668 vs 0.674; agreement 0.935 (floor 0.941, allowance 0.031); McNemar p=0.36 [frozen set] |
| order | new | **PASS** | net flip vs baseline: within tolerance |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 22.5%; out-of-scope recall 98.3% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.490 (always-flag 0.601), recall 33.7%, predicted yes 16.2% vs gold 42.9% |
| di_catchall_oos_rate | new | **INFO** | in-scope answered as the catch-all: 26.7%; out-of-scope recall 96.7% (pooled runs) |
| rag_dev_yes_bias | new | **INFO** | hallucinated-class F1 0.561 (always-flag 0.601), recall 41.2%, predicted yes 20.0% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.920–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | new | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | new | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | new | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | new | **INFO** | top probability on unambiguous wide-option probes: 0.958–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_kit_compat | prod | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| di_kit_compat | new | **INFO** | skipped: set DGEM_DI_KIT_DIR and DGEM_DI_COMPAT_ROWS to run the kit compatibility pass |
| bbox | prod | **INFO** | acc_at_50_expectation_pct=36.364, mean_expectation_iou=0.412 |
| bbox | new | **INFO** | acc_at_50_expectation_pct=36.364, mean_expectation_iou=0.413 |
| decision_index | prod | **INFO** | headline_decision_index=98.889, ece_10bin=0.008 |
| decision_index | new | **INFO** | headline_decision_index=100, ece_10bin=0.026 |
| latency | new | **PASS** | within tolerance |

Overall: **new: REVIEW**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items.

| suite | target | runs: correct / n | coverage | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 43, 45 / 50 | 1.000 | 0.880 | 0.730 | 0.054 | 0.219 | 0.900 | 220 |
| calibration | new | 43, 44, 43 / 50 | 1.000 | 0.867 | 0.698 | 0.050 | 0.236 | 0.894 | 191 |
| jev_native | prod | 185, 185, 186 / 231 | 1.000 | 0.802 | 0.774 | 0.080 | 0.265 | 0.864 | 129 |
| jev_native | new | 184, 191, 190 / 231 | 1.000 | 0.815 | 0.783 | 0.080 | 0.261 | 0.847 | 132 |
| jev_systemone | prod | 193, 198, 195 / 231 | 1.000 | 0.846 | 0.809 | 0.087 | 0.250 | 0.746 | 220 |
| jev_systemone | new | 194, 195, 194 / 231 | 1.000 | 0.841 | 0.797 | 0.067 | 0.243 | 0.756 | 224 |
| intents_banking77 | prod | 24, 23, 23 / 30 | 1.000 | 0.778 | 0.709 | 0.153 | 0.418 | 0.853 | 400 |
| intents_banking77 | new | 25, 24, 25 / 30 | 1.000 | 0.822 | 0.763 | 0.138 | 0.324 | 0.793 | 406 |
| intents_clinc150 | prod | 30, 30, 29 / 30 | 1.000 | 0.989 | 0.983 | 0.053 | 0.044 | 0.966 | 140 |
| intents_clinc150 | new | 30, 29, 29 / 30 | 1.000 | 0.978 | 0.966 | 0.047 | 0.046 | 0.983 | 141 |
| massive_spot | prod | 87 / 100 | 1.000 | 0.870 | 0.901 | 0.118 | 0.245 | 0.752 | 222 |
| massive_spot | new | 82 / 100 | 1.000 | 0.820 | 0.809 | 0.121 | 0.273 | 0.838 | 222 |
| di_wide | prod | 85, 87, 86 / 100 | 1.000 | 0.860 | 0.818 | 0.093 | 0.242 | 0.813 | 319 |
| di_wide | new | 84, 85, 86 / 100 | 1.000 | 0.850 | 0.820 | 0.093 | 0.239 | 0.820 | 327 |
| di_catchall | prod | 77, 74, 77 / 100 | 1.000 | 0.760 | 0.665 | 0.098 | 0.360 | 0.777 | 803 |
| di_catchall | new | 76, 75, 68 / 100 | 1.000 | 0.730 | 0.640 | 0.125 | 0.424 | 0.778 | 812 |
| rag_dev | prod | 142, 136, 137 / 198 | 1.000 | 0.699 | 0.638 | 0.214 | 0.492 | 0.643 | 256 |
| rag_dev | new | 144, 144, 142 / 198 | 1.000 | 0.724 | 0.680 | 0.189 | 0.459 | 0.634 | 262 |
| massive | prod | 4209 / 5100 | 1.000 | 0.825 | 0.791 | 0.078 | 0.266 | 0.908 | 220 |
| massive | new | 4236 / 5100 | 1.000 | 0.831 | 0.793 | 0.075 | 0.261 | 0.906 | 214 |
| xnli | prod | 2979 / 4500 | 1.000 | 0.662 | 0.663 | 0.261 | 0.586 | 0.624 | 208 |
| xnli | new | 3025 / 4500 | 1.000 | 0.672 | 0.676 | 0.249 | 0.565 | 0.625 | 210 |
| typed | prod | 1348 / 2000 | 1.000 | 0.674 | 0.539 | 0.228 | 0.537 | 0.737 | 262 |
| typed | new | 1337 / 2000 | 1.000 | 0.668 | 0.529 | 0.235 | 0.543 | 0.747 | 314 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | mean |
|---|---|---|---|---|---|---|---|---|---|
| prod | 0.960 | 0.968 | 0.964 | 0.856 | 0.978 | 0.973 | 0.833 | 0.919 | 0.931 |
| new | 0.987 | 0.955 | 0.945 | 0.978 | 0.978 | 0.980 | 0.853 | 0.923 | 0.950 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.083 | 0.100 | 0.384 | 0.373 | 1.35, 1.20, 1.25, 1.25, 1.35 |
| calibration | new | 50 | 0.079 | 0.085 | 0.430 | 0.411 | 1.45, 1.25, 1.30, 1.40, 1.45 |
| jev_native | prod | 231 | 0.082 | 0.037 | 0.517 | 0.474 | 1.50, 1.50, 1.60, 1.50, 1.50 |
| jev_native | new | 231 | 0.092 | 0.044 | 0.525 | 0.473 | 1.55, 1.55, 1.65, 1.55, 1.55 |
| jev_systemone | prod | 231 | 0.084 | 0.066 | 0.584 | 0.498 | 1.80, 1.55, 1.85, 1.75, 1.75 |
| jev_systemone | new | 231 | 0.076 | 0.041 | 0.600 | 0.503 | 1.85, 1.65, 1.90, 1.75, 1.80 |
| intents_banking77 | prod | 29 | 0.204 | 0.186 | 0.931 | 0.759 | 1.65, 1.95, 1.75, 2.00, 1.90 |
| intents_banking77 | new | 29 | 0.122 | 0.142 | 0.848 | 0.679 | 1.70, 2.00, 1.60, 2.00, 1.85 |
| intents_clinc150 | prod | 30 | 0.068 | 0.022 | 0.094 | 0.087 | 0.75, 0.50, 0.75, 0.65, 0.65 |
| intents_clinc150 | new | 30 | 0.054 | 0.040 | 0.081 | 0.104 | 0.85, 0.50, 0.90, 0.80, 0.80 |
| massive_spot | prod | 100 | 0.118 | 0.102 | 0.635 | 0.552 | 1.50, 1.45, 1.50, 1.50, 1.40 |
| massive_spot | new | 100 | 0.121 | 0.049 | 0.687 | 0.565 | 1.65, 1.55, 1.60, 1.60, 1.55 |
| di_wide | prod | 100 | 0.090 | 0.034 | 0.760 | 0.720 | 1.25, 1.20, 1.35, 1.25, 1.30 |
| di_wide | new | 99 | 0.085 | 0.084 | 0.635 | 0.603 | 1.20, 1.25, 1.30, 1.20, 1.30 |
| di_catchall | prod | 99 | 0.079 | 0.097 | 1.156 | 1.122 | 1.20, 1.15, 1.20, 1.25, 1.15 |
| di_catchall | new | 97 | 0.080 | 0.061 | 1.021 | 1.014 | 1.10, 1.10, 1.20, 1.25, 1.15 |
| rag_dev | prod | 198 | 0.200 | 0.021 | 0.938 | 0.581 | 3.80, 4.05, 3.20, 3.75, 3.85 |
| rag_dev | new | 198 | 0.181 | 0.025 | 0.895 | 0.566 | 3.60, 3.65, 3.35, 3.55, 3.45 |
| massive | prod | 5097 | 0.078 | 0.031 | 0.866 | 0.748 | 1.55, 1.50, 1.50, 1.50, 1.50 |
| massive | new | 5098 | 0.075 | 0.030 | 0.849 | 0.734 | 1.55, 1.50, 1.50, 1.50, 1.50 |
| xnli | prod | 4472 | 0.257 | 0.019 | 1.657 | 0.812 | 4.20, 4.30, 4.25, 4.20, 4.15 |
| xnli | new | 4489 | 0.247 | 0.019 | 1.624 | 0.811 | 4.10, 4.15, 4.15, 4.10, 4.05 |
| typed | prod | 1979 | 0.221 | 0.017 | 1.281 | 0.752 | 3.40, 3.40, 3.50, 3.50, 3.35 |
| typed | new | 1986 | 0.230 | 0.018 | 1.315 | 0.756 | 3.45, 3.50, 3.55, 3.60, 3.45 |

## massive per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| af | 0.75 / 0.86 | 0.76 / 0.87 |
| am | 0.80 / 0.89 | 0.83 / 0.91 |
| ar | 0.85 / 0.93 | 0.86 / 0.93 |
| az | 0.82 / 0.89 | 0.81 / 0.90 |
| bn | 0.81 / 0.90 | 0.81 / 0.90 |
| cy | 0.47 / 0.69 | 0.42 / 0.70 |
| da | 0.87 / 0.92 | 0.87 / 0.93 |
| de | 0.87 / 0.93 | 0.89 / 0.94 |
| el | 0.85 / 0.93 | 0.88 / 0.93 |
| en | 0.87 / 0.95 | 0.91 / 0.96 |
| es | 0.88 / 0.94 | 0.88 / 0.94 |
| fa | 0.89 / 0.92 | 0.89 / 0.93 |
| fi | 0.84 / 0.91 | 0.85 / 0.92 |
| fr | 0.84 / 0.92 | 0.85 / 0.92 |
| he | 0.87 / 0.91 | 0.89 / 0.94 |
| hi | 0.89 / 0.95 | 0.90 / 0.95 |
| hu | 0.72 / 0.81 | 0.71 / 0.81 |
| hy | 0.77 / 0.87 | 0.78 / 0.86 |
| id | 0.89 / 0.94 | 0.89 / 0.96 |
| is | 0.78 / 0.87 | 0.77 / 0.87 |
| it | 0.94 / 0.95 | 0.94 / 0.94 |
| ja | 0.93 / 0.96 | 0.94 / 0.95 |
| jv | 0.71 / 0.86 | 0.76 / 0.86 |
| ka | 0.76 / 0.88 | 0.75 / 0.87 |
| km | 0.77 / 0.88 | 0.78 / 0.88 |
| kn | 0.85 / 0.90 | 0.83 / 0.92 |
| ko | 0.89 / 0.95 | 0.90 / 0.95 |
| lv | 0.67 / 0.83 | 0.69 / 0.81 |
| ml | 0.88 / 0.92 | 0.87 / 0.92 |
| mn | 0.80 / 0.86 | 0.79 / 0.85 |
| ms | 0.88 / 0.92 | 0.84 / 0.92 |
| my | 0.87 / 0.91 | 0.87 / 0.91 |
| nb | 0.85 / 0.91 | 0.84 / 0.91 |
| nl | 0.82 / 0.92 | 0.82 / 0.92 |
| pl | 0.88 / 0.93 | 0.89 / 0.94 |
| pt | 0.83 / 0.94 | 0.86 / 0.94 |
| ro | 0.83 / 0.90 | 0.81 / 0.89 |
| ru | 0.87 / 0.96 | 0.90 / 0.94 |
| sl | 0.71 / 0.83 | 0.74 / 0.83 |
| sq | 0.68 / 0.82 | 0.66 / 0.83 |
| sv | 0.82 / 0.92 | 0.84 / 0.92 |
| sw | 0.78 / 0.87 | 0.79 / 0.88 |
| ta | 0.85 / 0.91 | 0.85 / 0.90 |
| te | 0.82 / 0.92 | 0.83 / 0.93 |
| th | 0.89 / 0.93 | 0.89 / 0.95 |
| tl | 0.85 / 0.93 | 0.86 / 0.93 |
| tr | 0.86 / 0.90 | 0.87 / 0.91 |
| ur | 0.84 / 0.92 | 0.86 / 0.92 |
| vi | 0.86 / 0.94 | 0.86 / 0.93 |
| zh | 0.89 / 0.93 | 0.89 / 0.94 |
| **macro** | **0.824** | **0.829** |

## xnli per language (accuracy / mean confidence)

| lang | prod | new |
|---|---|---|
| ar | 0.66 / 0.93 | 0.67 / 0.92 |
| bg | 0.68 / 0.91 | 0.69 / 0.90 |
| de | 0.71 / 0.93 | 0.72 / 0.92 |
| el | 0.68 / 0.92 | 0.68 / 0.91 |
| en | 0.81 / 0.93 | 0.81 / 0.93 |
| es | 0.68 / 0.93 | 0.69 / 0.92 |
| fr | 0.70 / 0.93 | 0.73 / 0.93 |
| hi | 0.59 / 0.92 | 0.63 / 0.92 |
| ru | 0.65 / 0.93 | 0.66 / 0.93 |
| sw | 0.61 / 0.92 | 0.62 / 0.92 |
| th | 0.63 / 0.92 | 0.62 / 0.92 |
| tr | 0.64 / 0.91 | 0.65 / 0.91 |
| ur | 0.60 / 0.92 | 0.63 / 0.92 |
| vi | 0.64 / 0.93 | 0.64 / 0.92 |
| zh | 0.65 / 0.92 | 0.67 / 0.92 |
| **macro** | **0.662** | **0.672** |

## typed-decisions (soft metrics vs the teacher distribution)

| target | accuracy | soft acc | soft Brier | score MAE |
|---|---|---|---|---|
| prod | 0.674 | 0.554 | 0.303 | 0.412 |
| new | 0.668 | 0.554 | 0.309 | 0.428 |

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.040 | 0.110 (+0.070) | 0.090 (+0.050) | 0.420 / 0.305 |
| prod | order_jev_choice | 139 | 0.058 | 0.101 (+0.043) | 0.094 (+0.036) | 0.230 / 0.216 |
| prod | order_massive_en | 200 | 0.030 | 0.075 (+0.045) | 0.095 (+0.065) | 0.050 / 0.045 |
| prod | order_xnli_en | 200 | 0.070 | 0.060 (-0.010) | 0.060 (-0.010) | 0.285 / 0.335 |
| new | order_emotion | 200 | 0.025 | 0.100 (+0.075) | 0.095 (+0.070) | 0.425 / 0.305 |
| new | order_jev_choice | 139 | 0.058 | 0.122 (+0.065) | 0.137 (+0.079) | 0.237 / 0.216 |
| new | order_massive_en | 200 | 0.005 | 0.080 (+0.075) | 0.085 (+0.080) | 0.055 / 0.045 |
| new | order_xnli_en | 200 | 0.055 | 0.055 (+0.000) | 0.055 (+0.000) | 0.310 / 0.335 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.983 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.934 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.984 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.920 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.996 |
| prod | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.977 |
| prod | batch_12q | 200 | yes |  |
| prod | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| prod | noul_criteria | 200 | yes |  |
| prod | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.981 |
| prod | long_9k | 422 | yes | marker=maximum context length |
| prod | context_refusal | 422 | yes | marker=maximum context length |
| new | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.969 |
| new | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.958 |
| new | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.994 |
| new | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.974 |
| new | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.995 |
| new | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.977 |
| new | batch_12q | 200 | yes |  |
| new | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| new | noul_criteria | 200 | yes |  |
| new | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.997 |
| new | long_9k | 422 | yes | marker=maximum context length |
| new | context_refusal | 422 | yes | marker=maximum context length |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 91.2 | 96.2 | 109.4 | 53.8 | — |
| prod | q5 | 100/100 | 95.4 | 98.8 | 103.0 | 57.8 | — |
| prod | q10 | 100/100 | 101.3 | 105.8 | 237.8 | 61.4 | — |
| prod | long_q1 | 100/100 | 110.6 | 114.3 | 119.7 | 62.6 | — |
| prod | q5_s4 | 100/100 | 134.8 | 142.7 | 150.5 | 95.1 | — |
| prod | sweep q1/w16 | 256/256 | 218.1 | 224.8 | 605.9 | 108.8 | 65.2 |
| prod | sweep q1/w32 | 256/256 | 461.0 | 789.9 | 1088.9 | 113.0 | 58.7 |
| prod | sweep q5/w16 | 256/256 | 226.6 | 240.1 | 722.2 | 112.0 | 61.7 |
| prod | sweep q5/w32 | 256/256 | 449.1 | 793.7 | 1146.4 | 111.6 | 59.7 |
| new | q1 | 100/100 | 91.6 | 95.7 | 105.8 | 54.9 | — |
| new | q5 | 100/100 | 94.9 | 98.6 | 159.7 | 57.9 | — |
| new | q10 | 100/100 | 101.4 | 110.2 | 114.4 | 61.9 | — |
| new | long_q1 | 100/100 | 111.8 | 114.7 | 121.0 | 63.9 | — |
| new | q5_s4 | 100/100 | 136.0 | 142.1 | 152.1 | 97.0 | — |
| new | sweep q1/w16 | 256/256 | 179.9 | 193.0 | 606.1 | 89.3 | 75.9 |
| new | sweep q1/w32 | 256/256 | 440.4 | 860.9 | 1069.0 | 109.5 | 59.9 |
| new | sweep q5/w16 | 256/256 | 223.3 | 239.8 | 645.8 | 111.2 | 63.5 |
| new | sweep q5/w32 | 256/256 | 451.9 | 910.6 | 1212.7 | 112.6 | 58.4 |

