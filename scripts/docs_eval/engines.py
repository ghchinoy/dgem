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

"""Engines that answer the EXP-25 rubric for one section.

Every engine returns {qid: {"answer": label, "probs": {label: p} or None, "ms": float}} plus request metadata.

  dgem     /v1/systemone on a serving target (scripts/matrix/targets.py auth). mode "joint" reads all questions of
           a policy in one request; "isolated" sends one request per question. order "shuffled" permutes choice
           options per section with a seeded RNG (the option-order check).
  gemini   Vertex AI generateContent (global location), JSON response schema with one enum per question.
  docstats deterministic regex rules from github.com/ghchinoy/docstats (optional; DOCSTATS_DIR or ../docstats;
           DOCSTATS_PYTHON names an interpreter with pydantic, otherwise `uv run` in the docstats checkout).
"""
import json
import math
import os
import random
import subprocess
import sys
import time
import urllib.request

from . import rubric

sys.path.insert(0, os.path.join(rubric.REPO, "scripts"))
from matrix import targets as _targets  # noqa: E402


def entropy(probs):
    return -sum(p * math.log(p) for p in probs.values() if p > 1e-12)


def hesitation(probs):
    k = len(probs)
    return entropy(probs) / math.log(k) if k > 1 else 0.0


def _dist(typ, labels, ans):
    if not isinstance(ans, dict):
        return None
    pr = ans.get("probabilities") or {}
    if typ == "score":
        d = {lab: float(pr.get(str(i), pr.get(lab, 0.0))) for i, lab in enumerate(labels)}
    else:
        d = {lab: float(pr.get(lab, 0.0)) for lab in labels}
    s = sum(d.values())
    if s <= 0:
        ch = ans.get("choice")
        return {k: float(k == ch) for k in labels} if ch in labels else None
    return {k: v / s for k, v in d.items()}


class Dgem:
    name = "dgem"

    def __init__(self, url, mode="joint", order="original", seed=25):
        self.target = _targets.Target("dgem", url)
        self.mode, self.order, self.seed = mode, order, seed

    def _order(self, policy, sid):
        if self.order != "shuffled":
            return None
        rng = random.Random(f"{self.seed}:{sid}:{policy}")
        out = {}
        for qid, typ, _i, labels, _d in rubric.questions(policy):
            if typ == "choice":
                lab = labels[:]
                while lab == labels:  # always a different order
                    rng.shuffle(lab)
                out[qid] = lab
        return out

    def run(self, policy, section):
        qs = rubric.questions(policy)
        order = self._order(policy, section["id"])
        groups = [[q[0] for q in qs]] if self.mode == "joint" else [[q[0]] for q in qs]
        out, meta = {}, {"requests": 0, "errors": []}
        for g in groups:
            body = rubric.systemone_body(policy, section["text"], qids=g, order=order)
            st, resp, ms, _ = self.target.systemone(body, timeout=300)
            meta["requests"] += 1
            if st != 200:
                meta["errors"].append(f"{st}: {str(resp)[:200]}")
                continue
            answers = resp.get("answers") or {}
            srv = ((resp.get("diagnostics") or {}).get("timing") or {}).get("total_ms")
            for qid, typ, _i, labels, _d in qs:
                if qid not in g:
                    continue
                d = _dist(typ, labels, answers.get(qid))
                if d is None:
                    meta["errors"].append(f"{qid}: bad answer")
                    continue
                out[qid] = {"answer": max(labels, key=lambda l: d[l]), "probs": d, "ms": round(ms, 1),
                            "server_ms": srv}
        if order:
            meta["order"] = order
        return out, meta


_TOKEN = {"t": None, "at": 0.0}


def _gcloud_token():
    if not _TOKEN["t"] or time.time() - _TOKEN["at"] > 1500:
        _TOKEN["t"] = subprocess.check_output(["gcloud", "auth", "application-default", "print-access-token"],
                                              text=True).strip()
        _TOKEN["at"] = time.time()
    return _TOKEN["t"]


