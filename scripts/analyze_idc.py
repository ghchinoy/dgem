#!/usr/bin/env python3
"""Offline analyses over dgem receipts (no GPU needed).

  cv-temperature   PROP-01: held-out temperature scaling (k-fold CV, repeated) + cross-suite transfer
  gates            PROP-02 (offline): cascade accuracy vs. escalation for entropy-only and
                   entropy-or-Mirror-TVD gates, using a Stage-2 receipt with answers for every item
  merge-rules      PROP-05: re-score dual-mirror receipts under alternative merge rules

All metrics are computed from per-item probabilities stored in the receipts, at T=1 unless stated.
Output is Markdown (and optionally JSON via --json-out) so it can be pasted into the experiment ledger.
"""

import argparse
import json
import math
import random
import statistics

GRID = [round(0.5 + 0.05 * i, 2) for i in range(61)]  # 0.50 .. 3.50, same range as --auto-temperature


def load_items(path):
    with open(path) as f:
        d = json.load(f)
    applied = (d.get("jev_parity") or {}).get("temperature_applied") or d.get("temperature_scale") or 1.0
    items = []
    for c in d.get("cases", []):
        tp = c.get("top_probabilities") or {}
        if not tp:
            continue
        raw = {k: max(v, 1e-12) ** applied for k, v in tp.items()}  # undo stored temperature
        s = sum(raw.values())
        p = {k: v / s for k, v in raw.items()}
        exp = str(c.get("expected", "")).strip().lower()
        alias = {"true": "yes", "false": "no"}.get(exp, exp)
        gold = next((k for k in p if k.lower() in (exp, alias)), None)
        items.append({"id": c["id"], "p": p, "acc": bool(c.get("accurate")), "gold": gold,
                      "k": c.get("vocab_cardinality") or len(p), "idc": c.get("idc")})
    return d, items


def scale(p, t):
    w = {k: v ** (1.0 / t) for k, v in p.items()}
    s = sum(w.values())
    return {k: v / s for k, v in w.items()}


def ece(pairs, bins=10):
    b = [[] for _ in range(bins)]
    for conf, acc in pairs:
        b[min(int(conf * bins), bins - 1)].append((conf, acc))
    n = len(pairs)
    return sum(len(x) / n * abs(sum(c for c, _ in x) / len(x) - sum(a for _, a in x) / len(x)) for x in b if x)


def metrics(items, t):
    pairs, nll, brier, nb = [], 0.0, 0.0, 0
    for it in items:
        q = scale(it["p"], t)
        pairs.append((max(q.values()), 1.0 if it["acc"] else 0.0))
        if it["gold"]:
            nll -= math.log(max(q[it["gold"]], 1e-12))
            brier += sum((v - (1.0 if k == it["gold"] else 0.0)) ** 2 for k, v in q.items())
            nb += 1
    return {"ece": ece(pairs), "nll": nll / nb if nb else None, "brier": brier / nb if nb else None}


def fit_t(items, criterion):
    best = min(GRID, key=lambda t: (metrics(items, t)[criterion] if metrics(items, t)[criterion] is not None else 9e9))
    return best


def cv_temperature(items, folds, repeats, criterion, seed):
    rng = random.Random(seed)
    base = metrics(items, 1.0)["ece"]
    insample_t = fit_t(items, criterion)
    insample = metrics(items, insample_t)["ece"]
    oof_eces, ts = [], []
    for _ in range(repeats):
        idx = list(range(len(items)))
        rng.shuffle(idx)
        parts = [idx[i::folds] for i in range(folds)]
        pairs = []
        for f in range(folds):
            test = [items[i] for i in parts[f]]
            train = [items[i] for j in range(folds) if j != f for i in parts[j]]
            t = fit_t(train, criterion)
            ts.append(t)
            for it in test:
                q = scale(it["p"], t)
                pairs.append((max(q.values()), 1.0 if it["acc"] else 0.0))
        oof_eces.append(ece(pairs))
    oof = statistics.mean(oof_eces)
    return {
        "n": len(items), "criterion": criterion, "folds": folds, "repeats": repeats,
        "ece_t1": base, "insample_t": insample_t, "ece_insample": insample,
        "ece_cv_mean": oof, "ece_cv_min": min(oof_eces), "ece_cv_max": max(oof_eces),
        "cv_t_median": statistics.median(ts), "cv_t_min": min(ts), "cv_t_max": max(ts),
        "reduction_insample_pct": 100 * (base - insample) / base if base else None,
        "reduction_cv_pct": 100 * (base - oof) / base if base else None,
        "cv_reductions_positive": sum(1 for e in oof_eces if e < base), 
    }


