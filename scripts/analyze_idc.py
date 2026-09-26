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


def fisher_one_sided(a, b, c, d):
    """P(X >= a) for the 2x2 table [[a,b],[c,d]] under the hypergeometric null (one-sided Fisher exact)."""
    from math import comb
    r1, c1, n = a + b, a + c, a + b + c + d
    denom = comb(n, c1)
    return sum(comb(r1, x) * comb(n - r1, c1 - x) for x in range(a, min(r1, c1) + 1)) / denom


def cmd_collision(args):
    """PROP-11: forward-slot accuracy and the letter-collision effect for dual-mirror receipts."""
    tasks = {}
    with open(args.dataset) as f:
        for line in f:
            t = json.loads(line)
            tasks[t["id"]] = t
    def load(p):
        with open(p) as f:
            return {c["id"]: c for c in json.load(f)["cases"]}
    bases = [load(p) for p in args.baselines]
    ids = list(bases[0])
    majority = {}
    for i in ids:
        votes = [b[i]["actual"] for b in bases]
        majority[i] = max(set(votes), key=votes.count)
    base_correct = [sum(1 for i in ids if b[i]["accurate"]) for b in bases]
    lo, hi = min(base_correct), max(base_correct)
    print("### PROP-11 letter collision\n")
    print(f"Baselines: {base_correct} → noise band [{lo}, {hi}] of {len(ids)}. 'Forward' = the forward slot's own raw reading "
          "(before merging with the mirror). 'Same position' = both slots chose the option at the same list position, i.e. "
          "the same label, which in reversed modes means different options.\n")
    print("| condition | merged correct | forward-slot correct | vs band | same position, different option | forward changed vs baseline majority: same-position / other | one-sided Fisher p |")
    print("| :--- | ---: | ---: | :--- | ---: | :--- | ---: |")
    out = []
    for p in args.receipts:
        mode = p.split("__")[-1].replace(".json", "")
        cs = load(p)
        fwd_ok = merged_ok = coll = 0
        a = b = c = d = 0
        for i in ids:
            x = cs[i]
            merged_ok += 1 if x["accurate"] else 0
            idc = x.get("idc") or {}
            fw, rv = idc.get("raw_fwd_probs"), idc.get("raw_rev_probs")
            if not fw or not rv:
                continue
            labels = tasks[i]["labels"]
            k = len(labels)
            fwin = max(fw, key=fw.get)
            rwin = max(rv, key=rv.get)
            exp = str(x["expected"])
            fwd_ok += 1 if fwin.lower() == exp.lower() else 0
            fi = labels.index(fwin) if fwin in labels else -1
            ri = labels.index(rwin) if rwin in labels else -1
            rpos = ri if "copy" in mode else k - 1 - ri  # position of the mirror's pick in the mirror's own list
            same_pos_diff = (fi == rpos) and (fi != ri)
            coll += same_pos_diff
            changed = fwin != majority[i]
            if same_pos_diff:
                a += changed; b += not changed
            else:
                c += changed; d += not changed
        band = "within" if lo <= fwd_ok <= hi else ("degraded" if fwd_ok < lo - 3 else ("below band" if fwd_ok < lo else "above band"))
        pval = fisher_one_sided(a, b, c, d) if (a + b) and (c + d) else None
        rate1 = f"{a}/{a+b} ({100*a/(a+b):.0f}%)" if a + b else "—"
        rate2 = f"{c}/{c+d} ({100*c/(c+d):.0f}%)" if c + d else "—"
        print(f"| {mode} | {merged_ok} | {fwd_ok} | {band} | {coll} | {rate1} / {rate2} | {'—' if pval is None else '%.2g' % pval} |")
        out.append({"receipt": p, "mode": mode, "merged_correct": merged_ok, "forward_correct": fwd_ok, "band": [lo, hi],
                    "verdict": band, "same_position_different_option": coll, "changed_same_pos": [a, a + b],
                    "changed_other": [c, c + d], "fisher_p": pval})
    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump({"baselines": args.baselines, "baseline_correct": base_correct, "conditions": out}, f, indent=2)


def _rank(xs):
    order = sorted(range(len(xs)), key=lambda i: xs[i])
    r = [0.0] * len(xs)
    i = 0
    while i < len(order):
        j = i
        while j + 1 < len(order) and xs[order[j + 1]] == xs[order[i]]:
            j += 1
        for k in range(i, j + 1):
            r[order[k]] = (i + j) / 2.0
        i = j + 1
    return r


