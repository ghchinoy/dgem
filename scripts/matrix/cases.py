"""Cases for the /v1/systemone suites, and scoring of their answers.

case = {"id", "suite", "subset", "tier", "state", "qs": [q, ...]}
q    = {"qid", "type" (noul|choice|score), "instructions", "criteria", "labels", "expected",
        "values" (score levels), "gold_soft" (optional {label: p}), "expected_value" (optional)}

The same request body works on dgem and on any server with the Jev /v1/systemone contract. Protocols for the
public sets follow the published ones they come from (MASSIVE: 20 options = gold + 19 distractors, Random(13)
per language; option-order suites: one shuffle per case from Random(99)).
"""
import json
import math
import os
import random

from . import datasets as ds

REPO = ds.REPO
EPS = 1e-6


# ---------------------------------------------------------------- development suites (in the repo)
def jevbench():
    out = []
    with open(os.path.join(REPO, "benchmarks", "jevbench", "jevbench_public.jsonl")) as f:
        rows = [json.loads(l) for l in f if l.strip()]
    for r in rows:
        q = r["question"]
        t = q["type"]
        qq = {"qid": "decision", "type": t, "instructions": q.get("instructions", ""), "criteria": q.get("criteria"),
              "values": None}
        exp = r["expected"]
        if t == "noul":
            qq["labels"] = ["yes", "no"]
            qq["expected"] = "yes" if str(exp).lower() in ("yes", "true", "1") else "no"
        elif t == "choice":
            qq["labels"] = list(q["criteria"].keys())
            qq["expected"] = str(exp)
        else:
            qq["labels"] = [str(i) for i in range(len(q["criteria"]))]
            qq["values"] = list(range(len(q["criteria"])))
            qq["expected"] = str(int(exp))
        out.append({"id": r["id"], "suite": "jev_systemone", "subset": r.get("family", ""), "tier": r.get("tier"),
                    "state": r["state"], "qs": [qq]})
    return out


# ---------------------------------------------------------------- MASSIVE
def _massive_case(lang, i, row, labels, rng, suite):
    gold = row["label_text"]
    pool = [x for x in labels if x != gold]
    keys = [gold] + rng.sample(pool, 19)
    rng.shuffle(keys)
    q = {"qid": "intent", "type": "choice", "instructions": "What is the user asking for in `utterance`?",
         "criteria": {k: k.replace("_", " ").replace(".", ": ") for k in keys}, "labels": keys, "expected": gold,
         "values": None}
    return {"id": f"{suite}-{lang}-{i:03d}", "suite": suite, "subset": lang, "tier": None,
            "state": {"utterance": row["text"]}, "qs": [q]}


def massive(split="test", langs=None, n=100, suite="massive"):
    out = []
    for lang in langs or ds.massive_langs():
        test = ds.jsonl_gz(ds.MASSIVE, f"test/{lang}.json.gz")
        labels = sorted({r["label_text"] for r in test})  # the option pool is the language's test label set
        rows = test if split == "test" else ds.jsonl_gz(ds.MASSIVE, f"{split}/{lang}.json.gz")
        rng = random.Random(13)
        out += [_massive_case(lang, i, r, labels, rng, suite) for i, r in enumerate(rows[:n])]
    return out


SPOT_LANGS = ["ru", "th", "hi", "ja", "es"]


def massive_spot():
    """Multilingual smoke check: MASSIVE validation split (never the frozen test split), 5 languages x 20."""
    return massive("validation", SPOT_LANGS, 20, suite="massive_spot")


# ---------------------------------------------------------------- XNLI
XNLI_CRIT = {"entailment": "the premise implies the hypothesis is true",
             "neutral": "the premise neither implies nor contradicts the hypothesis",
             "contradiction": "the premise implies the hypothesis is false"}


def xnli(langs=None, n=300, suite="xnli"):
    names = ds.xnli_label_names()
    out = []
    for lang in langs or ds.xnli_langs():
        rows = ds.parquet(ds.XNLI, f"{lang}/test-00000-of-00001.parquet")[:n]
        for i, r in enumerate(rows):
            q = {"qid": "relation", "type": "choice",
                 "instructions": "What is the relationship between `premise` and `hypothesis`?",
                 "criteria": dict(XNLI_CRIT), "labels": list(XNLI_CRIT), "expected": names[r["label"]], "values": None}
            out.append({"id": f"{suite}-{lang}-{i:03d}", "suite": suite, "subset": lang, "tier": None,
                        "state": {"premise": r["premise"], "hypothesis": r["hypothesis"]}, "qs": [q]})
    return out


