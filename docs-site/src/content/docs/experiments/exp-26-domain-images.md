---
title: "EXP-26: Guided Boxes and Image Questions on Domain Images"
description: "Pre-registered repeat of EXP-24 (dgem-guided Gemini boxes) and EXP-22 (categorical image questions) on mobile UI, web UI, document pages and synthetic PCBs, with gemini-3.7-flash and gemini-3.8-flash."
---

**Status:** pre-registered 2026-10-04, before any scored run (commit in PR #95); results added 2026-10-04 below without editing the design or rule.
**Issue:** #76. **Follows:** [EXP-22](/dgem/experiments/exp-22-image-readouts/), [EXP-24](/dgem/experiments/exp-24-guided-cascade/).

## Question

Do the EXP-22 and EXP-24 recommendations hold beyond RefCOCO and ScreenSpot, and with the cheaper Gemini model?
1. Is "skip when dgem confidently says absent" plus "Gemini at LOW thinking with dgem's grid cell as a hint" still
   as accurate as Gemini alone, and faster, on each domain?
2. Does it hold with `gemini-3.7-flash` as the Gemini step?
3. Which categorical image questions can dgem answer alone, which need the cascade, and which it should not answer,
   per domain?

## Data (`benchmarks/domain_images.jsonl`, `scripts/build_domain_images.py`, seed 26)

| Domain | Source (licence) | Positives | Negatives |
| :--- | :--- | :---: | :---: |
| `ui_mobile` | RICO referring expressions, `ivelin/rico_refexp_combined` test (CC BY 4.0) | 100 | 83 (Gemini-confirmed) |
| `ui_web` | ScreenSpot-v2 web split (Apache-2.0), disjoint from the EXP-22 ScreenSpot images | 100 | 90 (Gemini-confirmed) |
| `documents` | DocLayNet v1.1 test pages (CDLA-Permissive-1.0); targets are elements that occur once on the page ("the table") | 100 | 100 (category absent from the annotation) |
| `pcb` | synthetic boards (this repo); one defect per board: solder bridge, scratch, missing component, contamination | 100 | 100 (defect not drawn) |

- FUNSD is not used: its licence allows non-commercial research only.
- Ground truth: boxes from the annotations; grid cell and relation from geometry. Relations are to another annotated
  target on the same image (UI and documents) or to an IC chip (PCB).
- UI negatives: the image queried with another image's instruction, kept only when `gemini-3.8-flash` confirms it is
  absent. A hand check of 40 of them found 38 truly absent and 2 ambiguous (precision 0.95,
  `benchmarks/domain_images_handcheck.json`). As in EXP-22, this flatters Gemini's own presence score on UI.

## Design

- **dgem:** Vertex G4 production endpoint, serving version from `/health` recorded per receipt, `samples: 1`,
  template `vision_aspects_generic.json.tmpl`.
- **Boxes** (`dgem bench-guided`), per Gemini model (`gemini-3.8-flash`, `gemini-3.7-flash`):
  `full@default`, `full@low`, `hint@default`, `hint@low` on positives; `full@default`, `full@low` on negatives.
  Cropping is not repeated (no gain in EXP-24). MINIMAL thinking is unsupported on both models (#79).
- **Categorical questions** (`dgem bench-vision`): present, grid cell, relation. dgem ×2, dgem on blank images,
  `gemini-3.8-flash` ×1, `gemini-3.7-flash` ×1.

## Decision rule (fixed before scoring)

For each domain *d*:

- **R1, guided boxes.** "Gemini at LOW thinking with dgem's hint" (`hint@low`) with model *m* is recommended if the
  paired difference mIoU(`hint@low`, *m*) − mIoU(`full@default`, 3.8) has a 95% bootstrap lower bound ≥ −0.03 **and**
  its median end-to-end latency (dgem pass included) is lower than `full@default` with 3.8. The reference is 3.8 at
  default thinking, the current Stage-2 default.
- **R2, skip.** Skipping Gemini when dgem says "absent" with normalized entropy < 0.16 is recommended if it saves
  ≥ 15% of Gemini calls on the domain's item mix **and** wrongly skips ≤ 2% of positives.
- **R3, categorical questions.** Aspect *a* on domain *d* is
  - **supported (dgem alone)** if dgem accuracy ≥ 0.85, its lift over the majority-class baseline has a 95% lower
    bound > 0, and hesitation→error AUROC ≥ 0.70;
  - **cascade** if the lift bound is > 0 and AUROC ≥ 0.70 but accuracy < 0.85;
  - **not supported** otherwise.
- **R4, model.** `gemini-3.7-flash` is recommended for the guided step on *d* if R1 holds with *m* = 3.7.

The outcome of each rule per domain goes into `docs/policies/images.md`.

## Results

**Run:** [`benchmarks/runs/20261004-exp26-domains`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261004-exp26-domains/manifest.json)
(dgem on production Vertex G4; Gemini on Vertex). **Reproduce:** `./scripts/run_exp26_domains.sh`, then
`python3 scripts/exp26_decide.py benchmarks/runs/<run_id>` (writes `decision.json`).

Errors:
- 13 of 6,184 Gemini box calls failed and are excluded pairwise.
- 24 of 1,546 dgem `bench-vision` requests failed at the network level and are excluded.

### Pre-registered verdicts

| Domain | R1 hint@low, 3.8 | R1 hint@low, 3.7 (= R4) | R2 skip | Categorical questions (R3) |
| :--- | :---: | :---: | :---: | :--- |
| `ui_mobile` | fail | fail | **pass** (22% saved, 0–1 / 100 positives lost) | present: **cascade**; grid cell: **cascade**; relation: not supported |
| `ui_web` | fail | fail | fail (33% saved, 3–4 / 100 positives lost) | present: **supported**; grid cell, relation: not supported |
| `documents` | fail | fail | fail (10% saved) | grid cell: **cascade**; present, relation: not supported |
| `pcb` | fail | fail | fail (23% saved, 14–15 / 100 positives lost) | grid cell: **cascade**; present, relation: not supported |

**R1, guided boxes: fails on all four domains with both models.** Each paired difference against 3.8 at default
thinking:

| Domain | Δ mIoU, hint@low 3.8 | Δ mIoU, hint@low 3.7 | Median latency, hint@low 3.8 / 3.7 / reference |
| :--- | :---: | :---: | :---: |
| ui_mobile | −0.041 [−0.088, +0.005] | −0.009 [−0.052, +0.033] | 2.1 / 3.1 / 3.7 s |
| ui_web | +0.014 [−0.046, +0.072] | +0.015 [−0.039, +0.072] | 3.3 / 4.1 / 4.5 s |
| documents | −0.028 [−0.066, +0.007] | −0.029 [−0.067, +0.005] | 2.2 / 2.3 / 3.7 s |
| pcb | −0.048 [−0.100, −0.001] | −0.028 [−0.079, +0.022] | 3.9 / 3.7 / 6.4 s |

Latency always improved. Every failure is on the accuracy bound: the lower bound is below −0.03. On `ui_web` the
point estimate is positive, but the interval is too wide.

**R3, categorical questions (dgem ×2):**

| Domain | Aspect | dgem accuracy | Majority | Lift [95% CI] | Hesitation AUROC | Gemini 3.8 / 3.7 |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: |
| ui_web | present | 0.900 | 0.526 | +0.374 [+0.292, +0.453] | 0.85 | 0.995 / 0.989 |
| ui_mobile | present | 0.833 | 0.546 | +0.287 [+0.221, +0.350] | 0.78 | 0.951 / 0.956 |
| ui_mobile | grid cell | 0.590 | 0.220 | +0.370 [+0.250, +0.485] | 0.72 | 0.900 / 0.910 |
| documents | grid cell | 0.610 | 0.250 | +0.360 [+0.255, +0.460] | 0.79 | 0.910 / 0.900 |
| pcb | grid cell | 0.365 | 0.180 | +0.185 [+0.065, +0.300] | 0.72 | 0.790 / 0.770 |
| ui_web | grid cell | 0.480 | 0.200 | +0.280 [+0.155, +0.400] | 0.65 | 0.920 / 0.910 |
| documents | present | 0.667 | 0.500 | +0.168 [+0.040, +0.287] | 0.62 | 0.825 / 0.805 |
| pcb | present | 0.677 | 0.500 | +0.177 [+0.098, +0.260] | 0.64 | 0.845 / 0.790 |
| documents | relation | 0.817 | 0.463 | +0.354 [+0.220, +0.500] | 0.55 | 0.878 / 0.854 |
| ui_mobile | relation | 0.667 | 0.273 | +0.394 [+0.227, +0.561] | 0.60 | 0.636 / 0.667 |
| ui_web | relation | 0.588 | 0.309 | +0.279 [+0.140, +0.412] | 0.65 | 0.971 / 0.912 |
| pcb | relation | 0.621 | 0.439 | +0.182 [+0.061, +0.303] | 0.57 | 0.682 / 0.682 |

- dgem beats the majority baseline on every question, but its hesitation identifies its own errors well enough
  (AUROC ≥ 0.70) on only half of them.
- **Latency:** median for the three questions, 0.56 s for dgem, 7.6 s for Gemini 3.8 and 4.8 s for Gemini 3.7.

### Exploratory (not pre-registered)

- **Thinking level matters, the dgem hint does not.** For the same model, "LOW thinking, no hint" vs "default
  thinking" changes mIoU by −0.036 to +0.013 (only 3.8 on PCB is significant, −0.036 [−0.075, −0.006]). It cuts
  median latency by 30–45%.
- **The hint is −0.032 to +0.009** against LOW thinking without it (no interval excludes 0).
- **Why the hint doesn't help here:** dgem's grid cell was right on only 36–61% of these positives (EXP-24 photos:
  52%), and the hint helped only when the cell was right.
- **3.7 vs 3.8:** box accuracy is equivalent. 3.7 is faster on the categorical questions (4.8 s vs 7.6 s p50).
- **Skip on PCB:** dgem confidently answers "absent" for 14–15% of real defects. It does not recognise small
  defects, so the skip is unsafe there.

## Decision

- **Guided boxes:** don't send dgem's grid-cell hint by default. Across EXP-24 and EXP-26 it is neutral on average.
  The latency gain comes from Gemini's LOW thinking. For boxes, call Gemini at LOW thinking (3.7 or 3.8; same
  accuracy) without a hint. The exception is PCB-like defect images, where 3.8 at default thinking is more accurate.
- **Skip absent targets:** only on mobile-UI-like images. Do not use it on documents (it saves little) or on
  defect/inspection images (it loses real defects).
- **Categorical questions:** per the R3 table.
  - "Is X on the screen?" is dgem-alone on web UI and cascade on mobile UI.
  - Grid cell is cascade on mobile UI, documents and PCB.
  - Relations are not supported on any of these domains.
  - Presence is not supported on documents or PCB.
- **Follow-ups:** the guided-locate defaults in #98 (hint on, skip at 0.16) should change to no hint by default, with
  skipping off unless the caller opts in. This is noted on the PR. `docs/policies/images.md` now carries the
  per-domain table.
