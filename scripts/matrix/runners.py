"""Suite runners. Each writes one receipt file and returns its path.

Receipt kinds:
  health     {"kind": "health", "target", "status", "body"}
  contract   {"kind": "contract", "target", "cases": {case: summary}}       (cases from scripts/contract_diff.py)
  dgem       the native receipt written by `dgem bench-* -o`
  systemone  {"kind": "systemone", "suite", "target", "permute", "run", "cases": [row, ...]}
  latency    {"kind": "latency", "target", "modes": {...}, "sweep": {...}}
"""
import concurrent.futures as cf
import http.client
import json
import os
import random
import subprocess
import sys
import threading
import time
import urllib.parse

from . import cases as caselib
from . import net

SCRIPTS = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
REPO = os.path.dirname(SCRIPTS)


def _dump(path, obj):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = path + ".tmp"
    with open(tmp, "w") as f:
        json.dump(obj, f, indent=1, ensure_ascii=False)
    os.replace(tmp, path)
    return path


# ---------------------------------------------------------------- health
def health(target, out):
    st, body = target.health()
    ok = st == 200 and isinstance(body, dict) and body.get("status") == "ok" and body.get("vllm_ready", True) is not False
    keep = ("status", "server", "version", "revision", "vllm_commit", "vllm_ready", "warmed", "phase")
    body = {k: body.get(k) for k in keep if k in body} if isinstance(body, dict) else {"error": str(body)[:200]}
    return _dump(out, {"kind": "health", "target": target.name, "target_kind": target.kind, "status": st, "ok": ok,
                       "body": body})


# ---------------------------------------------------------------- contract (scripts/contract_diff.py cases)
def contract(target, out):
    sys.path.insert(0, SCRIPTS)
    import contract_diff as cd  # noqa: E402
    res = {}
    with cf.ThreadPoolExecutor(6) as ex:
        futs = {name: ex.submit(cd.run_case, target, name) for name in cd.CASES}
        for name, f in futs.items():
            try:
                res[name] = f.result()
            except Exception as e:
                res[name] = {"status": None, "error": repr(e)[:200]}
            if name in cd.EXPECT:
                res[name]["expected_labels"] = cd.EXPECT[name]
                res[name]["wrong"] = res[name].get("labels") != cd.EXPECT[name]
    return _dump(out, {"kind": "contract", "target": target.name, "cases": res})


# ---------------------------------------------------------------- dgem CLI harnesses
def dgem(target, args, out, dgem_bin, workers=4, timeout_s=3600):
    """Run `dgem <args...> -u <url> [-k token] --http-retries 3 [-w N] -o out`."""
    cmd = [dgem_bin, *args, "-u", target.cli_url, "--http-retries", "3", "-o", out]
    if workers and args[0] not in ("bench-bbox",):
        cmd += ["-w", str(workers)]
    tok = net.token_for(target.base)
    if tok:
        cmd += ["-k", tok]
    env = {k: v for k, v in os.environ.items() if not k.startswith("DGEM_VERTEX") and k != "DGEM_URL"}
    os.makedirs(os.path.dirname(out), exist_ok=True)
    log = out[:-5] + ".log"
    t0 = time.time()
    with open(log, "w") as lf:
        p = subprocess.run(cmd, cwd=REPO, env=env, stdout=lf, stderr=subprocess.STDOUT, timeout=timeout_s)
    if p.returncode != 0 or not os.path.exists(out):
        raise RuntimeError(f"{' '.join(args)} on {target.name} failed (exit {p.returncode}); see {log}")
    for p in (log, out):  # tokens and endpoint URLs never land in a receipt or log
        redact_file(p, target, tok)
    return out, time.time() - t0


def redact_file(path, target, tok=None):
    with open(path, encoding="utf-8", errors="replace") as f:
        txt = f.read()
    new = txt
    if tok:
        new = new.replace(tok, "<TOKEN>")
    host = urllib.parse.urlparse(target.base).netloc
    for s in (target.base, host):
        if s:
            new = new.replace(s, f"<{target.kind}:{target.name}>")
    if new != txt:
        with open(path, "w", encoding="utf-8") as f:
            f.write(new)


