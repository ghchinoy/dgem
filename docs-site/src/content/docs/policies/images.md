---
title: "What dgem Can Do With Images"
description: "Measured strengths and limits of DiffusionGemma on images: fast categorical questions (presence, 3x3 location, relations, count, quality) in one ~0.2 s pass, with hesitation to route hard cases to Gemini; not bounding boxes, occlusion or screenshot grounding."
---

DiffusionGemma reads an image with Gemma 4's vision tower and answers a policy's questions about it in one forward
pass. This page says which image questions that works for, based on the measurements in
[EXP-09](/dgem/experiments/exp-09-spatial-grounding/) and [EXP-22](/dgem/experiments/exp-22-image-readouts/).
Every number here comes from serving v0.2.0 on Vertex G4 (`samples: 1`). Re-check on your own images with
`dgem bench-vision`.

## The short version

- **Use dgem for fast categorical questions about an image.** It can answer several questions together, in a median
  of about 0.16–0.5 s per image. Gemini 3.8 Flash took 6.5–10 s for the same questions.
- **It is less accurate than Gemini on every question.** Its value is speed and cost, plus a hesitation score that
  says when to ask Gemini instead.
- **Do not use it for coordinates.** Use a detector or Gemini for bounding boxes and masks, for "is this hidden or cut
  off?", or for locating elements in app and web screenshots.

## Supported questions and measured accuracy

| Question | Photos (RefCOCO) | Synthetic UI scenes | Screenshots (ScreenSpot) | Gemini 3.8 Flash |
| :--- | :---: | :---: | :---: | :---: |
| Is the described thing in the image? | 0.94 | 0.95 | 0.86 | 0.97–0.98 |
| Is it left of / right of / above / below another thing? | 0.88 | 0.77 | 0.45 | 0.85–0.98 |
| Which cell of a 3×3 grid holds it? | 0.67 | 0.72 | 0.43 | 0.88–0.92 |
| How many elements (1–4)? | — | 0.68 | — | 0.98 |
| Image quality (clean, blurred, noisy, compressed, low contrast)? | — | 0.60 | — | 0.72 |
| Is part of it hidden or cut off? | 0.40 | 0.45 | — | 0.87–1.00 |

Notes on the table:
- Photos: 120 RefCOCO expressions. Screenshots: 108 ScreenSpot instructions. Synthetic: 230 generated UI scenes.
  Each question has its own count of scored items; see EXP-22.
- On real images, "Is the described thing in the image?" is scored on matched present and absent queries. The absent
  queries were kept only when Gemini confirmed absence, which flatters Gemini's presence score.
- The baselines that ignore the image are 0.18–0.21 for the grid cell, 0.36–0.39 for relations and 0.43 for counts.
- On a blank image, every answer collapses to a fixed default (absent, centre cell, one element, clean). The answers
  come from the image, not the prompt.
- Answers are stable across identical requests: 92–99% agree.

**Strong:** presence on photos and simple UI scenes; spatial relations between named things in photos.

**Usable with a cascade:** grid cell, counts, image quality.

**Not supported:**
- Occlusion: it answers "not hidden" almost every time.
- Grid cell and relations on app or web screenshots.
- Boxes. Centre hit is 75% on photos and 12% on screenshots. Boxes also move when the image is flipped or padded.

## Hesitation tells you when to ask Gemini

Each answer comes with a hesitation score (normalized entropy). High hesitation picks out wrong answers with an AUROC
of 0.60–0.87, depending on the question. The Stage-2 cascade uses this:
- slots at or above an entropy threshold go to Gemini 3.x;
- the image goes to Gemini too, since EXP-22;
- `cascade_mode: "entropy"` turns it on, on `POST /api/decide`, the MCP decide tools and Studio batch runs.

Measured live on the 230 synthetic scenes, asking only the questions each item is scored on:

| Setting | Requests answered by dgem alone | End-to-end p50 / p90 | Accuracy (all questions) |
| :--- | :---: | :---: | :---: |
| dgem only | 100% | 0.16 s / 0.33 s | 0.76 |
| Cascade at 0.35 nats (default) | 54% | **0.43 s** / 17 s | **0.85** |
| Cascade at 0.10 nats | 24% | 6.1 s / 29 s | 0.88 |
| Gemini 3.8 Flash only | 0% | 10.4 s / — | 0.94 |

