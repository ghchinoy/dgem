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

"""Compare bench-bbox receipts side by side (PROP-18). Standard library only.

  compare   per-variant mIoU, Acc@0.5, transform consistency and presence for each receipt
  tags      per-tag-value mIoU for each receipt (generated sweeps), with paired dgem-minus-other deltas
  occlusion dose-response on the sweep's occlusion set: per coverage, IoU against the full and the
            visible box, the occluded edge's entropy, and the share of placements whose occluded-edge
            entropy rises monotonically with coverage
  judge     summaries of bench-bbox-judge receipts
  cascade   bench-vision: accuracy when the most hesitant share of dgem answers per aspect is replaced by the
            Gemini reference's answer for the same item (offline simulation; repeat 0)

  python3 scripts/analyze_bbox_receipts.py compare dgem=<receipt> gemini38=<receipt> ...
  python3 scripts/analyze_bbox_receipts.py tags dgem=<receipt> gemini38=<receipt>
  python3 scripts/analyze_bbox_receipts.py occlusion dgem=<receipt> gemini38=<receipt>
  python3 scripts/analyze_bbox_receipts.py judge <judge receipt> ...
  python3 scripts/analyze_bbox_receipts.py cascade <dgem bench-vision receipt> <gemini bench-vision receipt>
"""

import json
import random
import statistics
import sys
from collections import defaultdict


def load(arg):
    name, _, path = arg.partition("=")
    if not path:
        name, path = arg, arg
    with open(path) as f:
        return name, json.load(f)


def ok(c):
    return not c.get("error") and not c.get("skipped")


def boot_ci(vals, seed=1, iters=2000):
    if not vals:
        return (float("nan"),) * 3
    m = statistics.fmean(vals)
    if len(vals) == 1:
        return m, m, m
    rng = random.Random(seed)
    ms = sorted(statistics.fmean(rng.choice(vals) for _ in vals) for _ in range(iters))
    return m, ms[int(0.025 * iters)], ms[int(0.975 * iters) - 1]


def fmt_ci(t):
    return f"{t[0]:.3f} [{t[1]:.3f}, {t[2]:.3f}]"


def per_case_iou(rec, variant="original", key="expectation_iou", filt=None):
    by = defaultdict(list)
    for c in rec["cases"]:
        if ok(c) and (c.get("variant") or "original") == variant and c["object_present_gt"] and (filt is None or filt(c)):
            v = c.get(key)
            if v is not None:
                by[c["id"]].append(v)
    return {k: statistics.fmean(v) for k, v in by.items()}


def cmd_compare(args):
    recs = [load(a) for a in args]
    print("| receipt | model | variant | box cases | mIoU expect [95% CI] | Acc@0.5 | consistency vs original | answered present |")
    print("| :--- | :--- | :--- | ---: | :---: | ---: | :---: | ---: |")
    for name, r in recs:
        for v in r.get("variants", []):
            cons = v.get("consistency_iou_vs_transformed_original")
            pr = v.get("predicted_present_rate")
            print(f"| {name} | {r.get('target_model')} | {v['variant']} | {v['box_cases']} | "
                  f"{v['mean_expectation_iou']['mean']:.3f} [{v['mean_expectation_iou']['ci95_lo']:.3f}, {v['mean_expectation_iou']['ci95_hi']:.3f}] | "
                  f"{v['acc_at_50_expectation_pct']:.1f}% | "
                  f"{'—' if not cons else format(cons['mean'], '.3f')} | "
                  f"{'—' if pr is None else f'{100 * pr:.0f}%'} |")
    if len(recs) >= 2:
        (n0, r0), rest = recs[0], recs[1:]
        a = per_case_iou(r0)
        for n1, r1 in rest:
            b = per_case_iou(r1)
            common = sorted(set(a) & set(b))
            d = [a[k] - b[k] for k in common]
            print(f"\nPaired original-variant mIoU, {n0} − {n1} over {len(common)} cases: {fmt_ci(boot_ci(d))}")


