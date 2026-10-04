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

"""Build the EXP-26 domain image set (seeded). Images are written under fixtures/domain_images/ (gitignored); the
manifest benchmarks/domain_images.jsonl records each image's SHA-256 and source so `fetch` can rebuild it.

Domains (each item: target text, ground-truth box when present, "aspects" ground truth, "aspect_sources"):
  ui_mobile  RICO referring expressions (Hugging Face ivelin/rico_refexp_combined, test split; CC BY 4.0)
  ui_web     ScreenSpot-v2 web split (OS-Copilot/ScreenSpot-v2; Apache-2.0)
  documents  DocLayNet v1.1 test pages (ds4sd/DocLayNet-v1.1; CDLA-Permissive-1.0). Targets are layout elements that
             occur exactly once on the page ("the table", "the page header"); negatives ask for a category absent
             from the page (natural ground truth). Relation: between two single-instance elements.
  pcb        synthetic boards from this script (components + one defect: solder bridge, scratch, missing component,
             contamination spot); negatives ask for a defect type not drawn (natural ground truth).

UI negatives: the same image queried with another image's instruction, kept only when a Gemini 3.x judge confirms it
is absent ("gemini_confirmed"); a 40-item hand check of those negatives is kept with the manifest.
FUNSD is not used: its licence is non-commercial research only.

  pip install -r scripts/requirements-vision.txt
  python3 scripts/build_domain_images.py build --project <PROJECT>   # needs ADC for the UI negatives
  python3 scripts/build_domain_images.py fetch                       # rebuild images from sources + verify hashes
  python3 scripts/build_domain_images.py sheet                       # contact sheets of UI negatives for hand check
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import io
import json
import random
import subprocess
import sys
import time
import urllib.request
import zipfile
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
CACHE = Path("/tmp/dgem-domain-cache")
HF = "https://huggingface.co/datasets"
SRC = {
    "rico": f"{HF}/ivelin/rico_refexp_combined/resolve/main/data/test-00000-of-00001-09f7223bee6a973d.parquet",
    "ss_json": f"{HF}/OS-Copilot/ScreenSpot-v2/resolve/main/screenspot_web_v2.json",
    "ss_zip": f"{HF}/OS-Copilot/ScreenSpot-v2/resolve/main/screenspotv2_image.zip",
    "doclaynet": f"{HF}/ds4sd/DocLayNet-v1.1/resolve/main/data/test-00000-of-00002-635b47e9044a436c.parquet",
}
LOCAL = {"rico": "rico_refexp_test.parquet", "ss_json": "screenspot_web_v2.json", "ss_zip": "screenspotv2_image.zip",
         "doclaynet": "doclaynet_test0.parquet"}
DOC_CATS = {1: "caption", 2: "footnote", 3: "formula", 4: "list item", 5: "page footer", 6: "page header", 7: "picture",
            8: "section header", 9: "table", 10: "text block", 11: "title"}
DOC_TARGET_CATS = [1, 3, 5, 6, 7, 9, 11]  # distinctive single-instance elements
OUT_DIR = "fixtures/domain_images"


def cached(key):
    p = CACHE / LOCAL[key]
    if not p.exists():
        CACHE.mkdir(parents=True, exist_ok=True)
        print(f"  downloading {SRC[key]}", file=sys.stderr)
        urllib.request.urlretrieve(SRC[key], p)
    return p


def sha(b):
    return hashlib.sha256(b).hexdigest()


def grid_cell(b):
    cy, cx = (b[0] + b[2]) / 200, (b[1] + b[3]) / 200
    return ["top", "middle", "bottom"][min(2, int(cy * 3))] + "_" + ["left", "center", "right"][min(2, int(cx * 3))]


def relation(t, r):
    dx = (t[1] + t[3]) / 2 - (r[1] + r[3]) / 2
    dy = (t[0] + t[2]) / 2 - (r[0] + r[2]) / 2
    if abs(dx) > 1.5 * abs(dy) and abs(dx) > 5:
        return "left_of" if dx < 0 else "right_of"
    if abs(dy) > 1.5 * abs(dx) and abs(dy) > 5:
        return "above" if dy < 0 else "below"
    return None


def size_bucket(b):
    a = (b[2] - b[0]) * (b[3] - b[1])
    return "small" if a < 75 else ("medium" if a < 400 else "large")


def positive(iid, domain, path, data, target, box, src, ref=None, extra_tags=None):
    asp = {"present": "yes", "grid_cell": grid_cell(box)}
    srcs = {"present": "annotation", "grid_cell": "geometry"}
    row = {"id": iid, "tier": domain, "template": "templates/multimodal/vision_aspects_generic.json.tmpl",
           "image_path": path, "target": target, "object_present": True, "gt_box_continuous": [round(v, 3) for v in box],
           "occluded_edge": "none", "aspects": asp, "aspect_sources": srcs,
           "tags": {"set": domain, "domain": domain, "polarity": "positive", "size": size_bucket(box), **(extra_tags or {})},
           "image_sha256": sha(data), "source": src}
    if ref:
        rel = relation(box, ref[1])
        if rel:
            asp["relation"], srcs["relation"], row["reference"] = rel, "geometry", ref[0]
    return row


def negative(iid, domain, path, data, target, src, how, extra_tags=None):
    return {"id": iid, "tier": domain, "template": "templates/multimodal/vision_aspects_generic.json.tmpl",
            "image_path": path, "target": target, "object_present": False, "gt_box_continuous": [0, 0, 0, 0],
            "occluded_edge": "none", "aspects": {"present": "no"}, "aspect_sources": {"present": how},
            "tags": {"set": domain, "domain": domain, "polarity": "negative", **(extra_tags or {})},
            "image_sha256": sha(data), "source": src}


def save(rel, data):
    p = REPO / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_bytes(data)


# ------------------------------------------------------------------ UI mobile (RICO refexp)
def build_rico(rng, n):
    import pyarrow.parquet as pq
    pf = pq.ParquetFile(cached("rico"))
    meta = pf.read(columns=["image_id", "prompt", "target_bounding_box"]).to_pylist()
    by_img = defaultdict(list)
    for i, r in enumerate(meta):
        by_img[r["image_id"]].append(i)
    imgs = sorted(by_img)
    rng.shuffle(imgs)
    chosen = imgs[:n]
    want = {by_img[im][0]: im for im in chosen}
    img_rows = {}
    row = 0
    for g in range(pf.num_row_groups):
        t = pf.read_row_group(g, columns=["image"])
        for j in range(t.num_rows):
            if row + j in want:
                img_rows[row + j] = t.slice(j, 1).to_pylist()[0]["image"]["bytes"]
        row += t.num_rows
    rows, pool = [], []
    for idx, im in want.items():
        r, data = meta[idx], img_rows[idx]
        bb = r["target_bounding_box"]
        box = [100 * bb["ymin"], 100 * bb["xmin"], 100 * bb["ymax"], 100 * bb["xmax"]]
        others = [meta[k] for k in by_img[im] if k != idx]
        ref = None
        for o in others:
            ob = o["target_bounding_box"]
            obox = [100 * ob["ymin"], 100 * ob["xmin"], 100 * ob["ymax"], 100 * ob["xmax"]]
            if relation(box, obox):
                ref = (o["prompt"], obox)
                break
        path = f"{OUT_DIR}/ui_mobile/rico_{im}.png"
        save(path, data)
        rows.append(positive(f"ui_mobile-{im}", "ui_mobile", path, data, r["prompt"], box,
                             {"dataset": "rico_refexp_combined", "row": idx}, ref))
        pool.append((path, data, im, r["prompt"]))
    return rows, pool


# ------------------------------------------------------------------ UI web (ScreenSpot-v2)
def build_ss_web(rng, n, exclude_files):
    from PIL import Image
    items = json.load(open(cached("ss_json")))
    items = [it for it in items if it["img_filename"] not in exclude_files]
    strata = defaultdict(list)
    for i, it in enumerate(items):
        strata[(it["data_source"], it["data_type"])].append(i)
    pools = {k: rng.sample(v, len(v)) for k, v in sorted(strata.items())}
    picked, files = [], set()
    while len(picked) < n and any(pools.values()):  # round-robin over strata, one item per screenshot
        for k in sorted(pools):
            while pools[k]:
                i = pools[k].pop()
                if items[i]["img_filename"] not in files:
                    files.add(items[i]["img_filename"])
                    picked.append(i)
                    break
            if len(picked) >= n:
                break
    zf = zipfile.ZipFile(cached("ss_zip"))
    names = {Path(x).name: x for x in zf.namelist()}
    by_file = defaultdict(list)
    for i, it in enumerate(items):
        by_file[it["img_filename"]].append(i)
    rows, pool = [], []
    seen_files = set()
    for i in picked:
        it = items[i]
        if it["img_filename"] in seen_files:
            continue
        seen_files.add(it["img_filename"])
        data = zf.read(names[it["img_filename"]])
        w, h = Image.open(io.BytesIO(data)).size
        x, y, bw, bh = it["bbox"]  # xywh pixels
        box = [100 * y / h, 100 * x / w, 100 * (y + bh) / h, 100 * (x + bw) / w]
        ref = None
        for k in by_file[it["img_filename"]]:
            if k == i:
                continue
            ox, oy, ow, oh = items[k]["bbox"]
            obox = [100 * oy / h, 100 * ox / w, 100 * (oy + oh) / h, 100 * (ox + ow) / w]
            if relation(box, obox):
                ref = (items[k]["instruction"], obox)
                break
        path = f"{OUT_DIR}/ui_web/{it['img_filename']}"
        save(path, data)
        rows.append(positive(f"ui_web-{i:03d}", "ui_web", path, data, it["instruction"], box,
                             {"dataset": "ScreenSpot-v2/web", "file": it["img_filename"], "index": i}, ref,
                             {"platform": it["data_source"], "element": it["data_type"]}))
        pool.append((path, data, it["data_source"], it["instruction"]))
    return rows, pool


# ------------------------------------------------------------------ documents (DocLayNet)
def build_doclaynet(rng, n_pos, n_neg):
    import pyarrow.parquet as pq
    pf = pq.ParquetFile(cached("doclaynet"))
    meta = []
    for g in range(pf.num_row_groups):
        t = pf.read_row_group(g, columns=["bboxes", "category_id", "metadata"]).to_pylist()
        for j, r in enumerate(t):
            meta.append((g, j, r))
    order = list(range(len(meta)))
    rng.shuffle(order)
    pos_plan, neg_plan = [], []
    used = set()
    for k in order:
        g, j, r = meta[k]
        counts = Counter(r["category_id"])
        singles = [c for c in DOC_TARGET_CATS if counts.get(c) == 1]
        absent = [c for c in DOC_TARGET_CATS if counts.get(c, 0) == 0]
        if singles and len(pos_plan) < n_pos:
            # Prefer distinctive elements; page header/footer only when nothing else is single on the page.
            pref = [c for c in singles if c not in (5, 6)] or singles
            pos_plan.append((k, rng.choice(pref), singles))
            used.add(k)
        elif absent and len(neg_plan) < n_neg and k not in used:
            neg_plan.append((k, rng.choice(absent)))
            used.add(k)
        if len(pos_plan) >= n_pos and len(neg_plan) >= n_neg:
            break
    need = defaultdict(list)
    for k, *_ in pos_plan + neg_plan:
        need[meta[k][0]].append(meta[k][1])
    imgs = {}
    for g, js in need.items():
        t = pf.read_row_group(g, columns=["image"])
        for j in js:
            imgs[(g, j)] = t.slice(j, 1).to_pylist()[0]["image"]["bytes"]
    rows = []

    def box_of(r, c):
        i = r["category_id"].index(c)
        x, y, w, h = r["bboxes"][i]
        W, H = r["metadata"]["coco_width"], r["metadata"]["coco_height"]
        return [100 * y / H, 100 * x / W, 100 * (y + h) / H, 100 * (x + w) / W]

    for k, c, singles in pos_plan:
        g, j, r = meta[k]
        data = imgs[(g, j)]
        box = box_of(r, c)
        ref = None
        for oc in singles:
            if oc != c and relation(box, box_of(r, oc)):
                ref = (f"the {DOC_CATS[oc]}", box_of(r, oc))
                break
        md = r["metadata"]
        path = f"{OUT_DIR}/documents/doclaynet_{md['page_hash'][:16]}.png"
        save(path, data)
        rows.append(positive(f"documents-{md['page_hash'][:12]}", "documents", path, data, f"the {DOC_CATS[c]}", box,
                             {"dataset": "DocLayNet-v1.1/test", "row_group": g, "row": j}, ref,
                             {"category": DOC_CATS[c], "doc_category": md["doc_category"]}))
    for k, c in neg_plan:
        g, j, r = meta[k]
        data = imgs[(g, j)]
        md = r["metadata"]
        path = f"{OUT_DIR}/documents/doclaynet_{md['page_hash'][:16]}.png"
        save(path, data)
        rows.append(negative(f"documents-{md['page_hash'][:12]}-neg", "documents", path, data, f"the {DOC_CATS[c]}",
                             {"dataset": "DocLayNet-v1.1/test", "row_group": g, "row": j}, "annotation_absent",
                             {"category": DOC_CATS[c], "doc_category": md["doc_category"]}))
    return rows


# ------------------------------------------------------------------ PCB (synthetic)
DEFECTS = {"solder_bridge": "the solder bridge between two pads", "scratch": "the scratch on the board",
           "missing_component": "the empty footprint where a component is missing",
           "contamination": "the contamination spot on the board"}


def draw_board(rng, W, H):
    from PIL import Image, ImageDraw
    img = Image.new("RGB", (W, H), (20, 90, 50))
    d = ImageDraw.Draw(img)
    for _ in range(rng.randint(25, 45)):  # copper traces
        x, y = rng.randint(0, W), rng.randint(0, H)
        pts = [(x, y)]
        for _ in range(rng.randint(2, 4)):
            if rng.random() < 0.5:
                x = max(0, min(W, x + rng.randint(-W // 3, W // 3)))
            else:
                y = max(0, min(H, y + rng.randint(-H // 3, H // 3)))
            pts.append((x, y))
        d.line(pts, fill=(196, 160, 70), width=rng.randint(3, 6))
    taken, comps = [], []

    def free(b, m=14):
        return not any(not (b[2] + m < t[0] or t[2] + m < b[0] or b[3] + m < t[1] or t[3] + m < b[1]) for t in taken)

    kinds = ["ic", "resistor", "capacitor", "connector"]
    for _ in range(rng.randint(6, 10)):
        k = rng.choice(kinds)
        for _ in range(100):
            if k == "ic":
                w, h = rng.randint(W // 9, W // 6), rng.randint(H // 9, H // 6)
            elif k == "connector":
                w, h = rng.randint(W // 5, W // 3), rng.randint(H // 18, H // 12)
            else:
                w, h = rng.randint(W // 22, W // 14), rng.randint(H // 34, H // 24)
                if rng.random() < 0.5:
                    w, h = h, w
            x0, y0 = rng.randint(20, W - w - 20), rng.randint(20, H - h - 20)
            b = (x0, y0, x0 + w, y0 + h)
            if free(b):
                taken.append(b)
                comps.append((k, b))
                break
    for k, (x0, y0, x1, y1) in comps:
        if k == "ic":
            for i in range(6):  # pins
                px = x0 + (i + 0.5) * (x1 - x0) / 6
                d.rectangle((px - 3, y0 - 8, px + 3, y0), fill=(200, 200, 200))
                d.rectangle((px - 3, y1, px + 3, y1 + 8), fill=(200, 200, 200))
            d.rectangle((x0, y0, x1, y1), fill=(25, 25, 25))
            d.ellipse((x0 + 6, y0 + 6, x0 + 14, y0 + 14), fill=(70, 70, 70))
        elif k == "resistor":
            d.rectangle((x0, y0, x1, y1), fill=(210, 180, 140))
        elif k == "capacitor":
            d.rectangle((x0, y0, x1, y1), fill=(150, 110, 70))
        else:
            d.rectangle((x0, y0, x1, y1), fill=(235, 235, 235), outline=(90, 90, 90), width=2)
    return img, d, taken, comps


def build_pcb(rng, n_pos, n_neg, out):
    W, H = 1200, 900
    rows = []
    types = list(DEFECTS)
    for i in range(n_pos + n_neg):
        img, d, taken, comps = draw_board(rng, W, H)
        pos = i < n_pos
        defect = types[i % len(types)]
        box = None
        if pos:
            if defect == "solder_bridge":
                k, (x0, y0, x1, y1) = rng.choice([c for c in comps if c[0] == "ic"] or comps)
                px = x0 + rng.randint(1, 4) * (x1 - x0) / 6
                bx = (px - 6, y1, px + (x1 - x0) / 6 + 6, y1 + 10)
                d.rectangle(bx, fill=(220, 220, 225))
                box = bx
            elif defect == "scratch":
                x, y = rng.randint(80, W - 300), rng.randint(80, H - 200)
                x2, y2 = x + rng.randint(120, 260), y + rng.randint(40, 160)
                d.line((x, y, x2, y2), fill=(150, 200, 160), width=4)
                box = (x - 3, y - 3, x2 + 3, y2 + 3)
            elif defect == "missing_component":
                k, (x0, y0, x1, y1) = rng.choice(comps)
                d.rectangle((x0, y0, x1, y1), fill=(20, 90, 50))
                pw = max(4, (x1 - x0) // 4)
                d.rectangle((x0, y0, x0 + pw, y1), fill=(196, 160, 70))
                d.rectangle((x1 - pw, y0, x1, y1), fill=(196, 160, 70))
                box = (x0, y0, x1, y1)
            else:
                for _ in range(100):
                    r = rng.randint(14, 28)
                    x, y = rng.randint(60, W - 60), rng.randint(60, H - 60)
                    if all(not (x + r > t[0] and x - r < t[2] and y + r > t[1] and y - r < t[3]) for t in taken):
                        break
                d.ellipse((x - r, y - r, x + r, y + r), fill=(110, 80, 40))
                box = (x - r, y - r, x + r, y + r)
        buf = io.BytesIO()
        img.save(buf, format="PNG", optimize=True)
        data = buf.getvalue()
        path = f"{OUT_DIR}/pcb/pcb_{i:03d}.png"
        save(path, data)
        src = {"dataset": "synthetic", "seed_index": i}
        if pos:
            pbox = [100 * box[1] / H, 100 * box[0] / W, 100 * box[3] / H, 100 * box[2] / W]
            ic = next((c for c in comps if c[0] == "ic" and (defect != "missing_component" or c[1] != tuple(int(v) for v in box))), None)
            ref = None
            if ic:
                b = ic[1]
                ref = ("the large black IC chip", [100 * b[1] / H, 100 * b[0] / W, 100 * b[3] / H, 100 * b[2] / W])
            rows.append(positive(f"pcb-{i:03d}", "pcb", path, data, DEFECTS[defect], pbox, src, ref, {"defect": defect}))
        else:
            rows.append(negative(f"pcb-{i:03d}-neg", "pcb", path, data, DEFECTS[defect], src, "synthetic_absent",
                                 {"defect": defect}))
    return rows


# ------------------------------------------------------------------ Gemini-confirmed UI negatives
_TOK = {"t": None, "at": 0.0}


def gemini_visible(project, model, data, text):
    if not _TOK["t"] or time.time() - _TOK["at"] > 1800:
        _TOK["t"] = subprocess.check_output(["gcloud", "auth", "application-default", "print-access-token"], text=True).strip()
        _TOK["at"] = time.time()
    url = f"https://aiplatform.googleapis.com/v1/projects/{project}/locations/global/publishers/google/models/{model}:generateContent"
    body = {"contents": [{"role": "user", "parts": [
        {"inlineData": {"mimeType": "image/png", "data": base64.b64encode(data).decode()}},
        {"text": f"Is there anything in this screenshot that matches the instruction target \"{text}\"? Answer \"no\" only if "
                 f"you are confident nothing on screen matches it; answer \"unsure\" if something might."}]}],
        "generationConfig": {"responseMimeType": "application/json", "responseSchema": {
            "type": "OBJECT", "properties": {"visible": {"type": "STRING", "enum": ["yes", "no", "unsure"]}}, "required": ["visible"]}}}
    for attempt in range(5):
        try:
            req = urllib.request.Request(url, data=json.dumps(body).encode(),
                                         headers={"Authorization": f"Bearer {_TOK['t']}", "Content-Type": "application/json"})
            with urllib.request.urlopen(req, timeout=300) as r:
                return json.loads(json.load(r)["candidates"][0]["content"]["parts"][0]["text"])["visible"]
        except Exception as e:  # noqa: BLE001
            err = e
            time.sleep(2 + 3 * attempt * attempt)
    raise RuntimeError(err)


def ui_negatives(rng, pool, domain, project, model, workers):
    cands = []
    for path, data, group, text in pool:
        others = [p for p in pool if p[0] != path and p[2] != group] or [p for p in pool if p[0] != path]
        cands.append((path, data, rng.choice(others)[3]))
    with ThreadPoolExecutor(workers) as ex:
        verdicts = list(ex.map(lambda c: gemini_visible(project, model, c[1], c[2]), cands))
    rows = []
    for (path, data, text), v in zip(cands, verdicts):
        if v == "no":
            stem = Path(path).stem
            rows.append(negative(f"{domain}-{stem}-neg", domain, path, data, text, {"derived_from": path},
                                 f"gemini_confirmed:{model}"))
    return rows


def build(a):
    rng = random.Random(a.seed)
    exclude = set()
    real = REPO / "benchmarks/bbox_real.jsonl"
    if real.exists():  # keep ScreenSpot-v2 web disjoint from the EXP-22 ScreenSpot sample
        exclude = {Path(json.loads(l)["image_path"]).name for l in open(real)}
    rows = []
    rico, rico_pool = build_rico(rng, a.n)
    web, web_pool = build_ss_web(rng, a.n, exclude)
    rows += rico + web
    rows += build_doclaynet(rng, a.n, a.n)
    rows += build_pcb(rng, a.n, a.n, None)
    if a.project:
        rows += ui_negatives(rng, rico_pool, "ui_mobile", a.project, a.model, a.workers)
        rows += ui_negatives(rng, web_pool, "ui_web", a.project, a.model, a.workers)
    with open(REPO / a.manifest, "w") as f:
        for r in rows:
            f.write(json.dumps(r) + "\n")
    c = Counter((r["tier"], r["tags"]["polarity"]) for r in rows)
    print(f"wrote {len(rows)} items to {a.manifest}: {dict(sorted(c.items()))}")


def fetch(a):
    rows = [json.loads(l) for l in open(REPO / a.manifest)]
    missing = [r for r in rows if not (REPO / r["image_path"]).exists() or
               sha((REPO / r["image_path"]).read_bytes()) != r["image_sha256"]]
    if missing:
        print(f"{len(missing)} images missing; rebuilding from sources (same seed)", file=sys.stderr)
        tmp = a.manifest + ".rebuild"
        b = argparse.Namespace(**{**vars(a), "manifest": tmp, "project": ""})
        build(b)
        (REPO / tmp).unlink()
    bad = [r["id"] for r in rows if sha((REPO / r["image_path"]).read_bytes()) != r["image_sha256"]]
    print("all images present and verified" if not bad else f"{len(bad)} hash mismatches: {bad[:5]}")
    sys.exit(1 if bad else 0)


def sheet(a):
    from PIL import Image, ImageDraw
    rows = [json.loads(l) for l in open(REPO / a.manifest)]
    negs = [r for r in rows if r["aspect_sources"]["present"].startswith("gemini_confirmed")]
    rng = random.Random(a.seed + 7)
    rng.shuffle(negs)
    negs = negs[:a.n_check]
    out = Path(a.sheet_dir)
    out.mkdir(parents=True, exist_ok=True)
    for s in range(0, len(negs), 4):
        canvas = Image.new("RGB", (1280, 1400), "white")
        for i, r in enumerate(negs[s:s + 4]):
            im = Image.open(REPO / r["image_path"]).convert("RGB")
            im.thumbnail((630, 640))
            x, y = (i % 2) * 640, (i // 2) * 700
            canvas.paste(im, (x, y + 50))
            ImageDraw.Draw(canvas).text((x + 6, y + 6), f'{r["id"]}\nQUERY: {r["target"][:80]}', fill="black")
        canvas.save(out / f"neg_{s // 4:02d}.png")
    json.dump([r["id"] for r in negs], open(out / "ids.json", "w"), indent=1)
    print(f"{len(negs)} negatives in {out}")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("cmd", choices=["build", "fetch", "sheet"])
    ap.add_argument("--seed", type=int, default=26)
    ap.add_argument("--n", type=int, default=100, help="positives per domain (and negatives for documents/pcb)")
    ap.add_argument("--project", default="")
    ap.add_argument("--model", default="gemini-3.8-flash")
    ap.add_argument("--workers", type=int, default=8)
    ap.add_argument("--manifest", default="benchmarks/domain_images.jsonl")
    ap.add_argument("--sheet-dir", default="/tmp/dgem-domain-handcheck")
    ap.add_argument("--n-check", type=int, default=40)
    a = ap.parse_args()
    {"build": build, "fetch": fetch, "sheet": sheet}[a.cmd](a)


if __name__ == "__main__":
    main()
