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
"""Preflight for a model comparison: is the target safe and repeatable under concurrent requests?

Sends the same items three times (serial, serial again, concurrent) and compares the returned distributions:

  serial vs serial      the server's own run-to-run noise (0.000 for a deterministic server)
  serial vs concurrent  must not be larger than the serial noise; if it is, concurrent requests interfere
                        (e.g. a thread pool sharing one model) and the comparison must run that target serialized

  scripts/compare/preflight.py --competitor strands=<url>#profile=strands-decider-2b
  scripts/compare/preflight.py --target dgem=<url> -n 60 --workers 8

Exit code 1 when concurrency changes answers beyond the serial noise. Run it before any scored comparison
(docs/operate/model-comparison.md, step 3).
"""
import argparse
import concurrent.futures as cf
import json
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
from matrix import cases as caselib  # noqa: E402
from matrix.targets import CompetitorTarget, Target  # noqa: E402


def items(n):
    """A mix of question types and lengths: JevBench (noul/choice/score, short to ~3k tokens) + the yes/no pairs."""
    jev = caselib.SUITES["jev_systemone"]()
    gate = caselib.SUITES["gate_mixed_noul"]()
    step = max(1, len(jev) // max(1, n - n // 4))
    return jev[::step][: n - n // 4] + gate[: n // 4]


def ask(target, case):
    st, resp, ms, _ = target.systemone(caselib.body(case))
    if st != 200 or not isinstance(resp, dict):
        return {"status": st}
    out = {}
    for q in case["qs"]:
        d = caselib.distribution(q, (resp.get("answers") or {}).get(q["qid"]))
        if d is not None:
            out[q["qid"]] = d
    return {"status": 200, "dist": out}


def run(target, cases, workers):
    with cf.ThreadPoolExecutor(workers) as ex:
        return list(ex.map(lambda c: ask(target, c), cases))


def compare(a, b):
    """-> (pairs compared, answers changed, max probability change)."""
    n = changed = 0
    mx = 0.0
    for x, y in zip(a, b):
        if x.get("status") != 200 or y.get("status") != 200:
            continue
        for qid, dx in x["dist"].items():
            dy = y["dist"].get(qid)
            if dy is None:
                continue
            n += 1
            changed += max(dx, key=dx.get) != max(dy, key=dy.get)
            mx = max(mx, max(abs(dx[k] - dy.get(k, 0.0)) for k in dx))
    return n, changed, mx


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--target", help="name=url of a dgem target")
    g.add_argument("--competitor", help="name=url[#profile=...] of another decision model")
    ap.add_argument("-n", type=int, default=40, help="items (default 40)")
    ap.add_argument("--workers", type=int, default=8, help="concurrency of the third pass (default 8)")
    ap.add_argument("-o", "--output", help="write the result as JSON")
    a = ap.parse_args()
    t = Target(*a.target.split("=", 1)) if a.target else CompetitorTarget(*a.competitor.split("=", 1))
    cases = items(a.n)
    s1, s2, cc = run(t, cases, 1), run(t, cases, 1), run(t, cases, a.workers)
    errors = sum(r.get("status") != 200 for r in s1 + s2 + cc)
    n0, ch0, mx0 = compare(s1, s2)
    n1, ch1, mx1 = compare(s1, cc)
    unsafe = ch1 > ch0 + max(1, round(0.02 * n1)) or (mx0 == 0.0 and mx1 > 1e-3)
    res = {"target": t.name, "role": t.role, "items": len(cases), "decisions": n0, "errors": errors,
           "serial_vs_serial": {"changed": ch0, "max_prob_change": round(mx0, 4)},
           "serial_vs_concurrent": {"workers": a.workers, "changed": ch1, "max_prob_change": round(mx1, 4)},
           "deterministic": mx0 == 0.0, "concurrency_safe": not unsafe}
    print(json.dumps(res, indent=1))
    if a.output:
        with open(a.output, "w") as f:
            json.dump(res, f, indent=1)
    if unsafe:
        print(f"\nUNSAFE: concurrent requests changed {ch1} answers (max {mx1:.3f}) against {ch0} between serial runs. "
              "Run this target serialized (--competitor-workers 1, or a lock in front of the model).", file=sys.stderr)
    return 1 if unsafe or errors else 0


if __name__ == "__main__":
    sys.exit(main())
