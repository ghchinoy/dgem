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

"""Score EXP-25 predictions and write report.md + summary.json into the run directory.

  python3 -m scripts.docs_eval.report --run-dir benchmarks/runs/<id> --set dev_injected
  python3 -m scripts.docs_eval.report --run-dir benchmarks/runs/<id> --set label_set --labels benchmarks/docs_eval/labels.jsonl
  python3 -m scripts.docs_eval.report --run-dir benchmarks/runs/<id> --set dev_pairs
  python3 -m scripts.docs_eval.report thresholds --run-dir benchmarks/runs/<id> [--write]

Gold per set:
  dev_injected  style problems only: an original is negative on all six problems, a variant is positive on its
                injected problem and negative on the other five (the seeds were filtered for docstats-clean prose).
  label_set     the human labels; the 15 repeat items are excluded from scoring and used for self-agreement.
  dev_pairs     no per-question gold: the share of pairs whose rewritten ("after") section gets the better verdict.
"""
import argparse
import collections
import glob
import json
import math
import os
import random

from . import rubric
from .build_sets import OUT
from .engines import hesitation

THRESHOLDS = (0.16, 0.25, 0.35, 0.50)
PREREG_THRESHOLD = 0.35


def _load(path):
    return [json.loads(l) for l in open(path)]


def gold_for(set_name, labels_path=None):
    """{section id: {qid: gold label}}"""
    qs = {q[1]: q for q in rubric.all_questions()}
    gold = {}
    if set_name == "dev_injected":
        for r in _load(os.path.join(OUT, "dev_injected.jsonl")):
            gold[r["id"]] = {q: qs[q][4][1] if q in r["labels"]["problems"] else qs[q][4][0]
                             for q in rubric.PROBLEM_QUESTIONS}
    elif set_name == "label_set":
        man = json.load(open(os.path.join(OUT, "label_set.manifest.json")))
        for r in _load(labels_path):
            if r["id"] in man["repeat_ids"]:
                continue
            g = {q: qs[q][4][1] if q in r.get("problems", []) else qs[q][4][0] for q in rubric.PROBLEM_QUESTIONS}
            for k in ("doc_type", "purity", "verdict", "evidence"):
                if r.get(k):
                    g[k] = r[k]
            g["_unsure"] = bool(r.get("unsure"))
            gold[r["id"]] = g
    return gold


def auroc(scores, positives):
    pos = [s for s, y in zip(scores, positives) if y]
    neg = [s for s, y in zip(scores, positives) if not y]
    if not pos or not neg:
        return None
    wins = sum((p > n) + 0.5 * (p == n) for p in pos for n in neg)
    return wins / (len(pos) * len(neg))


def f1(pred, gold, positive):
    tp = sum(p == positive and g == positive for p, g in zip(pred, gold))
    fp = sum(p == positive and g != positive for p, g in zip(pred, gold))
    fn = sum(p != positive and g == positive for p, g in zip(pred, gold))
    return 2 * tp / (2 * tp + fp + fn) if tp + fp + fn else None


def macro_f1(pred, gold, labels):
    vals = [f1(pred, gold, l) for l in labels if any(g == l for g in gold)]
    vals = [v for v in vals if v is not None]
    return sum(vals) / len(vals) if vals else None


def kappa(a, b, labels):
    n = len(a)
    if not n:
        return None
    po = sum(x == y for x, y in zip(a, b)) / n
    pe = sum((a.count(l) / n) * (b.count(l) / n) for l in labels)
    return (po - pe) / (1 - pe) if pe < 1 else None


def boot_ci(xs, n=2000, seed=7):
    if not xs:
        return None
    rng = random.Random(seed)
    means = sorted(sum(rng.choice(xs) for _ in xs) / len(xs) for _ in range(n))
    return [round(means[int(0.025 * n)], 3), round(means[int(0.975 * n)], 3)]


def _r(x, d=3):
    return None if x is None else round(x, d)