def _pearson(a, b):
    ma, mb = sum(a) / len(a), sum(b) / len(b)
    sa = math.sqrt(sum((x - ma) ** 2 for x in a)); sb = math.sqrt(sum((y - mb) ** 2 for y in b))
    return sum((x - ma) * (y - mb) for x, y in zip(a, b)) / (sa * sb) if sa and sb else 0.0


def partial_spearman(x, y, z):
    """Spearman correlation of x and y controlling for z (rank residuals)."""
    rx, ry, rz = _rank(x), _rank(y), _rank(z)
    def resid(a):
        mz, ma = sum(rz) / len(rz), sum(a) / len(a)
        vz = sum((q - mz) ** 2 for q in rz)
        beta = sum((q - mz) * (w - ma) for q, w in zip(rz, a)) / vz if vz else 0.0
        return [w - ma - beta * (q - mz) for q, w in zip(rz, a)]
    return _pearson(resid(rx), resid(ry))


def auroc(scores, labels):
    pos = [s for s, l in zip(scores, labels) if l]; neg = [s for s, l in zip(scores, labels) if not l]
    if not pos or not neg:
        return None
    return sum((p > n) + 0.5 * (p == n) for p in pos for n in neg) / (len(pos) * len(neg))


def _logit_fit(X, y, iters=3000, lr=0.5, l2=1e-3):
    import numpy as np
    X = np.column_stack([np.ones(len(X)), np.asarray(X, float)])
    mu, sd = X[:, 1:].mean(0), X[:, 1:].std(0) + 1e-9
    X[:, 1:] = (X[:, 1:] - mu) / sd
    w = np.zeros(X.shape[1]); y = np.asarray(y, float)
    for _ in range(iters):
        p = 1 / (1 + np.exp(-X @ w))
        w -= lr * (X.T @ (p - y) / len(y) + l2 * w)
    return w, mu, sd


def _logit_pred(model, X):
    import numpy as np
    w, mu, sd = model
    X = (np.asarray(X, float) - mu) / sd
    return 1 / (1 + np.exp(-(w[0] + X @ w[1:])))


def cv_auroc(features, labels, folds=5, repeats=20, seed=7):
    rng = random.Random(seed)
    vals = []
    n = len(labels)
    for _ in range(repeats):
        idx = list(range(n)); rng.shuffle(idx)
        preds = [0.0] * n
        for f in range(folds):
            test = idx[f::folds]; tset = set(test)
            train = [i for i in idx if i not in tset]
            m = _logit_fit([features[i] for i in train], [labels[i] for i in train])
            for i, p in zip(test, _logit_pred(m, [features[i] for i in test])):
                preds[i] = float(p)
        vals.append(auroc(preds, labels))
    return statistics.mean(vals), min(vals), max(vals)


