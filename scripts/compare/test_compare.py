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
"""Offline tests for the model-comparison tools (no network beyond a local stub server):
python3 -m unittest scripts/compare/test_compare.py"""
import http.server
import json
import os
import random
import sys
import tempfile
import threading
import time
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..")))
from compare import exposure  # noqa: E402
from compare import preflight  # noqa: E402
from compare import report as CR  # noqa: E402
from matrix import cases as C  # noqa: E402
from matrix import compare_cases as CC  # noqa: E402
from matrix import report as R  # noqa: E402
from matrix.targets import CompetitorTarget, load_profile  # noqa: E402

MATRIX_V2 = os.path.join(HERE, "..", "..", "benchmarks", "matrix", "matrix_v2.json")


class TestCases(unittest.TestCase):
    def test_sizes_and_bodies(self):
        self.assertEqual(len(C.SUITES["calib_systemone"]()), 50)
        self.assertEqual(len(C.SUITES["intents_systemone"]()), 60)
        gate = C.SUITES["gate_mixed_noul"]()
        self.assertEqual((len(gate), sum(len(c["qs"]) for c in gate)), (51, 102))
        self.assertEqual(len(C.SUITES["gate_mixed_noul_single"]()), 102)
        self.assertEqual(sum(c["subset"] == "A" for c in gate), 7)
        for c in C.SUITES["calib_systemone"]():
            q = c["qs"][0]
            self.assertIn(q["expected"], q["labels"], c["id"])
            b = C.body(c)["questions"]["decision"]
            if q["type"] == "score":
                self.assertIsInstance(b["criteria"], list)
        self.assertTrue(any(len(c["qs"][0]["labels"]) > 26 for c in C.SUITES["intents_systemone"]()))

    def test_coupling(self):
        rows = [{"status": 200, "case": "gate-00", "qid": "args_grounded", "actual": "yes"},
                {"status": 200, "case": "gate-00", "qid": "premature", "actual": "yes"},
                {"status": 200, "case": "gate-01", "qid": "args_grounded", "actual": "yes"},
                {"status": 200, "case": "gate-01", "qid": "premature", "actual": "no"}]
        self.assertEqual(CC.coupling(rows)["equal"], 1)


class TestCompetitorTarget(unittest.TestCase):
    def test_profile_and_body(self):
        t = CompetitorTarget("s", "http://127.0.0.1:1#profile=strands-decider-2b")
        self.assertEqual((t.role, t.max_options, t.model), ("competitor", 255, "strands-decider-2B-hobson-v19"))
        self.assertEqual(t.redacted()["profile"], "strands-decider-2b")
        with self.assertRaises(SystemExit):
            CompetitorTarget("s", "http://x#layout=document_first")

    def test_profiles_parse(self):
        d = os.path.join(HERE, "..", "..", "benchmarks", "competitors")
        for fn in os.listdir(d):
            if fn.endswith(".json"):
                p = load_profile(fn[:-5])
                self.assertEqual(p["schema"], "dgem.competitor/v1")
                for m in p.get("exposure", {}).values():
                    for k, v in m.items():
                        if not k.startswith("_"):
                            self.assertIn(v, exposure.TAGS, fn)

    def test_exposure(self):
        p = load_profile("strands-decider-2b")
        self.assertEqual(exposure.tag(p, {"suite": "calib_systemone", "subset": "boolq-passage-answer-check"}), "train")
        self.assertEqual(exposure.tag(p, {"suite": "calib_systemone", "subset": "prompt-injection-check"}), "unseen")
        self.assertEqual(exposure.tag(p, {"suite": "massive", "subset": "en", "id": "massive-en-001"}), "calib")
        self.assertEqual(exposure.tag(p, {"suite": "nope"}), "unknown")


def _rows(suite, n, acc, seed):
    rng = random.Random(seed)
    out = []
    for i in range(n):
        ok = rng.random() < acc
        p = 0.95 if ok else 0.7
        out.append({"id": f"{suite}-{i}", "case": f"{suite}-{i}", "qid": "decision", "suite": suite, "subset": "x",
                    "status": 200, "expected": "a", "actual": "a" if ok else "b", "accurate": ok, "confidence": p,
                    "probabilities": {"a": p if ok else 1 - p, "b": 1 - p if ok else p}, "wall_ms": 50.0})
    return out


