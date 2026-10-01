#!/usr/bin/env python3
"""dgem regression matrix: run a tier of benchmarks against one or more serving targets and report verdicts.

  # Compare a candidate with production in the same session (serving-image change, release gate)
  scripts/bench_matrix.py run --tier T1 --target base=<prod-url> --target new=<candidate-url> --baseline base \
      --label image-abc123

  # Smoke-check one target (every deploy), or evaluate your own GPU against the published reference ranges
  scripts/bench_matrix.py run --tier T0 --target gpu=http://<GPU_HOST>:8080

  # Full matrix incl. frozen multilingual / typed-decisions / option-order sets (~25k requests per target)
  scripts/bench_matrix.py run --tier T2 --target prod=<url> --confirm

  scripts/bench_matrix.py report benchmarks/runs/<run_id>      # (re)build report.md + summary.json
  scripts/bench_matrix.py fetch                                # download + verify the pinned T2 datasets
  scripts/bench_matrix.py list                                 # tiers and suites

Target URLs: a Vertex dedicated invoke base (https://<ID>.<REGION>-<NUM>.prediction.vertexai.goog/v1/projects/<P>/
locations/<R>/endpoints/<ID>/invoke), a Cloud Run URL (https://<service>-<hash>-<region>.run.app, tagged revision URLs
work), or a self-hosted server (http://<GPU_HOST>:8080). Auth: see scripts/matrix/net.py (DGEM_MATRIX_TOKEN for an
API key). The run directory never contains URLs or tokens: receipts are redacted to <kind:name>.

Matrix definition: benchmarks/matrix/matrix_v1.json. Docs: docs/operate/regression-matrix.md.
"""
import argparse
import datetime as dt
import hashlib
import json
import os
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from matrix import cases as caselib  # noqa: E402
from matrix import datasets  # noqa: E402
from matrix import report as reportlib  # noqa: E402
from matrix import runners  # noqa: E402
from matrix.targets import Target  # noqa: E402

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
MATRIX = os.path.join(REPO, "benchmarks", "matrix", "matrix_v1.json")


def load_matrix(path=MATRIX):
    with open(path) as f:
        return json.load(f)


def runs_for(suite, tier):
    r = suite.get("runs", 1)
    return r.get(tier, 1) if isinstance(r, dict) else r


def git(*a):
    try:
        return subprocess.check_output(["git", *a], cwd=REPO, text=True).strip()
    except Exception:
        return ""


def sha(p):
    h = hashlib.sha256()
    with open(p, "rb") as f:
        for c in iter(lambda: f.read(1 << 16), b""):
            h.update(c)
    return h.hexdigest()


def log_event(enabled, **kw):
    """One structured line per event (Cloud Logging parses these on Cloud Run jobs)."""
    if enabled:
        print(json.dumps({"matrix_event": kw.pop("event"), **kw}), flush=True)
    else:
        print("  " + " ".join(f"{k}={v}" for k, v in kw.items()), file=sys.stderr, flush=True)


def find_dgem(arg):
    for c in (arg, os.path.join(REPO, "bin", "dgem"), "dgem"):
        if c and (os.path.exists(c) or subprocess.run(["which", c], capture_output=True).returncode == 0):
            return c
    raise SystemExit("dgem binary not found: build it (make build) or pass --dgem")