def cmd_cv(args):
    results = []
    for path in args.receipts:
        _, items = load_items(path)
        for crit in args.criteria:
            r = cv_temperature(items, args.folds, args.repeats, crit, args.seed)
            r["receipt"] = path
            results.append(r)
    transfer = []
    if len(args.receipts) >= 2:
        a, b = args.receipts[0], args.receipts[1]
        _, ia = load_items(a)
        _, ib = load_items(b)
        for crit in args.criteria:
            for src, dst, isrc, idst in ((a, b, ia, ib), (b, a, ib, ia)):
                t = fit_t(isrc, crit)
                transfer.append({"fit_on": src, "apply_to": dst, "criterion": crit, "t": t,
                                 "ece_t1": metrics(idst, 1.0)["ece"], "ece_transfer": metrics(idst, t)["ece"]})
    print("### Held-out temperature scaling (PROP-01)\n")
    print(f"{args.folds}-fold CV repeated {args.repeats}× (seed {args.seed}); T grid 0.50–3.50. ECE is 10-bin, pooled over out-of-fold predictions.\n")
    print("| receipt | n | fit by | ECE T=1 | in-sample T* | ECE in-sample | ECE held-out (mean [min–max]) | held-out T* median [range] | reduction in-sample | reduction held-out | repeats better than T=1 |")
    print("| :--- | ---: | :--- | ---: | ---: | ---: | :--- | :--- | ---: | ---: | ---: |")
    for r in results:
        print(f"| `{r['receipt']}` | {r['n']} | {r['criterion']} | {r['ece_t1']:.3f} | {r['insample_t']:.2f} | {r['ece_insample']:.3f} | "
              f"{r['ece_cv_mean']:.3f} [{r['ece_cv_min']:.3f}–{r['ece_cv_max']:.3f}] | {r['cv_t_median']:.2f} [{r['cv_t_min']:.2f}–{r['cv_t_max']:.2f}] | "
              f"{r['reduction_insample_pct']:.0f}% | {r['reduction_cv_pct']:.0f}% | {r['cv_reductions_positive']}/{r['repeats']} |")
    if transfer:
        print("\n| fit on | apply to | fit by | T | ECE T=1 | ECE with transferred T |")
        print("| :--- | :--- | :--- | ---: | ---: | ---: |")
        for t in transfer:
            print(f"| `{t['fit_on']}` | `{t['apply_to']}` | {t['criterion']} | {t['t']:.2f} | {t['ece_t1']:.3f} | {t['ece_transfer']:.3f} |")
    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump({"cv": results, "transfer": transfer, "folds": args.folds, "repeats": args.repeats, "seed": args.seed}, f, indent=2)


def hes(p, k):
    h = -sum(v * math.log(v) for v in p.values() if v > 0)
    return h / math.log(max(2, k))


def cmd_gates(args):
    _, s1 = load_items(args.stage1)
    with open(args.stage2) as f:
        s2 = {c["id"]: c for c in json.load(f)["cases"]}
    s1 = [it for it in s1 if it["id"] in s2]
    n = len(s1)
    has_tvd = any((it["idc"] or {}).get("has_mirror") for it in s1)
    rows = []
    for tau in args.taus:
        for tvd_tau in ([None] + args.tvd_taus if has_tvd else [None]):
            esc = correct = 0
            for it in s1:
                go = hes(it["p"], it["k"]) >= tau
                if tvd_tau is not None:
                    go = go or (it["idc"] or {}).get("mirror_tvd", 0.0) >= tvd_tau
                if go:
                    esc += 1
                    correct += 1 if s2[it["id"]].get("accurate") else 0
                else:
                    correct += 1 if it["acc"] else 0
            rows.append({"tau": tau, "tvd_tau": tvd_tau, "escalated": esc, "correct": correct, "n": n})
    s1_acc = sum(it["acc"] for it in s1)
    s2_acc = sum(1 for it in s1 if s2[it["id"]].get("accurate"))
    print(f"### Cascade gates (offline): `{args.stage1}` → `{args.stage2}`\n")
    print(f"n={n}. Stage 1 alone {s1_acc}/{n}; Stage 2 alone {s2_acc}/{n}. Escalated items take the Stage-2 answer.\n")
    print("| hesitation gate | Mirror TVD gate | escalated | correct | accuracy |")
    print("| ---: | ---: | ---: | ---: | ---: |")
    for r in rows:
        print(f"| ≥ {r['tau']:.2f} | {'—' if r['tvd_tau'] is None else '≥ %.2f' % r['tvd_tau']} | {r['escalated']} ({100*r['escalated']/n:.0f}%) | {r['correct']} | {100*r['correct']/n:.1f}% |")
    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump({"stage1": args.stage1, "stage2": args.stage2, "rows": rows, "stage1_correct": s1_acc, "stage2_correct": s2_acc}, f, indent=2)


