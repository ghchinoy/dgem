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

"""Score an EXP-24 bench-guided receipt.

Boxes (positives): IoU, centre hit and Acc@0.5 per condition, overall and by set and size, with case-level
bootstrap 95% intervals; end-to-end latency (dgem pass + Gemini for guided strategies, Gemini only for "full");
Gemini tokens. Skip-absent: for each thinking level, Gemini is skipped when dgem says "absent" with normalized
entropy below --skip-h; reports the share of calls saved, positives lost and the latency/accuracy effect.
Masks: polygon masks from the "poly" strategy against COCO masks (needs Pillow + benchmarks/bbox_real_masks.jsonl);
optional SAM masks from a VM run (--sam).

  python3 scripts/analyze_guided.py <receipt> [--sam sam_masks.jsonl] [--export-sam-boxes out.jsonl --detector-preds preds.jsonl]
"""

import argparse
import json
import random
import statistics
import sys
from collections import defaultdict
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent


def iou(a, b):
    ih = max(0.0, min(a[2], b[2]) - max(a[0], b[0]))
    iw = max(0.0, min(a[3], b[3]) - max(a[1], b[1]))
    inter = ih * iw
    u = (a[2] - a[0]) * (a[3] - a[1]) + (b[2] - b[0]) * (b[3] - b[1]) - inter
    return inter / u if u > 0 else 0.0


def centre_hit(pred, gt):
    cy, cx = (pred[0] + pred[2]) / 2, (pred[1] + pred[3]) / 2
    return gt[0] <= cy <= gt[2] and gt[1] <= cx <= gt[3]


def boot(vals, seed=1, iters=2000):
    if not vals:
        return float("nan"), float("nan"), float("nan")
    m = statistics.fmean(vals)
    rng = random.Random(seed)
    ms = sorted(statistics.fmean(rng.choice(vals) for _ in vals) for _ in range(iters))
    return m, ms[int(0.025 * iters)], ms[int(0.975 * iters) - 1]


def ci(t, d=3):
    return f"{t[0]:.{d}f} [{t[1]:.{d}f}, {t[2]:.{d}f}]"


def pct(v, q):
    v = sorted(v)
    return v[int(q * (len(v) - 1))] if v else float("nan")


def size_bucket(b):
    a = (b[2] - b[0]) * (b[3] - b[1])
    return "small" if a < 75 else ("medium" if a < 400 else "large")


def end_to_end(call, dgem_ms):
    return call["gemini_ms"] + (dgem_ms if call["strategy"] in ("hint", "crop") else 0.0)


