---
title: "EXP-09: Single-Pass Bounding Boxes, Re-baselined"
description: "On serving v0.2.0, DiffusionGemma localizes the 11 synthetic EXP-09 targets well above image-free baselines (mIoU 0.60 vs 0.33), but the September claims about softmax-expectation sub-bin gains and occlusion entropy do not hold, and boxes are not stable under flips or padding."
---

# EXP-09: Single-Pass Bounding Boxes, Re-baselined

**Re-baseline date:** 2026-10-03 · **Backend:** Vertex AI G4 endpoint `<endpoint-id>` (`g4-standard-48`, RTX PRO 6000),
serving **v0.2.0** (`027b68e`, default layout `document_first`), `samples: 1` · **Code:** `a22657a` + the harness changes
described below.
**Run:** [`benchmarks/runs/20261003-exp09-rebaseline`](../../benchmarks/runs/20261003-exp09-rebaseline/manifest.json)
**Reproduce:** `ENDPOINT=<ENDPOINT_ID> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> ./scripts/run_exp09_rebaseline.sh`

The original EXP-09 (2026-09-21, Cloud Run, receipt `benchmarks/results_bbox_cloudrun.json`) is kept for history. Its
headline numbers are corrected [below](#what-changed-from-the-september-receipt).

## Question

A box is read as four coordinate slots `[ymin, xmin, ymax, xmax]` (21 labels each, 0–100 in 5% steps) plus an
`object_present` slot, all in one forward pass (`templates/multimodal/bbox_localization.json.tmpl`). The two-object
template (`bbox_multi_object_detr.json.tmpl`) reads two such boxes in decile bands. Three questions:

1. Does the model localize, or would an image-free guess score as well?
2. Does the probability-weighted coordinate (softmax expectation) beat the most likely label (argmax), and is per-edge
   entropy useful (does it flag wrong edges, or occluded ones)?
3. Are the boxes stable under changes that should not matter: option order, label style, flips and padding?

## Method

- **Suite:** the 12 synthetic UI fixtures in `benchmarks/bbox_suite.jsonl` (11 with a target, 1 absent). Small:
  every interval below is a case-level bootstrap over 11 cases and is wide.
- **Repeats:** every request ×3 to measure run-to-run stability (about 5–7% of answers change between identical requests
  on this model).
- **Baselines that use no image:** a fixed `[25, 25, 75, 75]` box; the mean ground-truth box of the other cases
  (leave-one-out); and the model's own coordinates for each prompt on a blank image of the same size. The model's
  *lift* is its IoU minus the baseline's, paired by case.
- **Probe variants** (`--variants all`):
  - `reversed`: the 21 coordinate options in reverse order, so letter `A` means 100 instead of 0.
  - `digits9`: the coordinates as a 9-level `score` question (labels `1`–`9`, 12.5% steps).
  - `digits9_reversed`: the same, with label `1` meaning 100.
  - `hflip` and `vflip`: the image mirrored, with the ground truth mirrored to match.
  - `pad_right`, `pad_left`, `pad_bottom`: the canvas extended by 50% with the background colour, with the ground truth
    rescaled to match.
  - `blank`: a blank image of the same size and colour.

  For the image transforms, *consistency* is the IoU between the box on the transformed image and the original box
  mapped through the same transform. It needs no ground truth.
- **Harness fixes** in `cmd/bench_bbox.go`. These change how results are scored, so they matter for comparing with the
  September numbers:
  - A slot the model does not return is scored as a failure. The old code filled it with the ground-truth coordinate.
  - `object_present` is no longer defaulted to the ground truth.
  - Decile-band labels (`"60"` = 60–70%) are read as band centres, not band starts.
  - Both objects of the two-object cases are scored, with order-free matching, plus their class slots.
  - Absent-target cases are excluded from box IoU. They used to count as IoU 1.
  - Entropy is normalized by each slot's own label count.

## Results

### 1. The model localizes, and clearly beats image-free baselines

| Predictor (11 cases, ×3) | mIoU [95% CI] | Acc@0.5 |
| :--- | :---: | :---: |
| Model, argmax box | 0.588 [0.494, 0.686] | — |
| **Model, expectation box** | **0.599 [0.516, 0.680]** | **81.8%** |
| Fixed box `[25, 25, 75, 75]` | 0.331 [0.178, 0.499] | 18.2% |
| Leave-one-out mean ground-truth box | 0.296 [0.161, 0.450] | 18.2% |
| The model's own box on a blank image | 0.057 [0.000, 0.121] | 0% |

The paired lift over the fixed box is **+0.27 [+0.11, +0.40]**. On blank images the model answered `object_present: no`
in 30 of 30 runs. On the absent-target fixture it raised no false positive in 3 runs. Run-to-run stability is high:
97.0% of edge argmaxes agree across repeats, and the mean IoU between repeat boxes is 0.945. An earlier session with
identical settings measured an expectation mIoU of 0.635, well inside this interval.

### 2. Softmax expectation does not reliably help, and occlusion does not raise entropy

- **Expectation vs argmax:** the mean IoU gain is +0.01.
  - On cases whose edges all sit on a label (no rounding error to recover) it is **−0.029 [−0.061, −0.003]**.
  - On off-grid cases it is **+0.044 [−0.035, +0.140]**.
  - Most edge error is not rounding error. A perfect argmax reader would still be off by 1.69 points on average; the
    model's argmax is off by 3.6–3.9 points.
  - Of the 156 scored edges, the argmax lands on the label nearest the ground truth for 68, one step away for 71, and
    two steps away for 17.
- **Entropy flags wrong edges:** per-edge normalized entropy ranks the 17 edges whose argmax is more than one step
  (5 points) from the truth above the other 139, with **AUROC 0.82**. It barely tracks the size of the error (Spearman
  0.12). Treat this as promising, not established: only 17 edges are wrong.
- **Occlusion:** compared with its unoccluded twin, the occluded edge's entropy went *down* in both pairs (ymax −0.03,
  xmax −0.06). It ranked 2nd and 3rd of the four edges in its own image. On these fixtures, edge entropy does not
  indicate occlusion.