# ---------------------------------------------------------------- /v1/systemone suites
def _run_case(target, case, perm):
    single = len(case["qs"]) == 1
    meta = []
    for q in case["qs"]:
        shown = (q.get("shown_order") or list(q["criteria"].keys()))[0] if q["type"] == "choice" else None
        meta.append({"id": case["id"] if single else f"{case['id']}/{q['qid']}", "case": case["id"], "qid": q["qid"],
                     "suite": case["suite"], "subset": case["subset"], "tier": case.get("tier"), "perm": perm,
                     "type": q["type"], "K": len(q["labels"]), "expected": q["expected"], "shown_first": shown})
    if any(q["type"] == "choice" and len(q["labels"]) > 26 for q in case["qs"]):
        return [{**m, "status": "n/a"} for m in meta]
    st, resp, ms, hdr = target.systemone(caselib.body(case))
    if st != 200:
        return [{**m, "status": st, "wall_ms": round(ms, 1), "error": str(resp)[:300]} for m in meta]
    answers = resp.get("answers") or {}
    timing = (resp.get("diagnostics") or {}).get("timing") or {}
    rows = []
    for q, m in zip(case["qs"], meta):
        d = caselib.distribution(q, answers.get(q["qid"]))
        if d is None:
            rows.append({**m, "status": "bad_answer", "error": json.dumps(answers.get(q["qid"]))[:300]})
            continue
        rows.append({**m, "status": 200, "wall_ms": round(ms, 1), "server_ms": timing.get("total_ms"),
                     "probabilities": d, **caselib.score(q, d)})
    return rows


def systemone(target, cases, out, workers=8, permute="none", run=1, suite=None):
    rng = random.Random(99)  # one Random(99) per suite pass, one shuffle per case
    todo = [caselib.permute(c, permute, rng) for c in cases]
    results = [None] * len(todo)
    with cf.ThreadPoolExecutor(workers) as ex:
        futs = {ex.submit(_run_case, target, c, permute): i for i, c in enumerate(todo)}
        for f in cf.as_completed(futs):
            i = futs[f]
            try:
                results[i] = f.result()
            except Exception as e:
                c = todo[i]
                results[i] = [{"id": c["id"], "suite": c["suite"], "status": "exception", "error": repr(e)[:300]}]
    rows = [r for rs in results for r in rs]
    return _dump(out, {"kind": "systemone", "suite": suite or (cases[0]["suite"] if cases else ""), "target": target.name,
                       "permute": permute, "run": run, "cases": rows})


# ---------------------------------------------------------------- latency (keep-alive, /v1/systemone)
_TRIAGE = {
    "urgent": {"type": "noul", "instructions": "Does this issue require immediate same-day escalation?"},
    "team": {"type": "choice", "instructions": "Which department owns resolution of this ticket?",
             "criteria": {"billing": "Charges, invoices, payment methods, renewals",
                          "support": "General questions, password resets, account setup",
                          "engineering": "System outages, 500 errors, bugs, API failures"}},
    "sentiment": {"type": "score", "instructions": "Customer anger or distress level",
                  "criteria": ["calm", "frustrated", "furious"]},
    "refund": {"type": "noul", "instructions": "Does the customer ask for a refund?"},
    "churn": {"type": "noul", "instructions": "Does the customer threaten to cancel?"},
}
_TICKET = {"ticket": "We were charged twice for our annual renewal and the API returns 500 on every invoice call. "
                     "Refund the duplicate today or we cancel."}


def _long_state():
    with open(os.path.join(REPO, "benchmarks", "jevbench", "jevbench_public.jsonl")) as f:
        rows = [json.loads(l) for l in f if l.strip()]
    r = max((r for r in rows if r["question"]["type"] == "choice"), key=lambda r: len(json.dumps(r["state"])))
    return r["state"], {"decision": r["question"]}