def config_name(rec):
    if rec["engine"] == "dgem":
        return f"dgem/{rec['mode']}/{rec['order']}/r{rec['repeat']}"
    return f"{rec['engine']}/r{rec['repeat']}"


def apply_thresholds(preds, thr):
    """dgem answers with the frozen per-question p(problem) thresholds (style questions only); probs unchanged."""
    qs = {q[1]: q for q in rubric.all_questions()}
    out = {}
    for i, r in preds.items():
        ans = {}
        for qid, a in r["answers"].items():
            a = dict(a)
            if qid in thr and a.get("probs"):
                clean, prob = qs[qid][4]
                a["answer"] = prob if a["probs"][prob] >= thr[qid] else clean
            ans[qid] = a
        out[i] = {**r, "answers": ans}
    return out


def load_preds(run_dir, set_name):
    out = {}
    for p in sorted(glob.glob(os.path.join(run_dir, f"preds__{set_name}__*.jsonl"))):
        rows = _load(p)
        if rows:
            out[config_name(rows[0])] = {r["id"]: r for r in rows}
    tpath = os.path.join(OUT, "thresholds.json")
    if os.path.exists(tpath):
        t = json.load(open(tpath))
        base = out.get(t["config"])
        if base is not None and t["rubric_hash"] == rubric.rubric_hash():
            out[t["config"] + "+thr"] = apply_thresholds(base, t["thresholds"])
    return out


def score_config(preds, gold, qmeta):
    res, err_scores = {}, []
    for qid, (_pol, _q, typ, _i, labels, _d) in qmeta.items():
        ids = [i for i in gold if qid in gold[i] and qid in (preds.get(i, {}).get("answers") or {})]
        if not ids:
            continue
        pred = [preds[i]["answers"][qid]["answer"] for i in ids]
        g = [gold[i][qid] for i in ids]
        acc = sum(p == y for p, y in zip(pred, g)) / len(ids)
        row = {"n": len(ids), "accuracy": _r(acc), "accuracy_ci": boot_ci([float(p == y) for p, y in zip(pred, g)])}
        if qid in rubric.PROBLEM_QUESTIONS:
            row["positives"] = sum(y == labels[1] for y in g)
            row["f1_problem"] = _r(f1(pred, g, labels[1]))
            probs = [preds[i]["answers"][qid].get("probs") for i in ids]
            if all(probs):
                row["auroc_problem"] = _r(auroc([pr[labels[1]] for pr in probs], [y == labels[1] for y in g]))
        else:
            row["macro_f1"] = _r(macro_f1(pred, g, labels))
            row["kappa"] = _r(kappa(pred, g, labels))
        for i, p, y in zip(ids, pred, g):
            pr = preds[i]["answers"][qid].get("probs")
            if pr:
                err_scores.append((hesitation(pr), p != y))
        res[qid] = row
    if err_scores:
        res["_hesitation_error_auroc"] = {"n": len(err_scores), "errors": sum(e for _, e in err_scores),
                                          "auroc": _r(auroc([h for h, _ in err_scores], [e for _, e in err_scores]))}
    return res


def cascade(dgem, gem, gold, qmeta, tau):
    """Per question: Gemini's answer when dgem's hesitation >= tau, else dgem's."""
    hits, handed, n = [], 0, 0
    for i, g in gold.items():
        for qid in qmeta:
            if qid not in g or qid not in dgem.get(i, {}).get("answers", {}) or qid not in gem.get(i, {}).get("answers", {}):
                continue
            a = dgem[i]["answers"][qid]
            use_g = hesitation(a["probs"]) >= tau
            ans = gem[i]["answers"][qid]["answer"] if use_g else a["answer"]
            handed += use_g
            n += 1
            hits.append(float(ans == g[qid]))
    return {"tau": tau, "n": n, "accuracy": _r(sum(hits) / n) if n else None, "accuracy_ci": boot_ci(hits),
            "handed_off": _r(handed / n) if n else None}


