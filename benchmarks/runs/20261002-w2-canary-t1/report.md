# Regression matrix report: 20261002-w2-canary

- Matrix **v1**, tier **T1**, started 2026-10-02T18:56:55+00:00, finished 2026-10-02T19:11:34+00:00
- Repository commit `e5e92ef8eb`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | v0.1.3 | e685ef9 | a9eafde59c |
| nvfp4 | vertex | v0.1.5-11-ga2d5b0b | a2d5b0b | a9eafde59c |
| docfirst | vertex | v0.1.5-11-ga2d5b0b | a2d5b0b | a9eafde59c |
| bf16 | vertex | v0.1.5-11-ga2d5b0b | a2d5b0b | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| health | nvfp4 | **PASS** | ready |
| health | docfirst | **PASS** | ready |
| health | bf16 | **PASS** | ready |
| contract | nvfp4 | **PASS** | all cases as expected |
| contract | docfirst | **PASS** | all cases as expected |
| contract | bf16 | **PASS** | all cases as expected |
| calibration | nvfp4 | **PASS** | acc 0.873 vs 0.867; agreement 0.929 (floor 0.947, allowance 0.084); McNemar p=1 |
| calibration | docfirst | **PASS** | acc 0.887 vs 0.867; agreement 0.947 (floor 0.947, allowance 0.084); McNemar p=0.38 |
| calibration | bf16 | **PASS** | acc 0.853 vs 0.867; agreement 0.942 (floor 0.953, allowance 0.080); McNemar p=0.69 |
| jev_native | nvfp4 | **PASS** | acc 0.812 vs 0.801; agreement 0.931 (floor 0.933, allowance 0.053); McNemar p=0.27 |
| jev_native | docfirst | **PASS** | acc 0.810 vs 0.801; agreement 0.933 (floor 0.935, allowance 0.052); McNemar p=0.43 |
| jev_native | bf16 | **REVIEW** | acc 0.830 vs 0.801; agreement 0.929 (floor 0.964, allowance 0.045); McNemar p=0.0022 — McNemar p=0.0022 (30 vs 10) |
| jev_systemone | nvfp4 | **PASS** | acc 0.850 vs 0.840; agreement 0.937 (floor 0.943, allowance 0.051); McNemar p=0.34 |
| jev_systemone | docfirst | **PASS** | acc 0.857 vs 0.840; agreement 0.912 (floor 0.955, allowance 0.047); McNemar p=0.11 |
| jev_systemone | bf16 | **PASS** | acc 0.846 vs 0.840; agreement 0.943 (floor 0.963, allowance 0.045); McNemar p=0.61 |
| intents_banking77 | nvfp4 | **PASS** | acc 0.822 vs 0.789; agreement 0.863 (floor 0.889, allowance 0.135); McNemar p=0.45 |
| intents_banking77 | docfirst | **PASS** | acc 0.800 vs 0.789; agreement 0.856 (floor 0.856, allowance 0.148); McNemar p=1 |
| intents_banking77 | bf16 | **PASS** | acc 0.811 vs 0.789; agreement 0.852 (floor 0.900, allowance 0.130); McNemar p=0.75 |
| intents_clinc150 | nvfp4 | **PASS** | acc 1.000 vs 1.000; agreement 1.000 (floor 1.000, allowance 0.020); McNemar p=1 |
| intents_clinc150 | docfirst | **PASS** | acc 0.978 vs 1.000; agreement 0.978 (floor 0.989, allowance 0.058); McNemar p=0.5 |
| intents_clinc150 | bf16 | **REVIEW** | acc 0.967 vs 1.000; agreement 0.967 (floor 1.000, allowance 0.020); McNemar p=0.25 — agreement 0.967 < noise floor 1.000 - 0.020 |
| massive_spot | nvfp4 | **PASS** | es 15/20, hi 17/20, ja 18/20, ru 17/20, th 16/20 |
| massive_spot | docfirst | **PASS** | es 18/20, hi 20/20, ja 19/20, ru 20/20, th 19/20 |
| massive_spot | bf16 | **PASS** | es 16/20, hi 18/20, ja 18/20, ru 19/20, th 18/20 |
| di_wide | nvfp4 | **PASS** | acc 0.863 vs 0.860; agreement 0.969 (floor 0.968, allowance 0.055); McNemar p=1 |
| di_wide | docfirst | **PASS** | acc 0.870 vs 0.860; agreement 0.916 (floor 0.958, allowance 0.060); McNemar p=0.66 |
| di_wide | bf16 | **PASS** | acc 0.857 vs 0.860; agreement 0.976 (floor 0.980, allowance 0.048); McNemar p=1 |
| di_catchall | nvfp4 | **PASS** | acc 0.790 vs 0.755; agreement 0.863 (floor 0.840, allowance 0.093); McNemar p=0.19 |
| di_catchall | docfirst | **FAIL** | acc 0.870 vs 0.755; agreement 0.765 (floor 0.890, allowance 0.083); McNemar p=0.00029 — agreement 0.765 < noise floor 0.890 - 0.083; McNemar p=0.00029 (31 vs 8) |
| di_catchall | bf16 | **PASS** | acc 0.790 vs 0.755; agreement 0.865 (floor 0.885, allowance 0.084); McNemar p=0.23 |
| rag_dev | nvfp4 | **PASS** | acc 0.705 vs 0.730; agreement 0.914 (floor 0.904, allowance 0.062); McNemar p=0.11 |
| rag_dev | docfirst | **PASS** | acc 0.770 vs 0.730; agreement 0.879 (floor 0.919, allowance 0.059); McNemar p=0.029 |
| rag_dev | bf16 | **PASS** | acc 0.712 vs 0.730; agreement 0.909 (floor 0.917, allowance 0.059); McNemar p=0.32 |
| di_catchall_oos_rate | prod | **INFO** | in-scope answered as the catch-all: 23.1%; out-of-scope recall 100.0% (pooled runs) |
| rag_dev_yes_bias | prod | **INFO** | hallucinated-class F1 0.580 (always-flag 0.601), recall 43.5%, predicted yes 21.5% vs gold 42.9% |
| di_catchall_oos_rate | nvfp4 | **INFO** | in-scope answered as the catch-all: 21.2%; out-of-scope recall 100.0% (pooled runs) |
| rag_dev_yes_bias | nvfp4 | **INFO** | hallucinated-class F1 0.522 (always-flag 0.601), recall 37.6%, predicted yes 18.9% vs gold 42.9% |
| di_catchall_oos_rate | docfirst | **INFO** | in-scope answered as the catch-all: 2.5%; out-of-scope recall 82.5% (pooled runs) |
| rag_dev_yes_bias | docfirst | **INFO** | hallucinated-class F1 0.676 (always-flag 0.601), recall 55.9%, predicted yes 28.0% vs gold 42.9% |
| di_catchall_oos_rate | bf16 | **INFO** | in-scope answered as the catch-all: 18.1%; out-of-scope recall 90.0% (pooled runs) |
| rag_dev_yes_bias | bf16 | **INFO** | hallucinated-class F1 0.558 (always-flag 0.601), recall 42.4%, predicted yes 22.2% vs gold 42.9% |
| di_probes | prod | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | prod | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | prod | **INFO** | top probability on unambiguous wide-option probes: 0.971–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | nvfp4 | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | nvfp4 | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | nvfp4 | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | nvfp4 | **INFO** | top probability on unambiguous wide-option probes: 0.971–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | docfirst | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | docfirst | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | docfirst | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | docfirst | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | bf16 | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | bf16 | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | bf16 | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | bf16 | **INFO** | top probability on unambiguous wide-option probes: 0.980–0.999 (a flat ceiling across K means bracket fusion is capping confidence) |
| latency | nvfp4 | **PASS** | within tolerance |
| latency | docfirst | **PASS** | within tolerance |
| latency | bf16 | **PASS** | within tolerance |