# ---------------------------------------------------------------- typed-decisions (multi-question cases)
def typed():
    out = []
    for idx, r in enumerate(ds.parquet(ds.TYPED, "all/test-00000-of-00001.parquet")):
        qs_raw, gold = json.loads(r["questions"]), json.loads(r["gold"])
        state = json.loads(r["state"]) if isinstance(r["state"], str) else r["state"]
        qs = []
        for qid, qd in qs_raw.items():
            g, t = gold[qid], qd["type"]
            pr = g.get("probabilities") or {}
            q = {"qid": qid, "type": t, "instructions": qd.get("instructions", ""), "criteria": qd.get("criteria"),
                 "values": None}
            if t == "noul":
                pt = pr.get("true", g.get("noul", 0.5))
                q.update(labels=["yes", "no"], gold_soft={"yes": pt, "no": 1 - pt},
                         expected="yes" if str(g["label"]).lower() == "true" else "no")
            elif t == "choice":
                keys = list(qd["criteria"].keys())
                q.update(labels=keys, gold_soft={k: pr.get(k, 0.0) for k in keys}, expected=str(g["label"]))
            else:
                k = len(qd["criteria"])
                q.update(labels=[str(i) for i in range(k)], values=list(range(k)),
                         gold_soft={str(i): pr.get(str(i), 0.0) for i in range(k)}, expected=str(int(g["label"])),
                         expected_value=float(g.get("score", g["label"])))
            s = sum(q["gold_soft"].values()) or 1.0
            q["gold_soft"] = {a: b / s for a, b in q["gold_soft"].items()}
            qs.append(q)
        out.append({"id": f"typed-{idx:03d}", "suite": "typed", "subset": r["workflow"], "tier": None,
                    "state": state, "qs": qs})
    return out


# ---------------------------------------------------------------- option-order suites
def order_cases():
    """MASSIVE-en 200 (20 options), dair-ai/emotion 200 (6), XNLI-en 200 (3), JevBench choice items."""
    out = massive("test", ["en"], 200, suite="order_massive_en")
    emo = ds.emotion_label_names()
    for i, r in enumerate(ds.parquet(ds.EMOTION, "split/test-00000-of-00001.parquet")[:200]):
        q = {"qid": "emotion", "type": "choice", "instructions": "Which emotion does the author of `text` express?",
             "criteria": {e: e for e in emo}, "labels": list(emo), "expected": emo[r["label"]], "values": None}
        out.append({"id": f"order_emotion-{i:03d}", "suite": "order_emotion", "subset": "en", "tier": None,
                    "state": {"text": r["text"]}, "qs": [q]})
    for c in xnli(["en"], 200, suite="order_xnli_en"):
        out.append(c)
    for c in jevbench():
        if c["qs"][0]["type"] == "choice":
            out.append({**c, "suite": "order_jev_choice"})
    return out


def permute(case, mode, rng=None):
    """Reorder choice criteria: none | random (one shuffle per case from a shared rng) | reverse."""
    if mode == "none":
        return case
    qs = []
    for q in case["qs"]:
        if q["type"] != "choice":
            qs.append(q)
            continue
        keys = list(q["criteria"].keys())
        if mode == "random":
            order = list(range(len(keys)))
            rng.shuffle(order)
            nk = [keys[i] for i in order]
        elif mode == "reverse":
            nk = keys[::-1]
        else:
            raise ValueError(mode)
        qs.append({**q, "criteria": {k: q["criteria"][k] for k in nk}, "shown_order": nk})
    return {**case, "qs": qs}


def body(case):
    return {"state": case["state"],
            "questions": {q["qid"]: {"type": q["type"], "instructions": q["instructions"],
                                     **({"criteria": q["criteria"]} if q["criteria"] is not None else {})}
                          for q in case["qs"]}}


# ---------------------------------------------------------------- answers -> distributions and per-item scores
def distribution(q, answer):
    """Normalize a /v1/systemone answer to {label: p} over q["labels"]. Never uses the server's `confidence`
    field (its meaning differs between servers); confidence is recomputed as max p."""
    if not isinstance(answer, dict):
        return None
    if q["type"] == "noul":
        p = answer.get("noul")
        if p is None:
            pr = answer.get("probabilities") or {}
            p = pr.get("true", pr.get("yes"))
        if p is None:
            return None
        p = min(max(float(p), 0.0), 1.0)
        return {"yes": p, "no": 1.0 - p}
    pr = answer.get("probabilities") or {}
    if q["type"] == "score":
        d = {lab: float(pr.get(str(i), 0.0)) for i, lab in enumerate(q["labels"])}
    else:
        d = {lab: float(pr.get(lab, 0.0)) for lab in q["labels"]}
    s = sum(d.values())
    if s <= 0:
        ch = answer.get("choice")
        return {k: (1.0 if k == ch else 0.0) for k in d} if ch in d else None
    return {k: v / s for k, v in d.items()}


def score(q, dist):
    labels = q["labels"]
    actual = max(labels, key=lambda l: dist[l])
    p_gold = dist.get(q["expected"], 0.0)
    h = -sum(p * math.log(p) for p in dist.values() if p > 1e-12)
    out = {"actual": actual, "accurate": actual == q["expected"], "confidence": dist[actual], "p_gold": p_gold,
           "nll": -math.log(max(p_gold, EPS)),
           "brier": sum((dist[l] - (1.0 if l == q["expected"] else 0.0)) ** 2 for l in labels),
           "entropy": h, "normalized_entropy": h / math.log(len(labels)) if len(labels) > 1 else 0.0}
    if q.get("values"):
        vals = dict(zip(labels, q["values"]))
        ev = sum(dist[l] * vals[l] for l in labels)
        out["abs_err_ev"] = abs(ev - q.get("expected_value", vals[q["expected"]]))
    gs = q.get("gold_soft")
    if gs:
        out["soft_acc"] = sum(dist[l] * gs.get(l, 0.0) for l in labels)
        out["soft_brier"] = sum((dist[l] - gs.get(l, 0.0)) ** 2 for l in labels)
        out["gold_soft_max"] = max(gs.values())
    return out


SUITES = {
    "jev_systemone": jevbench,
    "massive_spot": massive_spot,
    "massive": massive,
    "xnli": xnli,
    "typed": typed,
    "order": order_cases,
}