def boxes(rep):
    items = [it for it in rep["items"] if it["positive"] and it.get("gt_box")]
    conds = [c for c in rep["conditions"] if not c.startswith("poly")]
    print(f"## Boxes ({len(items)} positives; gemini {rep['gemini_model']}; dgem crop trusted below H~ {rep['confident_h_norm']})\n")
    print("| condition | mIoU [95% CI] | centre hit | Acc@0.5 | small mIoU | e2e p50 / p90 ms | Gemini ms p50 | thought tok | cropped / fell back |")
    print("| :--- | :---: | ---: | ---: | :---: | ---: | ---: | ---: | ---: |")
    out = {}
    for cond in conds:
        ious, hits, a50, small, e2e, gms, th = [], [], [], [], [], [], []
        cropped = fell = 0
        for it in items:
            c = next((x for x in it["calls"] if x["condition"] == cond), None)
            if not c or c.get("error"):
                continue
            gt = it["gt_box"]
            b = c.get("box_pct")
            v = iou(gt, b) if b else 0.0
            ious.append(v)
            hits.append(1.0 if b and centre_hit(b, gt) else 0.0)
            a50.append(1.0 if v >= 0.5 else 0.0)
            if size_bucket(gt) == "small":
                small.append(v)
            e2e.append(end_to_end(c, it["dgem"]["ms"]))
            gms.append(c["gemini_ms"])
            th.append(c["thought_tokens"])
            cropped += bool(c.get("cropped"))
            fell += bool(c.get("fell_back"))
        out[cond] = {"miou": statistics.fmean(ious), "e2e_p50": pct(e2e, .5)}
        print(f"| {cond} | {ci(boot(ious))} | {statistics.fmean(hits):.1%} | {statistics.fmean(a50):.1%} | "
              f"{ci(boot(small))} | {pct(e2e, .5):.0f} / {pct(e2e, .9):.0f} | {pct(gms, .5):.0f} | {statistics.fmean(th):.0f} | {cropped} / {fell} |")
    # Paired deltas vs full@default and by set.
    base = "full@default"
    if base in conds:
        print(f"\nPaired mIoU vs {base} (case-level bootstrap):")
        for cond in conds:
            if cond == base:
                continue
            d = []
            for it in items:
                a = next((x for x in it["calls"] if x["condition"] == cond and not x.get("error")), None)
                b = next((x for x in it["calls"] if x["condition"] == base and not x.get("error")), None)
                if a and b:
                    ga = iou(it["gt_box"], a["box_pct"]) if a.get("box_pct") else 0.0
                    gb = iou(it["gt_box"], b["box_pct"]) if b.get("box_pct") else 0.0
                    d.append(ga - gb)
            print(f"  {cond:14s} {ci(boot(d))}  (n={len(d)})")
    print("\nmIoU by set:")
    for s in sorted({it["tags"].get("set") for it in items}):
        row = []
        for cond in conds:
            v = [iou(it["gt_box"], c["box_pct"]) if c.get("box_pct") else 0.0
                 for it in items if it["tags"].get("set") == s
                 for c in it["calls"] if c["condition"] == cond and not c.get("error")]
            row.append(f"{cond} {statistics.fmean(v):.3f}")
        print(f"  {s}: " + " | ".join(row))
    dg = [it["dgem"] for it in items if not it["dgem"].get("error")]
    conf = [d for d in dg if d["grid_h_norm"] < rep["confident_h_norm"]]
    cc = [d for d in conf if d.get("cell_correct") is not None]
    print(f"\ndgem pass: median {pct([d['ms'] for d in dg], .5):.0f} ms; grid cell confident on {len(conf)}/{len(dg)}, "
          f"correct when confident {sum(d['cell_correct'] for d in cc)}/{len(cc)}")
    return out


def skip(rep, skip_h):
    print(f"\n## Skip Gemini when dgem confidently says absent (H~ < {skip_h})\n")
    print("| thinking | Gemini calls saved | negatives skipped | positives wrongly skipped | presence acc: Gemini only -> with skip | mean e2e ms: Gemini only -> with skip |")
    print("| :--- | ---: | ---: | ---: | :---: | :---: |")
    levels = sorted({c.split("@")[1] for c in rep["neg_conditions"] if c.startswith("full@")})
    for lv in levels:
        cond = f"full@{lv}"
        n = saved = negskip = poswrong = 0
        acc_g = acc_s = 0
        t_g, t_s = [], []
        npos = nneg = 0
        for it in rep["items"]:
            c = next((x for x in it["calls"] if x["condition"] == cond and not x.get("error")), None)
            d = it["dgem"]
            if not c or d.get("error"):
                continue
            n += 1
            pos = it["positive"]
            npos += pos
            nneg += not pos
            sk = d["present"] == "no" and d["present_h_norm"] < skip_h
            g_present = bool(c.get("box_pct"))
            acc_g += (g_present == pos)
            t_g.append(c["gemini_ms"])
            if sk:
                saved += 1
                negskip += not pos
                poswrong += pos
                acc_s += (not pos)
                t_s.append(d["ms"])
            else:
                acc_s += (g_present == pos)
                t_s.append(d["ms"] + c["gemini_ms"])
        print(f"| {lv} | {saved}/{n} ({saved / n:.0%}) | {negskip}/{nneg} | {poswrong}/{npos} | {acc_g / n:.3f} -> {acc_s / n:.3f} | "
              f"{statistics.fmean(t_g):.0f} -> {statistics.fmean(t_s):.0f} |")