def cmd_separate(args):
    """PROP-12: does forward/reversed disagreement from separate passes add error detection beyond hesitation?"""
    def load(p):
        with open(p) as f:
            return {c["id"]: c for c in json.load(f)["cases"]}
    F = [load(p) for p in args.forward]
    Rv = [load(p) for p in args.reversed]
    ids = [i for i in F[0] if all(i in x for x in F + Rv)]
    def dist(c):
        tp = c.get("top_probabilities") or {}
        s = sum(tp.values()) or 1.0
        return {k: v / s for k, v in tp.items()}
    def tvd(a, b):
        keys = set(a) | set(b)
        return 0.5 * sum(abs(a.get(k, 0) - b.get(k, 0)) for k in keys)
    def hes(p):
        k = max(2, len(p))
        return -sum(v * math.log(v) for v in p.values() if v > 0) / math.log(k)
    def stats_for(fwd, other, label):
        err = [0 if fwd[i]["accurate"] else 1 for i in ids]
        h = [hes(dist(fwd[i])) for i in ids]
        d = [tvd(dist(fwd[i]), dist(other[i])) for i in ids]
        ps = partial_spearman(d, err, h)
        rng = random.Random(11)
        boots = []
        for _ in range(args.boot):
            s_ = [rng.randrange(len(ids)) for _ in ids]
            boots.append(partial_spearman([d[j] for j in s_], [err[j] for j in s_], [h[j] for j in s_]))
        boots.sort()
        lo, hi = boots[int(0.025 * len(boots))], boots[int(0.975 * len(boots)) - 1]
        a_h = cv_auroc([[x] for x in h], err)
        a_hd = cv_auroc([[x, y] for x, y in zip(h, d)], err)
        return {"pair": label, "n": len(ids), "errors": sum(err), "partial_spearman": ps, "ci": [lo, hi],
                "auroc_hes_raw": auroc(h, err), "auroc_tvd_raw": auroc(d, err),
                "cv_auroc_hes": a_h, "cv_auroc_hes_tvd": a_hd, "mean_tvd": sum(d) / len(d)}
    rows = []
    for fi, fwd in enumerate(F):
        for ri, rev in enumerate(Rv):
            rows.append(stats_for(fwd, rev, f"F{fi+1}×R{ri+1}"))
    noise = [stats_for(F[a], F[b], f"F{a+1}×F{b+1} (noise)") for a in range(len(F)) for b in range(a + 1, len(F))]
    ens = []
    for fi, fwd in enumerate(F):
        for ri, rev in enumerate(Rv):
            ok = 0
            for i in ids:
                a, b = dist(fwd[i]), dist(rev[i])
                m = {k: 0.5 * (a.get(k, 0) + b.get(k, 0)) for k in set(a) | set(b)}
                ok += max(m, key=m.get).lower() == str(fwd[i]["expected"]).lower()
            ens.append({"pair": f"F{fi+1}+R{ri+1}", "correct": ok, "forward_correct": sum(fwd[i]["accurate"] for i in ids),
                        "reversed_correct": sum(rev[i]["accurate"] for i in ids)})
    # Pre-registered primary: mean partial Spearman over all forward x reversed pairings, item bootstrap.
    def pair_arrays(fwd, other):
        return ([tvd(dist(fwd[i]), dist(other[i])) for i in ids], [0 if fwd[i]["accurate"] else 1 for i in ids],
                [hes(dist(fwd[i])) for i in ids])
    arrs = [pair_arrays(f, r) for f in F for r in Rv]
    primary = statistics.mean(partial_spearman(*a) for a in arrs)
    rng = random.Random(23)
    boots = []
    for _ in range(args.boot):
        smp = [rng.randrange(len(ids)) for _ in ids]
        boots.append(statistics.mean(partial_spearman([d[j] for j in smp], [e[j] for j in smp], [h[j] for j in smp]) for d, e, h in arrs))
    boots.sort()
    prim_ci = [boots[int(0.025 * len(boots))], boots[int(0.975 * len(boots)) - 1]]
    print("### PROP-12 separate-pass mirror\n")
    print(f"**Primary (pre-registered):** mean partial Spearman over {len(arrs)} forward×reversed pairings = {primary:.3f}, "
          f"95% item-bootstrap CI [{prim_ci[0]:.3f}, {prim_ci[1]:.3f}].\n")
    print("| pair | n | errors | mean TVD | partial Spearman (TVD, error \\| hesitation) [95% CI] | AUROC hesitation | AUROC TVD | CV AUROC hesitation | CV AUROC hesitation + TVD |")
    print("| :--- | ---: | ---: | ---: | :--- | ---: | ---: | ---: | ---: |")
    for r in rows + noise:
        print(f"| {r['pair']} | {r['n']} | {r['errors']} | {r['mean_tvd']:.3f} | {r['partial_spearman']:.3f} [{r['ci'][0]:.3f}, {r['ci'][1]:.3f}] | "
              f"{r['auroc_hes_raw']:.3f} | {r['auroc_tvd_raw']:.3f} | {r['cv_auroc_hes'][0]:.3f} | {r['cv_auroc_hes_tvd'][0]:.3f} |")
    print("\n| averaged forward+reversed | correct | forward alone | reversed alone |")
    print("| :--- | ---: | ---: | ---: |")
    for e in ens:
        print(f"| {e['pair']} | {e['correct']} | {e['forward_correct']} | {e['reversed_correct']} |")
    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump({"primary_mean_partial_spearman": primary, "primary_ci": prim_ci, "order_pairs": rows, "noise_pairs": noise, "ensembles": ens}, f, indent=2)


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
    sp = sub.add_parser("separate-pass")
    sp.add_argument("--forward", nargs="+", required=True)
    sp.add_argument("--reversed", nargs="+", required=True)
    sp.add_argument("--boot", type=int, default=2000)
    sp.add_argument("--json-out")
    sp = sub.add_parser("collision")
    sp.add_argument("--baselines", nargs="+", required=True)
    sp.add_argument("--dataset", default="benchmarks/jevbench/jevbench_public.jsonl")
    sp.add_argument("--json-out")
    sp.add_argument("receipts", nargs="+")
    args = ap.parse_args()
    {"collision": cmd_collision, "cv-temperature": cmd_cv, "gates": cmd_gates, "merge-rules": cmd_merge, "separate-pass": cmd_separate}[args.cmd](args)


if __name__ == "__main__":
    main()
