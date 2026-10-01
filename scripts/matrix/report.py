"""Matrix report: verdicts per gate, per-suite tables and the measured noise floor -> report.md + summary.json.

Two modes:
  baseline mode (a run has a baseline target): every other target is a candidate compared with the baseline
                measured in the same session.
  single-target mode: the target is compared with the reference ranges in matrix_v1.json ("reference").
Verdicts: PASS, REVIEW (look before promoting), FAIL, INFO (reported, not gated).
"""
import json
import os
import statistics
from collections import defaultdict

from . import metrics as M

RANK = {"PASS": 0, "INFO": 0, "REVIEW": 1, "FAIL": 2}


def _f(x, d=3):
    if x is None:
        return "—"
    return f"{x:.{d}f}" if isinstance(x, float) else str(x)


def load_run(run_dir):
    with open(os.path.join(run_dir, "manifest.json")) as f:
        man = json.load(f)
    by = defaultdict(lambda: defaultdict(list))  # suite -> target -> [(entry, receipt)]
    for e in man["receipts"]:
        by[e["suite"]][e["config"]].append((e, M.load(os.path.join(run_dir, e["path"]))))
    for s in by.values():
        for lst in s.values():
            lst.sort(key=lambda x: (x[0].get("run", 1), x[0].get("perm") or ""))
    return man, by


def _within(runs):
    vals = [M.agreement(runs[i], runs[j]) for i in range(len(runs)) for j in range(i + 1, len(runs))]
    vals = [v for v in vals if v is not None]
    return statistics.mean(vals) if vals else None


def _cross(ra, rb):
    vals = [M.agreement(a, b) for a in ra for b in rb]
    vals = [v for v in vals if v is not None]
    return statistics.mean(vals) if vals else None


def _noise_table(rows, acc_suites, names, noise):
    multi = [s for s in acc_suites if any(len(rows[s].get(n, [])) > 1 for n in names)]
    out = ["## Measured noise floor (answer agreement between repeated identical runs)", "",
           "| target | " + " | ".join(multi) + " | mean |", "|---|" + "---|" * (len(multi) + 1)]
    for n in names:
        out.append(f"| {n} | " + " | ".join(_f(_within(rows[s].get(n, []))) for s in multi) + f" | {_f(noise.get(n))} |")
    return out + [""]