def masks(rep, sam_path):
    try:
        from PIL import Image, ImageDraw
        import numpy as np
    except ImportError:
        print("\n(masks skipped: pip install -r scripts/requirements-vision.txt)")
        return
    gt = {json.loads(l)["id"]: json.loads(l) for l in open(REPO / "benchmarks/bbox_real_masks.jsonl")}

    def raster(polys_px, w, h):
        im = Image.new("L", (w, h), 0)
        d = ImageDraw.Draw(im)
        for p in polys_px:
            if len(p) >= 6:
                d.polygon(p, fill=1)
        return np.asarray(im, dtype=bool)

    def miou(a, b):
        u = (a | b).sum()
        return float((a & b).sum() / u) if u else 0.0

    print("\n## Masks (RefCOCO, against COCO instance masks)\n")
    print("| source | n | mask IoU [95% CI] | box-of-mask IoU vs GT box | Gemini ms p50 |")
    print("| :--- | ---: | :---: | :---: | ---: |")
    for cond in [c for c in rep["conditions"] if c.startswith("poly")]:
        vals, bvals, ms = [], [], []
        for it in rep["items"]:
            if it["id"] not in gt:
                continue
            c = next((x for x in it["calls"] if x["condition"] == cond and not x.get("error")), None)
            if not c:
                continue
            g = gt[it["id"]]
            w, h = g["width"], g["height"]
            gm = raster(g["polygons"], w, h)
            poly = c.get("polygon_pct") or []
            pm = raster([[v for y, x in poly for v in (x / 100 * w, y / 100 * h)]], w, h) if len(poly) >= 3 else np.zeros_like(gm)
            vals.append(miou(gm, pm))
            if len(poly) >= 3:
                ys, xs = [p[0] for p in poly], [p[1] for p in poly]
                bvals.append(iou(it["gt_box"], [min(ys), min(xs), max(ys), max(xs)]))
            ms.append(c["gemini_ms"])
        print(f"| Gemini polygon ({cond}) | {len(vals)} | {ci(boot(vals))} | {statistics.fmean(bvals):.3f} | {pct(ms, .5):.0f} |")
    if sam_path:
        by = defaultdict(list)
        for l in open(sam_path):
            r = json.loads(l)
            by[r["source"]].append(r["mask_iou"])
        for src, v in sorted(by.items()):
            print(f"| SAM from {src} box | {len(v)} | {ci(boot(v))} | — | — |")


def export_sam_boxes(rep, out, det_path):
    """Boxes to refine with SAM on the VM: ground truth (upper bound), Gemini full@default and full@low, Grounding DINO."""
    gt = {json.loads(l)["id"] for l in open(REPO / "benchmarks/bbox_real_masks.jsonl")}
    rows = []
    for it in rep["items"]:
        if it["id"] not in gt:
            continue
        rows.append({"id": it["id"], "source": "ground_truth", "box_pct": it["gt_box"]})
        for cond in ("full@default", "full@low"):
            c = next((x for x in it["calls"] if x["condition"] == cond and x.get("box_pct")), None)
            if c:
                rows.append({"id": it["id"], "source": f"gemini_{cond.replace('@', '_')}", "box_pct": c["box_pct"]})
    if det_path and Path(det_path).exists():
        for l in open(det_path):
            p = json.loads(l)
            if p["model"] == "grounding_dino" and p["id"] in gt and p.get("found"):
                rows.append({"id": p["id"], "source": "grounding_dino", "box_pct": p["box_pct"]})
    with open(out, "w") as f:
        for r in rows:
            f.write(json.dumps(r) + "\n")
    print(f"\nwrote {len(rows)} SAM box prompts to {out}")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("receipt")
    ap.add_argument("--skip-h", type=float, default=0.16)
    ap.add_argument("--sam", default="")
    ap.add_argument("--export-sam-boxes", default="")
    ap.add_argument("--detector-preds", default="")
    a = ap.parse_args()
    rep = json.load(open(a.receipt))
    boxes(rep)
    skip(rep, a.skip_h)
    masks(rep, a.sam)
    if a.export_sam_boxes:
        export_sam_boxes(rep, a.export_sam_boxes, a.detector_preds)


if __name__ == "__main__":
    sys.exit(main())
