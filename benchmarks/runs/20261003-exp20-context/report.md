# Regression matrix report: 20261003-exp20-context-t1

- Matrix **v1**, tier **T1**, started 2026-10-03T06:11:12+00:00, finished 2026-10-03T08:45:57+00:00
- Repository commit `a22657a5e1`
- Mode: baseline `ctx4k` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| ctx4k (baseline) | vertex | v0.2.0 | 027b68e | a9eafde59c |
| ctx8k | vertex | v0.2.0 | 027b68e | a9eafde59c |
| ctx32k | vertex | v0.2.0 | 027b68e | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | ctx4k | **PASS** | ready |
| health | ctx8k | **PASS** | ready |
| health | ctx32k | **PASS** | ready |
| contract | ctx8k | **PASS** | all cases as expected |
| contract | ctx32k | **PASS** | all cases as expected |
| calibration | ctx8k | **PASS** | acc 0.860 vs 0.880; agreement 0.980 (floor 0.987, allowance 0.052); McNemar p=0.25 |
| calibration | ctx32k | **PASS** | acc 0.860 vs 0.880; agreement 0.980 (floor 0.987, allowance 0.052); McNemar p=0.25 |
| jev_native | ctx8k | **PASS** | acc 0.810 vs 0.807; agreement 0.957 (floor 0.965, allowance 0.044); McNemar p=1 |
| jev_native | ctx32k | **PASS** | acc 0.807 vs 0.807; agreement 0.957 (floor 0.962, allowance 0.045); McNemar p=1 |
| jev_systemone | ctx8k | **PASS** | acc 0.847 vs 0.846; agreement 0.937 (floor 0.939, allowance 0.051); McNemar p=1 |
| jev_systemone | ctx32k | **PASS** | acc 0.841 vs 0.846; agreement 0.944 (floor 0.944, allowance 0.050); McNemar p=0.74 |
| intents_banking77 | ctx8k | **PASS** | acc 0.778 vs 0.778; agreement 0.922 (floor 0.900, allowance 0.130); McNemar p=1 |
| intents_banking77 | ctx32k | **PASS** | acc 0.789 vs 0.778; agreement 0.919 (floor 0.911, allowance 0.124); McNemar p=1 |
| intents_clinc150 | ctx8k | **PASS** | acc 0.956 vs 0.967; agreement 0.989 (floor 0.989, allowance 0.058); McNemar p=1 |
| intents_clinc150 | ctx32k | **PASS** | acc 0.967 vs 0.967; agreement 1.000 (floor 1.000, allowance 0.020); McNemar p=1 |
| massive_spot | ctx8k | **PASS** | es 19/20, hi 20/20, ja 18/20, ru 20/20, th 19/20 |
| massive_spot | ctx32k | **PASS** | es 19/20, hi 20/20, ja 19/20, ru 20/20, th 19/20 |
| di_wide | ctx8k | **PASS** | acc 0.873 vs 0.870; agreement 0.944 (floor 0.940, allowance 0.067); McNemar p=1 |
| di_wide | ctx32k | **PASS** | acc 0.873 vs 0.870; agreement 0.939 (floor 0.935, allowance 0.069); McNemar p=1 |
| di_catchall | ctx8k | **PASS** | acc 0.865 vs 0.850; agreement 0.930 (floor 0.935, allowance 0.069); McNemar p=0.55 |
| di_catchall | ctx32k | **PASS** | acc 0.870 vs 0.850; agreement 0.953 (floor 0.955, allowance 0.061); McNemar p=0.22 |
| rag_dev | ctx8k | **PASS** | acc 0.778 vs 0.763; agreement 0.932 (floor 0.939, allowance 0.054); McNemar p=0.36 |
| rag_dev | ctx32k | **PASS** | acc 0.773 vs 0.763; agreement 0.932 (floor 0.934, allowance 0.055); McNemar p=0.54 |
| coverage | ctx8k | **PASS** | no coverage loss |
| coverage | ctx32k | **PASS** | no coverage loss |
| di_catchall_oos_rate | ctx4k | **INFO** | in-scope answered as the catch-all: 3.8%; out-of-scope recall 87.5% (pooled runs) |
| rag_dev_yes_bias | ctx4k | **INFO** | hallucinated-class F1 0.667 (always-flag 0.601), recall 55.3%, predicted yes 28.3% vs gold 42.9% |
| di_catchall_oos_rate | ctx8k | **INFO** | in-scope answered as the catch-all: 2.5%; out-of-scope recall 85.0% (pooled runs) |
| rag_dev_yes_bias | ctx8k | **INFO** | hallucinated-class F1 0.688 (always-flag 0.601), recall 57.1%, predicted yes 28.3% vs gold 42.9% |
| di_catchall_oos_rate | ctx32k | **INFO** | in-scope answered as the catch-all: 3.8%; out-of-scope recall 90.0% (pooled runs) |
| rag_dev_yes_bias | ctx32k | **INFO** | hallucinated-class F1 0.685 (always-flag 0.601), recall 57.6%, predicted yes 29.3% vs gold 42.9% |
| di_probes | ctx4k | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | ctx4k | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | ctx4k | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | ctx4k | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | ctx8k | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | ctx8k | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | ctx8k | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | ctx8k | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| di_probes | ctx32k | **PASS** | all 10 gated probes as expected |
| di_probe_noul_criteria | ctx32k | **INFO** | as expected (known adapter gap; not gated) |
| di_probe_wide_catchall_151 | ctx32k | **INFO** | as expected (known adapter gap; not gated) |
| di_confidence_cap | ctx32k | **INFO** | top probability on unambiguous wide-option probes: 0.909–0.998 (a flat ceiling across K means bracket fusion is capping confidence) |
| latency | ctx8k | **PASS** | within tolerance |
| latency | ctx32k | **FAIL** | 1124 errors |

