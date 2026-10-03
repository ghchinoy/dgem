---
title: "EXP-22: Validating Image Readouts (PROP-18 Phases 2–4)"
description: "Gemini 3.x validated as a box judge; a generated sweep, RefCOCO and ScreenSpot show DiffusionGemma's boxes are coarse (centre hit 45% vs Gemini 91%) and fail on small GUI elements, while one-pass categorical image questions (grid cell, relation, count, quality) beat baselines by 0.25–0.52 at ~300 ms, and a hesitation-gated Gemini cascade closes much of the gap."
---

**Date:** 2026-10-03.

**Backends:**
- Vertex AI G4 endpoint `<endpoint-id>`, serving **v0.2.0** (`027b68e`), `samples: 1`;
- Vertex Gemini `gemini-3.8-flash` and `gemini-3.7-flash` (default thinking);
- OWLv2 / Grounding DINO / SAM on a short-lived `g2-standard-8` + L4 VM, deleted after the run.

**Follows:** the [EXP-09 re-baseline](/dgem/experiments/exp-09-spatial-grounding/) (phase 1).
**Plan:** [`PROP-18`](/dgem/experiments/proposed/#prop-18-validate-image-metrics-beyond-exp-09).

| Phase | Run | Reproduce |
| :--- | :--- | :--- |
| 2. Gemini reference + judge (EXP-09 fixtures) | [`20261003-prop18-gemini`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-gemini/manifest.json) | `./scripts/run_prop18_phase2.sh` |
| 3a. Generated sweep (230 items) | [`20261003-prop18-sweep`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-sweep/manifest.json) | `SUITE=sweep ./scripts/run_prop18_phase3.sh` |
| 3b. RefCOCO + ScreenSpot (228 items) | [`20261003-prop18-real-v2`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-real-v2/manifest.json) | `SUITE=real ./scripts/run_prop18_phase3.sh` |
| 3c. Open-vocabulary detectors | [`20261003-prop18-detectors`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-detectors/manifest.json) | `./scripts/run_detectors_gce.sh` |
| 4. Multi-aspect pilot (230 items) | [`20261003-prop18-vision`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-vision/manifest.json) | `./scripts/run_prop18_phase4.sh` |

`benchmarks/runs/20261003-prop18-real` is superseded: its manifest gave 8 ScreenSpot rows duplicate ids, because one
screenshot can carry several instructions.

## Question

Phase 1 found that DiffusionGemma localizes the 11 EXP-09 targets above image-free baselines, but that its boxes move
under flips and padding. This experiment asks four follow-up questions:

1. Can a Gemini 3.x model stand in for ground truth, as a reference localizer or as a judge of box overlays?
2. How do dgem's boxes hold up on harder synthetic scenes and on real images, against Gemini and classical
   open-vocabulary detectors?
3. Which factors (target size, aspect ratio, grid lines, occlusion, degradation, distractors) drive the errors?
4. Are categorical image questions (where, how many, what quality) a better fit for a one-pass decision model than
   coordinates?

## Data

- **EXP-09 fixtures:** the 12 synthetic UI images used in phase 1.
- **Generated sweep** (`scripts/generate_bbox_sweep.py`, seed 18, 230 items, byte-reproducible; images gitignored). It
  has four sets, each varying one factor:
  - *Geometry* (120): aspect ratio (1:1, 16:9, 9:16, 4:3) × 10% grid lines on/off, random placement, size and 0–2
    distractors.
  - *Occlusion* (40): 10 placements × coverage 0/25/50/75% of one edge, with the same scene at every coverage.
  - *Degradation* (50): 10 placements × none, blur, noise, JPEG and low contrast.
  - *Absent* (20): distractors only.

  Every row carries ground truth for six questions: `present`, `grid_cell`, `element_count`, `relation`,
  `image_quality` and `occluded`.
- **Real images** (`scripts/fetch_bbox_real.py`, seed 18). Only the manifest is committed; images are downloaded and
  checked by SHA-256.
  - 120 RefCOCO validation referring expressions on COCO photos.
  - 108 ScreenSpot test instructions (Apache-2.0): Windows, macOS, iOS, Android, GitLab; text and icon targets.

## Results

### 1. Gemini 3.x is a reliable reference and judge on these images

**As a localizer** (EXP-09 fixtures, ×3, same scorer as dgem):

| Localizer | mIoU [95% CI] | Acc@0.5 | Consistency under flips/padding |
| :--- | :---: | :---: | :---: |
| dgem v0.2.0 | 0.599 [0.516, 0.680] | 81.8% | 0.34–0.53 |
| `gemini-3.8-flash` | 0.892 [0.809, 0.969] | 97.0% | 0.85–0.93 |
| `gemini-3.7-flash` | 0.877 [0.769, 0.966] | 93.9% | 0.79–0.96 |

So the instability under flips and padding is specific to dgem, not a property of the task. Gemini returned only one
object on 2 of 3 runs of each two-object case.

**As a judge** (`dgem bench-bbox-judge`). It was tested on 225 ground-truth boxes with one edge shifted by 5, 10 or
20 points (plus exact boxes), and on dgem's own 27 boxes:

| Judge | Edges, 3-class accuracy (κ) | Wrong edges (>5 pts) caught | Correct edges accepted | Direction right | "Acceptable" vs all edges within 2.5 pts |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `gemini-3.8-flash`, calibration | 0.959 (0.90) | 100% | 94.9% | 100% | 0.991 |
| `gemini-3.7-flash`, calibration | 0.960 (0.90) | 98.6% | 95.3% | 100% | 0.982 |
| `gemini-3.8-flash`, dgem boxes | 0.796 (0.68) | 89.7% | 58.7% | 100% | 1.000 |

- **Resolution:** the judge flags 0–5% of edges within 0.5 points of the truth, 30–35% at 0.5–1.5 points, and 100%
  from 1.5 points up.
- **Why "correct edges accepted" drops on dgem's boxes:** its edges in the 1.5–2.5 point band are visibly off, so the
  judge flags them. That is a mismatch with the tolerance, not a judge mistake.
- **Box level:** the "acceptable" verdict means "tight" (all edges within tolerance). Do not read it as IoU ≥ 0.75;
  agreement with that is only 0.52.

**Verdict:** either Gemini model can label boxes on images without ground truth, to about 1.5-point resolution.

### 2. On harder synthetic scenes, dgem's boxes are coarse; Gemini's are not

Generated sweep, 210 items with a target (dgem ×2, Gemini ×1):

| Localizer | mIoU [95% CI] | Centre hit | Absent targets answered "present" | Median latency |
| :--- | :---: | :---: | :---: | :---: |
| Fixed `[25, 25, 75, 75]` box | 0.062 | 11.4% | — | — |
| dgem | 0.197 [0.170, 0.226] | 43.3% | 17 / 40 | 0.77 s |
| `gemini-3.8-flash` | 0.782 [0.733, 0.827] | 86.7% | 1 / 20 | 3.4 s |

*Centre hit* is whether the box centre lands inside the true box, as in ScreenSpot. Latencies are with 8 concurrent
requests.

- **Size drives dgem's error.** mIoU is 0.06 / 0.15 / 0.40 for small / medium / large targets, against 0.64 / 0.79 /
  0.99 for Gemini. The EXP-09 fixtures are mostly large, which is why phase 1 looked better.
- **Grid lines** (the "ruler" every EXP-09 fixture carries) do not significantly help: 0.196 vs 0.141, with overlapping
  intervals.
- **Aspect ratio:** square and portrait (9:16) images are worst for dgem (0.12 and 0.11).
- **Degradation** barely changes dgem's (already low) scores.
- **Run-to-run stability** is much lower than on EXP-09: the mean IoU between repeat boxes is 0.42, and 82% of edge
  argmaxes agree across repeats.
- **Occlusion:**
  - Gemini tracks the *visible* part of the target: IoU with the visible box stays at 0.85–0.87 up to 50% coverage.
  - dgem tracks neither the visible box nor the full box. Its error on the occluded edge grows from 5.6 to 24.4
    points.
  - dgem's entropy on the occluded edge rises with coverage (0.41 → 0.64 at 50%), but every edge rises (+0.10). The
    occluded-edge-specific part is +0.06 to +0.11, with intervals that include 0 (10 placements). Edge entropy does
    not reliably say *which* edge is hidden.
- **Edge entropy as an error flag** weakens on this set: AUROC 0.65 (against 0.82 on the EXP-09 fixtures).

### 3. Real images: usable on photos, not on GUI elements

| Set | dgem mIoU | dgem centre hit | Gemini 3.8 mIoU | Gemini centre hit |
| :--- | :---: | :---: | :---: | :---: |
| RefCOCO (120) | 0.393 [0.353, 0.432] | 75.4% | 0.781 [0.725, 0.836] | 90.0% |
| ScreenSpot (108) | 0.059 [0.037, 0.083] | 11.6% | 0.516 [0.457, 0.574] | 92.6% |
| All (fixed box: 0.119, 17.1%) | 0.235 [0.204, 0.268] | 45.2% | 0.656 [0.612, 0.699] | 91.2% |

On the full set, per-edge entropy flags wrong edges with AUROC 0.73.

**Open-vocabulary detectors** (top-scoring box per query; L4 GPU, one image at a time, deleted-after-use VM):

| Localizer | Sweep mIoU [95% CI] | Sweep centre hit | RefCOCO IoU / centre hit | ScreenSpot IoU / centre hit | Median latency |
| :--- | :---: | :---: | :---: | :---: | :---: |
| dgem | 0.197 [0.170, 0.226] | 43.3% | 0.393 / 75.4% | 0.059 / 11.6% | 0.74–0.77 s |
| OWLv2 (base) | 0.657 [0.604, 0.709] | 82.9% | 0.443 / 60.8% | 0.042 / 8.3% | 0.49 s |
| Grounding DINO (base) | 0.641 [0.589, 0.695] | 73.3% | **0.816 / 89.2%** | 0.032 / 4.6% | 0.65 s |
| Grounding DINO + SAM | 0.687 [0.631, 0.744] | 73.3% | 0.802 / 89.2% | 0.030 / 4.6% | 0.90 s |
| `gemini-3.8-flash` | **0.782** [0.733, 0.827] | **86.7%** | 0.781 / 90.0% | **0.516 / 92.6%** | 3.4–3.8 s |

- **Object queries on photos:** a ~200M-parameter open detector matches Gemini (Grounding DINO on RefCOCO).
- **Synthetic UI elements:** the detectors beat dgem by about 0.45 mIoU.
- **ScreenSpot instructions:** only Gemini handles them ("raise air conditioner temperature", "use fire emoji"). They
  need reading and reasoning, and every detector fails as dgem does.
- **Presence:** detector scores barely separate absent targets from present ones (AUROC 0.47–0.62 on the sweep),
  because the top box always exists.
- **SAM refinement** tightens boxes slightly on the sweep (+0.05 mIoU) and does not help on photos.

### 4. Categorical image questions suit a one-pass decision model better than coordinates

`dgem bench-vision`, 230 sweep items. All six questions are answered in one pass (`vision_aspects.json.tmpl`); dgem
×2, Gemini 3.8 ×1:

| Aspect | Options | dgem accuracy [95% CI] | Majority baseline | dgem lift | Hesitation AUROC | Gemini 3.8 | dgem − Gemini (paired) |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 3×3 grid cell (210) | 9 | 0.724 [0.667, 0.783] | 0.205 | **+0.52** | 0.76 | 0.919 | −0.20 [−0.26, −0.13] |
| Spatial relation (93) | 4 | 0.774 [0.688, 0.849] | 0.355 | **+0.42** | 0.61 | 0.935 | −0.16 [−0.24, −0.09] |
| Image quality (50) | 5 | 0.600 [0.460, 0.740] | 0.200 | **+0.40** | 0.68 | 0.720 | −0.12 [−0.24, 0.00] |
| Element count (230) | 4 | 0.683 [0.624, 0.739] | 0.430 | **+0.25** | 0.73 | 0.978 | −0.30 [−0.36, −0.24] |
| Target present (230) | 2 | 0.954 [0.928, 0.978] | 0.913 | +0.04 | 0.80 | 0.974 | −0.02 [−0.05, 0.00] |
| Target partly hidden (40) | 2 | 0.450 [0.300, 0.600] | 0.750 | **−0.30** | 0.71 | 1.000 | −0.55 [−0.70, −0.40] |

- **Speed:** dgem answers all six questions in one call with a median of **0.30 s**. Gemini's median is 10.4 s.
- **Blank images:** every dgem answer collapses to a single default (`present: no`, `grid_cell: middle_center`,
  `element_count: 1`, `image_quality: clean`). The scores above come from the image, not from the prompt.
- **Weak spots:**
  - Absent targets: only 55% of absent-set answers are right.
  - Occlusion: dgem answers "not hidden" 80% of the time.
  - Grid cell and relation drop to 0.48–0.50 on the occlusion set, where the occluding panel confuses the scene.

**Hesitation-gated cascade.** This is an offline simulation on the same items
(`python3 scripts/analyze_bbox_receipts.py cascade`). For each aspect, the most hesitant share of dgem's answers is
replaced with Gemini's answer for that item:

| Aspect | dgem only | 10% escalated | 20% | 30% | 50% | Gemini only |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| grid_cell | 0.729 | 0.771 | 0.805 | 0.857 | 0.895 | 0.919 |
| relation | 0.763 | 0.806 | 0.817 | 0.849 | 0.892 | 0.935 |
| element_count | 0.683 | 0.735 | 0.765 | 0.843 | 0.896 | 0.978 |
| image_quality | 0.600 | 0.680 | 0.740 | 0.740 | 0.720 | 0.720 |
| present | 0.957 | 0.978 | 0.974 | 0.974 | 0.970 | 0.974 |

### 5. Multi-aspect questions on real images

`bench-vision` on RefCOCO and ScreenSpot (`templates/multimodal/vision_aspects_generic.json.tmpl`): dgem ×2, dgem on
blank images, and Gemini 3.8 ×1. Run:
[`20261003-prop18-vision-real`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-vision-real/manifest.json).

**Ground truth** (`scripts/build_bbox_real_aspects.py`, `benchmarks/bbox_real_aspects.jsonl`):
- *Grid cell and relation* come from the annotated boxes. The relation is to another annotated target in the same
  image.
- *Presence* is scored on 228 positives plus 204 negatives. A negative is the same image queried with another item's
  text, kept only when Gemini confirmed it is absent. This flatters Gemini's own presence score.
- *Occlusion* ("hidden or cut off by the border") comes from a Gemini judge looking at the box overlay. It agreed with
  blind hand labels on 38 of 40 items (κ 0.89, `benchmarks/bbox_real_aspects_handcheck.json`).

| Aspect | Photos: dgem / Gemini | Screenshots: dgem / Gemini | dgem hesitation AUROC (all) |
| :--- | :---: | :---: | :---: |
| present | 0.937 / 0.973 | 0.864 / 0.981 | 0.87 |
| relation | 0.875 / 0.981 | 0.447 / 0.848 | 0.87 |
| grid cell | 0.671 / 0.875 | 0.426 / 0.880 | 0.74 |
| occluded | 0.396 / 0.867 | 0.995 / 1.000 (almost all "no") | 0.60 |

- **Blank images** fall to the majority-class baseline or below on every aspect.
- **Latency:** median 0.51 s for dgem (these are larger images), 6.5 s for Gemini.
- **Photos:** presence and relation are reliable.
- **Screenshots:** dgem's location answers are not reliable, consistent with its ScreenSpot boxes.

### 6. Live hesitation cascade, with the image

Stage 2 now receives the image (`ExecuteStage2GeminiCascadeWithImages`). This is the path that `POST /api/decide`,
both MCP decide tools and Studio batch items with `image_url` use.

Measured with `dgem bench-vision --cascade-threshold` on the 230 synthetic scenes. Run:
[`20261003-prop18-cascade`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-prop18-cascade/manifest.json),
reproduce with `scripts/run_prop18_cascade.sh`.

| Config | Requests that called Gemini | Slots escalated | End-to-end p50 / p90 | Accuracy dgem → cascade |
| :--- | :---: | :---: | :---: | :---: |
| six questions, 0.35 nats | 69% | 18% | 7.4 s / 35.8 s | 0.754 → 0.825 |
| scored questions only, 0.35 nats | **46%** | 19% | **0.43 s** / 17.4 s | 0.762 → **0.846** |
| six questions, 0.10 nats | 90% | 29% | 9.4 s / 35.0 s | 0.753 → 0.858 |
| scored questions only, 0.10 nats | 76% | 37% | 6.1 s / 29.2 s | 0.760 → 0.875 |
| Gemini 3.8 only (phase 4) | 100% | — | 10.4 s | 0.944 |

dgem's own part took a median 0.16 s. Per aspect (scored questions only, 0.35 nats; dgem → cascade, with the share
escalated):

