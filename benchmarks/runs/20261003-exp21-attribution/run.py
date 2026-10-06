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

"""EXP-21 arms via local adapters (dgem systemone serve --prompt-layout X) in front of one upstream.
usage: run.py <upstream invoke base> <out.jsonl>"""
import sys, json, subprocess, time, socket, os, concurrent.futures as cf
sys.path.insert(0, "scripts")
from matrix import net
UP, OUT = sys.argv[1].rstrip("/"), sys.argv[2]
DGEM = "bin/dgem"
def adapter(layout):
    s = socket.socket(); s.bind(("127.0.0.1", 0)); port = s.getsockname()[1]; s.close()
    tok = net.token_for(UP)
    env = {k: v for k, v in os.environ.items() if not k.startswith("DGEM_")}
    p = subprocess.Popen([DGEM, "systemone", "serve", "--host", "127.0.0.1", "--port", str(port), "--upstream", UP + "/v1",
                          "--temperature", "1.0", "--http-retries", "3", "--prompt-layout", layout] + (["-k", tok] if tok else []),
                         env=env, stdout=open(f"./adapter_{layout}.log", "w"), stderr=subprocess.STDOUT)
    for _ in range(60):
        st, *_ = net.request(f"http://127.0.0.1:{port}/health", timeout=5, retries=0)
        if st == 200: return p, f"http://127.0.0.1:{port}"
        time.sleep(0.5)
    raise SystemExit("adapter failed")
A = {l: adapter(l) for l in ("schema_first", "document_first")}
fc = json.load(open("./forecast_dev.json")); ph = json.load(open("./phish_dev.json"))
def strip(qs):
    return {k: ({kk: vv for kk, vv in q.items() if kk != "criteria"} if q["type"] == "noul" else q) for k, q in qs.items()}
jobs = [("forecast", it, l, "sent") for it in fc for l in A] + [("phish", it, l, c) for it in ph for l in A for c in ("sent", "dropped")]
def go(j):
    fam, it, l, c = j
    qs = it["questions"] if c == "sent" else strip(it["questions"])
    st, b, ms, _ = net.request(A[l][1] + "/v1/systemone", {"state": it["state"], "questions": qs}, timeout=600, retries=2)
    return {"family": fam, "id": it["id"], "layout": l, "criteria": c, "status": st, "ms": round(ms),
            "answers": (b or {}).get("answers") if st == 200 else None, "error": None if st == 200 else str(b)[:200],
            "gold": it.get("outcome", it.get("label"))}
with cf.ThreadPoolExecutor(4) as ex, open(OUT, "w") as f:
    for r in ex.map(go, jobs): f.write(json.dumps(r) + "\n")
for p, _ in A.values(): p.terminate()