Overall: **nvfp4: PASS**, **docfirst: FAIL**, **bf16: REVIEW**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items.

| suite | target | runs: correct / n | coverage | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|
| calibration | prod | 44, 43, 43 / 50 | 1.000 | 0.867 | 0.715 | 0.077 | 0.230 | 0.902 | 215 |
| calibration | nvfp4 | 43, 45, 43 / 50 | 1.000 | 0.873 | 0.738 | 0.049 | 0.201 | 0.940 | 185 |
| calibration | docfirst | 44, 46, 43 / 50 | 1.000 | 0.887 | 0.731 | 0.060 | 0.208 | 0.898 | 213 |
| calibration | bf16 | 43, 42, 43 / 50 | 1.000 | 0.853 | 0.695 | 0.072 | 0.228 | 0.949 | 184 |
| jev_native | prod | 181, 186, 188 / 231 | 1.000 | 0.801 | 0.774 | 0.084 | 0.267 | 0.864 | 130 |
| jev_native | nvfp4 | 187, 189, 187 / 231 | 1.000 | 0.812 | 0.784 | 0.079 | 0.267 | 0.845 | 129 |
| jev_native | docfirst | 184, 188, 189 / 231 | 1.000 | 0.810 | 0.780 | 0.082 | 0.273 | 0.842 | 130 |
| jev_native | bf16 | 190, 193, 192 / 231 | 1.000 | 0.830 | 0.802 | 0.088 | 0.258 | 0.818 | 151 |
| jev_systemone | prod | 193, 194, 195 / 231 | 1.000 | 0.840 | 0.797 | 0.072 | 0.248 | 0.759 | 225 |
| jev_systemone | nvfp4 | 199, 197, 193 / 231 | 1.000 | 0.850 | 0.810 | 0.074 | 0.235 | 0.749 | 226 |
| jev_systemone | docfirst | 198, 196, 200 / 231 | 1.000 | 0.857 | 0.818 | 0.065 | 0.222 | 0.832 | 235 |
| jev_systemone | bf16 | 196, 195, 195 / 231 | 1.000 | 0.846 | 0.814 | 0.090 | 0.237 | 0.792 | 245 |
| intents_banking77 | prod | 25, 21, 25 / 30 | 1.000 | 0.789 | 0.730 | 0.157 | 0.340 | 0.856 | 403 |
| intents_banking77 | nvfp4 | 25, 23, 26 / 30 | 1.000 | 0.822 | 0.765 | 0.138 | 0.303 | 0.790 | 433 |
| intents_banking77 | docfirst | 25, 22, 25 / 30 | 1.000 | 0.800 | 0.741 | 0.160 | 0.341 | 0.816 | 405 |
| intents_banking77 | bf16 | 25, 24, 24 / 30 | 1.000 | 0.811 | 0.748 | 0.131 | 0.324 | 0.816 | 417 |
| intents_clinc150 | prod | 30, 30, 30 / 30 | 1.000 | 1.000 | 1.000 | 0.053 | 0.046 | — | 140 |
| intents_clinc150 | nvfp4 | 30, 30, 30 / 30 | 1.000 | 1.000 | 1.000 | 0.037 | 0.045 | — | 143 |
| intents_clinc150 | docfirst | 29, 30, 29 / 30 | 1.000 | 0.978 | 0.966 | 0.037 | 0.061 | 0.862 | 142 |
| intents_clinc150 | bf16 | 29, 29, 29 / 30 | 1.000 | 0.967 | 0.949 | 0.040 | 0.076 | 0.851 | 136 |
| massive_spot | prod | 86 / 100 | 1.000 | 0.860 | 0.849 | 0.106 | 0.212 | 0.871 | 230 |
| massive_spot | nvfp4 | 83 / 100 | 1.000 | 0.830 | 0.818 | 0.137 | 0.296 | 0.818 | 231 |
| massive_spot | docfirst | 96 / 100 | 1.000 | 0.960 | 0.968 | 0.029 | 0.073 | 0.846 | 255 |
| massive_spot | bf16 | 89 / 100 | 1.000 | 0.890 | 0.910 | 0.105 | 0.209 | 0.768 | 225 |
| di_wide | prod | 87, 85, 86 / 100 | 1.000 | 0.860 | 0.830 | 0.084 | 0.237 | 0.801 | 322 |
| di_wide | nvfp4 | 87, 87, 85 / 100 | 1.000 | 0.863 | 0.828 | 0.091 | 0.233 | 0.814 | 344 |
| di_wide | docfirst | 89, 86, 86 / 100 | 1.000 | 0.870 | 0.832 | 0.079 | 0.195 | 0.922 | 447 |
| di_wide | bf16 | 86, 86, 85 / 100 | 1.000 | 0.857 | 0.814 | 0.093 | 0.229 | 0.833 | 368 |
| di_catchall | prod | 76, 75 / 100 | 1.000 | 0.755 | 0.655 | 0.102 | 0.377 | 0.762 | 806 |
| di_catchall | nvfp4 | 77, 81 / 100 | 1.000 | 0.790 | 0.699 | 0.101 | 0.358 | 0.737 | 809 |
| di_catchall | docfirst | 89, 85 / 100 | 1.000 | 0.870 | 0.834 | 0.077 | 0.238 | 0.692 | 812 |
| di_catchall | bf16 | 78, 80 / 100 | 1.000 | 0.790 | 0.723 | 0.123 | 0.329 | 0.811 | 1231 |
| rag_dev | prod | 144, 145 / 198 | 1.000 | 0.730 | 0.691 | 0.173 | 0.448 | 0.599 | 257 |
| rag_dev | nvfp4 | 140, 139 / 198 | 1.000 | 0.705 | 0.654 | 0.208 | 0.484 | 0.665 | 266 |
| rag_dev | docfirst | 153, 152 / 198 | 1.000 | 0.770 | 0.749 | 0.183 | 0.411 | 0.665 | 289 |
| rag_dev | bf16 | 140, 142 / 198 | 1.000 | 0.712 | 0.672 | 0.178 | 0.451 | 0.642 | 392 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | mean |
|---|---|---|---|---|---|---|---|---|---|
| prod | 0.960 | 0.951 | 0.952 | 0.867 | 1.000 | 0.973 | 0.840 | 0.894 | 0.930 |
| nvfp4 | 0.933 | 0.915 | 0.934 | 0.911 | 1.000 | 0.963 | 0.840 | 0.914 | 0.926 |
| docfirst | 0.933 | 0.919 | 0.957 | 0.844 | 0.978 | 0.943 | 0.940 | 0.944 | 0.932 |
| bf16 | 0.947 | 0.977 | 0.974 | 0.933 | 1.000 | 0.987 | 0.930 | 0.939 | 0.961 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | prod | 50 | 0.064 | 0.060 | 0.365 | 0.357 | 1.25, 1.15, 1.25, 1.20, 1.25 |
| calibration | nvfp4 | 50 | 0.078 | 0.083 | 0.341 | 0.341 | 1.25, 1.10, 1.15, 1.15, 1.25 |
| calibration | docfirst | 50 | 0.076 | 0.031 | 0.381 | 0.387 | 1.35, 1.05, 1.25, 1.25, 1.35 |
| calibration | bf16 | 50 | 0.090 | 0.081 | 0.359 | 0.345 | 1.40, 1.25, 1.25, 1.30, 1.40 |
| jev_native | prod | 231 | 0.102 | 0.063 | 0.533 | 0.480 | 1.55, 1.55, 1.65, 1.55, 1.55 |
| jev_native | nvfp4 | 231 | 0.081 | 0.041 | 0.522 | 0.475 | 1.50, 1.50, 1.65, 1.50, 1.55 |
| jev_native | docfirst | 231 | 0.090 | 0.046 | 0.559 | 0.497 | 1.60, 1.60, 1.70, 1.55, 1.60 |
| jev_native | bf16 | 231 | 0.084 | 0.051 | 0.562 | 0.470 | 1.80, 1.75, 1.90, 1.80, 1.80 |
| jev_systemone | prod | 231 | 0.088 | 0.061 | 0.595 | 0.501 | 1.75, 1.65, 1.85, 1.75, 1.80 |
| jev_systemone | nvfp4 | 231 | 0.072 | 0.056 | 0.569 | 0.490 | 1.75, 1.55, 1.80, 1.70, 1.75 |
| jev_systemone | docfirst | 231 | 0.054 | 0.034 | 0.446 | 0.408 | 1.45, 1.45, 1.60, 1.50, 1.55 |
| jev_systemone | bf16 | 231 | 0.089 | 0.021 | 0.574 | 0.457 | 1.90, 1.80, 2.00, 1.90, 1.95 |
| intents_banking77 | prod | 30 | 0.130 | 0.125 | 1.077 | 0.822 | 1.70, 2.15, 1.95, 1.90, 2.00 |
| intents_banking77 | nvfp4 | 29 | 0.157 | 0.116 | 0.939 | 0.774 | 1.55, 2.15, 1.80, 2.10, 1.95 |
| intents_banking77 | docfirst | 29 | 0.142 | 0.078 | 0.953 | 0.815 | 1.45, 2.10, 1.80, 2.00, 1.90 |
| intents_banking77 | bf16 | 28 | 0.145 | 0.089 | 0.917 | 0.700 | 1.70, 2.05, 1.80, 2.10, 1.95 |
| intents_clinc150 | prod | 30 | 0.047 | 0.035 | 0.097 | 0.075 | 0.70, 0.50, 0.65, 0.60, 0.60 |
| intents_clinc150 | nvfp4 | 30 | 0.063 | 0.043 | 0.088 | 0.086 | 0.75, 0.50, 0.75, 0.70, 0.70 |
| intents_clinc150 | docfirst | 30 | 0.021 | 0.028 | 0.147 | 0.251 | 1.00, 0.50, 1.00, 0.95, 0.95 |
| intents_clinc150 | bf16 | 30 | 0.043 | 0.039 | 0.196 | 0.383 | 1.30, 0.50, 1.30, 1.25, 1.25 |
| massive_spot | prod | 100 | 0.106 | 0.050 | 0.528 | 0.476 | 1.45, 1.35, 1.40, 1.45, 1.35 |
| massive_spot | nvfp4 | 100 | 0.137 | 0.064 | 0.791 | 0.658 | 1.65, 1.55, 1.65, 1.60, 1.45 |
| massive_spot | docfirst | 100 | 0.029 | 0.050 | 0.235 | 0.252 | 1.30, 1.00, 1.20, 1.35, 1.40 |
| massive_spot | bf16 | 100 | 0.105 | 0.045 | 0.581 | 0.477 | 1.60, 1.55, 1.55, 1.55, 1.50 |
| di_wide | prod | 100 | 0.077 | 0.057 | 0.698 | 0.667 | 1.25, 1.15, 1.30, 1.25, 1.30 |
| di_wide | nvfp4 | 99 | 0.068 | 0.076 | 0.758 | 0.725 | 1.15, 1.30, 1.35, 1.25, 1.35 |
| di_wide | docfirst | 100 | 0.054 | 0.057 | 0.573 | 0.543 | 1.25, 1.20, 1.35, 1.35, 1.35 |
| di_wide | bf16 | 100 | 0.098 | 0.076 | 0.789 | 0.713 | 1.30, 1.35, 1.45, 1.35, 1.45 |
| di_catchall | prod | 99 | 0.081 | 0.051 | 1.039 | 1.027 | 1.15, 1.15, 1.25, 1.20, 1.10 |
| di_catchall | nvfp4 | 99 | 0.094 | 0.062 | 1.119 | 1.093 | 1.15, 1.25, 1.25, 1.20, 1.15 |
| di_catchall | docfirst | 100 | 0.049 | 0.055 | 0.788 | 0.776 | 1.15, 1.20, 1.25, 1.20, 1.10 |
| di_catchall | bf16 | 99 | 0.126 | 0.043 | 0.859 | 0.806 | 1.20, 1.30, 1.30, 1.35, 1.30 |
| rag_dev | prod | 198 | 0.187 | 0.092 | 0.937 | 0.584 | 3.85, 3.75, 3.60, 3.85, 3.75 |
| rag_dev | nvfp4 | 198 | 0.222 | 0.030 | 0.952 | 0.570 | 3.95, 4.00, 3.50, 3.70, 3.70 |
| rag_dev | docfirst | 198 | 0.177 | 0.011 | 0.834 | 0.509 | 3.30, 3.45, 3.05, 3.35, 3.30 |
| rag_dev | bf16 | 198 | 0.179 | 0.007 | 0.842 | 0.573 | 3.40, 3.40, 3.05, 3.25, 3.25 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| prod | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.983 |
| prod | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.990 |
| prod | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.978 |
| prod | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.971 |
| prod | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.985 |
| prod | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.985 |
| prod | batch_12q | 200 | yes |  |
| prod | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| prod | noul_criteria | 200 | yes |  |
| prod | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.981 |
| prod | long_9k | 422 | yes | marker=maximum context length |
| prod | context_refusal | 422 | yes | marker=maximum context length |
| nvfp4 | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.983 |
| nvfp4 | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.990 |
| nvfp4 | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.978 |
| nvfp4 | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.971 |
| nvfp4 | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.985 |
| nvfp4 | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.985 |
| nvfp4 | batch_12q | 200 | yes |  |
| nvfp4 | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| nvfp4 | noul_criteria | 200 | yes |  |
| nvfp4 | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.981 |
| nvfp4 | long_9k | 422 | yes | marker=maximum context length |
| nvfp4 | context_refusal | 422 | yes | marker=maximum context length |
| docfirst | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| docfirst | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| docfirst | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| docfirst | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| docfirst | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| docfirst | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| docfirst | batch_12q | 200 | yes |  |
| docfirst | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| docfirst | noul_criteria | 200 | yes |  |
| docfirst | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
| docfirst | long_9k | 422 | yes | marker=maximum context length |
| docfirst | context_refusal | 422 | yes | marker=maximum context length |
| bf16 | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.989 |
| bf16 | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.989 |
| bf16 | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.980 |
| bf16 | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.991 |
| bf16 | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.994 |
| bf16 | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.982 |
| bf16 | batch_12q | 200 | yes |  |
| bf16 | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.999 |
| bf16 | noul_criteria | 200 | yes |  |
| bf16 | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.994 |
| bf16 | long_9k | 422 | yes | marker=maximum context length |
| bf16 | context_refusal | 422 | yes | marker=maximum context length |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 91.2 | 98.0 | 103.0 | 53.9 | — |
| prod | q5 | 100/100 | 93.0 | 97.1 | 102.5 | 55.7 | — |
| prod | q10 | 100/100 | 98.8 | 103.1 | 206.0 | 59.0 | — |
| prod | long_q1 | 100/100 | 109.1 | 113.2 | 118.3 | 59.9 | — |
| prod | q5_s4 | 100/100 | 134.2 | 138.0 | 144.8 | 95.1 | — |
| prod | sweep q1/w16 | 256/256 | 219.4 | 226.1 | 645.4 | 109.6 | 64.3 |
| prod | sweep q1/w32 | 256/256 | 442.4 | 826.0 | 1144.6 | 109.4 | 59.2 |
| prod | sweep q5/w16 | 256/256 | 218.7 | 232.4 | 621.7 | 108.9 | 64.6 |
| prod | sweep q5/w32 | 256/256 | 453.3 | 779.2 | 1099.0 | 112.9 | 59.2 |
| nvfp4 | q1 | 100/100 | 92.6 | 98.9 | 111.0 | 55.7 | — |
| nvfp4 | q5 | 100/100 | 96.7 | 101.6 | 107.1 | 58.1 | — |
| nvfp4 | q10 | 100/100 | 101.7 | 104.7 | 113.6 | 62.4 | — |
| nvfp4 | long_q1 | 100/100 | 110.2 | 115.5 | 143.1 | 62.8 | — |
| nvfp4 | q5_s4 | 100/100 | 136.6 | 139.9 | 147.1 | 97.4 | — |
| nvfp4 | sweep q1/w16 | 256/256 | 227.3 | 269.7 | 585.8 | 113.2 | 62.8 |
| nvfp4 | sweep q1/w32 | 256/256 | 445.3 | 746.6 | 1096.7 | 110.6 | 59.5 |
| nvfp4 | sweep q5/w16 | 256/256 | 229.3 | 244.1 | 669.4 | 114.1 | 61.9 |
| nvfp4 | sweep q5/w32 | 256/256 | 468.6 | 779.1 | 1078.3 | 116.6 | 58.2 |
| docfirst | q1 | 100/100 | 93.1 | 98.0 | 102.2 | 55.6 | — |
| docfirst | q5 | 100/100 | 95.7 | 171.0 | 256.4 | 58.7 | — |
| docfirst | q10 | 100/100 | 101.5 | 108.2 | 111.9 | 62.6 | — |
| docfirst | long_q1 | 100/100 | 109.8 | 114.4 | 128.5 | 63.3 | — |
| docfirst | q5_s4 | 100/100 | 135.8 | 138.6 | 146.4 | 98.1 | — |
| docfirst | sweep q1/w16 | 256/256 | 184.8 | 195.7 | 551.3 | 91.5 | 75.4 |
| docfirst | sweep q1/w32 | 256/256 | 456.3 | 745.2 | 1015.3 | 113.1 | 58.8 |
| docfirst | sweep q5/w16 | 256/256 | 233.2 | 238.8 | 641.1 | 116.2 | 60.6 |
| docfirst | sweep q5/w32 | 256/256 | 466.8 | 875.1 | 1230.3 | 115.9 | 56.9 |
| bf16 | q1 | 100/100 | 86.8 | 93.5 | 105.0 | 49.5 | — |
| bf16 | q5 | 100/100 | 91.5 | 95.7 | 100.9 | 52.0 | — |
| bf16 | q10 | 100/100 | 96.0 | 99.5 | 125.1 | 55.3 | — |
| bf16 | long_q1 | 100/100 | 105.2 | 109.3 | 115.0 | 57.3 | — |
| bf16 | q5_s4 | 100/100 | 128.2 | 131.1 | 134.6 | 89.2 | — |
| bf16 | sweep q1/w16 | 256/256 | 211.2 | 254.2 | 712.2 | 105.5 | 66.2 |
| bf16 | sweep q1/w32 | 256/256 | 356.1 | 813.5 | 1190.0 | 85.7 | 72.2 |
| bf16 | sweep q5/w16 | 256/256 | 206.1 | 214.9 | 568.4 | 102.7 | 69.2 |
| bf16 | sweep q5/w32 | 256/256 | 419.1 | 665.0 | 902.9 | 104.4 | 64.7 |

