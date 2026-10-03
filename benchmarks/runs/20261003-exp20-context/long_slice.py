"""EXP-20 long-prompt slice (dev data only): JevBench long_policy items and the 60 longest RAGTruth-train items, each
padded with unrelated RAGTruth-train documents (a 'background' field placed before the original state) to roughly
+0 / +5k / +10k / +20k tokens. Labels are unchanged. Usage: long_slice.py run <out.jsonl> name=url [name=url ...]
                                                              long_slice.py load <out.json> name=url  (w8 sweep, 8k prompts)"""
import sys, json, random, time, concurrent.futures as cf
sys.path.insert(0, "scripts")
from matrix import cases as C, datasets as ds, net
from matrix.targets import Target
PAD = {"p0": 0, "p5k": 5000, "p10k": 10000, "p20k": 20000}   # extra tokens, ~4 chars/token
rows = ds.parquet(ds.RAGTRUTH, "data/train-00000-of-00001.parquet")
rng = random.Random(20261003)
used = {c["id"].split("-", 2)[-1] for c in C.rag_dev()}
pool = [r for r in rows if str(r["id"]) not in used]
pool.sort(key=lambda r: -len((r["query"] or "") + (r["context"] or "")))
rag_ids = {str(r["id"]) for r in pool[:60]}
filler_docs = [r["context"] for r in rows if r["task_type"] == "Summary" and r["context"] and str(r["id"]) not in rag_ids]
rng.shuffle(filler_docs)
def filler(tokens):
    out, n, i = [], 0, 0
    while n < tokens * 4:
        d = filler_docs[i % len(filler_docs)]; out.append(d); n += len(d); i += 1
    return "\n\n".join(out)
def base_cases():
    jev = [c for c in C.jevbench() if "long_policy" in c["id"]]
    rag_all = {c["id"]: c for c in []}
    # build RAGTruth cases the same way rag_dev does, for the 60 longest unused rows
    rag = []
    import ast
    for r in pool[:60]:
        lab = r["hallucination_labels_processed"]
        lab = lab if isinstance(lab, dict) else ast.literal_eval(lab)
        hall = (int(lab.get("evident_conflict", 0)) + int(lab.get("baseless_info", 0))) > 0
        prompt = r["query"] + "\n" + r["context"] if r["context"] else r["query"]
        q = {"qid": "q", "type": "noul", "instructions": C.RAG_INS,
             "criteria": {"true": C.RAG_INS, "false": "All content of the response is supported by the context in the prompt."},
             "labels": ["yes", "no"], "expected": "yes" if hall else "no", "values": None}
        rag.append({"id": f"rag-{r['task_type']}-{r['id']}", "suite": "long", "subset": "rag", "state": {"prompt": prompt, "response": r["output"]}, "qs": [q]})
    for c in jev: c["subset"] = "jev"
    return jev + rag
def padded(c, lvl):
    if not PAD[lvl]: return c
    bg = "Background documents (unrelated to the questions):\n\n" + filler(PAD[lvl])
    st = c["state"]
    st = {"background": bg, **st} if isinstance(st, dict) else {"background": bg, "state": st}
    return {**c, "state": st}
def score(c, resp):
    out = []
    for q in c["qs"]:
        a = (resp.get("answers") or {}).get(q["qid"])
        d = C.distribution(q, a)
        if d is None: out.append(None); continue
        out.append(max(d, key=d.get) == q["expected"])
    return out
def run(out, specs):
    targets = [Target(*s.split("=", 1)) for s in specs]
    jobs = [(t, c, lvl) for c in base_cases() for lvl in PAD for t in targets]
    def go(j):
        t, c, lvl = j; p = padded(c, lvl)
        st, resp, ms, _ = t.systemone(C.body(p), timeout=600)
        return {"target": t.name, "id": c["id"], "subset": c["subset"], "pad": lvl, "status": st, "ms": round(ms, 1),
                "chars": len(json.dumps(p["state"], ensure_ascii=False)),
                "error": None if st == 200 else str(resp)[:300], "hits": score(c, resp) if st == 200 else None}
    with cf.ThreadPoolExecutor(6) as ex, open(out, "w") as f:
        for r in ex.map(go, jobs): f.write(json.dumps(r) + "\n")
def load(out, spec):
    t = Target(*spec.split("=", 1)); cs = [padded(c, "p5k") for c in base_cases() if c["subset"] == "rag"][:64]
    t0 = time.time()
    def go(c):
        st, resp, ms, _ = t.systemone(C.body(c), timeout=600); return st, ms
    with cf.ThreadPoolExecutor(8) as ex: res = list(ex.map(go, cs))
    ok = [ms for st, ms in res if st == 200]; ok.sort()
    r = {"target": t.name, "n": len(res), "ok": len(ok), "errors": sorted({st for st, _ in res if st != 200}, key=str),
         "p50_ms": ok[len(ok)//2] if ok else None, "p90_ms": ok[int(len(ok)*.9)] if ok else None, "wall_s": round(time.time()-t0, 1)}
    json.dump(r, open(out, "w")); print(r)
if __name__ == "__main__":
    {"run": lambda: run(sys.argv[2], sys.argv[3:]), "load": lambda: load(sys.argv[2], sys.argv[3])}[sys.argv[1]]()
