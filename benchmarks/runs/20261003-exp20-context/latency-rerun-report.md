# Regression matrix report: 20261003-exp20b-latency

- Matrix **v1**, tier **T1**, started 2026-10-03T13:53:06+00:00, finished 2026-10-03T13:56:01+00:00
- Repository commit `a22657a5e1`
- Mode: baseline `ctx8k` measured in the same session

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| ctx8k (baseline) | vertex | — | — | — |
| ctx32k | vertex | — | — | — |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| coverage | ctx32k | **PASS** | no coverage loss |
| latency | ctx32k | **PASS** | within tolerance |

Overall: **ctx32k: PASS**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|

## Latency (/v1/systemone, keep-alive, seed 42)

| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |
|---|---|---|---|---|---|---|---|
| ctx8k | q1 | 100/100 | 128.7 | 133.0 | 139.6 | 52.4 | — |
| ctx8k | q5 | 100/100 | 132.6 | 136.8 | 156.1 | 55.9 | — |
| ctx8k | q10 | 100/100 | 118.8 | 141.0 | 191.9 | 58.8 | — |
| ctx8k | long_q1 | 100/100 | 108.3 | 113.8 | 134.4 | 59.9 | — |
| ctx8k | q5_s4 | 100/100 | 171.1 | 175.4 | 184.4 | 93.1 | — |
| ctx8k | sweep q1/w16 | 256/256 | 213.3 | 255.2 | 561.3 | 106.4 | 67.6 |
| ctx8k | sweep q1/w32 | 256/256 | 449.0 | 693.1 | 992.3 | 111.9 | 61.3 |
| ctx8k | sweep q5/w16 | 256/256 | 226.3 | 235.8 | 522.8 | 113.0 | 65.0 |
| ctx8k | sweep q5/w32 | 256/256 | 454.6 | 714.3 | 953.9 | 112.7 | 61.0 |
| ctx32k | q1 | 100/100 | 130.3 | 134.7 | 182.3 | 53.6 | — |
| ctx32k | q5 | 100/100 | 135.4 | 139.6 | 177.8 | 57.2 | — |
| ctx32k | q10 | 100/100 | 121.3 | 143.7 | 177.0 | 61.0 | — |
| ctx32k | long_q1 | 100/100 | 108.7 | 113.4 | 126.2 | 61.6 | — |
| ctx32k | q5_s4 | 100/100 | 173.2 | 175.5 | 187.8 | 93.9 | — |
| ctx32k | sweep q1/w16 | 256/256 | 218.9 | 304.2 | 549.1 | 108.0 | 64.3 |
| ctx32k | sweep q1/w32 | 256/256 | 428.7 | 744.3 | 1001.9 | 107.1 | 61.9 |
| ctx32k | sweep q5/w16 | 256/256 | 219.1 | 229.2 | 527.6 | 108.7 | 66.9 |
| ctx32k | sweep q5/w32 | 256/256 | 477.0 | 679.5 | 941.0 | 113.4 | 59.6 |