| Aspect | dgem → cascade | Escalated |
| :--- | :---: | :---: |
| element_count | 0.69 → 0.83 | 32% |
| grid_cell | 0.68 → 0.81 | 23% |
| image_quality | 0.62 → 0.76 | 18% |
| present | 0.97 → 0.98 | 4% |
| relation | 0.85 → 0.85 | 19% |
| occluded | 0.40 → 0.50 | 15% |

- **Gemini is called per request.** One unsure slot sends the whole request to Gemini, so asking only the questions
  you need is the main latency lever.
- **The Stage-1 hint costs accuracy at low thresholds.** At 0.35 nats, escalated answers were 0.945 correct, close to
  Gemini alone on the same answers (0.957). At 0.10, they were 0.860 against 0.946, and Gemini kept dgem's answer 68%
  of the time. The Stage-2 prompt includes dgem's candidate and distribution, which anchors Gemini when dgem is only
  mildly unsure. EXP-23 tests hint against no hint.

The user-facing summary is in [What dgem Can Do With Images](/dgem/policies/images/).

## Decision

- **Boxes:** do not position DiffusionGemma as a localizer.
  - Its boxes beat image-free baselines, but are coarse on photos (centre hit 75% on RefCOCO).
  - They fail on small GUI elements (12% on ScreenSpot) and are unstable across repeats and transforms.
  - For boxes, use Grounding DINO for object queries on photos (matches Gemini at 0.65 s on one L4), and Gemini 3.x
    for GUI instructions and anything that needs reading or reasoning.
