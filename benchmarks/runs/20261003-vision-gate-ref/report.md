# Regression matrix report: 20261003-vision-gate-ref

- Matrix **v1**, tier **T1**, started 2026-10-03T16:38:38+00:00, finished 2026-10-03T16:39:21+00:00
- Repository commit `d30ec61550`
- Mode: single target, compared with the reference ranges

| target | kind | version | revision | vLLM commit |
|---|---|---|---|---|
| prod | vertex | v0.2.0 | 027b68e | a9eafde59c |

## Verdicts

| gate | target | verdict | detail |
|---|---|---|---|
| health | prod | **PASS** | ready |
| vision | prod | **PASS** | acc 0.755; reference 0.750–0.761 (v0.2.0) |
| vision_spot | prod | **PASS** | acc 0.787; reference 0.787–0.801 (v0.2.0) |
| coverage | prod | **PASS** | no coverage loss |

Overall: **prod: PASS**

## Suites

Coverage = answered / attempted items (refusals such as HTTP 422 and errors count as unanswered, as the Decision Index scores them). Accuracy, ECE and the other metrics are over answered items. Unanswered reasons: context (prompt longer than the served context), capacity (server shape limits), na (skipped by the matrix), error. The coverage gate reviews any suite that answers fewer items than the baseline or reference.

| suite | target | runs: correct / n | coverage | unanswered (all runs) | mean accuracy | macro-F1 | ECE10 | Brier | AUROC | wall p50 ms |
|---|---|---|---|---|---|---|---|---|---|---|
| vision | prod | 640, 648 / 853 | 1.000 | — | 0.755 | 0.662 | 0.189 | 0.413 | 0.810 | 348 |
| vision_spot | prod | 111 / 141 | 1.000 | — | 0.787 | 0.781 | 0.169 | 0.373 | 0.820 | 361 |

## Measured noise floor (answer agreement between repeated identical runs)

| target | vision | mean |
|---|---|---|
| prod | 0.967 | 0.967 |

## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)

| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |
|---|---|---|---|---|---|---|---|
| vision | prod | 853 | 0.193 | 0.051 | 1.021 | 0.615 | 2.65, 2.75, 2.65, 2.75, 2.75 |
| vision_spot | prod | 141 | 0.169 | 0.046 | 1.021 | 0.595 | 2.80, 2.70, 2.50, 2.90, 2.75 |