def latency_payloads():
    ls, lq = _long_state()
    q5 = dict(_TRIAGE)
    return {
        "q1": {"state": _TICKET, "questions": {"team": _TRIAGE["team"]}, "samples": 1},
        "q5": {"state": _TICKET, "questions": q5, "samples": 1},
        "q10": {"state": _TICKET, "questions": {**q5, **{k + "_b": v for k, v in q5.items()}}, "samples": 1},
        "long_q1": {"state": ls, "questions": lq, "samples": 1},
        "q5_s4": {"state": _TICKET, "questions": q5, "samples": 4},
    }


class _KeepAlive:
    def __init__(self, base):
        u = urllib.parse.urlparse(base)
        self.https, self.host, self.prefix, self.base = u.scheme == "https", u.netloc, u.path.rstrip("/"), base
        self.local = threading.local()

    def _conn(self):
        c = getattr(self.local, "c", None)
        if c is None:
            cls = http.client.HTTPSConnection if self.https else http.client.HTTPConnection
            c = self.local.c = cls(self.host, timeout=300)
        return c

    def call(self, payload):
        data = json.dumps({"model": "dgemma", "seed": 42, **payload}).encode()
        hdr = {"Content-Type": "application/json", "Connection": "keep-alive"}
        tok = net.token_for(self.base)
        if tok:
            hdr["Authorization"] = f"Bearer {tok}"
        for attempt in range(2):
            t0 = time.perf_counter()
            try:
                c = self._conn()
                c.request("POST", self.prefix + "/v1/systemone", body=data, headers=hdr)
                r = c.getresponse()
                raw = r.read()
                rec = {"status": r.status, "wall_ms": (time.perf_counter() - t0) * 1000}
                if r.status == 200:
                    t = (json.loads(raw).get("diagnostics") or {}).get("timing") or {}
                    rec["server_ms"] = t.get("total_ms")
                else:
                    rec["error"] = raw[:200].decode(errors="replace")
                return rec
            except Exception as e:
                self.local.c = None
                if attempt:
                    return {"status": None, "wall_ms": (time.perf_counter() - t0) * 1000, "error": repr(e)[:200]}


def _pct(xs, p):
    xs = sorted(x for x in xs if x is not None)
    return xs[min(len(xs) - 1, int(round(p / 100 * (len(xs) - 1))))] if xs else None


def _summ(recs, elapsed=None):
    ok = [r for r in recs if r["status"] == 200]
    w = [r["wall_ms"] for r in ok]
    s = [r.get("server_ms") for r in ok]
    out = {"n": len(recs), "ok": len(ok), "errors": len(recs) - len(ok), "wall_p50": _pct(w, 50),
           "wall_p90": _pct(w, 90), "wall_p99": _pct(w, 99), "server_p50": _pct(s, 50)}
    if elapsed:
        out["rps"] = len(ok) / elapsed
    errs = [r.get("error") for r in recs if r["status"] != 200][:3]
    if errs:
        out["sample_errors"] = errs
    return out


def latency(target, out, n=100, warmup=5, sweep_workers=(16, 32), sweep_modes=("q1", "q5"), min_requests=256):
    P = latency_payloads()
    ka = _KeepAlive(target.base)
    res = {"kind": "latency", "target": target.name, "modes": {}, "sweep": {}}
    for m, payload in P.items():
        for _ in range(warmup):
            ka.call(payload)
        res["modes"][m] = _summ([ka.call(payload) for _ in range(n)])
    for m in sweep_modes:
        for w in sweep_workers:
            k = max(min_requests, w * 8)
            for _ in range(min(w, 8)):
                ka.call(P[m])
            t0 = time.perf_counter()
            with cf.ThreadPoolExecutor(w) as ex:
                recs = list(ex.map(lambda _: ka.call(P[m]), range(k)))
            res["sweep"][f"{m}/w{w}"] = _summ(recs, time.perf_counter() - t0)
    return _dump(out, res)