class TestReports(unittest.TestCase):
    def _run(self, tmp):
        man = {"run_id": "t", "git_commit": "0" * 40, "receipts": [],
               "matrix": {"matrix_version": "v2", "tier": "TC", "baseline": "dgem",
                          "targets": [{"name": "dgem", "kind": "vertex", "role": "dgem"},
                                      {"name": "strands", "kind": "cloudrun", "role": "competitor",
                                       "profile": "strands-decider-2b"}]}}
        for t, acc in (("dgem", 0.85), ("strands", 0.7)):
            for r in (1, 2, 3):
                p = f"jev_systemone__{t}__r{r}.json"
                with open(os.path.join(tmp, p), "w") as f:
                    json.dump({"kind": "systemone", "cases": _rows("jev_systemone", 231, acc, r)}, f)
                man["receipts"].append({"path": p, "suite": "jev_systemone", "config": t, "run": r, "perm": None})
        with open(os.path.join(tmp, "manifest.json"), "w") as f:
            json.dump(man, f)
        with open(MATRIX_V2) as f:
            md, summ = R.build(tmp, json.load(f))
        with open(os.path.join(tmp, "summary.json"), "w") as f:
            json.dump(summ, f)
        return md, summ

    def test_competitor_never_gated(self):
        with tempfile.TemporaryDirectory() as tmp:
            md, summ = self._run(tmp)
            self.assertNotIn("strands", summ["overall"])
            self.assertFalse([g for g in summ["gates"] if g["target"] == "strands"])
            self.assertIn("## Competitors", md)
            self.assertIn("jev_systemone", summ["competitors"]["strands"])
            cmp = CR.build(tmp)
            self.assertIn("## 7. Findings for dgem", cmp)
            self.assertIn("By training exposure of strands", cmp)


class _Stub(http.server.BaseHTTPRequestHandler):
    """A /v1/systemone server; with `interfere`, overlapping requests corrupt answers (the bug preflight catches)."""
    interfere = False
    active = 0
    lock = threading.Lock()

    def log_message(self, *a):
        pass

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with _Stub.lock:
            _Stub.active += 1
            busy = _Stub.active > 1
        time.sleep(0.01)
        ans = {}
        for qid, q in body["questions"].items():
            p = 0.8 if not (self.interfere and busy) else 0.2
            if q["type"] == "noul":
                ans[qid] = {"type": "noul", "noul": p}
            elif q["type"] == "choice":
                ks = list(q["criteria"])
                ans[qid] = {"type": "choice", "choice": ks[0], "probabilities": {k: (p if i == 0 else (1 - p) / (len(ks) - 1)) for i, k in enumerate(ks)}}
            else:
                n = len(q["criteria"])
                ans[qid] = {"type": "score", "probabilities": {str(i): (p if i == 0 else (1 - p) / (n - 1)) for i in range(n)}}
        with _Stub.lock:
            _Stub.active -= 1
        raw = json.dumps({"answers": ans}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


class TestPreflight(unittest.TestCase):
    def _check(self, interfere):
        _Stub.interfere = interfere
        srv = http.server.ThreadingHTTPServer(("127.0.0.1", 0), _Stub)
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        try:
            t = CompetitorTarget("stub", f"http://127.0.0.1:{srv.server_address[1]}")
            cases = preflight.items(16)
            s1, s2, cc = preflight.run(t, cases, 1), preflight.run(t, cases, 1), preflight.run(t, cases, 8)
            return preflight.compare(s1, s2), preflight.compare(s1, cc)
        finally:
            srv.shutdown()
            srv.server_close()

    def test_safe_server(self):
        (_, ch0, mx0), (_, ch1, mx1) = self._check(False)
        self.assertEqual((ch0, mx0, ch1, mx1), (0, 0.0, 0, 0.0))

    def test_interfering_server(self):
        (_, ch0, _), (_, ch1, mx1) = self._check(True)
        self.assertEqual(ch0, 0)
        self.assertGreater(mx1, 0.1)


if __name__ == "__main__":
    unittest.main()