def cmd_run(a):
    mx = load_matrix(a.matrix)
    tier = a.tier
    if tier in mx.get("confirm_required", []) and not a.confirm:
        raise SystemExit(f"{tier} sends ~25k requests per target to shared GPUs; re-run with --confirm")
    suites = [s for s in mx["tiers"][tier] if not a.only or s in a.only]
    suites = [s for s in suites if s not in (a.skip or [])]
    specs = a.target or [t for t in os.environ.get("MATRIX_TARGETS", "").split(",") if t]
    if not specs:
        raise SystemExit("give --target name=url (or MATRIX_TARGETS=name=url,name=url)")
    targets = [Target(*t.split("=", 1)) for t in specs]
    if a.baseline and a.baseline not in [t.name for t in targets]:
        raise SystemExit(f"--baseline {a.baseline} is not one of the targets")
    run_id = f"{dt.datetime.now(dt.timezone.utc):%Y%m%d}-{a.label or tier.lower()}"
    run_dir = os.path.abspath(a.out_dir or os.path.join(REPO, "benchmarks", "runs", run_id))
    if os.path.exists(os.path.join(run_dir, "manifest.json")) and not a.resume:
        raise SystemExit(f"{run_dir} exists; choose another --label or pass --resume")
    os.makedirs(run_dir, exist_ok=True)
    needs_dgem = any(mx["suites"][s]["kind"] == "dgem" for s in suites)
    dgem_bin = find_dgem(a.dgem) if needs_dgem else None
    man_path = os.path.join(run_dir, "manifest.json")
    man = json.load(open(man_path)) if a.resume and os.path.exists(man_path) else {
        "run_id": run_id, "created": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
        "git_commit": git("rev-parse", "HEAD"), "notes": a.note or "", "receipts": [],
        "matrix": {"matrix_version": mx["version"], "tier": tier, "suites": suites, "baseline": a.baseline,
                   "targets": [t.redacted() for t in targets],
                   "started": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")}}
    done = {(e["suite"], e["config"], e.get("run", 1), e.get("perm")) for e in man["receipts"]}

    def save():
        with open(man_path, "w") as f:
            json.dump(man, f, indent=2)
            f.write("\n")

    def record(suite, t, run, perm, path, seconds, kind):
        runners.redact_file(path, t)
        man["receipts"].append({"path": os.path.relpath(path, run_dir), "suite": suite, "config": t.name, "run": run,
                                "perm": perm, "kind": kind, "sha256": sha(path), "seconds": round(seconds, 1)})
        save()

    case_cache = {}

    def cases_for(name):
        if name not in case_cache:
            case_cache[name] = caselib.SUITES[name]()
        return case_cache[name]

    log_event(a.log_json, event="dgem.matrix.start", run_id=run_id, tier=tier, targets=len(targets),
              suites=",".join(suites))
    max_runs = max([runs_for(mx["suites"][s], tier) for s in suites] + [1])
    for r in range(1, max_runs + 1):
        order = targets if r % 2 else list(reversed(targets))  # alternate target order between runs
        for s in suites:
            spec = mx["suites"][s]
            if r > runs_for(spec, tier) or (spec["kind"] in ("health", "contract", "latency") and r > 1):
                continue
            if spec["kind"] == "latency":
                continue  # after all accuracy work, so it is not measured under the matrix's own load
            for t in order:
                perms = spec.get("permutes", [None])
                for perm in perms:
                    if (s, t.name, r, perm) in done:
                        continue
                    stem = f"{s}{'__' + perm if perm else ''}__{t.name}__r{r}"
                    out = os.path.join(run_dir, stem + ".json")
                    t0 = time.time()
                    try:
                        if spec["kind"] == "health":
                            runners.health(t, out)
                            hb = json.load(open(out))["body"]
                            for tt in man["matrix"]["targets"]:
                                if tt["name"] == t.name:
                                    tt["health"] = {k: hb.get(k) for k in ("version", "revision", "vllm_commit")}
                        elif spec["kind"] == "contract":
                            runners.contract(t, out)
                        elif spec["kind"] == "dgem":
                            runners.dgem(t, spec["args"], out, dgem_bin, workers=a.workers)
                        elif spec["kind"] == "systemone":
                            p = "none" if perm in (None, "none2") else perm
                            runners.systemone(t, cases_for(spec["cases"]), out, workers=a.workers * 2, permute=p, run=r,
                                              suite=s)
                        record(s, t, r, perm, out, time.time() - t0, spec["kind"])
                        log_event(a.log_json, event="dgem.matrix.suite", run_id=run_id, suite=s, target=t.name, run=r,
                                  perm=perm or "", seconds=round(time.time() - t0, 1), status="ok")
                    except Exception as e:
                        log_event(a.log_json, event="dgem.matrix.suite", run_id=run_id, suite=s, target=t.name, run=r,
                                  perm=perm or "", status="error", error=str(e)[:300])
                        if a.fail_fast:
                            raise
    if "latency" in suites:
        for t in targets:
            if ("latency", t.name, 1, None) in done:
                continue
            out = os.path.join(run_dir, f"latency__{t.name}__r1.json")
            t0 = time.time()
            runners.latency(t, out, n=a.latency_n)
            record("latency", t, 1, None, out, time.time() - t0, "latency")
            log_event(a.log_json, event="dgem.matrix.suite", run_id=run_id, suite="latency", target=t.name, run=1,
                      seconds=round(time.time() - t0, 1), status="ok")
    man["matrix"]["finished"] = dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")
    save()
    summary = write_report(run_dir, mx)
    for g in summary["gates"]:
        log_event(a.log_json, event="dgem.matrix.gate", run_id=run_id, tier=tier, **g)
    log_event(a.log_json, event="dgem.matrix.done", run_id=run_id, tier=tier,
              overall=json.dumps(summary["overall"]), report=os.path.join(run_dir, "report.md"))
    if a.upload:
        dest = upload(run_dir, a.upload.rstrip("/") + "/" + os.path.basename(run_dir))
        log_event(a.log_json, event="dgem.matrix.uploaded", run_id=run_id, destination=dest)
    print(f"\n{run_dir}/report.md\n" + "\n".join(f"  {n}: {v}" for n, v in summary["overall"].items()), file=sys.stderr)
    return 1 if any(v == "FAIL" for v in summary["overall"].values()) else 0


def upload(run_dir, dest):
    """Copy a run directory to gs://bucket/prefix with the Cloud Storage JSON API (standard library only)."""
    import urllib.parse
    from matrix import net
    if not dest.startswith("gs://"):
        raise SystemExit("--upload needs gs://bucket/prefix")
    bucket, _, prefix = dest[5:].partition("/")
    tok = net._mint("access", None) if not os.environ.get("DGEM_MATRIX_TOKEN") else os.environ["DGEM_MATRIX_TOKEN"]
    import urllib.request
    for root, _, files in os.walk(run_dir):
        for fn in files:
            p = os.path.join(root, fn)
            name = f"{prefix}/{os.path.relpath(p, run_dir)}".lstrip("/")
            ctype = "text/markdown" if fn.endswith(".md") else "application/json" if fn.endswith(".json") else "text/plain"
            url = (f"https://storage.googleapis.com/upload/storage/v1/b/{bucket}/o?uploadType=media&name="
                   + urllib.parse.quote(name, safe=""))
            with open(p, "rb") as f:
                req = urllib.request.Request(url, data=f.read(), method="POST",
                                             headers={"Authorization": f"Bearer {tok}", "Content-Type": ctype})
            urllib.request.urlopen(req, timeout=120).read()
    return f"gs://{bucket}/{prefix}"


def write_report(run_dir, mx):
    md, summary = reportlib.build(run_dir, mx)
    with open(os.path.join(run_dir, "report.md"), "w") as f:
        f.write(md)
    with open(os.path.join(run_dir, "summary.json"), "w") as f:
        json.dump(summary, f, indent=1)
    return summary


def cmd_report(a):
    s = write_report(os.path.abspath(a.run_dir), load_matrix(a.matrix))
    print(open(os.path.join(a.run_dir, "report.md")).read())
    return 1 if any(v == "FAIL" for v in s["overall"].values()) else 0


def cmd_fetch(a):
    bad = 0
    for repo, file, ok in datasets.fetch(a.repo):
        if ok is not True:
            bad += 1
            print(f"FAIL {repo}/{file}: {ok}")
    print(f"{'all files verified' if not bad else f'{bad} files failed'} (cache {datasets.CACHE})")
    return 1 if bad else 0


def cmd_list(a):
    mx = load_matrix(a.matrix)
    for t, ss in mx["tiers"].items():
        print(f"{t}{' (needs --confirm)' if t in mx.get('confirm_required', []) else ''}: {', '.join(ss)}")
    print()
    for s, spec in mx["suites"].items():
        print(f"  {s:18s} {spec['kind']:9s} {'frozen ' if spec.get('frozen') else '       '}{spec.get('description', '')}")
    return 0


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--matrix", default=MATRIX)
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run")
    r.add_argument("--tier", required=True, choices=["T0", "T1", "T2"])
    r.add_argument("--target", action="append", help="name=url (repeat; or MATRIX_TARGETS=name=url,...)")
    r.add_argument("--baseline", help="target name to compare the others against (same session)")
    r.add_argument("--label", help="run id suffix (default: tier)")
    r.add_argument("--out-dir", help="run directory (default benchmarks/runs/<date>-<label>)")
    r.add_argument("--only", nargs="*", help="run only these suites of the tier")
    r.add_argument("--skip", nargs="*", help="skip these suites")
    r.add_argument("--workers", type=int, default=4, help="workers per dgem harness (systemone suites use 2x)")
    r.add_argument("--latency-n", type=int, default=100)
    r.add_argument("--dgem", help="path to the dgem binary (default bin/dgem)")
    r.add_argument("--confirm", action="store_true", help="required for T2")
    r.add_argument("--resume", action="store_true", help="continue an interrupted run in the same directory")
    r.add_argument("--fail-fast", action="store_true")
    r.add_argument("--log-json", action="store_true", help="structured JSON event lines on stdout (scheduled jobs)")
    r.add_argument("--note", help="free text stored in the manifest")
    r.add_argument("--upload", help="gs://bucket/prefix: copy the run directory there when done (scheduled jobs)")
    p = sub.add_parser("report")
    p.add_argument("run_dir")
    f = sub.add_parser("fetch")
    f.add_argument("--repo", nargs="*")
    sub.add_parser("list")
    a = ap.parse_args()
    sys.exit({"run": cmd_run, "report": cmd_report, "fetch": cmd_fetch, "list": cmd_list}[a.cmd](a))


if __name__ == "__main__":
    main()
