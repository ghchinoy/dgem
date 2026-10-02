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

"""Collect regression-matrix run summaries into one time series (trends.json + trends.md).

Reads every `summary.json` under a Cloud Storage prefix (where the scheduled T0/T1 jobs upload their runs) or a local
directory of run directories, and writes:

  trends.json  {"generated", "source", "runs": [...], "series": {"<tier>/<suite>/<target>": [{run_id, finished,
               version, accuracy, coverage, macro_f1, ece10, verdict}, ...]}, "gates": {"<tier>/<gate>/<target>":
               [{run_id, finished, version, verdict}, ...]}}
  trends.md    latest verdict per gate and the last N points per suite, for a quick read

Works with summary schema v1 (verdicts only) and v2 (per-suite metrics; `dgem.matrix.summary/v2`).

  scripts/matrix_trends.py --source gs://<bucket> --out trends/            # scheduled runs (T0/, T1/ prefixes)
  scripts/matrix_trends.py --source benchmarks/runs --out /tmp/trends/     # committed runs
  scripts/matrix_trends.py --source gs://<bucket> --tier T0 --since 2026-10-01 --out trends/

Stdlib only; Cloud Storage access uses the same token logic as the matrix (scripts/matrix/net.py).
"""
import argparse
import datetime as dt
import glob
import json
import os
import sys
import urllib.parse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from matrix import net  # noqa: E402

METRICS = ("accuracy", "coverage", "macro_f1", "ece10", "brier", "auroc")


def _gcs_token():
    return os.environ.get("DGEM_MATRIX_TOKEN") or net._mint("access", None)


def _gcs_list(bucket, prefix, tok):
    import urllib.request
    names, page = [], None
    while True:
        q = {"prefix": prefix, "fields": "items(name),nextPageToken"}
        if page:
            q["pageToken"] = page
        url = f"https://storage.googleapis.com/storage/v1/b/{bucket}/o?" + urllib.parse.urlencode(q)
        req = urllib.request.Request(url, headers={"Authorization": f"Bearer {tok}"})
        with urllib.request.urlopen(req, timeout=60) as r:
            d = json.load(r)
        names += [i["name"] for i in d.get("items", [])]
        page = d.get("nextPageToken")
        if not page:
            return names


def _gcs_get(bucket, name, tok):
    import urllib.request
    url = f"https://storage.googleapis.com/storage/v1/b/{bucket}/o/{urllib.parse.quote(name, safe='')}?alt=media"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {tok}"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.load(r)


def load_summaries(source):
    """-> list of summary dicts (each with a '_path')."""
    out = []
    if source.startswith("gs://"):
        bucket, _, prefix = source[5:].partition("/")
        tok = _gcs_token()
        for name in _gcs_list(bucket, prefix, tok):
            if name.endswith("/summary.json"):
                try:
                    s = _gcs_get(bucket, name, tok)
                except Exception as e:  # one unreadable object must not stop the trend
                    print(f"skip {name}: {e}", file=sys.stderr)
                    continue
                s["_path"] = f"gs://{bucket}/{name}"
                out.append(s)
    else:
        for p in sorted(glob.glob(os.path.join(source, "**", "summary.json"), recursive=True)):
            with open(p) as f:
                s = json.load(f)
            s["_path"] = p
            out.append(s)
    return out


def _when(s):
    """Sortable ISO-8601 time of a run. v1 summaries carry no timestamp: derive it from the run id (YYYYMMDD and an
    optional -HHMM, e.g. 20261001-scheduled-t0-2249), else from the object path; unparseable ids sort first."""
    t = s.get("finished") or s.get("started") or s.get("created")
    if t:
        return t
    import re
    for text in (s.get("run_id") or "", s.get("_path") or ""):
        m = re.search(r"(20\d{6})(?:\D.*?(?<!\d)(\d{4})(?!\d))?", text)
        if m:
            d, hm = m.group(1), m.group(2) or "0000"
            return f"{d[:4]}-{d[4:6]}-{d[6:]}T{hm[:2]}:{hm[2:]}:00+00:00"
    return "0000-00-00T00:00:00+00:00"