def cmd_tags(args):
    recs = [load(a) for a in args]
    tags = defaultdict(set)
    for _, r in recs:
        for c in r["cases"]:
            for k, v in (c.get("tags") or {}).items():
                if k != "placement":
                    tags[k].add(v)
    names = [n for n, _ in recs]
    print("| tag | value | " + " | ".join(f"{n} mIoU" for n in names) +
          (f" | {names[0]} − {names[1]} (paired)" if len(recs) > 1 else "") + " | " +
          " | ".join(f"{n} present" for n in names) + " |")
    print("| :--- | :--- |" + " :---: |" * len(names) + (" :---: |" if len(recs) > 1 else "") + " ---: |" * len(names))
    for t in sorted(tags):
        for v in sorted(tags[t], key=lambda x: (not x.lstrip("-").isdigit(), int(x) if x.lstrip("-").isdigit() else 0, x)):
            filt = lambda c, t=t, v=v: (c.get("tags") or {}).get(t) == v
            ious = [per_case_iou(r, filt=filt) for _, r in recs]
            cells = [fmt_ci(boot_ci(list(x.values()))) if x else "—" for x in ious]
            delta = ""
            if len(recs) > 1:
                common = sorted(set(ious[0]) & set(ious[1]))
                delta = " | " + (fmt_ci(boot_ci([ious[0][k] - ious[1][k] for k in common])) if common else "—")
            pres = []
            for _, r in recs:
                cs = [c for c in r["cases"] if ok(c) and (c.get("variant") or "original") == "original"
                      and filt(c) and c.get("presence_asked")]
                pres.append(f"{100 * sum(c['object_present_pred'] for c in cs) / len(cs):.0f}%" if cs else "—")
            print(f"| {t} | {v} | " + " | ".join(cells) + delta + " | " + " | ".join(pres) + " |")


def cmd_occlusion(args):
    for name, r in (load(a) for a in args):
        rows = [c for c in r["cases"] if ok(c) and (c.get("variant") or "original") == "original"
                and (c.get("tags") or {}).get("set") == "occlusion"]
        if not rows:
            print(f"{name}: no occlusion-set cases")
            continue
        print(f"\n{name} ({r.get('target_model')})")
        print("| coverage | n | IoU vs full box | IoU vs visible box | occluded-edge H~ | occluded-edge error vs full box (pts) |")
        print("| ---: | ---: | :---: | :---: | :---: | ---: |")
        by_cov = defaultdict(list)
        for c in rows:
            by_cov[int(c["tags"]["coverage"])].append(c)
        edge_idx = {"ymin": 0, "xmin": 1, "ymax": 2, "xmax": 3}
        for cov in sorted(by_cov):
            cs = by_cov[cov]
            full = [c["expectation_iou"] for c in cs]
            vis = [c["visible_iou"] for c in cs if c.get("visible_iou") is not None]
            hs, ef = [], []
            for c in cs:
                e = c["tags"]["edge"]
                tel = (c.get("edges") or {}).get(e)
                if tel and not tel.get("missing"):
                    hs.append(tel.get("normalized_entropy", 0.0))
                    ef.append(abs(tel["expected_coord"] - c["gt_box"][edge_idx[e]]))
            print(f"| {cov}% | {len(cs)} | {fmt_ci(boot_ci(full))} | {fmt_ci(boot_ci(vis)) if vis else '—'} | "
                  f"{statistics.fmean(hs):.3f} | {statistics.fmean(ef):.1f} |" if hs else
                  f"| {cov}% | {len(cs)} | {fmt_ci(boot_ci(full))} | {fmt_ci(boot_ci(vis)) if vis else '—'} | — | — |")
        # Per placement: does the occluded edge's entropy rise with coverage (mean over repeats)?
        place = defaultdict(lambda: defaultdict(list))
        for c in rows:
            e = c["tags"]["edge"]
            tel = (c.get("edges") or {}).get(e)
            if tel and not tel.get("missing"):
                place[c["tags"]["placement"]][int(c["tags"]["coverage"])].append(tel.get("normalized_entropy", 0.0))
        rising = up75 = n = 0
        for p, d in place.items():
            if len(d) < 4:
                continue
            seq = [statistics.fmean(d[k]) for k in sorted(d)]
            n += 1
            rising += all(b >= a for a, b in zip(seq, seq[1:]))
            up75 += seq[-1] > seq[0]
        if n:
            print(f"Placements with occluded-edge entropy non-decreasing in coverage: {rising}/{n}; higher at 75% than 0%: {up75}/{n}")


