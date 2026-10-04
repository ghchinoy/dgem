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

"""Build the EXP-25 data sets from this repository's docs.

  python3 -m scripts.docs_eval.build_sets label-set            # frozen test set to label (150 + 15 repeats)
  python3 -m scripts.docs_eval.build_sets inject --project P   # dev: clean sections + one injected problem each
  python3 -m scripts.docs_eval.build_sets pairs --range A..B[,C..D]  # dev: sections before/after docs rewrites

The label set and the dev sets never share a section (dev seeds are drawn from what the label set leaves).
"""
import argparse
import collections
import glob
import json
import os
import random
import subprocess

from . import rubric, sections

OUT = os.path.join(rubric.REPO, "benchmarks", "docs_eval")
SEED = 2025
N_TEST, N_REPEAT = 150, 15
PER_PAGE = 5
# Share of the test set per docs area (sums to 1). Experiments are capped: they are research logs, not product docs.
STRATA = {"deploy": 0.17, "operate": 0.17, "policies": 0.22, "reference": 0.14, "confidence": 0.10, "root": 0.10,
          "experiments": 0.10}
EXCLUDE = ("docs/history/", "docs/confidence-beyond-shannon.md")


def area(page):
    parts = page.split("/")
    return parts[1] if len(parts) > 2 else "root"


def all_sections(rev=None):
    """Every reviewable section of docs/ at the working tree (rev=None) or at a git revision."""
    if rev is None:
        files = sorted(f for f in glob.glob(os.path.join(rubric.REPO, "docs", "**", "*.md"), recursive=True))
        pages = [(os.path.relpath(f, rubric.REPO), open(f).read()) for f in files]
    else:
        names = subprocess.check_output(["git", "ls-tree", "-r", "--name-only", rev, "docs/"], cwd=rubric.REPO,
                                        text=True).split()
        pages = [(n, subprocess.check_output(["git", "show", f"{rev}:{n}"], cwd=rubric.REPO, text=True))
                 for n in names if n.endswith(".md")]
    out = []
    for page, md in pages:
        if page.startswith(EXCLUDE):
            continue
        out += [s for s in sections.split_page(page, md) if s["prose_words"] >= sections.MIN_PROSE_WORDS]
    return out


def _write_jsonl(path, rows):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        for r in rows:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")


def _git_sha():
    return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=rubric.REPO, text=True).strip()


def label_set(args):
    secs = all_sections()
    rng = random.Random(SEED)
    by = collections.defaultdict(list)
    for s in secs:
        by[area(s["page"])].append(s)
    picked, per_page = [], collections.Counter()
    for a, share in STRATA.items():
        pool = by.get(a, [])
        rng.shuffle(pool)
        # at most PER_PAGE sections per page so long pages don't dominate
        take = []
        for s in pool:
            if len(take) < round(N_TEST * share) and per_page[s["page"]] < PER_PAGE:
                take.append(s)
                per_page[s["page"]] += 1
        picked += take
    # strata that ran short (few pages) are filled from the other product-docs areas, not from experiments
    ids = {s["id"] for s in picked}
    rest = [s for s in secs if s["id"] not in ids and area(s["page"]) != "experiments"]
    rng.shuffle(rest)
    for s in rest:
        if len(picked) >= N_TEST:
            break
        if per_page[s["page"]] < PER_PAGE:
            picked.append(s)
            per_page[s["page"]] += 1
    picked = picked[:N_TEST]
    repeats = rng.sample(picked, N_REPEAT)
    items = [dict(s) for s in picked] + [dict(s, id=s["id"] + "r") for s in repeats]
    rng.shuffle(items)
    rows = [{"id": s["id"], "page": s["page"], "heading": s["heading"], "text": s["text"]} for s in items]
    _write_jsonl(os.path.join(OUT, "label_set.jsonl"), rows)
    manifest = {"git_sha": _git_sha(), "seed": SEED, "n_test": len(picked), "n_repeat": N_REPEAT,
                "strata": dict(collections.Counter(area(s["page"]) for s in picked)),
                "repeat_ids": {s["id"] + "r": s["id"] for s in repeats},
                "rubric_hash": rubric.rubric_hash(), "sections_total": len(secs)}
    json.dump(manifest, open(os.path.join(OUT, "label_set.manifest.json"), "w"), indent=2)
    used = {s["id"] for s in picked}
    rest = [s for s in secs if s["id"] not in used]
    _write_jsonl(os.path.join(OUT, "dev_pool.jsonl"), rest)
    print(f"label set: {len(rows)} rows ({len(picked)} + {N_REPEAT} repeats) from {len(secs)} sections; "
          f"dev pool {len(rest)}; strata {manifest['strata']}")


