# Regression matrix report: 20261001-v013-reference-t1

- Matrix **v1**, tier **T1**, started 2026-10-01T22:37:42+00:00, finished 2026-10-01T22:42:55+00:00
- Repository commit `8608361d9b`
- Mode: baseline `a` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| a (baseline) | vertex | v0.1.3 | e685ef9 | a9eafde59c |
| b | vertex | v0.1.3 | e685ef9 | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | a | **PASS** | ready |
| health | b | **PASS** | ready |
| contract | b | **PASS** | all cases as expected |
| calibration | b | **PASS** | acc 0.880 vs 0.867; agreement 0.987 (floor 0.993, allowance 0.043); McNemar p=0.5 |
| jev_native | b | **PASS** | acc 0.805 vs 0.811; agreement 0.948 (floor 0.939, allowance 0.052); McNemar p=0.54 |
| jev_systemone | b | **PASS** | acc 0.830 vs 0.833; agreement 0.961 (floor 0.952, allowance 0.048); McNemar p=0.8 |
| intents_banking77 | b | **PASS** | acc 0.789 vs 0.778; agreement 0.852 (floor 0.839, allowance 0.154); McNemar p=1 |
| intents_clinc150 | b | **PASS** | acc 0.978 vs 0.989; agreement 0.981 (floor 0.978, allowance 0.074); McNemar p=1 |
| massive_spot | b | **PASS** | es 15/20, hi 18/20, ja 15/20, ru 19/20, th 16/20 |
| latency | b | **PASS** | within tolerance |

Overall: **b: PASS**

## Suites

| suite | target | runs: correct / n | mean accuracy | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|
| calibration | a | 43, 44, 43 / 50 | 0.867 | 0.069 | 0.218 | 0.900 | 185 |
| calibration | b | 44, 44, 44 / 50 | 0.880 | 0.042 | 0.215 | 0.879 | 164 |
| jev_native | a | 187, 188, 187 / 231 | 0.811 | 0.076 | 0.266 | 0.850 | 131 |
| jev_native | b | 183, 185, 190 / 231 | 0.805 | 0.080 | 0.267 | 0.855 | 126 |
| jev_systemone | a | 193, 191, 193 / 231 | 0.833 | 0.084 | 0.252 | 0.770 | 264 |
| jev_systemone | b | 193, 190, 192 / 231 | 0.830 | 0.095 | 0.252 | 0.776 | 219 |
| intents_banking77 | a | 24, 22, 24 / 30 | 0.778 | 0.156 | 0.392 | 0.870 | 404 |
| intents_banking77 | b | 24, 23, 24 / 30 | 0.789 | 0.139 | 0.398 | 0.825 | 442 |
| intents_clinc150 | a | 30, 29, 30 / 30 | 0.989 | 0.049 | 0.049 | 0.828 | 142 |
| intents_clinc150 | b | 29, 30, 29 / 30 | 0.978 | 0.045 | 0.053 | 0.914 | 151 |
| massive_spot | a | 83 / 100 | 0.830 | 0.098 | 0.237 | 0.888 | 221 |
| massive_spot | b | 83 / 100 | 0.830 | 0.099 | 0.228 | 0.872 | 216 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | calibration | jev_native | jev_systemone | intents_banking77 | intents_clinc150 | mean |
|---|---|---|---|---|---|---|
| a | 0.987 | 0.945 | 0.961 | 0.856 | 0.978 | 0.945 |
| b | 1.000 | 0.932 | 0.942 | 0.822 | 0.978 | 0.935 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| calibration | a | 50 | 0.080 | 0.067 | 0.376 | 0.371 | 1.30, 1.10, 1.25, 1.25, 1.30 |
| calibration | b | 50 | 0.065 | 0.046 | 0.339 | 0.339 | 1.25, 1.05, 1.20, 1.20, 1.20 |
| jev_native | a | 231 | 0.083 | 0.029 | 0.530 | 0.481 | 1.55, 1.55, 1.60, 1.55, 1.50 |
| jev_native | b | 231 | 0.091 | 0.045 | 0.551 | 0.493 | 1.60, 1.60, 1.65, 1.55, 1.55 |
| jev_systemone | a | 231 | 0.078 | 0.054 | 0.604 | 0.507 | 1.85, 1.60, 1.85, 1.80, 1.85 |
| jev_systemone | b | 231 | 0.096 | 0.056 | 0.601 | 0.502 | 1.85, 1.60, 1.85, 1.80, 1.85 |
| intents_banking77 | a | 29 | 0.155 | 0.143 | 0.951 | 0.799 | 1.65, 2.00, 1.60, 1.95, 1.85 |
| intents_banking77 | b | 29 | 0.223 | 0.150 | 1.038 | 0.800 | 1.80, 2.05, 1.80, 2.00, 1.95 |
| intents_clinc150 | a | 30 | 0.067 | 0.021 | 0.093 | 0.087 | 0.75, 0.50, 0.75, 0.65, 0.65 |
| intents_clinc150 | b | 30 | 0.038 | 0.038 | 0.113 | 0.154 | 0.85, 0.50, 0.90, 0.80, 0.80 |
| massive_spot | a | 100 | 0.098 | 0.074 | 0.572 | 0.512 | 1.45, 1.35, 1.40, 1.45, 1.40 |
| massive_spot | b | 100 | 0.099 | 0.061 | 0.575 | 0.508 | 1.45, 1.40, 1.45, 1.45, 1.40 |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| a | q1 | 100/100 | 91.9 | 115.6 | 131.0 | 53.7 | — |
| a | q5 | 100/100 | 95.2 | 109.6 | 140.0 | 57.0 | — |
| a | q10 | 100/100 | 101.4 | 111.1 | 150.8 | 60.1 | — |
| a | long_q1 | 100/100 | 110.7 | 136.1 | 317.2 | 61.5 | — |
| a | q5_s4 | 100/100 | 134.1 | 147.8 | 159.4 | 95.2 | — |
| a | sweep q1/w16 | 256/256 | 227.6 | 269.4 | 553.5 | 112.7 | 63.7 |
| a | sweep q1/w32 | 256/256 | 451.5 | 628.5 | 955.0 | 111.0 | 62.1 |
| a | sweep q5/w16 | 256/256 | 222.8 | 241.8 | 514.7 | 110.8 | 65.8 |
| a | sweep q5/w32 | 256/256 | 459.8 | 656.8 | 935.0 | 114.3 | 60.5 |
| b | q1 | 100/100 | 91.8 | 99.4 | 155.1 | 53.7 | — |
| b | q5 | 100/100 | 94.3 | 98.4 | 113.5 | 55.7 | — |
| b | q10 | 100/100 | 101.2 | 108.5 | 178.1 | 59.4 | — |
| b | long_q1 | 100/100 | 116.6 | 177.6 | 281.3 | 61.8 | — |
| b | q5_s4 | 100/100 | 134.6 | 141.2 | 156.2 | 95.0 | — |
| b | sweep q1/w16 | 256/256 | 217.7 | 228.4 | 549.7 | 108.2 | 66.3 |
| b | sweep q1/w32 | 256/256 | 436.2 | 675.6 | 919.3 | 108.2 | 62.9 |
| b | sweep q5/w16 | 256/256 | 218.0 | 233.4 | 525.4 | 108.4 | 66.8 |
| b | sweep q5/w32 | 256/256 | 447.3 | 629.1 | 885.9 | 111.2 | 61.7 |

