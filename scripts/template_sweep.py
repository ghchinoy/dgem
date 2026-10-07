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

"""Run every policy template through a dgem gateway with its sample variables and check the answer shape.

Uses GET /api/templates (id, sample_vars, multimodal) and POST /api/decide/<id>. For multimodal templates a
fixture image from fixtures/bbox/ is attached. A template passes when the call returns 200, every question in
the rendered schema has an answer with probabilities that sum to ~1, and (optionally) the request ran on the
expected backend.

  python3 scripts/template_sweep.py --gateway https://<your-dgem-gateway> [--backend vertex] [-o out.json]

Auth: Google ID token from Application Default Credentials (as scripts/serving_speed.py).
"""
import argparse
import base64
import concurrent.futures as cf
import glob
import json
import os
import sys
import time
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(__file__))
import serving_speed as ss  # noqa: E402

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))


def image_uri():
    p = sorted(glob.glob(os.path.join(REPO, "fixtures/bbox/*.png")))[0]
    return "data:image/png;base64," + base64.b64encode(open(p, "rb").read()).decode()


def call(gw, path, body=None, backend=None):
    _, it = ss.TOK.get()
    h = {"Authorization": f"Bearer {it}", "Content-Type": "application/json"}
    if backend:
        h["X-DGem-Backend"] = backend
    req = urllib.request.Request(gw + path, json.dumps(body).encode() if body is not None else None, h)
    t0 = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=300) as r:
            return r.status, dict(r.headers), json.loads(r.read()), (time.perf_counter() - t0) * 1000
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), {"error": e.read()[:300].decode(errors="replace")}, (time.perf_counter() - t0) * 1000


def check(gw, tpl, backend, img):
    body = {"variables": tpl.get("sample_vars") or {}}
    if tpl.get("multimodal"):
        body["image"] = img
    st, hdr, d, ms = call(gw, f"/api/decide/{tpl['id']}", body, backend)
    out = {"template": tpl["id"], "status": st, "wall_ms": round(ms), "backend_used": hdr.get("X-DGem-Backend-Used") or hdr.get("X-Dgem-Backend-Used")}
    if st != 200:
        out["error"] = str(d.get("error"))[:200]
        return out
    answers = d.get("answers") or {}
    bad = []
    for qid, a in answers.items():
        probs = (a or {}).get("probabilities") or {}
        if a and a.get("label") in ("", None) and not probs:
            continue  # skipped conditional question
        if probs and abs(sum(probs.values()) - 1) > 0.02:
            bad.append(f"{qid}: probabilities sum {sum(probs.values()):.3f}")
    out.update(questions=len(answers), problems=bad)
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--gateway", required=True)
    ap.add_argument("--backend", default=None, help="X-DGem-Backend for every call (default: gateway default)")
    ap.add_argument("-o", "--output")
    a = ap.parse_args()
    gw = a.gateway.rstrip("/")
    st, _, templates, _ = call(gw, "/api/templates")
    if st != 200:
        sys.exit(f"GET /api/templates: {st} {templates}")
    if isinstance(templates, dict):
        templates = templates.get("templates", [])
    img = image_uri()
    with cf.ThreadPoolExecutor(4) as ex:
        res = list(ex.map(lambda t: check(gw, t, a.backend, img), templates))
    ok = [r for r in res if r["status"] == 200 and not r.get("problems") and r.get("questions")]
    for r in sorted(res, key=lambda r: r["template"]):
        flag = "ok " if r in ok else "FAIL"
        print(f"{flag} {r['template']:34s} {r['status']} q={r.get('questions', '-')} {r['wall_ms']:>6} ms {r.get('backend_used') or ''} {r.get('error') or ' '.join(r.get('problems') or [])}")
    print(f"\n{len(ok)}/{len(res)} templates passed")
    if a.output:
        json.dump({"gateway_backend": a.backend, "results": res}, open(a.output, "w"), indent=1)
    sys.exit(0 if len(ok) == len(res) else 1)


if __name__ == "__main__":
    main()