INJECT = {
    "openers": "Add one or two throat-clearing openers or significance flourishes, such as 'It's worth noting that', "
               "'Here's the thing:', 'This matters because' or 'That distinction matters.'",
    "framing": "Rewrite one or two claims as dramatic contrast frames, such as 'This isn't X, it's Y' or "
               "'X isn't the problem, Y is.'",
    "actors": "Rewrite several sentences so the actor is hidden: use passive voice ('was changed', 'is handled') or "
              "false agency ('the data tells us', 'the logs are loud about').",
    "sentences": "Add two or three short sentence fragments for effect, such as 'That's it.' or 'Fast. Simple.'",
    "reader": "Add hand-holding meta-commentary, such as 'Don't worry,', 'As you can see,', 'Simply' or "
              "'In this section we will'.",
    "tone": "Add promotional hype: superlatives and words like 'blazing fast', 'seamless', 'revolutionary', or a claim "
            "of superiority with no measurement.",
}


def inject(args):
    from .engines import gemini_generate, Docstats
    pool = [json.loads(l) for l in open(os.path.join(OUT, "dev_pool.jsonl"))]
    try:
        ds = Docstats()
        clean = [s for s in pool if ds.counts(s["text"])["ai_tell_score"] >= 8.0]
    except FileNotFoundError:
        clean = pool
    rng = random.Random(SEED + 1)
    rng.shuffle(clean)
    seeds = clean[:args.n]
    import concurrent.futures as cf

    def variant(job):
        s, prob = job
        prompt = ("Edit this section of technical documentation. " + INJECT[prob] + " Change nothing else: keep every "
                  "fact, number, heading, link, code block and table exactly as it is. Return only the edited "
                  "section.\n\n<<<\n" + s["text"] + "\n>>>")
        txt, _u, _ms = gemini_generate(args.project, args.model, prompt)
        txt = txt.strip().removeprefix("<<<").removesuffix(">>>").strip()
        return {**s, "id": f"{s['id']}-{prob}", "text": txt, "source": s["id"], "variant": prob,
                "labels": {"problems": [prob]}}

    rows = [{**s, "source": s["id"], "variant": "original", "labels": {"problems": []}} for s in seeds]
    with cf.ThreadPoolExecutor(8) as ex:
        rows += list(ex.map(variant, [(s, p) for s in seeds for p in INJECT]))
    _write_jsonl(os.path.join(OUT, "dev_injected.jsonl"), rows)
    print(f"dev_injected: {len(rows)} rows from {len(seeds)} seeds")


def pairs(args):
    test_ids = {json.loads(l)["id"] for l in open(os.path.join(OUT, "label_set.jsonl"))}
    test_keys = {sections.section_key(s) for s in all_sections() if s["id"] in test_ids}
    rows, seen = [], set()
    for rng_ in args.range.split(","):
        a, b = rng_.split("..")
        rows += _pairs(a, b, test_ids, test_keys, seen)
    _write_jsonl(os.path.join(OUT, "dev_pairs.jsonl"), rows)
    print(f"dev_pairs: {len(rows)} pairs from {args.range}")


def _pairs(a, b, test_ids, test_keys, seen):
    import difflib
    before = {sections.section_key(s): s for s in all_sections(a)}
    after = {sections.section_key(s): s for s in all_sections(b)}
    rows = []
    for k, s1 in before.items():
        s2 = after.get(k)
        if not s2 or s1["text"] == s2["text"] or s2["id"] in test_ids or k in test_keys or k in seen:
            continue
        ratio = difflib.SequenceMatcher(None, s1["text"], s2["text"]).ratio()
        if 0.2 <= ratio <= 0.9:
            seen.add(k)
            rows.append({"id": s2["id"] + "-pair", "page": s2["page"], "heading": s2["heading"], "ratio": round(ratio, 3),
                         "range": f"{a}..{b}",
                         "before": {**s1, "id": s1["id"] + "-before"}, "after": {**s2, "id": s2["id"] + "-after"}})
    return rows


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("label-set")
    p = sub.add_parser("inject")
    p.add_argument("--project", default=os.environ.get("GOOGLE_CLOUD_PROJECT"))
    p.add_argument("--model", default="gemini-3.8-flash")
    p.add_argument("-n", type=int, default=25)
    p = sub.add_parser("pairs")
    p.add_argument("--range", required=True)
    a = ap.parse_args()
    {"label-set": label_set, "inject": inject, "pairs": pairs}[a.cmd](a)


if __name__ == "__main__":
    main()
