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

"""Evaluate the EXP-26 pre-registered decision rule (docs/experiments/exp-26-domain-images.md) on a run directory.

  python3 scripts/exp26_decide.py benchmarks/runs/<run_id>            # markdown tables + verdicts
  python3 scripts/exp26_decide.py benchmarks/runs/<run_id> --json out.json

Standard library only. Reads guided__gemini_3_8_flash.json, guided__gemini_3_7_flash.json and vision__*.json.
"""

import argparse
import json
import math
import random
import statistics
from collections import defaultdict
from pathlib import Path

REF_MODEL = "gemini_3_8_flash"
MODELS = ["gemini_3_8_flash", "gemini_3_7_flash"]
SKIP_H = 0.16


def iou(a, b):
    ih = max(0.0, min(a[2], b[2]) - max(a[0], b[0]))
    iw = max(0.0, min(a[3], b[3]) - max(a[1], b[1]))
    inter = ih * iw
    u = (a[2] - a[0]) * (a[3] - a[1]) + (b[2] - b[0]) * (b[3] - b[1]) - inter
    return inter / u if u > 0 else 0.0


def boot(vals, seed=26, iters=2000):
    if not vals:
        return (float("nan"),) * 3
    m = statistics.fmean(vals)
    rng = random.Random(seed)
    ms = sorted(statistics.fmean(rng.choice(vals) for _ in vals) for _ in range(iters))
    return m, ms[int(0.025 * iters)], ms[int(0.975 * iters) - 1]


def med(v):
    return statistics.median(v) if v else float("nan")


def ci(t):
    return f"{t[0]:.3f} [{t[1]:.3f}, {t[2]:.3f}]"


def auroc(pos, neg):
    if not pos or not neg:
        return float("nan")
    w = sum((p > q) + 0.5 * (p == q) for p in pos for q in neg)
    return w / (len(pos) * len(neg))


def call(it, cond):
    return next((c for c in it["calls"] if c["condition"] == cond and not c.get("error")), None)


def box_iou(it, cond):
    c = call(it, cond)
    if c is None:
        return None
    return iou(it["gt_box"], c["box_pct"]) if c.get("box_pct") else 0.0


def e2e(it, cond):
    c = call(it, cond)
    if c is None:
        return None
    return c["gemini_ms"] + (it["dgem"]["ms"] if c["strategy"] in ("hint", "crop") else 0.0)


def guided(run):
    out = {}
    recs = {m: json.load(open(run / f"guided__{m}.json")) for m in MODELS if (run / f"guided__{m}.json").exists()}
    ref = {it["id"]: it for it in recs[REF_MODEL]["items"]}
    domains = sorted({it["tags"]["domain"] for it in recs[REF_MODEL]["items"]})
    for d in domains:
        res = {"boxes": {}, "R1": {}, "R2": {}}
        for m, rec in recs.items():
            items = [it for it in rec["items"] if it["tags"]["domain"] == d]
            pos = [it for it in items if it["positive"]]
            for cond in ("full@default", "full@low", "hint@default", "hint@low"):
                v = [x for x in (box_iou(it, cond) for it in pos) if x is not None]
                lat = [x for x in (e2e(it, cond) for it in pos) if x is not None]
                res["boxes"][f"{m}:{cond}"] = {"miou": boot(v), "e2e_p50": med(lat), "n": len(v)}
            # R1: hint@low (model m) vs full@default (reference model), paired by item.
            diffs, lat_h = [], []
            for it in pos:
                a, b = box_iou(it, "hint@low"), box_iou(ref[it["id"]], "full@default")
                if a is not None and b is not None:
                    diffs.append(a - b)
                x = e2e(it, "hint@low")
                if x is not None:
                    lat_h.append(x)
            lat_ref = [x for x in (e2e(ref[it["id"]], "full@default") for it in pos) if x is not None]
            dci = boot(diffs)
            r1 = (dci[1] >= -0.03) and (med(lat_h) < med(lat_ref))
            res["R1"][m] = {"diff": dci, "lat_hint_low": med(lat_h), "lat_ref": med(lat_ref), "pass": r1}
            # R2: skip on this model's full@default (the call that skip saves).
            n = saved = wrong = 0
            for it in items:
                dg = it["dgem"]
                if dg.get("error") or call(it, "full@default") is None:
                    continue
                n += 1
                if dg["present"] == "no" and dg["present_h_norm"] < SKIP_H:
                    saved += 1
                    wrong += it["positive"]
            npos = sum(1 for it in items if it["positive"])
            r2 = n > 0 and saved / n >= 0.15 and wrong / max(1, npos) <= 0.02
            res["R2"][m] = {"saved": saved, "n": n, "wrong_pos": wrong, "npos": npos, "pass": r2}
        out[d] = res
    return out