- **Systematic bias:** the right edge (`xmax`) is pulled left by 7.1 points on average, so boxes come out too narrow.
  The other edges are within ±1.7 points.

### 3. Boxes move under changes that should not matter

| Variant | mIoU (expectation) [95% CI] | Acc@0.5 | Consistency with the original |
| :--- | :---: | :---: | :---: |
| original | 0.599 [0.516, 0.680] | 81.8% | — |
| `reversed` (letter A = 100) | 0.550 [0.466, 0.641] | 66.7% | — |
| `digits9` (labels 1–9) | 0.622 [0.562, 0.690] | 77.8% | — |
| `digits9_reversed` (label 1 = 100) | **0.024** [0.003, 0.054] | 0% | — |
| `hflip` | 0.499 [0.399, 0.592] | 57.6% | 0.384 [0.266, 0.484] |
| `vflip` | 0.519 [0.429, 0.608] | 57.6% | 0.525 [0.396, 0.648] |
| `pad_right` | 0.567 [0.475, 0.660] | 69.7% | 0.467 [0.373, 0.573] |
| `pad_left` | 0.426 [0.343, 0.506] | 33.3% | 0.475 [0.366, 0.577] |
| `pad_bottom` | **0.307** [0.196, 0.412] | 15.2% | 0.339 [0.224, 0.451] |

`digits9` variants skip the two band-labelled two-object cases (9 cases).

- **Letter order:** reversing the options moves the top and left edges by about +5 points toward 100, against +0.6 and
  −1.7 in forward order. This is a small pull toward early letters, like the first-choice bias in EXP-13 and EXP-15. It
  is not large enough to explain the original errors.
- **Digit labels:** forward digit labels work as well as 21 letters, despite the coarser 12.5% grid. **Reversed digit
  levels break localization completely.** The model reads `score` levels as an ordinal scale whatever their names say.
  Never give coordinate scores in descending order.
- **Flips and padding:** a mirrored or padded image gives a box that agrees with the original box at IoU 0.34–0.53.
  - With `pad_bottom`, predicted y coordinates sit between the original frame and the padded frame (+10 to +12 points
    too low). The model partly reports coordinates relative to the content rather than to the full image.
  - With `hflip`, the narrow-box bias grows (xmax −11.5).
  - Coordinates on non-square or mirrored inputs need their own validation before use.

## What changed from the September receipt

| Claim (2026-09-21, Cloud Run, ×1) | Re-analysis of that receipt | v0.2.0, Vertex G4, ×3 |
| :--- | :--- | :--- |
| Expectation lifts mIoU 0.290 → 0.377 (+30% relative); Acc@0.5 0% → 18.2% | A fixed centre box scored 0.331 with Acc@0.5 18.2%. The model's lift over it was +0.05 [−0.10, +0.16], not significant. | The model localizes (0.60, lift +0.27). Expectation adds +0.01 overall and nothing on on-grid cases. |
| Occluded edges have 1.37× higher entropy | 2 occluded edges against 38 visible ones. Paired with their twins: +0.27 and −0.14. | No increase in either pair. |
| Entropy marks uncertain edges | AUROC 0.51 for wrong edges (no signal) | AUROC 0.82 |
| `008.png` stemware (+50.4% IoU) and ruler overlay (3.97×) | Not in any committed receipt | Not re-run. Treat as unverified. |

The improvement between the two sessions coincides with the serving changes that shipped in between:
- full label distributions (`logprob_token_ids`);
- the v0.2.0 `document_first` prompt layout;
- Vertex G4 instead of Cloud Run.

This experiment does not isolate which change was responsible.

## Decision

- Report EXP-09 localization only with image-free baselines, intervals and repeats. Use `bench-bbox` defaults for new
  runs.
- Drop the "sub-bin regression" and "occlusion entropy" claims from user-facing docs. Keep softmax expectation as an
  option, not a recommended default.
- Per-edge entropy as an error flag is worth confirming on a larger set.
- Validation continues in [`PROP-18`](proposed.md#prop-18-validate-image-metrics-beyond-exp-09), adjusted by these
  results:
  - a Gemini 3.x reference model and judge;
  - a larger synthetic set that varies aspect ratio, horizontal position and occluder coverage;
  - real images (RefCOCO, ScreenSpot);
  - OWLv2 / Grounding DINO / SAM references run on a short-lived GCE VM.

## Files

- Harness: `cmd/bench_bbox.go`, `cmd/bench_bbox_analysis.go`, `cmd/bench_bbox_variants.go` (tests in
  `cmd/bench_bbox_test.go`)
- Reproduction: `scripts/run_exp09_rebaseline.sh`; `dgem bench-bbox --from-receipt <receipt>` re-analyzes any receipt
- Receipts (`benchmarks/runs/20261003-exp09-rebaseline/`):
  - `bbox__vertex_g4_all_x3.json`: the live run.
  - `bbox__legacy_cloudrun_20260921_reanalyzed.json`: the September receipt re-analyzed.
  - `bbox__simulated_all_variants.json`: a harness self-check.