- **Categorical image questions are the supported shape:** grid cell, spatial relation, count, quality and presence in
  one ~300 ms pass, with hesitation that flags errors (AUROC 0.61–0.80).
  - Route hesitant answers to Gemini with the existing slot cascade (`cascade_mode: entropy`). On these items,
    escalating 10–30% of answers recovers most of Gemini's accuracy.
  - Do not use dgem alone for "is X present?" when distractors are present, or for "is X occluded?".
- **Validation tooling:** keep the Gemini judge (validated here) as the default labeller for image sets without ground
  truth. Re-check it on each new image domain with `bench-bbox-judge --calibrate`.
- **Done in the follow-up:**
  - `bench-vision` on real images (section 5);
  - the regression-matrix image gate (`vision_spot` in T0, `vision` ×2 in T1, with reference ranges from four v0.2.0
    runs);
  - the live cascade with images (section 6).
- **Next:** EXP-23, Gemini guided by dgem for boxes and masks (skip absent targets, crop to the grid cell, hint or no
  hint, thinking level).

## Files

- **Harnesses:** `cmd/bench_bbox*.go` (with `--engine dgem|gemini|predictions`), `cmd/bench_bbox_judge.go`,
  `cmd/bench_vision.go`.
- **Template:** `templates/multimodal/vision_aspects.json.tmpl`.
- **Data:**
  - `scripts/generate_bbox_sweep.py` → `benchmarks/bbox_sweep.jsonl`;
  - `scripts/fetch_bbox_real.py` → `benchmarks/bbox_real.jsonl`;
  - `scripts/requirements-vision.txt`.
- **Detectors:** `scripts/detectors/run_detectors.py`, `scripts/detectors/requirements.txt`,
  `scripts/run_detectors_gce.sh`. The VM is deleted on exit; inputs and outputs go through Cloud Storage, with no SSH.
- **Analysis:** `scripts/analyze_bbox_receipts.py` (`compare`, `tags`, `occlusion`, `judge`, `cascade`).
