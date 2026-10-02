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

"""Per-item agreement between groups of benchmark receipts (bench-jev / bench-calibration).

Compares answers item by item. Within-group pairs (same image, repeated runs) give the noise floor;
cross-group pairs (old vs new image) should look the same if the change is neutral.

  python3 scripts/receipt_agreement.py --group old=a_r1.json,a_r2.json --group new=b_r1.json,b_r2.json
"""
import argparse
import itertools
import json
import os
import statistics
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from matrix import metrics as M  # noqa: E402


def load(path):
    """Per-item answers from any receipt the regression matrix understands (bench-jev, bench-calibration,
    bench-intents, matrix /v1/systemone receipts), via scripts/matrix/metrics.rows."""
    with open(path) as f:
        d = json.load(f)
    out = {}
    for r in M.rows(d):
        out[r["id"]] = {"actual": r["actual"], "ok": r["accurate"], "conf": r.get("confidence"),
                        "wall": r.get("wall_ms"), "brier": r.get("brier")}
    return out


def pair(a, b):
    ids = sorted(set(a) & set(b))
    same = sum(a[i]["actual"] == b[i]["actual"] for i in ids)
    dconf = [abs(a[i]["conf"] - b[i]["conf"]) for i in ids if a[i]["conf"] is not None and b[i]["conf"] is not None]
    return len(ids), same, (statistics.mean(dconf) if dconf else float("nan"))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--group", action="append", required=True, help="name=r1.json,r2.json,...")
    a = ap.parse_args()
    groups = {}
    for g in a.group:
        n, files = g.split("=", 1)
        groups[n] = [(f, load(f)) for f in files.split(",")]
    print(f"{'group':14s} {'runs':>4s} {'correct per run':>22s} {'mean Brier':>10s} {'wall p50 ms':>11s}")
    for n, runs in groups.items():
        corr = [sum(v["ok"] for v in r.values()) for _, r in runs]
        brier = [statistics.mean(v["brier"] for v in r.values() if v["brier"] is not None) for _, r in runs]
        wall = [statistics.median(v["wall"] for v in r.values() if v["wall"]) for _, r in runs]
        print(f"{n:14s} {len(runs):4d} {str(corr):>22s} {statistics.mean(brier):10.4f} {statistics.mean(wall):11.0f}")
    print(f"\n{'pair type':30s} {'pairs':>5s} {'answer agreement':>17s} {'mean |Δconf|':>12s}")
    names = list(groups)
    for n in names:
        rs = [pair(x[1], y[1]) for x, y in itertools.combinations(groups[n], 2)]
        if rs:
            print(f"{'within ' + n:30s} {len(rs):5d} {statistics.mean(s / t for t, s, _ in rs):17.1%} {statistics.mean(d for *_, d in rs):12.4f}")
    for n1, n2 in itertools.combinations(names, 2):
        rs = [pair(x[1], y[1]) for x in groups[n1] for y in groups[n2]]
        print(f"{'cross ' + n1 + ' vs ' + n2:30s} {len(rs):5d} {statistics.mean(s / t for t, s, _ in rs):17.1%} {statistics.mean(d for *_, d in rs):12.4f}")


if __name__ == "__main__":
    main()
