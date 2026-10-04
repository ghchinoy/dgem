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

"""Run one engine over an EXP-25 data set and write per-section predictions.

  python3 -m scripts.docs_eval.run --run-dir benchmarks/runs/<id> --set dev_injected --engine dgem --target <URL>
  python3 -m scripts.docs_eval.run --run-dir ... --set dev_injected --engine dgem --target <URL> --order shuffled
  python3 -m scripts.docs_eval.run --run-dir ... --set dev_injected --engine dgem --target <URL> --mode isolated
  python3 -m scripts.docs_eval.run --run-dir ... --set dev_injected --engine gemini --project <PROJECT>
  python3 -m scripts.docs_eval.run --run-dir ... --set dev_injected --engine docstats

Sets: label_set (the frozen test; refuses to run without --frozen-ok), dev_injected, dev_pairs (both sides), dev_pool.
The target URL is never written to the run directory (only its kind).
"""
import argparse
import concurrent.futures as cf
import datetime
import json
import os
import subprocess

from . import engines, rubric
from .build_sets import OUT


def load_set(name):
    path = os.path.join(OUT, f"{name}.jsonl")
    rows = [json.loads(l) for l in open(path)]
    if name == "dev_pairs":
        return [s for r in rows for s in (r["before"], r["after"])]
    return rows


def _engine(a):
    if a.engine == "dgem":
        return engines.Dgem(a.target, mode=a.mode, order=a.order)
    if a.engine == "gemini":
        return engines.Gemini(a.project, a.model, a.thinking)
    return engines.Docstats()


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--run-dir", required=True)
    ap.add_argument("--set", required=True, choices=["label_set", "dev_injected", "dev_pairs", "dev_pool"])
    ap.add_argument("--engine", required=True, choices=["dgem", "gemini", "docstats"])
    ap.add_argument("--target", default=os.environ.get("DGEM_DOCS_TARGET"))
    ap.add_argument("--mode", default="joint", choices=["joint", "isolated"])
    ap.add_argument("--order", default="original", choices=["original", "shuffled"])
    ap.add_argument("--repeat", type=int, default=1, help="run index (identical repeats measure the noise floor)")
    ap.add_argument("--project", default=os.environ.get("GOOGLE_CLOUD_PROJECT"))
    ap.add_argument("--model", default="gemini-3.8-flash")
    ap.add_argument("--thinking", default=None, help="Gemini thinkingLevel (e.g. LOW); default: model default")
    ap.add_argument("--policies", default=",".join(rubric.POLICIES))
    ap.add_argument("--workers", type=int, default=8)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--frozen-ok", action="store_true", help="required for label_set: the frozen test is scored once")
    a = ap.parse_args()
    if a.set == "label_set" and not a.frozen_ok:
        raise SystemExit("label_set is the frozen test set; pass --frozen-ok only for the single pre-registered run")
    if a.engine == "dgem" and not a.target:
        raise SystemExit("--target (or DGEM_DOCS_TARGET) is required for dgem")
    rows = load_set(a.set)[: a.limit or None]
    eng = _engine(a)
    pols = a.policies.split(",")
    tag = eng.name.replace(":", "-") + (f"__{a.mode}__{a.order}" if a.engine == "dgem" else "") + f"__r{a.repeat}"
    os.makedirs(a.run_dir, exist_ok=True)
    out = os.path.join(a.run_dir, f"preds__{a.set}__{tag}.jsonl")

    def one(s):
        rec = {"set": a.set, "id": s["id"], "engine": eng.name, "mode": a.mode if a.engine == "dgem" else None,
               "order": a.order if a.engine == "dgem" else None, "repeat": a.repeat, "answers": {}, "meta": {}}
        for p in pols:
            ans, meta = eng.run(p, s)
            rec["answers"].update(ans)
            rec["meta"][p] = meta
        return rec

    done = 0
    with open(out, "w") as f, cf.ThreadPoolExecutor(a.workers if a.engine != "docstats" else 1) as ex:
        for rec in ex.map(one, rows):
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")
            done += 1
            if done % 25 == 0:
                print(f"{tag}: {done}/{len(rows)}", flush=True)
    man_path = os.path.join(a.run_dir, "manifest.json")
    man = json.load(open(man_path)) if os.path.exists(man_path) else {"experiment": "EXP-25", "runs": []}
    man.setdefault("runs", []).append({
        "file": os.path.basename(out), "set": a.set, "engine": eng.name, "mode": a.mode, "order": a.order,
        "repeat": a.repeat, "n": len(rows), "policies": pols, "rubric_hash": rubric.rubric_hash(),
        "target_kind": engines._targets.Target("t", a.target).kind if a.engine == "dgem" else None,
        "model": a.model if a.engine == "gemini" else None, "thinking": a.thinking if a.engine == "gemini" else None,
        "git_sha": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=rubric.REPO, text=True).strip(),
        "utc": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")})
    json.dump(man, open(man_path, "w"), indent=2)
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