Overall: **ctx8k: PASS**, **ctx32k: FAIL**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| calibration | ctx4k | 44, 45, 43 / 50 | 1.000 | — | 0.880 | 0.677 | 0.077 | 0.243 | 0.797 | 219 |
| calibration | ctx8k | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.664 | 0.084 | 0.255 | 0.829 | 220 |
| calibration | ctx32k | 43, 43, 43 / 50 | 1.000 | — | 0.860 | 0.664 | 0.073 | 0.268 | 0.816 | 225 |
| jev_native | ctx4k | 185, 186, 186 / 230 | 0.996 | context 3 | 0.807 | 0.798 | 0.097 | 0.286 | 0.803 | 133 |
| jev_native | ctx8k | 186, 188, 187 / 231 | 1.000 | — | 0.810 | 0.800 | 0.093 | 0.289 | 0.790 | 130 |
| jev_native | ctx32k | 187, 184, 188 / 231 | 1.000 | — | 0.807 | 0.804 | 0.093 | 0.282 | 0.810 | 144 |
| jev_systemone | ctx4k | 196, 197, 193 / 231 | 1.000 | — | 0.846 | 0.818 | 0.062 | 0.224 | 0.857 | 226 |
| jev_systemone | ctx8k | 196, 196, 195 / 231 | 1.000 | — | 0.847 | 0.813 | 0.065 | 0.219 | 0.865 | 223 |
| jev_systemone | ctx32k | 193, 194, 196 / 231 | 1.000 | — | 0.841 | 0.806 | 0.074 | 0.229 | 0.850 | 233 |
| intents_banking77 | ctx4k | 24, 23, 23 / 30 | 1.000 | — | 0.778 | 0.720 | 0.206 | 0.358 | 0.722 | 399 |
| intents_banking77 | ctx8k | 24, 24, 22 / 30 | 1.000 | — | 0.778 | 0.715 | 0.180 | 0.301 | 0.743 | 424 |
| intents_banking77 | ctx32k | 24, 23, 24 / 30 | 1.000 | — | 0.789 | 0.728 | 0.164 | 0.276 | 0.723 | 424 |
| intents_clinc150 | ctx4k | 29, 29, 29 / 30 | 1.000 | — | 0.967 | 0.949 | 0.048 | 0.023 | 0.931 | 149 |
| intents_clinc150 | ctx8k | 28, 29, 29 / 30 | 1.000 | — | 0.956 | 0.932 | 0.051 | 0.025 | 0.948 | 150 |
| intents_clinc150 | ctx32k | 29, 29, 29 / 30 | 1.000 | — | 0.967 | 0.949 | 0.051 | 0.025 | 0.931 | 155 |
| massive_spot | ctx4k | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.020 | 0.065 | 0.753 | 250 |
| massive_spot | ctx8k | 96 / 100 | 1.000 | — | 0.960 | 0.960 | 0.044 | 0.086 | 0.833 | 241 |
| massive_spot | ctx32k | 97 / 100 | 1.000 | — | 0.970 | 0.967 | 0.031 | 0.073 | 0.746 | 254 |
| di_wide | ctx4k | 87, 87, 87 / 100 | 1.000 | — | 0.870 | 0.836 | 0.073 | 0.197 | 0.935 | 471 |
| di_wide | ctx8k | 87, 86, 89 / 100 | 1.000 | — | 0.873 | 0.831 | 0.068 | 0.201 | 0.909 | 448 |
| di_wide | ctx32k | 88, 88, 86 / 100 | 1.000 | — | 0.873 | 0.840 | 0.083 | 0.204 | 0.911 | 453 |
| di_catchall | ctx4k | 84, 86 / 100 | 1.000 | — | 0.850 | 0.790 | 0.092 | 0.260 | 0.721 | 793 |
| di_catchall | ctx8k | 87, 86 / 100 | 1.000 | — | 0.865 | 0.817 | 0.071 | 0.242 | 0.716 | 813 |
| di_catchall | ctx32k | 86, 88 / 100 | 1.000 | — | 0.870 | 0.812 | 0.072 | 0.239 | 0.668 | 791 |
| rag_dev | ctx4k | 148, 154 / 198 | 1.000 | — | 0.763 | 0.741 | 0.192 | 0.419 | 0.608 | 287 |
| rag_dev | ctx8k | 156, 152 / 198 | 1.000 | — | 0.778 | 0.758 | 0.182 | 0.404 | 0.640 | 286 |
| rag_dev | ctx32k | 152, 154 / 198 | 1.000 | — | 0.773 | 0.754 | 0.175 | 0.402 | 0.639 | 284 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | di_wide | di_catchall | rag_dev | mean |
|---|---|---|---|---|---|---|---|---|---|
| ctx4k | 0.973 | 0.964 | 0.944 | 0.867 | 1.000 | 0.927 | 0.930 | 0.919 | 0.940 |
| ctx8k | 1.000 | 0.965 | 0.935 | 0.933 | 0.978 | 0.953 | 0.940 | 0.960 | 0.958 |
| ctx32k | 1.000 | 0.961 | 0.945 | 0.956 | 1.000 | 0.943 | 0.980 | 0.949 | 0.967 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | ctx4k | 50 | 0.082 | 0.094 | 0.402 | 0.472 | 1.45, 0.85, 1.45, 1.35, 1.45 |
| calibration | ctx8k | 50 | 0.104 | 0.090 | 0.413 | 0.424 | 1.40, 1.05, 1.45, 1.40, 1.50 |
| calibration | ctx32k | 50 | 0.109 | 0.118 | 0.449 | 0.507 | 1.45, 0.90, 1.55, 1.45, 1.55 |
| jev_native | ctx4k | 230 | 0.094 | 0.043 | 0.578 | 0.495 | 1.75, 1.70, 1.75, 1.75, 1.70 |
| jev_native | ctx8k | 231 | 0.095 | 0.064 | 0.659 | 0.536 | 1.95, 1.85, 1.95, 1.80, 1.95 |
| jev_native | ctx32k | 231 | 0.086 | 0.061 | 0.611 | 0.511 | 1.80, 1.75, 1.85, 1.75, 1.85 |
| jev_systemone | ctx4k | 231 | 0.063 | 0.024 | 0.448 | 0.409 | 1.45, 1.45, 1.60, 1.55, 1.55 |
| jev_systemone | ctx8k | 231 | 0.063 | 0.066 | 0.438 | 0.405 | 1.45, 1.40, 1.55, 1.45, 1.50 |
| jev_systemone | ctx32k | 231 | 0.071 | 0.042 | 0.462 | 0.416 | 1.50, 1.50, 1.65, 1.50, 1.60 |
| intents_banking77 | ctx4k | 29 | 0.148 | 0.116 | 0.797 | 0.674 | 1.50, 1.65, 1.60, 1.80, 1.70 |
| intents_banking77 | ctx8k | 29 | 0.122 | 0.118 | 0.539 | 0.555 | 1.45, 1.05, 1.45, 1.55, 1.45 |
| intents_banking77 | ctx32k | 29 | 0.111 | 0.123 | 0.373 | 0.456 | 1.35, 0.75, 1.25, 1.30, 1.25 |
| intents_clinc150 | ctx4k | 30 | 0.059 | 0.021 | 0.071 | 0.024 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| intents_clinc150 | ctx8k | 30 | 0.065 | 0.025 | 0.079 | 0.028 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| intents_clinc150 | ctx32k | 30 | 0.050 | 0.016 | 0.060 | 0.019 | 0.50, 0.50, 0.50, 0.50, 0.50 |
| massive_spot | ctx4k | 100 | 0.020 | 0.046 | 0.254 | 0.269 | 1.35, 1.05, 1.20, 1.40, 1.45 |
| massive_spot | ctx8k | 100 | 0.044 | 0.046 | 0.254 | 0.273 | 1.30, 1.00, 1.15, 1.30, 1.40 |
| massive_spot | ctx32k | 100 | 0.031 | 0.049 | 0.266 | 0.279 | 1.35, 1.05, 1.25, 1.40, 1.45 |
| di_wide | ctx4k | 100 | 0.066 | 0.058 | 0.606 | 0.559 | 1.30, 1.25, 1.40, 1.35, 1.40 |
| di_wide | ctx8k | 100 | 0.074 | 0.067 | 0.656 | 0.590 | 1.30, 1.35, 1.45, 1.40, 1.45 |
| di_wide | ctx32k | 100 | 0.066 | 0.098 | 0.631 | 0.580 | 1.30, 1.25, 1.40, 1.40, 1.45 |
| di_catchall | ctx4k | 100 | 0.090 | 0.057 | 0.833 | 0.789 | 1.20, 1.25, 1.25, 1.25, 1.25 |
| di_catchall | ctx8k | 100 | 0.082 | 0.074 | 0.679 | 0.671 | 1.10, 1.10, 1.15, 1.15, 1.10 |
| di_catchall | ctx32k | 97 | 0.052 | 0.061 | 0.674 | 0.669 | 1.10, 1.10, 1.20, 1.10, 1.15 |
| rag_dev | ctx4k | 198 | 0.191 | 0.023 | 0.994 | 0.550 | 4.00, 4.30, 3.90, 3.85, 4.05 |
| rag_dev | ctx8k | 198 | 0.182 | 0.045 | 0.854 | 0.511 | 3.15, 3.75, 3.15, 3.50, 3.40 |
| rag_dev | ctx32k | 198 | 0.186 | 0.024 | 0.917 | 0.523 | 3.60, 3.85, 3.55, 3.65, 3.60 |

