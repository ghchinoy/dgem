#!/usr/bin/env python3
"""Versioned benchmark runs: record receipts into benchmarks/runs/<run_id>/ and compare them.

Layout
------
benchmarks/runs/<run_id>/manifest.json   run metadata + one entry per receipt
benchmarks/runs/<run_id>/<suite>__<config>.json   receipts written by `dgem bench-*`

A run id is "<YYYYMMDD>-<label>", e.g. "20260925-vertex-idc". Receipts are never overwritten;
start a new run id for a new session. Legacy receipts are registered in the
"20260920-legacy" run by path (they are not moved).

Commands
--------
  exec    run a dgem command whose output flag points into the run dir, then record it
  record  register an already-written receipt
  compare print a comparison table across runs/receipts for one suite
  list    list runs and receipts

Examples
--------
  scripts/bench_runs.py exec --run 20260925-vertex-idc --suite calibration --config baseline -- \
      ./bin/dgem bench-calibration --vertex-url 4217256562927861760 --gcp-auth -w 4 -o {out}
  scripts/bench_runs.py compare --suite calibration
"""

import argparse
import datetime as dt
import hashlib
import json
import math
import os
import subprocess
import sys

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
RUNS = os.path.join(REPO, "benchmarks", "runs")


def git(*args):
    try:
        return subprocess.check_output(["git", *args], cwd=REPO, text=True).strip()
    except Exception:
        return ""


def manifest_path(run_id):
    return os.path.join(RUNS, run_id, "manifest.json")


def load_manifest(run_id, create=False):
    p = manifest_path(run_id)
    if os.path.exists(p):
        with open(p) as f:
            return json.load(f)
    if not create:
        raise SystemExit(f"unknown run {run_id}")
    os.makedirs(os.path.dirname(p), exist_ok=True)
    return {
        "run_id": run_id,
        "created": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
        "git_commit": git("rev-parse", "HEAD"),
        "git_dirty_paths": [l[3:] for l in git("status", "--porcelain", "--", "cmd", "pkg", "templates").splitlines()],
        "notes": "",
        "receipts": [],
    }


def save_manifest(m):
    with open(manifest_path(m["run_id"]), "w") as f:
        json.dump(m, f, indent=2)
        f.write("\n")


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 16), b""):
            h.update(chunk)
    return h.hexdigest()


def summarize(path):
    """Suite-agnostic headline numbers recomputed from per-case data at T=1."""
    with open(path) as f:
        d = json.load(f)
    cases = d.get("cases", [])  # errored cases count as incorrect, as in the harness
    out = {
        "timestamp": d.get("timestamp"),
        "target_model": d.get("target_model") or d.get("model"),
        "idc_config": d.get("idc_config"),
        "n": len(cases),
        "errors": sum(1 for c in cases if c.get("error")),
    }
    if not cases or "accurate" not in cases[0]:
        return out
    applied = (d.get("jev_parity") or {}).get("temperature_applied") or d.get("temperature_scale") or 1.0
    correct = sum(1 for c in cases if c.get("accurate"))
    briers, confs, accs, tvds = [], [], [], []
    for c in cases:
        tp = c.get("top_probabilities") or {}
        if tp:
            raw = {k: max(v, 1e-12) ** applied for k, v in tp.items()}
            s = sum(raw.values())
            p = {k: v / s for k, v in raw.items()}
            conf = max(p.values())
            exp = str(c.get("expected", "")).lower()
            match = [k for k in p if k.lower() == exp]
            if match:
                briers.append(sum((v - (1.0 if k == match[0] else 0.0)) ** 2 for k, v in p.items()))
        else:
            conf = c.get("confidence", 0.0)
        confs.append(conf)
        accs.append(1.0 if c.get("accurate") else 0.0)
        idc = c.get("idc") or {}
        if idc.get("has_mirror"):
            tvds.append(idc.get("mirror_tvd", 0.0))
    bins = [[] for _ in range(10)]
    for cf, a in zip(confs, accs):
        bins[min(int(cf * 10), 9)].append((cf, a))
    ece = sum(len(b) / len(confs) * abs(sum(x for x, _ in b) / len(b) - sum(y for _, y in b) / len(b)) for b in bins if b)
    hi = [a for cf, a in zip(confs, accs) if cf > 0.9]
    jp = d.get("jev_parity") or {}
    out.update({
        "correct": correct,
        "accuracy": correct / len(cases),
        # Prefer the harness's own T=1 metrics (these handle yes/no vs true/false label mapping).
        "brier_t1": jp.get("raw_t1_brier_mean", (sum(briers) / len(briers)) if briers else None),
        "ece10_t1": jp.get("raw_t1_ece_10bin", ece),
        "hi_conf": f"{int(sum(hi))}/{len(hi)}",
        "mean_mirror_tvd": (sum(tvds) / len(tvds)) if tvds else None,
    })
    return out


