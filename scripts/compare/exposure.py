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
"""Training-exposure tags for comparison rows, from a competitor profile (benchmarks/competitors/<name>.json).

Tags: train (same dataset in the competitor's training mix), calib (used to fit its temperatures or select it),
near (a closely related task is trained), unseen (no known relation), unknown (not audited).
A profile's "exposure" maps suite -> {"*": default tag, "<subset>": tag}. dgem is zero-shot on every suite.
"""
import statistics

TAGS = ("train", "calib", "near", "unseen", "unknown")


def tag(profile, row):
    m = (profile or {}).get("exposure", {}).get(row.get("suite"), {})
    sub = row.get("subset") or ""
    if sub in m:
        return m[sub]
    if row.get("suite") in ("massive", "massive_spot") and "-" in str(row.get("id", "")):
        lang = str(row["id"]).split("-")[1]
        if lang in m:
            return m[lang]
    return m.get("*", "unknown")


def split(profile, rows_by_target, competitor):
    """rows_by_target: {target: [rows]} (first run, all suites pooled). Tags come from the competitor's profile and
    are applied to every target's rows, so each line compares the same items. -> [(tag, n, {target: accuracy})]."""
    out = []
    for t in TAGS:
        per = {}
        ids = None
        for name, rows in rows_by_target.items():
            sel = {(r["suite"], r["id"]): r for r in rows if tag(profile, r) == t}
            ids = set(sel) if ids is None else ids & set(sel)
            per[name] = sel
        if not ids:
            continue
        out.append((t, len(ids), {n: statistics.mean(1.0 if per[n][k]["accurate"] else 0.0 for k in ids) for n in per}))
    return out