Guidance:
- **Ask only the questions you need.** Gemini is called whenever *any* slot of a request is unsure. With six questions
  per image instead of only the scored ones, 69% of requests reached Gemini (instead of 46%), and the p50 rose from
  0.43 s to 7.4 s.
- **Keep the threshold at the default 0.35 nats.** At 0.10, Gemini kept dgem's (wrong) answer more often: escalated
  answers were 0.86 correct, against 0.95 when Gemini answered alone. In the cascade prompt, dgem's answer and its
  distribution are given as the candidate, and that can anchor Gemini. A softer hint is different: in
  [EXP-24](/dgem/experiments/exp-24-guided-cascade/), "probably in the top-left; may be wrong" never hurt Gemini's boxes.

## By domain (EXP-26)

dgem was measured on mobile UI (RICO), web UI (ScreenSpot-v2), document pages (DocLayNet) and synthetic circuit boards,
with presence, 3×3 grid cell and relation questions ([EXP-26](/dgem/experiments/exp-26-domain-images/), rule fixed before scoring):

| Domain | dgem alone | With a Gemini cascade | Not supported | Skip Gemini on "absent"? |
| :--- | :--- | :--- | :--- | :--- |
| Web UI | present (0.90) | — | grid cell, relation | no (loses 3–4% of targets) |
| Mobile UI | — | present (0.83), grid cell (0.59) | relation | yes (22% of calls saved) |
| Document pages | — | grid cell (0.61) | present, relation | no (saves only 10%) |
| Circuit-board defects | — | grid cell (0.37) | present, relation | no (misses 15% of defects) |

Gemini 3.7 Flash matched 3.8 on these questions and boxes at lower latency (4.8 s vs 7.6 s median for three questions).

## Boxes, masks and screenshots: use something else

| Need | Use | Measured (EXP-22) |
| :--- | :--- | :--- |
| Box for an object described in a photo | Grounding DINO (open source, ~0.65 s on one L4) or Gemini 3.x | centre hit 89–90% on RefCOCO |
| Box for a UI element from an instruction ("raise the temperature") | Gemini 3.x | centre hit 93% on ScreenSpot; open detectors 5–8% |
| Grade or label boxes without ground truth | Gemini 3.x as a judge (`dgem bench-bbox-judge`) | catches 99–100% of edges more than 5 points off |
| Faster Gemini boxes | `dgem locate`, `POST /api/locate` or MCP `locate_object` (defaults: LOW thinking, no hint, no skip): Gemini 3.x (3.7 or 3.8) at LOW thinking, without a dgem hint; skip Gemini when dgem confidently says absent only on photos and mobile UI | LOW thinking cuts median latency 30–45% at the same accuracy (except defect images); dgem's grid-cell hint helped on photos (EXP-24) but not on UI, documents or PCB ([EXP-26](/dgem/experiments/exp-26-domain-images/)) |
| Masks | SAM prompted with the Gemini (or Grounding DINO) box | mask IoU 0.73 (0.69 with Grounding DINO) vs 0.64 for Gemini's own polygon outline |

## Validate on your images

```bash
# Ground truth in a JSONL manifest: {"id", "image_path", "target", "aspects": {"present": "yes", "grid_cell": "top_left", ...}}
./bin/dgem bench-vision -d my_images.jsonl -t templates/multimodal/vision_aspects_generic.json.tmpl \
  --vertex-url <ENDPOINT_ID> --gcp-auth --repeat 2 -o mine.json

# The same questions to Gemini, and dgem with the live cascade
./bin/dgem bench-vision -d my_images.jsonl -t templates/multimodal/vision_aspects_generic.json.tmpl --engine gemini -o mine_gemini.json
./bin/dgem bench-vision -d my_images.jsonl -t templates/multimodal/vision_aspects_generic.json.tmpl \
  --vertex-url <ENDPOINT_ID> --gcp-auth --only-scored --cascade-threshold 0.35 -o mine_cascade.json
```

- **Templates:** `templates/multimodal/vision_aspects.json.tmpl` (UI scenes) and `vision_aspects_generic.json.tmpl`
  (photos and screenshots).
- **Regression matrix:** the T1 `vision` suite tracks this on every serving change.
