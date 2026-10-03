#!/usr/bin/env python3
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Open-vocabulary detector references for PROP-18 (runs on a GPU VM; see scripts/run_detectors_gce.sh).

For every manifest item the target text is the query, and the single highest-scoring box is kept:
  owlv2           google/owlv2-base-patch16-ensemble
  grounding_dino  IDEA-Research/grounding-dino-base
  gdino_sam       the Grounding DINO box refined to the tight box of its SAM mask (facebook/sam-vit-base)

Output: JSONL rows {id, model, box_pct [ymin, xmin, ymax, xmax] in percent, score, found, latency_ms},
scored on the workstation with `dgem bench-bbox --engine predictions --predictions <file> --pred-model <model>`.

  pip install -r scripts/detectors/requirements.txt
  python3 scripts/detectors/run_detectors.py --manifest benchmarks/bbox_sweep.jsonl benchmarks/bbox_real.jsonl -o preds.jsonl

EXP-23 masks from boxes: SAM refines each prompt box (any source) and the mask is scored against the COCO mask:
  python3 scripts/detectors/run_detectors.py --manifest benchmarks/bbox_real.jsonl --sam-boxes sam_boxes.jsonl \
      --gt-masks benchmarks/bbox_real_masks.jsonl -o sam_masks.jsonl
