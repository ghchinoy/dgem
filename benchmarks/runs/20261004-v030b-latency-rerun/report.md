# Regression matrix report: 20261004-v030b-latency-rerun

- Matrix **v2**, tier **T2**, started 2026-10-04T13:21:55+00:00, finished 2026-10-04T13:33:40+00:00
- Repository commit `d8484cb485`
- Mode: baseline `prod` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod (baseline) | vertex | — | — | — |
| new | vertex | — | — | — |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| coverage | new | **PASS** | no coverage loss |
| order | new | **REVIEW** | net flip vs baseline: order_jev_choice +0.108 |
| latency | new | **REVIEW** | q5_s4 p50 x1.19 |

Overall: **new: REVIEW**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|

## Option order (flip = answer changes; net = minus the identical-repeat flip)

| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |
|---|---|---|---|---|---|---|
| prod | order_emotion | 200 | 0.045 | 0.100 (+0.055) | 0.080 (+0.035) | 0.455 / 0.305 |
| prod | order_jev_choice | 139 | 0.086 | 0.094 (+0.007) | 0.122 (+0.036) | 0.223 / 0.216 |
| prod | order_massive_en | 200 | 0.005 | 0.075 (+0.070) | 0.065 (+0.060) | 0.045 / 0.045 |
| prod | order_xnli_en | 200 | 0.030 | 0.060 (+0.030) | 0.045 (+0.015) | 0.280 / 0.335 |
| new | order_emotion | 200 | 0.035 | 0.095 (+0.060) | 0.075 (+0.040) | 0.445 / 0.305 |
| new | order_jev_choice | 139 | 0.029 | 0.144 (+0.115) | 0.144 (+0.115) | 0.216 / 0.216 |
| new | order_massive_en | 112 | 0.009 | 0.036 (+0.027) | 0.080 (+0.071) | 0.045 / 0.045 |
| new | order_xnli_en | 200 | 0.030 | 0.070 (+0.040) | 0.055 (+0.025) | 0.265 / 0.335 |

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| prod | q1 | 100/100 | 95.4 | 103.2 | 195.5 | 56.3 | — |
| prod | q5 | 100/100 | 113.5 | 206.3 | 242.0 | 60.0 | — |
| prod | q10 | 100/100 | 106.0 | 114.3 | 156.1 | 63.7 | — |
| prod | long_q1 | 100/100 | 114.9 | 176.9 | 273.6 | 64.9 | — |
| prod | q5_s4 | 100/100 | 141.7 | 232.5 | 423.0 | 98.7 | — |
| prod | sweep q1/w16 | 256/256 | 240.2 | 382.5 | 637.3 | 109.8 | 56.9 |
| prod | sweep q1/w32 | 256/256 | 528.6 | 699.3 | 1004.6 | 115.6 | 51.9 |
| prod | sweep q5/w16 | 256/256 | 291.5 | 464.1 | 572.9 | 118.9 | 50.1 |
| prod | sweep q5/w32 | 256/256 | 546.9 | 725.4 | 1103.2 | 116.3 | 52.6 |
| new | q1 | 100/100 | 95.6 | 105.7 | 185.6 | 55.1 | — |
| new | q5 | 100/100 | 100.9 | 115.3 | 209.2 | 58.7 | — |
| new | q10 | 100/100 | 107.0 | 137.0 | 193.5 | 62.3 | — |
| new | long_q1 | 100/100 | 118.1 | 176.8 | 302.9 | 64.1 | — |
| new | q5_s4 | 100/100 | 168.2 | 240.3 | 295.3 | 97.7 | — |
| new | sweep q1/w16 | 256/256 | 227.5 | 267.2 | 492.4 | 113.5 | 64.2 |
| new | sweep q1/w32 | 256/256 | 459.0 | 603.3 | 914.5 | 114.3 | 61.4 |
| new | sweep q5/w16 | 256/256 | 222.5 | 262.4 | 514.5 | 110.8 | 65.5 |
| new | sweep q5/w32 | 256/256 | 453.1 | 671.2 | 998.9 | 112.6 | 61.3 |