def build(summaries, tier=None, since=None):
    runs, series, gates = [], {}, {}
    for s in sorted(summaries, key=_when):
        t = s.get("tier")
        if tier and t != tier:
            continue
        when = _when(s)
        if since and when[:10] < since:
            continue
        versions = {x["name"]: (x.get("health") or {}).get("version") for x in s.get("targets", [])}
        runs.append({"run_id": s.get("run_id"), "tier": t, "finished": when, "schema": s.get("schema", "v1"),
                     "overall": s.get("overall"), "noise_floor": s.get("noise_floor"), "versions": versions})
        for g in s.get("gates", []):
            gates.setdefault(f"{t}/{g['gate']}/{g['target']}", []).append(
                {"run_id": s.get("run_id"), "finished": when, "version": versions.get(g["target"]),
                 "verdict": g["verdict"], "detail": g.get("detail")})
        verdict = {(g["gate"], g["target"]): g["verdict"] for g in s.get("gates", [])}
        for suite, by_target in (s.get("suites") or {}).items():
            for target, blk in by_target.items():
                pt = {"run_id": s.get("run_id"), "finished": when, "version": versions.get(target),
                      "verdict": verdict.get((suite, target))}
                pt.update({m: blk.get(m) for m in METRICS})
                pt["runs"] = len(blk.get("runs") or [])
                series.setdefault(f"{t}/{suite}/{target}", []).append(pt)
    return {"generated": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"), "runs": runs,
            "series": series, "gates": gates}


def _f(x, d=3):
    return "—" if x is None else (f"{x:.{d}f}" if isinstance(x, float) else str(x))


def markdown(tr, last=8):
    L = [f"# Regression matrix trends ({len(tr['runs'])} runs, generated {tr['generated']})", "",
         "## Latest verdict per gate", "", "| tier/gate/target | latest | since | history (oldest → newest) |",
         "|---|---|---|---|"]
    for k, pts in sorted(tr["gates"].items()):
        hist = " ".join({"PASS": "✓", "INFO": "·", "REVIEW": "R", "FAIL": "✗"}.get(p["verdict"], "?") for p in pts[-20:])
        L.append(f"| {k} | **{pts[-1]['verdict']}** | {pts[0]['finished'][:10]} | {hist} |")
    L += ["", f"## Suite metrics (last {last} runs)", ""]
    for k, pts in sorted(tr["series"].items()):
        L += [f"### {k}", "", "| finished | version | accuracy | coverage | macro-F1 | ECE10 | verdict |", "|---|---|---|---|---|---|---|"]
        for p in pts[-last:]:
            L.append(f"| {p['finished'][:16]} | {p['version'] or '—'} | {_f(p['accuracy'])} | {_f(p['coverage'])} | "
                     f"{_f(p['macro_f1'])} | {_f(p['ece10'])} | {p['verdict'] or '—'} |")
        L.append("")
    return "\n".join(L) + "\n"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--source", required=True, help="gs://bucket[/prefix] or a local directory of run directories")
    ap.add_argument("--out", required=True, help="output directory for trends.json and trends.md")
    ap.add_argument("--tier", choices=["T0", "T1", "T2"])
    ap.add_argument("--since", help="YYYY-MM-DD")
    a = ap.parse_args()
    tr = build(load_summaries(a.source), a.tier, a.since)
    tr["source"] = "gcs" if a.source.startswith("gs://") else os.path.basename(os.path.normpath(a.source))
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "trends.json"), "w") as f:
        json.dump(tr, f, indent=1)
    with open(os.path.join(a.out, "trends.md"), "w") as f:
        f.write(markdown(tr))
    print(f"{len(tr['runs'])} runs, {len(tr['series'])} suite series, {len(tr['gates'])} gate series -> {a.out}",
          file=sys.stderr)


if __name__ == "__main__":
    main()