"""

import argparse
import json
import sys
import time

import numpy as np
import torch
from PIL import Image
from transformers import (AutoModelForZeroShotObjectDetection, AutoProcessor, Owlv2ForObjectDetection,
                          Owlv2Processor, SamModel, SamProcessor)

DEV = "cuda" if torch.cuda.is_available() else "cpu"


def query_text(target):
    return target.replace("_", " ").strip()


def pct(box, w, h):
    x0, y0, x1, y1 = [float(v) for v in box]
    x0, x1 = max(0.0, min(x0, w)), max(0.0, min(x1, w))
    y0, y1 = max(0.0, min(y0, h)), max(0.0, min(y1, h))
    return [100 * y0 / h, 100 * x0 / w, 100 * y1 / h, 100 * x1 / w]


class Owl:
    name = "owlv2"

    def __init__(self):
        self.p = Owlv2Processor.from_pretrained("google/owlv2-base-patch16-ensemble")
        self.m = Owlv2ForObjectDetection.from_pretrained("google/owlv2-base-patch16-ensemble").to(DEV).eval()

    @torch.no_grad()
    def __call__(self, img, text):
        w, h = img.size
        # OWLv2's text encoder takes at most 16 tokens; longer queries are truncated.
        inputs = self.p(text=[[text]], images=img, return_tensors="pt", truncation=True, max_length=16,
                        padding="max_length").to(DEV)
        out = self.m(**inputs)
        side = max(w, h)  # OWLv2 pads to a square; boxes are relative to the padded image
        res = self.p.post_process_object_detection(out, threshold=0.0, target_sizes=torch.tensor([[side, side]]).to(DEV))[0]
        if len(res["scores"]) == 0:
            return None, 0.0
        i = int(res["scores"].argmax())
        return res["boxes"][i].tolist(), float(res["scores"][i])


class GDino:
    name = "grounding_dino"

    def __init__(self):
        mid = "IDEA-Research/grounding-dino-base"
        self.p = AutoProcessor.from_pretrained(mid)
        self.m = AutoModelForZeroShotObjectDetection.from_pretrained(mid).to(DEV).eval()

    @torch.no_grad()
    def __call__(self, img, text):
        w, h = img.size
        t = text.lower().rstrip(".") + "."
        inputs = self.p(images=img, text=t, return_tensors="pt").to(DEV)
        out = self.m(**inputs)
        try:
            res = self.p.post_process_grounded_object_detection(
                out, inputs.input_ids, threshold=0.0, text_threshold=0.0, target_sizes=[(h, w)])[0]
        except TypeError:  # older transformers
            res = self.p.post_process_grounded_object_detection(
                out, inputs.input_ids, box_threshold=0.0, text_threshold=0.0, target_sizes=[(h, w)])[0]
        if len(res["scores"]) == 0:
            return None, 0.0
        i = int(res["scores"].argmax())
        return res["boxes"][i].tolist(), float(res["scores"][i])


class Sam:
    def __init__(self):
        self.p = SamProcessor.from_pretrained("facebook/sam-vit-base")
        self.m = SamModel.from_pretrained("facebook/sam-vit-base").to(DEV).eval()

    @torch.no_grad()
    def refine(self, img, box):
        inputs = self.p(img, input_boxes=[[box]], return_tensors="pt").to(DEV)
        out = self.m(**inputs, multimask_output=False)
        masks = self.p.image_processor.post_process_masks(
            out.pred_masks.cpu(), inputs["original_sizes"].cpu(), inputs["reshaped_input_sizes"].cpu())
        m = masks[0][0][0].numpy()
        ys, xs = np.where(m)
        if len(xs) == 0:
            return box
        return [float(xs.min()), float(ys.min()), float(xs.max() + 1), float(ys.max() + 1)]


def sam_from_boxes(args):
    from PIL import ImageDraw
    paths = {}
    for m in args.manifest:
        for l in open(m):
            if l.strip():
                r = json.loads(l)
                paths[r["id"]] = r["image_path"]
    gt = {json.loads(l)["id"]: json.loads(l) for l in open(args.gt_masks)}
    sam = Sam()
    rows = [json.loads(l) for l in open(args.sam_boxes) if l.strip()]
    print(f"{len(rows)} SAM box prompts on {DEV}", file=sys.stderr)
    cache = {}
    with open(args.output, "w") as f:
        for n, r in enumerate(rows):
            g = gt[r["id"]]
            w, h = g["width"], g["height"]
            if r["id"] not in cache:
                img = Image.open(paths[r["id"]]).convert("RGB")
                gm = Image.new("L", (w, h), 0)
                d = ImageDraw.Draw(gm)
                for poly in g["polygons"]:
                    if len(poly) >= 6:
                        d.polygon(poly, fill=1)
                cache = {r["id"]: (img, np.asarray(gm, dtype=bool))}
            img, gmask = cache[r["id"]]
            b = r["box_pct"]
            box = [b[1] / 100 * w, b[0] / 100 * h, b[3] / 100 * w, b[2] / 100 * h]
            t0 = time.time()
            with torch.no_grad():
                inputs = sam.p(img, input_boxes=[[box]], return_tensors="pt").to(DEV)
                out = sam.m(**inputs, multimask_output=False)
                pm = sam.p.image_processor.post_process_masks(
                    out.pred_masks.cpu(), inputs["original_sizes"].cpu(), inputs["reshaped_input_sizes"].cpu())[0][0][0].numpy()
            u = (pm | gmask).sum()
            f.write(json.dumps({"id": r["id"], "source": r["source"], "mask_iou": float((pm & gmask).sum() / u) if u else 0.0,
                                "latency_ms": round((time.time() - t0) * 1e3, 1)}) + "\n")
            if n % 50 == 0:
                print(f"  {n}/{len(rows)}", file=sys.stderr, flush=True)
    print("done", file=sys.stderr)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--manifest", nargs="+", required=True)
    ap.add_argument("-o", "--output", required=True)
    ap.add_argument("--models", default="owlv2,grounding_dino,gdino_sam")
    ap.add_argument("--sam-boxes", default="", help="JSONL of {id, source, box_pct}: refine each with SAM and score the mask")
    ap.add_argument("--gt-masks", default="benchmarks/bbox_real_masks.jsonl")
    args = ap.parse_args()
    if args.sam_boxes:
        return sam_from_boxes(args)
    want = set(args.models.split(","))

    items = []
    for m in args.manifest:
        items += [json.loads(l) for l in open(m) if l.strip()]
    print(f"{len(items)} items on {DEV} ({torch.cuda.get_device_name(0) if DEV == 'cuda' else 'cpu'})", file=sys.stderr)

    owl = Owl() if "owlv2" in want else None
    gd = GDino() if want & {"grounding_dino", "gdino_sam"} else None
    sam = Sam() if "gdino_sam" in want else None

    with open(args.output, "w") as f:
        for n, it in enumerate(items):
            img = Image.open(it["image_path"]).convert("RGB")
            w, h = img.size
            text = query_text(it["target"])
            rows = []

            def safe(fn, *a):
                # One failing item is recorded as "not found" with the error instead of stopping the run.
                try:
                    return fn(*a) + (None,)
                except Exception as e:  # noqa: BLE001
                    return None, 0.0, f"{type(e).__name__}: {e}"[:300]

            if owl:
                t0 = time.time()
                box, score, err = safe(owl, img, text)
                rows.append(("owlv2", box, score, (time.time() - t0) * 1e3, err))
            if gd:
                t0 = time.time()
                box, score, err = safe(gd, img, text)
                ms = (time.time() - t0) * 1e3
                if "grounding_dino" in want:
                    rows.append(("grounding_dino", box, score, ms, err))
                if sam:
                    t1 = time.time()
                    rbox, err2 = box, err
                    if box:
                        try:
                            rbox = sam.refine(img, box)
                        except Exception as e:  # noqa: BLE001
                            rbox, err2 = None, f"{type(e).__name__}: {e}"[:300]
                    rows.append(("gdino_sam", rbox, score, ms + (time.time() - t1) * 1e3, err2))
            for model, box, score, ms, err in rows:
                row = {"id": it["id"], "model": model, "found": box is not None,
                       "box_pct": pct(box, w, h) if box else [0, 0, 0, 0], "score": score, "latency_ms": round(ms, 1)}
                if err:
                    row["error"] = err
                f.write(json.dumps(row) + "\n")
            if n % 50 == 0:
                print(f"  {n}/{len(items)}", file=sys.stderr, flush=True)
    print("done", file=sys.stderr)


if __name__ == "__main__":
    main()
