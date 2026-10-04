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

"""Acceptance check for guided locate (#74): send the EXP-24 items through POST /api/locate on a running gateway and
compare with the EXP-24 table (hint@low mIoU 0.654, skip saves 35% of calls, 3/228 positives skipped).

  ./bin/dgem serve --port 8099 --vertex-url <ENDPOINT_ID> &
  python3 scripts/locate_acceptance.py --gateway http://localhost:8099 -o locate_acceptance.json

Standard library only. Images are sent as data: URIs (fixtures/bbox_real/, from scripts/fetch_bbox_real.py fetch).
"""

import argparse
import base64
import json
import mimetypes
import random
import statistics
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent


def iou(a, b):
    ih = max(0.0, min(a[2], b[2]) - max(a[0], b[0]))
    iw = max(0.0, min(a[3], b[3]) - max(a[1], b[1]))
    inter = ih * iw
    u = (a[2] - a[0]) * (a[3] - a[1]) + (b[2] - b[0]) * (b[3] - b[1]) - inter
    return inter / u if u > 0 else 0.0


def boot(v, iters=2000):
    rng = random.Random(74)
    ms = sorted(statistics.fmean(rng.choice(v) for _ in v) for _ in range(iters))
    return statistics.fmean(v), ms[int(0.025 * iters)], ms[int(0.975 * iters) - 1]


def call(gw, it, model):
    p = REPO / it["image_path"]
    mime = mimetypes.guess_type(p.name)[0] or "image/png"
    body = json.dumps({"image": f"data:{mime};base64,{base64.b64encode(p.read_bytes()).decode()}",
                       "target": it["target"], "gemini_model": model}).encode()
    for attempt in range(3):
        try:
            req = urllib.request.Request(gw.rstrip("/") + "/api/locate", data=body, headers={"Content-Type": "application/json"})
            t0 = time.time()
            with urllib.request.urlopen(req, timeout=300) as r:
                res = json.load(r)
            res["_client_ms"] = (time.time() - t0) * 1000
            return res
        except Exception as e:  # noqa: BLE001
            err = str(e)
            time.sleep(2 + 3 * attempt)
    return {"error": err}


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--gateway", default="http://localhost:8099")
    ap.add_argument("--dataset", default="benchmarks/bbox_real_aspects.jsonl")
    ap.add_argument("--gemini-model", default="gemini-3.8-flash")
    ap.add_argument("-w", "--workers", type=int, default=8)
    ap.add_argument("-o", "--output", default="")
    a = ap.parse_args()
    items = [json.loads(l) for l in open(REPO / a.dataset) if l.strip()]
    with ThreadPoolExecutor(a.workers) as ex:
        res = list(ex.map(lambda it: call(a.gateway, it, a.gemini_model), items))
    rows, ious, lat, errs = [], [], [], 0
    skipped = pos_skipped = 0
    for it, r in zip(items, res):
        pos = it["aspects"]["present"] == "yes"
        rows.append({"id": it["id"], "positive": pos, **{k: r.get(k) for k in ("path", "box_pct", "total_ms", "error")},
                     "dgem_ms": (r.get("dgem") or {}).get("ms"), "gemini_ms": (r.get("gemini") or {}).get("ms")})
        if r.get("error"):
            errs += 1
            continue
        if r["path"] == "skipped_absent":
            skipped += 1
            pos_skipped += pos
        if pos:
            ious.append(iou(it["gt_box_continuous"], r["box_pct"]) if r.get("box_pct") else 0.0)
            lat.append(r["total_ms"])
    m = boot(ious)
    summary = {"items": len(items), "errors": errs, "positives": len(ious), "miou": m, "e2e_p50_ms": statistics.median(lat),
               "skipped": skipped, "positives_skipped": pos_skipped, "gemini_model": a.gemini_model,
               "exp24_reference": {"hint_low_miou": [0.654, 0.611, 0.696], "e2e_p50_ms": 2111, "skip_saved": "152/432",
                                   "positives_skipped": "3/228"}}
    print(json.dumps(summary, indent=1))
    if a.output:
        json.dump({"summary": summary, "rows": rows}, open(a.output, "w"), indent=1)


if __name__ == "__main__":
    main()
