---
title: "EXP-24: dgem-Guided Gemini Boxes and Masks"
description: "A 0.19 s dgem pass in front of Gemini 3.8 Flash: skipping confidently-absent targets saves 35% of Gemini calls for 1.3% lost positives; a grid-cell hint plus LOW thinking matches or beats Gemini alone (mIoU 0.654 vs 0.639) at 46% lower latency; cropping to dgem's cell does not help; SAM from Gemini's box beats Gemini's own polygon masks (0.73 vs 0.64)."
---

**Date:** 2026-10-03.

**Backends:**
- dgem: Vertex AI G4 endpoint `<endpoint-id>`, serving v0.2.0, `samples: 1`.
- Gemini: `gemini-3.8-flash` on Vertex.
- SAM (`facebook/sam-vit-base`): on a short-lived `g2-standard-8` + L4 VM, deleted after the run.

**Run:** [`benchmarks/runs/20261003-exp24-guided`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261003-exp24-guided/manifest.json).
**Reproduce:** `./scripts/run_exp24_guided.sh`, then `SAM_BOXES=… ./scripts/run_detectors_gce.sh`, then
`python3 scripts/analyze_guided.py <receipt> --sam <sam_masks.jsonl>`.
**Follows:** [EXP-22](/dgem/experiments/exp-22-image-readouts/), which found dgem fast and calibrated on categorical image questions
but not a localizer.

## Question

Can a fast dgem pass make Gemini's boxes and masks faster, cheaper or better? dgem first answers two questions in one
pass (median **0.19 s**): "is the target present?" and "which 3×3 cell holds it?". Gemini then works under one of four
strategies:

| Strategy | What Gemini gets |
| :--- | :--- |
| `full` | the whole image, no guidance (Gemini alone) |
| `skip` | nothing, when dgem says "absent" with hesitation below 16%; otherwise `full` (scored offline) |
| `hint` | the whole image, plus dgem's cell as text ("a fast first-pass model thinks … it may be wrong") |
| `crop` | the image cropped to dgem's cell plus half a cell of margin, when dgem is confident. If Gemini finds nothing, or its box touches a crop edge inside the image, the whole image is used instead |

Each strategy ran at Gemini thinking levels `default`, `MEDIUM` (`full` only) and `LOW`. `MINIMAL` is rejected by
`gemini-3.8-flash` ("Thinking level is unsupported").

For masks, two routes were compared:
- Gemini's own outline: a polygon of 8–40 points (`poly@low`);
- SAM prompted with a box: from Gemini, from Grounding DINO, or the ground-truth box as an upper bound.

**Data:** the EXP-22 real set (`benchmarks/bbox_real_aspects.jsonl`): 228 positives (120 RefCOCO photos, 108
ScreenSpot screenshots) and 204 negatives (absent targets, Gemini-confirmed). Masks were scored against COCO instance
masks for the 120 RefCOCO items (`benchmarks/bbox_real_masks.jsonl`).

## Results

### Boxes (228 positives)

| Condition | mIoU [95% CI] | Centre hit | Small-target mIoU | End-to-end p50 / p90 | Gemini thought tokens |
| :--- | :---: | :---: | :---: | :---: | :---: |
| full @ default (Gemini alone) | 0.639 [0.593, 0.683] | 91.2% | 0.43 | 3.9 s / 8.2 s | 498 |
| full @ medium | 0.641 [0.597, 0.686] | 91.7% | 0.41 | 3.6 s / 7.6 s | 518 |
| full @ low | 0.627 [0.580, 0.671] | 91.7% | 0.43 | 2.4 s / 4.4 s | 177 |
| hint @ default | **0.668** [0.625, 0.711] | 93.4% | 0.39 | 4.0 s / 12.2 s | 641 |
| **hint @ low** | 0.654 [0.611, 0.696] | **93.9%** | 0.41 | **2.1 s / 4.2 s** | **135** |
| crop @ default | 0.635 [0.589, 0.679] | 93.0% | 0.42 | 4.1 s / 10.7 s | 667 |
| crop @ low | 0.638 [0.593, 0.680] | 93.4% | 0.41 | 2.8 s / 5.1 s | 181 |

End-to-end latency includes dgem's 0.19 s for `hint` and `crop`.

Paired differences (case-level bootstrap):
- hint @ default − full @ default: **+0.029 [+0.004, +0.056]**.
- hint @ low − full @ low: +0.027 [0.000, +0.056].
- hint @ low − full @ default: +0.015 [−0.007, +0.037].