def mean_acc(preds, gold, qmeta):
    hits = [float(preds[i]["answers"][q]["answer"] == g[q]) for i, g in gold.items() for q in qmeta
            if q in g and q in preds.get(i, {}).get("answers", {})]
    return (_r(sum(hits) / len(hits)) if hits else None), len(hits)


def flips(a, b, qmeta):
    n = f = 0
    for i in a:
        if i not in b:
            continue
        for q in qmeta:
            if q in a[i]["answers"] and q in b[i]["answers"]:
                n += 1
                f += a[i]["answers"][q]["answer"] != b[i]["answers"][q]["answer"]
    return {"n": n, "flip_rate": _r(f / n) if n else None}


def latency(preds):
    ms = sorted(r["answers"][q]["ms"] for r in preds.values() for q in r["answers"] if r["answers"][q].get("ms"))
    req = collections.defaultdict(float)
    for r in preds.values():
        seen = set()
        for q, a in r["answers"].items():
            key = (r["id"], a.get("ms"))
            if key not in seen:
                req[r["id"]] += a.get("ms") or 0.0
                seen.add(key)
    per_sec = sorted(req.values())
    p50 = lambda xs: xs[len(xs) // 2] if xs else None
    return {"per_request_p50_ms": _r(p50(ms), 1), "per_section_p50_ms": _r(p50(per_sec), 1)}


def pair_preference(preds):
    rows = _load(os.path.join(OUT, "dev_pairs.jsonl"))
    labels = next(q[3] for q in rubric.questions("docs_style_audit") if q[0] == "verdict")
    def val(rec):
        a = rec["answers"].get("verdict")
        if not a:
            return None
        if a.get("probs"):
            return sum(k * a["probs"][l] for k, l in enumerate(labels))
        return float(labels.index(a["answer"]))
    better = worse = tie = 0
    for r in rows:
        b, a = preds.get(r["before"]["id"]), preds.get(r["after"]["id"])
        if not a or not b or val(a) is None or val(b) is None:
            continue
        d = val(a) - val(b)
        better += d < -1e-6
        worse += d > 1e-6
        tie += abs(d) <= 1e-6
    n = better + worse + tie
    return {"pairs": n, "after_better": better, "after_worse": worse, "tie": tie,
            "after_better_share": _r(better / n) if n else None}


def self_agreement(labels_path):
    man = json.load(open(os.path.join(OUT, "label_set.manifest.json")))
    lab = {r["id"]: r for r in _load(labels_path)}
    out = collections.defaultdict(lambda: [0, 0])
    for rid, orig in man["repeat_ids"].items():
        a, b = lab.get(rid), lab.get(orig)
        if not a or not b:
            continue
        for k in ("doc_type", "purity", "verdict", "evidence"):
            out[k][0] += a.get(k) == b.get(k)
            out[k][1] += 1
        for q in rubric.PROBLEM_QUESTIONS:
            out[q][0] += (q in a.get("problems", [])) == (q in b.get("problems", []))
            out[q][1] += 1
    return {k: {"agree": v[0], "n": v[1], "rate": _r(v[0] / v[1]) if v[1] else None} for k, v in out.items()}


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--run-dir", required=True)
    ap.add_argument("--set", required=True, choices=["label_set", "dev_injected", "dev_pairs"])
    ap.add_argument("--labels")
    a = ap.parse_args()
    qmeta = {q[1]: q for q in rubric.all_questions()}
    preds = load_preds(a.run_dir, a.set)
    summary = {"experiment": "EXP-25", "set": a.set, "rubric_hash": rubric.rubric_hash(), "configs": {}}
    md = [f"# EXP-25 report: `{a.set}`", "", f"Rubric hash `{rubric.rubric_hash()}`. Configurations: "
          + ", ".join(f"`{c}`" for c in preds) + ".", ""]
    if a.set == "dev_pairs":
        md += ["| Configuration | Pairs | After better | After worse | Tie | Share better |", "| :--- | ---: | ---: | ---: | ---: | ---: |"]
        for c, p in preds.items():
            r = pair_preference(p)
            summary["configs"][c] = r
            md.append(f"| `{c}` | {r['pairs']} | {r['after_better']} | {r['after_worse']} | {r['tie']} | {r['after_better_share']} |")
    else:
        gold = gold_for(a.set, a.labels)
        if a.set == "label_set":
            summary["self_agreement"] = self_agreement(a.labels)
        for c, p in preds.items():
            summary["configs"][c] = {"scores": score_config(p, gold, qmeta), "latency": latency(p)}
            summary["configs"][c]["mean_accuracy"], summary["configs"][c]["answers_scored"] = mean_acc(p, gold, qmeta)
        qids = [q for q in qmeta if any(q in summary["configs"][c]["scores"] for c in preds)]
        md += ["## Accuracy per question", "", "| Question | " + " | ".join(f"`{c}`" for c in preds) + " |",
               "| :--- | " + " | ".join("---:" for _ in preds) + " |"]
        for q in qids:
            cells = []
            for c in preds:
                s = summary["configs"][c]["scores"].get(q)
                if not s:
                    cells.append("—")
                    continue
                extra = f" F1 {s['f1_problem']}" if "f1_problem" in s and s["f1_problem"] is not None else ""
                extra += f" AUROC {s['auroc_problem']}" if s.get("auroc_problem") is not None else ""
                extra += f" κ {s['kappa']}" if s.get("kappa") is not None else ""
                cells.append(f"{s['accuracy']}{extra} (n={s['n']})")
            md.append(f"| `{q}` | " + " | ".join(cells) + " |")
        md += ["", "| Configuration | Mean accuracy (answers) | Hesitation AUROC for errors | p50 per request / per section |",
               "| :--- | ---: | ---: | ---: |"]
        for c in preds:
            s = summary["configs"][c]
            h = s["scores"].get("_hesitation_error_auroc") or {}
            md.append(f"| `{c}` | {s['mean_accuracy']} ({s['answers_scored']}) | {h.get('auroc', '—')} "
                      f"({h.get('errors', 0)} errors) | {s['latency']['per_request_p50_ms']} / {s['latency']['per_section_p50_ms']} ms |")
        d0, gem = preds.get("dgem/joint/original/r1"), next((v for k, v in preds.items() if k.startswith("gemini")), None)
        dthr = preds.get("dgem/joint/original/r1+thr", d0)
        if d0 and gem:
            # hesitation is computed from the raw probabilities; the answer kept below the gate is the thresholded one
            summary["cascade"] = [cascade(dthr, gem, gold, qmeta, t) for t in THRESHOLDS]
            md += ["", f"## Hesitation-gated cascade (offline; pre-registered threshold {PREREG_THRESHOLD}; "
                   f"kept answers from `{'dgem/joint/original/r1+thr' if dthr is not d0 else 'dgem/joint/original/r1'}`)", "",
                   "| Hesitation ≥ | Accuracy | 95% CI | Handed to Gemini |", "| ---: | ---: | :--- | ---: |"]
            md += [f"| {r['tau']} | {r['accuracy']} | {r['accuracy_ci']} | {r['handed_off']} |" for r in summary["cascade"]]
        rows = []
        for other, label in (("dgem/joint/shuffled/r1", "option order shuffled"), ("dgem/joint/original/r2", "identical repeat"),
                             ("dgem/isolated/original/r1", "one question per request")):
            if d0 and other in preds:
                f = flips(d0, preds[other], qmeta)
                summary.setdefault("flips", {})[label] = f
                rows.append(f"| {label} | {f['flip_rate']} | {f['n']} |")
        if rows:
            md += ["", "## Answer changes vs `dgem/joint/original/r1`", "", "| Comparison | Flip rate | Answers |", "| :--- | ---: | ---: |"] + rows
        if "self_agreement" in summary:
            md += ["", "## Labeller self-agreement (15 repeats)", "", "| Label | Agree | n |", "| :--- | ---: | ---: |"]
            md += [f"| `{k}` | {v['rate']} | {v['n']} |" for k, v in summary["self_agreement"].items()]
    json.dump(summary, open(os.path.join(a.run_dir, f"summary__{a.set}.json"), "w"), indent=2)
    open(os.path.join(a.run_dir, f"report__{a.set}.md"), "w").write("\n".join(md) + "\n")
    print("\n".join(md))




# ---------------------------------------------------------------- per-question decision thresholds (dev only)
def fit_thresholds(preds, set_name="dev_injected", folds=5, seed=25):
    """Fit p(problem) thresholds per style question on dev_injected, maximising F1; report 5-fold CV grouped by seed
    section (an original and its six variants stay in one fold). Returns (thresholds fitted on all dev, cv rows)."""
    rows = _load(os.path.join(OUT, f"{set_name}.jsonl"))
    src = {r["id"]: r["source"] for r in rows}
    gold = gold_for(set_name)
    qs = {q[1]: q for q in rubric.all_questions()}
    sources = sorted(set(src.values()))
    random.Random(seed).shuffle(sources)
    fold_of = {s: k % folds for k, s in enumerate(sources)}
    def best_t(ids, qid, pos):
        # candidate thresholds: midpoints between observed probabilities (probabilities pile up near 0 and 1)
        ps = sorted({preds[i]["answers"][qid]["probs"][pos] for i in ids})
        grid = [(x + y) / 2 for x, y in zip(ps, ps[1:])] or [0.5]

        def f(t):
            pred = [pos if preds[i]["answers"][qid]["probs"][pos] >= t else "-" for i in ids]
            return f1(pred, [gold[i][qid] for i in ids], pos) or 0.0
        return max(grid, key=lambda t: (f(t), -abs(t - 0.5)))

    fitted, cv = {}, {}
    for qid in rubric.PROBLEM_QUESTIONS:
        pos = qs[qid][4][1]
        ids = [i for i in gold if i in preds and qid in preds[i]["answers"] and preds[i]["answers"][qid].get("probs")]
        if not ids:
            continue
        fitted[qid] = best_t(ids, qid, pos)
        pred, g = [], []
        for k in range(folds):
            tr = [i for i in ids if fold_of[src[i]] != k]
            te = [i for i in ids if fold_of[src[i]] == k]
            t = best_t(tr, qid, pos)
            pred += [pos if preds[i]["answers"][qid]["probs"][pos] >= t else "-" for i in te]
            g += [gold[i][qid] for i in te]
        cv[qid] = {"threshold_all_dev": fitted[qid], "cv_f1": _r(f1(pred, g, pos)),
                   "cv_accuracy": _r(sum((p == pos) == (y == pos) for p, y in zip(pred, g)) / len(g)), "n": len(g)}
    return fitted, cv


def thresholds_main():
    ap = argparse.ArgumentParser(description="Fit and freeze dgem per-question thresholds on dev_injected.")
    ap.add_argument("--run-dir", required=True)
    ap.add_argument("--config", default="dgem/joint/original/r1")
    ap.add_argument("--write", action="store_true", help="freeze to benchmarks/docs_eval/thresholds.json")
    a = ap.parse_args()
    preds = load_preds(a.run_dir, "dev_injected")[a.config]
    fitted, cv = fit_thresholds(preds)
    print(json.dumps(cv, indent=2))
    if a.write:
        json.dump({"config": a.config, "rubric_hash": rubric.rubric_hash(), "fitted_on": "dev_injected",
                   "thresholds": fitted, "cv": cv}, open(os.path.join(OUT, "thresholds.json"), "w"), indent=2)
        print("froze", os.path.join(OUT, "thresholds.json"))


if __name__ == "__main__":
    import sys
    if len(sys.argv) > 1 and sys.argv[1] == "thresholds":
        sys.argv.pop(1)
        thresholds_main()
    else:
        main()
