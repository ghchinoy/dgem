---
title: "EXP-17: Separate-Pass Mirror (PROP-12)"
description: "Reading the options reversed in a second, separate pass gives an order-disagreement signal that is statistically related to errors beyond hesitation, but the practical gain in error detection is small and not clearly larger than ordinary run-to-run disagreement."
---

# EXP-17: Separate-Pass Mirror (`PROP-12`)

**Date:** 2026-09-26 · **Backend:** Vertex AI G4 endpoint `4423577720856772608` (image `dgemma:ab208dd`) ·
**Run:** [`benchmarks/runs/20260926-prop12-separate-pass`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260926-prop12-separate-pass/manifest.json)
**Reproduce:** `./bin/dgem bench-jev --vertex-url 4423577720856772608 --gcp-auth -w 4 [--flip-options]`, then
`python3 scripts/analyze_idc.py separate-pass --forward <F1> <F2> --reversed <R1> <R2>`

## Question and pre-registered rule

EXP-15 showed that a mirror on the *same* canvas interferes with the forward answer. A reversed reading in its own
request cannot interfere. **H12:** disagreement between a forward pass and a separate reversed pass (TVD) adds error
detection beyond hesitation. Supported if the mean partial Spearman correlation between TVD and forward-pass error,
controlling for hesitation, is above 0 with a 95% bootstrap interval excluding 0. Noise control: the same statistic for two
*forward* passes, because Vertex is not fully deterministic.

## Results (JevBench v1.3.1, 231 items; run order F1, R1, F2, R2)

Accuracy: forward **187, 185**; reversed **187, 181** (reversing the options costs little on its own).

**Primary:** mean partial Spearman over the 4 forward×reversed pairings = **0.133, 95% CI [0.025, 0.240]**.

| Pair | Mean TVD | Partial Spearman [95% CI] | AUROC hesitation | AUROC TVD | CV AUROC hesitation | CV AUROC hesitation + TVD |
| :--- | ---: | :--- | ---: | ---: | ---: | ---: |
| F1×R1 | 0.134 | 0.146 [0.030, 0.265] | 0.857 | 0.840 | 0.834 | 0.839 |
| F1×R2 | 0.128 | 0.082 [−0.045, 0.218] | 0.857 | 0.816 | 0.834 | 0.834 |
| F2×R1 | 0.130 | 0.152 [0.025, 0.288] | 0.849 | 0.836 | 0.821 | 0.837 |
| F2×R2 | 0.129 | 0.151 [0.023, 0.279] | 0.849 | 0.828 | 0.821 | 0.836 |
| **F1×F2 (noise control)** | 0.059 | 0.083 [−0.057, 0.218] | 0.857 | 0.831 | 0.834 | 0.835 |

Averaging the forward and reversed distributions: **187–190** correct vs. 185–187 forward alone (+1 to +4 items, within
the ±3-item noise floor).

## Verdict

**Supported by the pre-registered test, but the practical effect is small.**

- Order disagreement is related to errors beyond hesitation (CI excludes 0), and reversed passes disagree with forward
  passes about twice as much as two forward passes do (mean TVD 0.13 vs 0.06).
- But the noise-control pair (two forward passes) shows a similar, non-significant partial correlation (0.083), so we
  cannot say order disagreement is clearly more informative than simply asking twice.
- Cross-validated error detection improves by at most 0.016 AUROC (0.821–0.834 → 0.834–0.839).
- It doubles the cost (a second pass).

## Consequences

- The IDC idea of an order check survives only in a weak form: as a separate pass, it adds a little information at double
  the cost. It is not worth enabling by default.
- `PROP-13` (does it improve the Gemini hand-off at a matched hand-off rate?) is the deciding test; given these numbers,
  a large gain is unlikely. Hesitation alone already detects most errors (AUROC ≈ 0.85).