def vision(run):
    dg = json.load(open(run / "vision__vertex_g4_x2.json"))
    refs = {m: json.load(open(run / f"vision__reference_{m}.json")) for m in MODELS
            if (run / f"vision__reference_{m}.json").exists()}
    out = defaultdict(dict)

    def per(rec, d, a):
        by = defaultdict(list)
        for c in rec["cases"]:
            if c.get("error") or c["tags"].get("domain") != d:
                continue
            for x in c["answers"]:
                if x["aspect"] == a:
                    by[c["id"]].append(x)
        return by

    domains = sorted({c["tags"]["domain"] for c in dg["cases"]})
    aspects = sorted({x["aspect"] for c in dg["cases"] if not c.get("error") for x in c["answers"] or []})
    for d in domains:
        for a in aspects:
            by = per(dg, d, a)
            if not by:
                continue
            ids = sorted(by)
            acc = [statistics.fmean(1.0 if x["correct"] else 0.0 for x in by[i]) for i in ids]
            exp = {i: by[i][0]["expected"] for i in ids}
            counts = defaultdict(int)
            for v in exp.values():
                counts[v] += 1
            maj = max(sorted(counts), key=lambda k: counts[k])
            lift = [acc[k] - (1.0 if exp[i] == maj else 0.0) for k, i in enumerate(ids)]
            pos_h = [x.get("h_norm", 0) for i in ids for x in by[i] if not x["correct"]]
            neg_h = [x.get("h_norm", 0) for i in ids for x in by[i] if x["correct"]]
            accm, lci, au = statistics.fmean(acc), boot(lift), auroc(pos_h, neg_h)
            if accm >= 0.85 and lci[1] > 0 and au >= 0.70:
                verdict = "supported"
            elif lci[1] > 0 and au >= 0.70:
                verdict = "cascade"
            else:
                verdict = "not supported"
            gem = {}
            for m, rec in refs.items():
                gb = per(rec, d, a)
                gem[m] = statistics.fmean(1.0 if gb[i][0]["correct"] else 0.0 for i in ids if gb.get(i)) if gb else float("nan")
            out[d][a] = {"n": len(ids), "acc": accm, "majority": counts[maj] / len(ids), "lift": lci, "auroc": au,
                         "wrong": len(pos_h), "verdict": verdict, "gemini": gem}
    lat = {"dgem_p50_ms": med([c["wall_time_ms"] for c in dg["cases"] if not c.get("error")])}
    for m, rec in refs.items():
        lat[f"{m}_p50_ms"] = med([c["wall_time_ms"] for c in rec["cases"] if not c.get("error")])
    return out, lat


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("run")
    ap.add_argument("--json", default="")
    a = ap.parse_args()
    run = Path(a.run)
    g = guided(run)
    print("## Guided boxes (positives), mIoU [95% CI] and end-to-end p50\n")
    print("| domain | model | full@default | full@low | hint@default | hint@low |")
    print("| :--- | :--- | :---: | :---: | :---: | :---: |")
    for d, r in g.items():
        for m in MODELS:
            cells = []
            for cond in ("full@default", "full@low", "hint@default", "hint@low"):
                b = r["boxes"].get(f"{m}:{cond}")
                cells.append(f"{ci(b['miou'])} · {b['e2e_p50'] / 1000:.1f} s" if b else "—")
            print(f"| {d} | {m} | " + " | ".join(cells) + " |")
    print("\n## R1 (hint@low vs 3.8 full@default) and R2 (skip)\n")
    print("| domain | model | Δ mIoU [95% CI] | p50 hint@low / ref | R1 | skip saved | positives skipped | R2 | R4 (3.7 ok) |")
    print("| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |")
    for d, r in g.items():
        for m in MODELS:
            r1, r2 = r["R1"].get(m), r["R2"].get(m)
            if not r1:
                continue
            r4 = ("yes" if r1["pass"] else "no") if m == "gemini_3_7_flash" else "—"
            print(f"| {d} | {m} | {ci(r1['diff'])} | {r1['lat_hint_low'] / 1000:.1f} / {r1['lat_ref'] / 1000:.1f} s | "
                  f"{'PASS' if r1['pass'] else 'FAIL'} | {r2['saved']}/{r2['n']} ({r2['saved'] / max(1, r2['n']):.0%}) | "
                  f"{r2['wrong_pos']}/{r2['npos']} | {'PASS' if r2['pass'] else 'FAIL'} | {r4} |")
    v, lat = vision(run)
    print("\n## R3 categorical questions (dgem ×2)\n")
    print("| domain | aspect | n | dgem acc | majority | lift [95% CI] | H AUROC (wrong) | verdict | Gemini 3.8 | Gemini 3.7 |")
    print("| :--- | :--- | ---: | :---: | :---: | :---: | :---: | :--- | :---: | :---: |")
    for d, asp in v.items():
        for k, x in asp.items():
            print(f"| {d} | {k} | {x['n']} | {x['acc']:.3f} | {x['majority']:.3f} | {ci(x['lift'])} | {x['auroc']:.2f} ({x['wrong']}) | "
                  f"**{x['verdict']}** | {x['gemini'].get('gemini_3_8_flash', float('nan')):.3f} | {x['gemini'].get('gemini_3_7_flash', float('nan')):.3f} |")
    print("\nLatency p50 (bench-vision, all questions): " + ", ".join(f"{k} {val:.0f}" for k, val in lat.items()))
    if a.json:
        def clean(o):
            if isinstance(o, float) and math.isnan(o):
                return None
            if isinstance(o, dict):
                return {k: clean(x) for k, x in o.items()}
            if isinstance(o, (list, tuple)):
                return [clean(x) for x in o]
            return o
        json.dump(clean({"guided": g, "vision": v, "vision_latency": lat}), open(a.json, "w"), indent=1)


if __name__ == "__main__":
    main()