def cmd_record(args, receipt_path=None, command=None):
    m = load_manifest(args.run, create=True)
    path = receipt_path or args.path
    rel = os.path.relpath(os.path.abspath(path), os.path.join(RUNS, args.run))
    entry = {
        "suite": args.suite,
        "config": args.config,
        "path": rel,
        "sha256": sha256(path),
        "recorded": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
        "command": command or args.command or "",
        "summary": summarize(path),
    }
    m["receipts"] = [r for r in m["receipts"] if not (r["suite"] == args.suite and r["config"] == args.config)]
    m["receipts"].append(entry)
    save_manifest(m)
    s = entry["summary"]
    print(f"recorded {args.run}/{args.suite}/{args.config}: n={s.get('n')} acc={s.get('accuracy')} brier={s.get('brier_t1')}")


def cmd_exec(args):
    run_dir = os.path.join(RUNS, args.run)
    os.makedirs(run_dir, exist_ok=True)
    out = os.path.join(run_dir, f"{args.suite}__{args.config}.json")
    if os.path.exists(out) and not args.force:
        raise SystemExit(f"{out} exists; receipts are immutable (use a new run id, or --force)")
    cmd = [c.replace("{out}", out) for c in args.cmd]
    if cmd and cmd[0] == "--":
        cmd = cmd[1:]
    print("+", " ".join(cmd), flush=True)
    rc = subprocess.call(cmd, cwd=REPO)
    if rc != 0 or not os.path.exists(out):
        raise SystemExit(f"command failed (rc={rc}); nothing recorded")
    cmd_record(args, receipt_path=out, command=" ".join(cmd))


def fmt(v, spec):
    return "—" if v is None else format(v, spec)


def cmd_compare(args):
    rows = []
    for run_id in sorted(os.listdir(RUNS)):
        mp = manifest_path(run_id)
        if not os.path.exists(mp):
            continue
        with open(mp) as f:
            m = json.load(f)
        for r in m["receipts"]:
            if args.suite and r["suite"] != args.suite:
                continue
            rows.append((run_id, r))
    print(f"| run | suite | config | n | correct | acc | Brier (T=1) | ECE-10 (T=1) | >0.9 conf correct | mean Mirror TVD |")
    print("| :--- | :--- | :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
    for run_id, r in rows:
        s = r["summary"]
        print(f"| {run_id} | {r['suite']} | {r['config']} | {s.get('n')} | {s.get('correct', '—')} | {fmt(s.get('accuracy'), '.3f')} | "
              f"{fmt(s.get('brier_t1'), '.3f')} | {fmt(s.get('ece10_t1'), '.3f')} | {s.get('hi_conf', '—')} | {fmt(s.get('mean_mirror_tvd'), '.3f')} |")


def cmd_list(args):
    for run_id in sorted(os.listdir(RUNS)):
        mp = manifest_path(run_id)
        if os.path.exists(mp):
            with open(mp) as f:
                m = json.load(f)
            print(f"{run_id}  commit={m.get('git_commit', '')[:8]}{' (dirty)' if m.get('git_dirty_paths') or m.get('git_dirty') else ''}  {m.get('notes', '')}")
            for r in m["receipts"]:
                print(f"    {r['suite']:<14} {r['config']:<22} {r['path']}")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd_name", required=True)
    for name in ("exec", "record"):
        sp = sub.add_parser(name)
        sp.add_argument("--run", required=True)
        sp.add_argument("--suite", required=True)
        sp.add_argument("--config", required=True)
        if name == "record":
            sp.add_argument("--path", required=True)
            sp.add_argument("--command", default="")
        else:
            sp.add_argument("--force", action="store_true")
            sp.add_argument("cmd", nargs=argparse.REMAINDER)
    sp = sub.add_parser("compare")
    sp.add_argument("--suite", default="")
    sub.add_parser("list")
    sp = sub.add_parser("note")
    sp.add_argument("--run", required=True)
    sp.add_argument("text")
    args = ap.parse_args()
    if args.cmd_name == "exec":
        cmd_exec(args)
    elif args.cmd_name == "record":
        cmd_record(args)
    elif args.cmd_name == "compare":
        cmd_compare(args)
    elif args.cmd_name == "list":
        cmd_list(args)
    elif args.cmd_name == "note":
        m = load_manifest(args.run, create=True)
        m["notes"] = args.text
        save_manifest(m)


if __name__ == "__main__":
    main()