def build(run_dir, matrix):
    man, by = load_run(run_dir)
    mx = man["matrix"]
    th = matrix["thresholds"]
    ref = (matrix.get("reference") or {}).get("suites", {})
    names = [t["name"] for t in mx["targets"]]
    base = mx.get("baseline")
    cands = [n for n in names if n != base] if base else names
    gates = []  # (gate, target, verdict, detail)
    L = [f"# Regression matrix report: {man['run_id']}", "",
         f"- Matrix **{mx['matrix_version']}**, tier **{mx['tier']}**, started {mx.get('started')}, finished {mx.get('finished')}",
         f"- Repository commit `{man.get('git_commit', '')[:10]}`",
         f"- Mode: {'baseline `' + base + '` measured in the same session' if base else 'single target, compared with the reference ranges'}",
         "", "| target | kind | version | revision | vLLM commit |", "|---|---|---|---|---|"]
    for t in mx["targets"]:
        h = t.get("health") or {}
        L.append(f"| {t['name']}{' (baseline)' if t['name'] == base else ''} | {t['kind']} | {h.get('version', '—')} | "
                 f"{h.get('revision', '—')} | {str(h.get('vllm_commit', '—'))[:10]} |")
    L.append("")

    # ---------------- health
    for n, lst in by.get("health", {}).items():
        ok = all(r.get("ok") for _, r in lst)
        gates.append(("health", n, "PASS" if ok else "FAIL", "ready" if ok else f"status {lst[0][1].get('status')}"))

    # ---------------- contract
    con = {n: lst[0][1]["cases"] for n, lst in by.get("contract", {}).items()}
    for n in cands:
        if n not in con:
            continue
        wrong = sorted(k for k, v in con[n].items() if v.get("wrong"))
        diffs = []
        if base and base in con:
            diffs = sorted(k for k in con[n] if k in con[base] and con[n][k].get("status") != con[base][k].get("status"))
        v = "FAIL" if wrong or diffs else "PASS"
        gates.append(("contract", n, v, (f"wrong: {', '.join(wrong)}; " if wrong else "") +
                      (f"status differs: {', '.join(diffs)}" if diffs else "") or "all cases as expected"))

    # ---------------- per-item accuracy suites
    acc_suites = [s for s in ("calibration", "jev_native", "jev_systemone", "intents_banking77", "intents_clinc150",
                              "massive_spot", "massive", "xnli", "typed") if s in by]
    rows = {s: {n: [M.rows(r) for _, r in lst] for n, lst in by[s].items()} for s in acc_suites}
    noise = {}
    for n in names:
        ws = [_within(rows[s][n]) for s in acc_suites if n in rows[s] and len(rows[s][n]) > 1]
        ws = [w for w in ws if w is not None]
        noise[n] = statistics.mean(ws) if ws else None
    L += ["## Suites", "", "| suite | target | runs: correct / n | mean accuracy | ECE10 | Brier | AUROC | wall p50 ms |",
          "|---|---|---|---|---|---|---|---|"]
    for s in acc_suites:
        for n in names:
            if n not in rows[s]:
                continue
            sums = [M.summary(r) for r in rows[s][n]]
            ok = [x for x in sums if x.get("n")]
            if not ok:
                continue
            accs = [x["accuracy"] for x in ok]
            avg = lambda k: (statistics.mean([x[k] for x in ok if x.get(k) is not None])  # noqa: E731
                             if any(x.get(k) is not None for x in ok) else None)
            L.append(f"| {s} | {n} | {', '.join(str(x['correct']) for x in ok)} / {ok[0]['n']} | {statistics.mean(accs):.3f} | "
                     f"{_f(avg('ece10'))} | {_f(avg('brier'))} | {_f(avg('auroc'))} | {_f(avg('wall_p50'), 0)} |")
    L.append("")

    for s in acc_suites:
        for n in cands:
            if n not in rows[s] or not rows[s][n]:
                continue
            cand = rows[s][n]
            mean_c = statistics.mean([M.summary(r)["accuracy"] for r in cand if r])
            if s == "massive_spot":
                bylang = defaultdict(list)
                for r in cand[0]:
                    bylang[r["id"].split("-")[1]].append(r["accurate"])
                low = {lg: sum(v) / len(v) for lg, v in bylang.items() if sum(v) / len(v) < th["multilingual_min_accuracy"]}
                gates.append((s, n, "FAIL" if low else "PASS",
                              ", ".join(f"{lg} {sum(v)}/{len(v)}" for lg, v in sorted(bylang.items()))))
                continue
            if base and base in rows[s]:
                bl = rows[s][base]
                mean_b = statistics.mean([M.summary(r)["accuracy"] for r in bl if r])
                within = [w for w in (_within(cand), _within(bl)) if w is not None]
                floor = statistics.mean(within) if within else statistics.mean([v for v in noise.values() if v] or [0.94])
                cross = _cross(cand, bl)
                mc = M.mcnemar(list(zip(cand, bl)))
                fails = []
                # allowance = fixed margin + sampling error of an agreement rate on n items (matters for small suites)
                n_items = len(cand[0]) or 1
                allow = th["agreement_margin_pp"] / 100 + 2 * (max(floor * (1 - floor), 0.0) / n_items) ** 0.5
                if cross is not None and cross < floor - allow:
                    fails.append(f"agreement {cross:.3f} < noise floor {floor:.3f} - {allow:.3f}")
                if mc["p"] < th["mcnemar_p"]:
                    fails.append(f"McNemar p={mc['p']:.2g} ({mc['a_only']} vs {mc['b_only']})")
                v = "PASS" if not fails else ("REVIEW" if len(fails) == 1 else "FAIL")
                frozen = " [frozen set]" if s in ("massive", "xnli", "typed") else ""
                gates.append((s, n, v, f"acc {mean_c:.3f} vs {mean_b:.3f}; agreement {_f(cross)} (floor {_f(floor)}, allowance {allow:.3f}); "
                                       f"McNemar p={mc['p']:.2g}{frozen}" + (" — " + "; ".join(fails) if fails else "")))
            elif s in ref and "accuracy" in ref[s]:
                lo, hi = ref[s]["accuracy"]
                slack = th["reference_slack_items_frac"]
                v = "PASS" if mean_c >= lo - 1e-4 else ("REVIEW" if mean_c >= lo - slack else "FAIL")  # 1e-4: ranges are rounded
                gates.append((s, n, v, f"acc {mean_c:.3f}; reference {lo:.3f}–{hi:.3f} ({matrix['reference'].get('image')})"))
            else:
                gates.append((s, n, "INFO", f"acc {mean_c:.3f}; no baseline or reference"))

    # ---------------- noise floor
    if any(len(rows[s].get(n, [])) > 1 for s in acc_suites for n in names):
        L += _noise_table(rows, acc_suites, names, noise)

    # ---------------- held-out calibration (T-cal)
    L += ["## Calibration: raw vs held-out temperature (5-fold, folds by case; first run)", "",
          "| suite | target | n | ECE raw | ECE held-out | NLL raw | NLL held-out | T per fold |", "|---|---|---|---|---|---|---|---|"]
    for s in acc_suites:
        for n in names:
            if rows[s].get(n):
                h = M.heldout_temperature(rows[s][n][0])
                if h:
                    L.append(f"| {s} | {n} | {h['n']} | {_f(h['ece_raw'])} | {_f(h['ece_heldout'])} | {_f(h['nll_raw'])} | "
                             f"{_f(h['nll_heldout'])} | {', '.join(f'{t:.2f}' for t in h['T'])} |")
    L.append("")

    # ---------------- multilingual per language
    for s in ("massive", "xnli"):
        if s not in rows:
            continue
        L += [f"## {s} per language (accuracy / mean confidence)", "", "| lang | " + " | ".join(rows[s]) + " |",
              "|---|" + "---|" * len(rows[s])]
        per = {}
        for n, rr in rows[s].items():
            g = defaultdict(list)
            for r in rr[0]:
                g[r["id"].split("-")[1]].append(r)
            per[n] = {lg: (sum(x["accurate"] for x in v) / len(v), sum(x["confidence"] for x in v) / len(v)) for lg, v in g.items()}
        for lg in sorted(next(iter(per.values()))):
            L.append(f"| {lg} | " + " | ".join(f"{per[n][lg][0]:.2f} / {per[n][lg][1]:.2f}" if lg in per[n] else "—" for n in per) + " |")
        L.append("| **macro** | " + " | ".join(f"**{statistics.mean(a for a, _ in per[n].values()):.3f}**" for n in per) + " |")
        L.append("")

    # ---------------- typed-decisions soft metrics
    if "typed" in rows:
        L += ["## typed-decisions (soft metrics vs the teacher distribution)", "",
              "| target | accuracy | soft acc | soft Brier | score MAE |", "|---|---|---|---|---|"]
        for n, rr in rows["typed"].items():
            x = M.summary(rr[0])
            L.append(f"| {n} | {_f(x.get('accuracy'))} | {_f(x.get('soft_acc'))} | {_f(x.get('soft_brier'))} | {_f(x.get('abs_err_ev'))} |")
        L.append("")

    # ---------------- option order
    if "order" in by:
        L += ["## Option order (flip = answer changes; net = minus the identical-repeat flip)", "",
              "| target | subset | n | identity flip | random flip (net) | reverse flip (net) | picks first shown / gold first shown |",
              "|---|---|---|---|---|---|---|"]
        net_by = defaultdict(dict)
        for n, lst in by["order"].items():
            perm = {e.get("perm"): M.rows(r) for e, r in lst}
            for sub in sorted({r["suite"] for r in perm.get("none", [])}):
                P = {k: [r for r in v if r["suite"] == sub] for k, v in perm.items()}
                fi = M.flip_rate(P["none"], P.get("none2", [])) or 0.0
                fr = M.flip_rate(P["none"], P.get("random", []))
                fv = M.flip_rate(P["none"], P.get("reverse", []))
                first = statistics.mean([r["actual"] == r.get("shown_first") for r in P["none"]])
                gfirst = statistics.mean([r["expected"] == r.get("shown_first") for r in P["none"]])
                net_by[n][sub] = (fr or 0) - fi
                L.append(f"| {n} | {sub} | {len(P['none'])} | {fi:.3f} | {_f(fr)} ({(fr or 0) - fi:+.3f}) | "
                         f"{_f(fv)} ({(fv or 0) - fi:+.3f}) | {first:.3f} / {gfirst:.3f} |")
        L.append("")
        for n in cands:
            if base and base in net_by and n in net_by:
                worse = [f"{s} {net_by[n][s] - net_by[base][s]:+.3f}" for s in net_by[n] if s in net_by[base]
                         and net_by[n][s] - net_by[base][s] > th["order_net_flip_delta_review"]]
                gates.append(("order", n, "REVIEW" if worse else "PASS", "net flip vs baseline: " + (", ".join(worse) or "within tolerance")))
            elif n in net_by:
                gates.append(("order", n, "INFO", ", ".join(f"{s} net {v:+.3f}" for s, v in net_by[n].items())))

    # ---------------- summary-only suites
    for s, keys in (("bbox", ("acc_at_50_expectation_pct", "mean_expectation_iou")),
                    ("decision_index", ("headline_decision_index", "ece_10bin"))):
        if s in by:
            for n, lst in by[s].items():
                r = lst[0][1]
                r = r.get("dgem_wide_canvas_report", r)
                gates.append((s, n, "INFO", ", ".join(f"{k}={_f(r.get(k))}" for k in keys)))

    # ---------------- latency
    if "latency" in by:
        lat = {n: lst[0][1] for n, lst in by["latency"].items()}
        L += ["## Latency (/v1/systemone, keep-alive, seed 42)", "", "| target | mode | ok/n | wall p50 | wall p90 | wall p99 | server p50 | rps |",
              "|---|---|---|---|---|---|---|---|"]
        for n, d in lat.items():
            for m, x in {**d["modes"], **{"sweep " + k: v for k, v in d["sweep"].items()}}.items():
                L.append(f"| {n} | {m} | {x['ok']}/{x['n']} | {_f(x['wall_p50'], 1)} | {_f(x['wall_p90'], 1)} | {_f(x['wall_p99'], 1)} | "
                         f"{_f(x['server_p50'], 1)} | {_f(x.get('rps'), 1)} |")
        L.append("")
        for n in cands:
            if n not in lat:
                continue
            d = lat[n]
            errs = sum(x["errors"] for x in list(d["modes"].values()) + list(d["sweep"].values()))
            notes, v = [], "PASS"
            if errs:
                v, notes = "FAIL", [f"{errs} errors"]
            if base and base in lat:
                for m, x in d["modes"].items():
                    b = lat[base]["modes"].get(m)
                    if b and b["wall_p50"] and x["wall_p50"]:
                        r = x["wall_p50"] / b["wall_p50"]
                        if r > th["latency_p50_ratio_fail"]:
                            v, notes = "FAIL", notes + [f"{m} p50 x{r:.2f}"]
                        elif r > th["latency_p50_ratio_review"]:
                            v = "REVIEW" if v == "PASS" else v
                            notes.append(f"{m} p50 x{r:.2f}")
                for k, x in d["sweep"].items():
                    b = lat[base]["sweep"].get(k)
                    if b and b.get("rps") and x.get("rps") and x["rps"] / b["rps"] < th["throughput_ratio_min"]:
                        v = "REVIEW" if v == "PASS" else v
                        notes.append(f"{k} throughput x{x['rps'] / b['rps']:.2f}")
            gates.append(("latency", n, v, "; ".join(notes) or "within tolerance"))

    overall = {}
    for g, n, v, _ in gates:
        if RANK[v] > RANK[overall.get(n, "PASS")]:
            overall[n] = v
    head = ["## Verdicts", "", "| gate | target | verdict | detail |", "|---|---|---|---|"]
    head += [f"| {g} | {n} | **{v}** | {d} |" for g, n, v, d in gates]
    head += ["", "Overall: " + ", ".join(f"**{n}: {overall.get(n, 'PASS')}**" for n in cands), ""]
    i = L.index("## Suites")
    L = L[:i] + head + L[i:]
    summary = {"run_id": man["run_id"], "matrix_version": mx["matrix_version"], "tier": mx["tier"], "baseline": base,
               "overall": {n: overall.get(n, "PASS") for n in cands}, "noise_floor": noise,
               "gates": [{"gate": g, "target": n, "verdict": v, "detail": d} for g, n, v, d in gates]}
    return "\n".join(L) + "\n", summary