Findings:
- **The hint helps only when dgem's cell is right.** On the 119 items where it was right, IoU rose by 0.05. On the 109
  where it was wrong, the change was −0.01 to +0.01. Gemini does not follow a wrong hint.
- **The hint helps photos and screenshots alike.** RefCOCO: 0.784 → 0.815. ScreenSpot: 0.477 → 0.504.
- **Best trade-off: `hint @ low`.** It matches Gemini alone at default thinking on accuracy, with a 46% lower median
  latency (2.1 s vs 3.9 s), a 49% lower p90, and 73% fewer thought tokens. dgem's call adds about 0.2 s of GPU time.
- **Cropping does not help.**
  - dgem's cell was confident on only 49 of 228 items, and right on 36 of those 49.
  - Half of the crops fell back to the whole image.
  - Gemini bills an image at about the same token count whatever its size (≈1,064 tokens here), so a crop saves no
    input cost.
- **Small targets** stay hard for every condition (0.39–0.43).

### Skip Gemini when dgem confidently says "absent" (432 items, 47% absent)

| Thinking | Gemini calls saved | Absent targets skipped | Present targets wrongly skipped | Presence accuracy | Mean end-to-end |
| :--- | :---: | :---: | :---: | :---: | :---: |
| default | 35% | 149 / 204 | 3 / 228 (1.3%) | 0.958 → 0.956 | 4.2 s → 3.5 s |
| low | 35% | 149 / 204 | 3 / 228 | 0.961 → 0.958 | 2.5 s → 2.1 s |

The absent targets were chosen because Gemini agreed they were absent. That probably makes Gemini's own presence
accuracy look higher than it is, but it does not change the skip arithmetic. The saving scales with the share of
requests whose target is really absent.

### Masks (120 RefCOCO items, against COCO masks)

| Source | Mask IoU [95% CI] | Latency |
| :--- | :---: | :---: |
| Gemini polygon outline (`poly @ low`) | 0.640 [0.593, 0.682] | 2.8 s (Gemini) |
| **SAM from Gemini's box (`full @ default`)** | **0.731** [0.685, 0.775] (n = 113) | 3.9 s + 0.23 s (SAM, L4) |
| SAM from Gemini's box (`full @ low`) | 0.716 [0.665, 0.764] (n = 114) | 2.4 s + 0.23 s |
| SAM from Grounding DINO's box | 0.691 [0.635, 0.745] | 0.65 s + 0.23 s |
| SAM from the ground-truth box (upper bound) | 0.771 [0.727, 0.807] | — |

- The SAM rows from Gemini exclude the 6–7 items where Gemini returned no box. Counting those as 0, the scores are
  0.688 and 0.680.
- **Gemini's native masks** (base64 PNG, the usual Gemini segmentation format) are not practical on
  `gemini-3.8-flash`. A single item took 72 s and hit the 16k output-token limit. That is why the polygon route was
  measured instead.

## Decision

The recommended image pipeline is "dgem first, Gemini when needed":
1. **dgem answers the categorical questions** (presence, location, relations) in about 0.2 s (EXP-22).
2. **Skip Gemini** when dgem confidently says the target is absent.
3. **For boxes, call Gemini at LOW thinking with dgem's grid cell as a hint.** This is as accurate as Gemini alone at
   default thinking, at about half the latency.
4. **For masks, run SAM on that box.** It is more accurate than asking Gemini for a polygon. Gemini's PNG masks are
   too slow.

Do not crop to dgem's cell. For photos with nameable objects and no instructions to follow, Grounding DINO + SAM is
the fastest route (about 0.9 s on one L4) at 0.69 mask IoU.

Follow-ups:
- implement steps 2–4 as a `locate` mode on the gateway and MCP (the `locate_bounding_boxes` tool still uses dgem's
  own boxes);
- repeat with `gemini-3.7-flash`, and on a domain set (UI review, documents);
- re-measure when a Gemini model supports `MINIMAL` thinking.

## Files

- `cmd/bench_guided.go` (`dgem bench-guided`)
- `scripts/analyze_guided.py`
- `scripts/run_exp24_guided.sh`
- SAM mode in `scripts/detectors/run_detectors.py` and `scripts/run_detectors_gce.sh` (`SAM_BOXES`)
- Mask ground truth: `scripts/build_bbox_real_aspects.py masks` → `benchmarks/bbox_real_masks.jsonl`