def gemini_generate(project, model, prompt, schema=None, thinking=None, timeout=300):
    """Returns (parsed JSON or text, usage dict, ms)."""
    url = (f"https://aiplatform.googleapis.com/v1/projects/{project}/locations/global/publishers/google/models/"
           f"{model}:generateContent")
    cfg = {}
    if schema is not None:
        cfg = {"responseMimeType": "application/json", "responseSchema": schema}
    if thinking:
        cfg["thinkingConfig"] = {"thinkingLevel": thinking}
    body = {"contents": [{"role": "user", "parts": [{"text": prompt}]}], "generationConfig": cfg}
    err = None
    for attempt in range(5):
        t0 = time.perf_counter()
        try:
            req = urllib.request.Request(url, data=json.dumps(body).encode(), method="POST", headers={
                "Authorization": f"Bearer {_gcloud_token()}", "Content-Type": "application/json"})
            with urllib.request.urlopen(req, timeout=timeout) as r:
                d = json.load(r)
            ms = (time.perf_counter() - t0) * 1000
            parts = d["candidates"][0]["content"]["parts"]
            txt = "".join(p.get("text", "") for p in parts if not p.get("thought"))
            return (json.loads(txt) if schema is not None else txt), d.get("usageMetadata", {}), ms
        except Exception as e:  # noqa: BLE001
            err = e
            time.sleep(2 + 3 * attempt * attempt)
    raise RuntimeError(f"gemini call failed: {err}")


class Gemini:
    def __init__(self, project, model="gemini-3.8-flash", thinking=None):
        self.project, self.model, self.thinking = project, model, thinking
        self.name = f"gemini:{model}" + (f":{thinking}" if thinking else "")

    def run(self, policy, section):
        prompt, schema = rubric.gemini_request(policy, section["text"])
        meta = {"requests": 1, "errors": []}
        try:
            ans, usage, ms = gemini_generate(self.project, self.model, prompt, schema, self.thinking)
        except RuntimeError as e:
            meta["errors"].append(str(e)[:200])
            return {}, meta
        meta["usage"] = usage
        out = {}
        for qid, _t, _i, labels, _d in rubric.questions(policy):
            a = ans.get(qid)
            if a in labels:
                out[qid] = {"answer": a, "probs": None, "ms": round(ms, 1)}
            else:
                meta["errors"].append(f"{qid}: {a!r}")
        return out, meta


class Docstats:
    """Maps docstats' deterministic counts onto the style questions it can see. No answer for reader/tone/verdict."""
    name = "docstats"
    MAP = {"openers": ("throat_clearing_count", "vague_declarative_count"), "framing": ("binary_contrast_count",),
           "sentences": ("fragment_count",), "actors": ("passive_hint_count",)}

    def __init__(self, path=None, passive_min=2):
        path = path or os.environ.get("DOCSTATS_DIR") or os.path.join(rubric.REPO, "..", "docstats")
        if not os.path.exists(os.path.join(path, "ai_patterns.py")):
            raise FileNotFoundError(f"docstats not found at {path} (set DOCSTATS_DIR)")
        self.passive_min, self._detect, self._proc = passive_min, None, None
        try:
            sys.path.insert(0, os.path.abspath(path))
            import ai_patterns  # noqa: E402
            self._detect = ai_patterns.detect_ai_patterns
        except ImportError:  # docstats' own environment (pydantic etc.) via uv, one long-lived worker
            code = ("import sys,json\nfrom ai_patterns import detect_ai_patterns as d\nfor l in sys.stdin:\n"
                    " t=json.loads(l)\n print(json.dumps(d(t,max(1,len(t.split()))).model_dump()),flush=True)\n")
            py = os.environ.get("DOCSTATS_PYTHON")  # any interpreter with pydantic installed
            cmd = [py, "-c", code] if py else ["uv", "run", "--quiet", "--directory", os.path.abspath(path), "python",
                                               "-c", code]
            self._proc = subprocess.Popen(cmd, cwd=os.path.abspath(path), stdin=subprocess.PIPE,
                                          stdout=subprocess.PIPE, text=True)

    def counts(self, text):
        prose = "\n".join(l for l in text.split("\n") if not l.startswith("Page: "))
        if self._detect:
            return self._detect(prose, max(1, len(prose.split()))).model_dump()
        self._proc.stdin.write(json.dumps(prose) + "\n")
        self._proc.stdin.flush()
        return json.loads(self._proc.stdout.readline())

    def run(self, policy, section):
        if policy != "docs_style_audit":
            return {}, {"requests": 0, "errors": []}
        c = self.counts(section["text"])
        out = {}
        for qid, keys in self.MAP.items():
            labels = next(q[3] for q in rubric.questions(policy) if q[0] == qid)
            n = sum(c.get(k, 0) for k in keys)
            hit = n >= (self.passive_min if qid == "actors" else 1)
            out[qid] = {"answer": labels[1] if hit else labels[0], "probs": None, "ms": 0.0, "count": n}
        return out, {"requests": 0, "errors": [], "counts": {k: c[k] for k in c if k.endswith("_count")},
                     "ai_tell_score": c.get("ai_tell_score")}