def cmd_merge(args):
    _, items = load_items(args.receipt)
    rules = {
        "forward_only": lambda f, r: f,
        "reversed_only": lambda f, r: r,
        "mean_50_50": lambda f, r: {k: 0.5 * f[k] + 0.5 * r.get(k, 0) for k in f},
        "geometric": lambda f, r: {k: math.sqrt(max(f[k], 1e-12) * max(r.get(k, 1e-12), 1e-12)) for k in f},
        "fwd70_rev30": lambda f, r: {k: 0.7 * f[k] + 0.3 * r.get(k, 0) for k in f},
    }
    print(f"### Dual-mirror merge rules (offline): `{args.receipt}`\n")
    print("| rule | n | correct | Brier | ECE-10 | >0.9 conf correct |")
    print("| :--- | ---: | ---: | ---: | ---: | ---: |")
    out = []
    use = [it for it in items if (it["idc"] or {}).get("has_mirror") and it["gold"]]
    for name, fn in rules.items():
        pairs, br, corr, hi, hic = [], 0.0, 0, 0, 0
        for it in use:
            d = it["idc"]
            f, r = d.get(args.stage + "fwd_probs"), d.get(args.stage + "rev_probs")
            if not f or not r:
                continue
            m = fn(f, r)
            s = sum(m.values())
            m = {k: v / s for k, v in m.items()}
            win = max(m, key=m.get)
            ok = win == it["gold"]
            corr += ok
            conf = m[win]
            pairs.append((conf, 1.0 if ok else 0.0))
            br += sum((v - (1.0 if k == it["gold"] else 0.0)) ** 2 for k, v in m.items())
            if conf > 0.9:
                hi += 1
                hic += ok
        k = len(pairs)
        if k:
            out.append({"rule": name, "n": k, "correct": corr, "brier": br / k, "ece": ece(pairs), "hi": f"{hic}/{hi}"})
            print(f"| {name} | {k} | {corr} | {br/k:.3f} | {ece(pairs):.3f} | {hic}/{hi} |")
    print("\nNote: scored on items whose gold label maps to a single option (exact-match grading), so counts can differ from the harness's accuracy.")
    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump({"receipt": args.receipt, "stage": args.stage, "rules": out}, f, indent=2)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    sp = sub.add_parser("cv-temperature")
    sp.add_argument("receipts", nargs="+")
    sp.add_argument("--folds", type=int, default=5)
    sp.add_argument("--repeats", type=int, default=20)
    sp.add_argument("--seed", type=int, default=13)
    sp.add_argument("--criteria", nargs="+", default=["nll", "ece"])
    sp.add_argument("--json-out")
    sp = sub.add_parser("gates")
    sp.add_argument("--stage1", required=True)
    sp.add_argument("--stage2", required=True)
    sp.add_argument("--taus", type=float, nargs="+", default=[0.10, 0.16, 0.25, 0.35, 0.50])
    sp.add_argument("--tvd-taus", type=float, nargs="+", default=[0.15, 0.25])
    sp.add_argument("--json-out")
    sp = sub.add_parser("merge-rules")
    sp.add_argument("receipt")
    sp.add_argument("--stage", choices=["raw_", ""], default="raw_", help="raw_ = before null-prior; '' = after")
    sp.add_argument("--json-out")
    args = ap.parse_args()
    {"cv-temperature": cmd_cv, "gates": cmd_gates, "merge-rules": cmd_merge}[args.cmd](args)


if __name__ == "__main__":
    main()