## Decision Index adapter probes (`dgem systemone serve` from this checkout)

| target | probe | HTTP | ok | detail |
|---|---|---|---|---|
| ctx4k | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| ctx4k | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| ctx4k | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| ctx4k | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| ctx4k | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| ctx4k | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| ctx4k | batch_12q | 200 | yes |  |
| ctx4k | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| ctx4k | noul_criteria | 200 | yes |  |
| ctx4k | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
| ctx4k | long_9k | 422 | yes | marker=maximum context length |
| ctx4k | context_refusal | 422 | yes | marker=maximum context length |
| ctx8k | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| ctx8k | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| ctx8k | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| ctx8k | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| ctx8k | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| ctx8k | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| ctx8k | batch_12q | 200 | yes |  |
| ctx8k | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| ctx8k | noul_criteria | 200 | yes |  |
| ctx8k | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
| ctx8k | long_9k | 422 | yes | marker=maximum context length |
| ctx8k | context_refusal | 422 | yes | marker=maximum context length |
| ctx32k | wide_27 | 200 | yes | keys=27, prob_sum=1.000, top_p=0.939 |
| ctx32k | wide_41 | 200 | yes | keys=41, prob_sum=1.000, top_p=0.960 |
| ctx32k | wide_61 | 200 | yes | keys=61, prob_sum=1.000, top_p=0.937 |
| ctx32k | wide_101 | 200 | yes | keys=101, prob_sum=1.000, top_p=0.909 |
| ctx32k | wide_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.972 |
| ctx32k | wide_255 | 200 | yes | keys=255, prob_sum=1.000, top_p=0.953 |
| ctx32k | batch_12q | 200 | yes |  |
| ctx32k | criteria_objects | 200 | yes | keys=3, prob_sum=1.000, top_p=0.998 |
| ctx32k | noul_criteria | 200 | yes |  |
| ctx32k | wide_catchall_151 | 200 | yes | keys=151, prob_sum=1.000, top_p=0.971 |
| ctx32k | long_9k | 200 | yes |  |
| ctx32k | context_refusal | 200 | yes |  |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| ctx4k | q1 | 100/100 | 93.0 | 106.6 | 113.6 | 53.3 | — |
| ctx4k | q5 | 100/100 | 93.4 | 98.4 | 102.8 | 56.7 | — |
| ctx4k | q10 | 100/100 | 98.2 | 102.8 | 109.3 | 59.9 | — |
| ctx4k | long_q1 | 100/100 | 108.3 | 112.0 | 118.3 | 62.2 | — |
| ctx4k | q5_s4 | 100/100 | 133.3 | 136.6 | 142.3 | 95.1 | — |
| ctx4k | sweep q1/w16 | 256/256 | 220.6 | 260.6 | 528.9 | 109.1 | 66.1 |
| ctx4k | sweep q1/w32 | 256/256 | 446.4 | 596.9 | 897.6 | 109.0 | 63.1 |
| ctx4k | sweep q5/w16 | 256/256 | 226.3 | 234.4 | 608.9 | 112.0 | 63.4 |
| ctx4k | sweep q5/w32 | 256/256 | 423.0 | 514.6 | 848.7 | 112.0 | 7.1 |
| ctx8k | q1 | 100/100 | 90.0 | 96.0 | 102.6 | 52.9 | — |
| ctx8k | q5 | 100/100 | 94.3 | 101.3 | 157.5 | 57.0 | — |
| ctx8k | q10 | 100/100 | 100.1 | 104.8 | 114.2 | 61.0 | — |
| ctx8k | long_q1 | 100/100 | 111.1 | 177.3 | 244.6 | 62.8 | — |
| ctx8k | q5_s4 | 100/100 | 135.8 | 142.2 | 217.3 | 96.7 | — |
| ctx8k | sweep q1/w16 | 256/256 | 223.9 | 263.9 | 487.5 | 111.2 | 65.5 |
| ctx8k | sweep q1/w32 | 256/256 | 462.3 | 616.9 | 890.3 | 114.3 | 61.1 |
| ctx8k | sweep q5/w16 | 256/256 | 223.7 | 293.6 | 497.1 | 111.1 | 66.2 |
| ctx8k | sweep q5/w32 | 256/256 | 445.4 | 719.3 | 894.9 | 110.2 | 62.5 |
| ctx32k | q1 | 100/100 | 90.6 | 94.2 | 99.2 | 53.8 | — |
| ctx32k | q5 | 100/100 | 95.0 | 103.3 | 151.4 | 57.2 | — |
| ctx32k | q10 | 100/100 | 104.7 | 174.6 | 245.7 | 60.9 | — |
| ctx32k | long_q1 | 100/100 | 107.9 | 111.4 | 117.3 | 61.4 | — |
| ctx32k | q5_s4 | 0/100 | — | — | — | — | — |
| ctx32k | sweep q1/w16 | 0/256 | — | — | — | — | 0.0 |
| ctx32k | sweep q1/w32 | 0/256 | — | — | — | — | 0.0 |
| ctx32k | sweep q5/w16 | 0/256 | — | — | — | — | 0.0 |
| ctx32k | sweep q5/w32 | 0/256 | — | — | — | — | 0.0 |

