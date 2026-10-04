"""PROP-20: leave-one-suite-out temperature fits on dev suites (prod side of the v2 reference run, runs 1-3 pooled per
suite for fitting, run 1 for scoring). Candidates: raw, global T, per-type T (noul / choice<=5 / choice 6-26 /
choice>26 / score)."""
import json, glob, math, sys
sys.path.insert(0, "scripts")
from matrix import metrics as M

RUN = "benchmarks/runs/20261003-v2-reference-v021-t2 (receipts in the matrix bucket or a local re-run)"
SUITES = ["jev_systemone", "calib_systemone", "intents_systemone", "di_wide", "di_catchall", "rag_dev",
          "gate_mixed_noul", "massive_spot"]
GRID = [round(0.5 + 0.05 * i, 2) for i in range(111)]  # 0.5 .. 6.0


def bucket(r):
    t, k = r.get("type"), r.get("K") or 2
    if t == "noul":
        return "noul"
    if t == "score":
        return "score"
    return "choice<=5" if k <= 5 else ("choice6-26" if k <= 26 else "choice>26")


def load(s, runs):
    out = []
    for f in sorted(glob.glob(f"{RUN}/{s}__prod__r*.json")):
        r = int(f.rsplit("__r", 1)[1].split(".")[0])
        if r in runs:
            out += [dict(x, suite=s) for x in M.rows(json.load(open(f))) if x.get("probabilities") and x.get("expected") in x["probabilities"]]
    return out


def temper(p, T):
    lp = {k: math.log(max(v, 1e-12)) / T for k, v in p.items()}
    m = max(lp.values()); e = {k: math.exp(v - m) for k, v in lp.items()}; z = sum(e.values())
    return {k: v / z for k, v in e.items()}


def nll(rows, T):
    return sum(-math.log(max(temper(r["probabilities"], T)[r["expected"]], 1e-12)) for r in rows) / max(1, len(rows))


def fitT(rows):
    return min(GRID, key=lambda t: (nll(rows, t), abs(t - 1))) if rows else 1.0


def score(rows, Tof):
    conf, hit = [], []
    for r in rows:
        p = temper(r["probabilities"], Tof(r)); top = max(p, key=p.get)
        conf.append(p[top]); hit.append(top == r["expected"])
    return M.ece(conf, hit), M.auroc(conf, hit) if hasattr(M, "auroc") else None


fitrows = {s: load(s, {1, 2, 3}) for s in SUITES}
testrows = {s: load(s, {1}) for s in SUITES}
res = {}
print(f"{'suite':18s} {'n':>4} {'raw ECE':>8} {'glob T':>7} {'glob ECE':>9} {'type ECE':>9} {'Ts':>40}  AUROC raw/glob/type")
for s in SUITES:
    train = [r for o in SUITES if o != s for r in fitrows[o]]
    Tg = fitT(train)
    Tt = {b: fitT([r for r in train if bucket(r) == b]) for b in ("noul", "choice<=5", "choice6-26", "choice>26", "score")}
    raw = score(testrows[s], lambda r: 1.0); g = score(testrows[s], lambda r: Tg); t = score(testrows[s], lambda r: Tt[bucket(r)])
    used = sorted({bucket(r) for r in testrows[s]})
    res[s] = {"raw": raw, "global": g, "type": t, "Tg": Tg, "Tt": {b: Tt[b] for b in used}}
    print(f"{s:18s} {len(testrows[s]):4d} {raw[0]:8.3f} {Tg:7.2f} {g[0]:9.3f} {t[0]:9.3f} {str({b: Tt[b] for b in used}):>40}  "
          f"{raw[1]:.3f}/{g[1]:.3f}/{t[1]:.3f}")
allrows = [r for s in SUITES for r in fitrows[s]]
print("full-data fits: global", fitT(allrows), {b: fitT([r for r in allrows if bucket(r) == b]) for b in ("noul", "choice<=5", "choice6-26", "choice>26", "score")})
for name in ("global", "type"):
    ok = all(res[s][name][0] <= 0.7 * res[s]["raw"][0] for s in SUITES)
    da = max(abs(res[s][name][1] - res[s]["raw"][1]) for s in SUITES)
    worse = [s for s in SUITES if res[s][name][0] > res[s]["raw"][0]]
    print(f"{name}: >=30% ECE cut on every suite: {ok}; max |dAUROC| {da:.3f}; worse than raw on {worse}")
json.dump(res, open("loso_results.json", "w"), indent=1)