def cmd_judge(args):
    for path in args:
        with open(path) as f:
            r = json.load(f)
        print(f"\n{path}: judge {r['judge_model']} (tol {r['tol']}, off {r['off']}); receipt boxes from {r.get('receipt_target_model') or '—'}")
        print("| source | items | errors | edge acc (3-class) | κ | wrong-edge recall | specificity | binary κ | direction right | box agree (IoU≥0.75) | box κ | judge pass | actual pass |")
        print("| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
        for s in r["summaries"]:
            print(f"| {s['source']} | {s['items']} | {s['errors']} | {s['edge_accuracy_3class']:.3f} | {s['edge_kappa_3class']:.3f} | "
                  f"{s['edge_recall_wrong']:.3f} | {s['edge_specificity']:.3f} | {s['edge_binary_kappa']:.3f} | {s['direction_accuracy_on_flagged_wrong']:.3f} | "
                  f"{s['box_agreement_iou75']:.3f} | {s['box_kappa_iou75']:.3f} | {s['judge_pass_rate']:.3f} | {s['actual_pass_rate_iou75']:.3f} |")
            if s.get("judged_correct_rate_by_shift"):
                ks = sorted(s["judged_correct_rate_by_shift"], key=lambda k: -1 if k == "exact" else float(k))
                print("  judged correct by shift: " + ", ".join(f"{k}: {s['judged_correct_rate_by_shift'][k]:.2f}" for k in ks))


def cmd_cascade(args):
    with open(args[0]) as f:
        d = json.load(f)
    with open(args[1]) as f:
        g = json.load(f)
    gm = {(c["id"], a["aspect"]): a["correct"] for c in g["cases"] if not c.get("error") for a in c["answers"]}
    rows = defaultdict(list)
    for c in d["cases"]:
        if c.get("repeat", 0) != 0 or c.get("error"):
            continue
        for a in c["answers"]:
            k = (c["id"], a["aspect"])
            if k in gm:
                rows[a["aspect"]].append((a.get("h_norm", 0.0), a["correct"], gm[k]))
    shares = (0, 0.1, 0.2, 0.3, 0.5, 1)
    print("| aspect | n | " + " | ".join("dgem only" if f == 0 else "Gemini only" if f == 1 else f"{int(f * 100)}% escalated" for f in shares) + " |")
    print("| :--- | ---: |" + " ---: |" * len(shares))
    for asp, rs in sorted(rows.items()):
        rs.sort(key=lambda x: -x[0])
        n = len(rs)
        cells = []
        for f in shares:
            k = round(f * n)
            cells.append(f"{(sum(r[2] for r in rs[:k]) + sum(r[1] for r in rs[k:])) / n:.3f}")
        print(f"| {asp} | {n} | " + " | ".join(cells) + " |")


def main():
    cmds = {"compare": cmd_compare, "tags": cmd_tags, "occlusion": cmd_occlusion, "judge": cmd_judge, "cascade": cmd_cascade}
    if len(sys.argv) < 3 or sys.argv[1] not in cmds:
        print(__doc__)
        sys.exit(2)
    cmds[sys.argv[1]](sys.argv[2:])


if __name__ == "__main__":
    main()
