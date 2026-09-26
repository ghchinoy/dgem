---
title: "EXP-15: Letter Collision in the Dual-Mirror (PROP-11)"
description: "The same-canvas mirror hurts the forward answer because the two slots share letters that mean different options. An identical copy or a digit-labelled reversed slot causes no measurable harm; the letter-labelled reversed slot costs 28 JevBench items."
---

# EXP-15: Letter Collision in the Dual-Mirror (`PROP-11`)

**Date:** 2026-09-25 · **Backend:** Vertex AI G4 endpoint `4423577720856772608` (g4-standard-48 + RTX PRO 6000, image `dgemma:ab208dd`, 1 replica available) ·
**Run:** [`benchmarks/runs/20260925-prop11-letter-collision`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-prop11-letter-collision/manifest.json) · **Code:** commit `1a7c13f` (`--mirror-mode`)
**Reproduce:** `./bin/dgem bench-jev --vertex-url 4423577720856772608 --gcp-auth -w 4 --dual-mirror --mirror-mode <mode>`, then
`python3 scripts/analyze_idc.py collision --baselines <b1> <b2> <b3> -- <receipts...>`

## Question and pre-registered rule

[EXP-14](/dgem/experiments/exp-14-idc-rerun/#why-the-mirror-degrades-and-null-prior-overcorrects-analysis) found that when the forward and
reversed slots picked the same letter (which in the reversed slot means a different option), the forward answer changed
far more often. **H11:** the mirror's damage comes from shared letters, not from having a second slot.

Pass rule (amended before data, see the [register](/dgem/experiments/proposed/#prop-11-letter-collision-in-the-mirror)): with at least 3
same-session baselines defining a noise band, H11 is supported if `copy` and `reversed-digits` fall inside the band while
`reversed` falls more than 3 items below it; and, per item, the forward answer changes much more often on same-letter
items than on others.

## Conditions (all JevBench v1.3.1, 231 items, one session, run order as listed)

| Condition | Second slot | Labels in second slot |
| :--- | :--- | :--- |
| `baseline_b1/b2/b3` | none | — |
| `reversed` | options reversed, placed after forward | A, B, C… (collide with forward) |
| `copy` | identical copy, same order | A, B, C… (same meaning) |
| `reversed-digits` | options reversed, sent as a `score` slot | 1, 2, 3… (no shared letters) |
| `reversed-first` | options reversed, placed **before** forward | A, B, C… (collide) |

## Results

Baselines: **189, 182, 186** → noise band **[182, 189]**.

| Condition | Forward-slot correct | vs band | Same label, different option | Forward answer changed (vs baseline majority): same-label items / other items | One-sided Fisher p |
| :--- | ---: | :--- | ---: | :--- | ---: |
| `copy` | **184** | within | 0 | — / 17/230 (7%) | — |
| `reversed-digits` | **182** | within (at the edge) | 7 | 3/7 / 15/223 (7%) | 0.012 |
| `reversed` | **154** | degraded (−28 vs band min) | 54 | **36/54 (67%)** / 10/176 (6%) | 1e-19 |
| `reversed-first` | **168** | degraded (−14) | 44 | **33/44 (75%)** / 27/186 (15%) | 2e-14 |

(`reversed-digits` can still pick the "same position" by coincidence; with digit labels this is not a shared label, and n = 7.)

## Verdict

**H11 supported.** A second slot by itself costs nothing measurable (`copy` within noise). Reversing the options costs
nothing measurable **when the reversed slot uses different labels** (`reversed-digits` within noise). Reversing with
shared letters costs 28 items, and the harm is concentrated on items where the slots chose the same letter (67% vs 6%
changed). Putting the reversed slot first still collides and still hurts, though less.

## Exploratory (not pre-registered)

Does the uncoupled mirror's disagreement help find errors? AUROC for detecting forward-slot errors on the same run:

| Condition | Hesitation alone | Mirror TVD alone | max(hesitation, TVD) |
| :--- | ---: | ---: | ---: |
| `reversed-digits` | 0.852 | 0.800 | 0.854 |
| `copy` | 0.865 | 0.755 | 0.858 |

On one run of 230 items, digit-mirror disagreement adds almost nothing on top of hesitation. `PROP-12`/`PROP-13` test
this properly (separate passes, held-out thresholds, cascade accuracy).

## Consequences

- If a same-canvas mirror is used, it should be **`--mirror-mode reversed-digits`** (≤ 9 options), never lettered
  `reversed`. The default stays `reversed` for reproducibility of older receipts, and the warning stays on for it.
- The server-side fix (explicit labels that keep each option's original letter) is no longer needed to *diagnose* the
  problem; it remains optional for questions with more than 9 options.
- Next: `PROP-16` (slot names), then `PROP-12` (separate-pass mirror), then `PROP-13` (does any mirror improve the cascade).
