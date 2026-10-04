---
title: "EXP-26: Guided Boxes and Image Questions on Domain Images"
description: "Pre-registered repeat of EXP-24 (dgem-guided Gemini boxes) and EXP-22 (categorical image questions) on mobile UI, web UI, document pages and synthetic PCBs, with gemini-3.7-flash and gemini-3.8-flash."
---

**Status:** pre-registered 2026-10-04, before any scored run. Results are added below this section without editing it.
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

Pending.
