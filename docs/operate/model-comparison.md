---
title: "Comparing dgem with Another Decision Model"
description: "The standard process when a new decision model appears: evidence levels, the step-by-step protocol, framing rules, the report template, and where each finding for dgem goes (issue, proposed experiment, matrix suite or re-run)."
---

# Comparing dgem with Another Decision Model

New decision models ("system one" models) appear every few weeks. This page is the standard process for comparing one
with dgem, so that every comparison is run the same way, reported in the same shape, and feeds dgem improvements
through one intake path. The measurement itself is the [regression matrix](regression-matrix.md): a comparison is a
matrix run in which the other model is a **competitor target**.

Worked example: [EXP-23, dgem vs Strands Decider 2B](../experiments/exp-23-strands-decider.md).

---

## 1. Evidence levels

Every number we cite about another model carries one of these levels, in the text or in the table.

| Level | What it means | How it is shown |
|---|---|---|
| **E0, self-reported** | The model's authors published it (blog, model card, leaderboard JSON) | Quoted with source and date in `benchmarks/competitors/<model>.json`. Never ranked against our measurements, never mixed into our tables |
| **E1, reproduced** | We ran their model with their harness and matched their published per-item results | State how many items matched (e.g. 229 of 231 predictions) |
| **E2, lab head-to-head** | Both models, same request bodies, same GPU type, same session, at least 3 runs, paired tests | The default for any claim that one model is better |
| **E3, frozen sets** | E2 plus one scored run on the matrix's frozen T2 sets, after the E2 report is written | Reported once; never used to tune anything |

A proxy is never a substitute: a community harness that serves the same base model is a different system and is
reported under its own name.

---

## 2. Protocol

Each step names its tool. Steps 1–4 happen before any scored run.

Comparisons run on demand, when a new model appears: one owner per comparison works in its own git worktree from
`origin/main`, reserves the EXP number in a small pull request at intake, and hands serving or matrix changes to the
regression-matrix owner rather than editing them in the comparison branch.

1. **Intake.** Create `benchmarks/competitors/<model>.json` from an existing profile: identity (code, weights and base
   model revisions, licence, architecture, size class), serving facts (command, `model` field, option and context
   limits), E0 numbers with sources, and the training-exposure map (step 4). Reserve the next free `EXP-XX` number (unused on `main` and in open pull requests) in the
   [ledger](../experiments/README.md) now, not when writing up.
2. **Deploy.** Pin every revision; bake weights into the image. Deploy only in a testing project (never the
   distribution project), on the **same GPU type** as the dgem target, IAM-only, `min-instances 0`. Note any change you
   had to make to run it, and tear it down at the end (step 9).
3. **Preflight.** `scripts/compare/preflight.py --competitor <name>=<url>#profile=<model>` sends the same items serially
   twice and concurrently once. If concurrency changes answers beyond the serial noise, the server is unsafe under load:
   run it serialized (`--competitor-workers 1`, or a lock in front of the model) and report that. Run it on the dgem
   target too; dgem's serial noise (about 6% of answers at `samples=1`) is part of the noise floor.
4. **Exposure.** Read the competitor's training-data inventory and fill the profile's `exposure` map for every matrix
   suite: `train` (same dataset), `calib` (used to fit its temperatures or select checkpoints), `near` (related task
   trained), `unseen`, `unknown` (not audited). The report splits accuracy by these tags, because in-domain and
   out-of-domain results answer different questions.
5. **Reproduce (E1, if they publish per-item results).** Run their published harness at their pinned commit against your
   deployment and match per-item predictions. This catches deployment mistakes before anything is compared.
6. **Head-to-head (E2).** One command, same session:

   ```bash
   scripts/bench_matrix.py --matrix benchmarks/matrix/matrix_v2.json run --tier TC \
     --target dgem=<dgem-url> --baseline dgem \
     --competitor <name>=<competitor-url>#profile=<model> --label compare-<model>
   ```

   Tier TC runs JevBench, the calibration and intent suites as `/v1/systemone` bodies, the mixed-polarity yes/no suite
   (in one request and one question per request), a multilingual spot check and latency, 3 runs each. Competitors run
   only `/v1/systemone` suites, directly (never through dgem's adapter), never receive dgem-only fields, and never get
   a verdict. Run the latency client in the same region as both services.
7. **Frozen sets (E3, optional).** After the E2 report is written, one T2 run with `--confirm` and the same
   `--competitor`. Report it once.
8. **Report.** `scripts/compare/report.py benchmarks/runs/<run_id>` writes `comparison.md` with every measured table of
   the template below and TODO markers for the narrative. Keep the internal draft in the gitignored `scratch/` until
   reviewed; the public version is the EXP page.
9. **Tear down and record.** Delete the service, jobs and images; list them in the report's last section.

---

## 3. Framing rules

- State the **size class and hardware** of both models next to the headline numbers. dgem (26B-A4B MoE, 3.8B active) is
  rarely in the same size class as the model it is compared with.
- Report **in-domain and out-of-domain separately** (step 4). A small fine-tuned model matching dgem on its own training
  tasks is a different result from dgem leading on unseen tasks; both are usually true.
- Report **calibration as served and with a held-out temperature**, plus the share of answers at confidence ≥ 0.9 and
  their accuracy: that is what an "act automatically above 0.9" policy delivers.
- Use **paired tests** on shared items (exact McNemar) and treat differences inside the measured noise floor as
  unresolved. On JevBench that is about 10 of 231 items between single runs.
- **Label post-hoc diagnostics** (anything designed after seeing a result) as such. They explain; they do not score.
- **Suites we wrote ourselves** are named as ours, with n and how they were labelled.
- Public pages use placeholders only (`<PROJECT>`, `<ENDPOINT_ID>`, `<your-dgem-gateway>`); run `make check-public`.

---

## 4. Report template

Internal draft and public EXP page use the same sections, in this order:

1. **Bottom line**: 4–6 numbered findings, each with n and a paired test.
2. **What we compared**: identity, size class, hardware, revisions, server changes, preflight, runs per suite.
3. **Accuracy**: per suite, paired tests, split by training exposure, where each side wins.
4. **Calibration**: raw and held-out ECE, Brier, AUROC, coverage and accuracy at ≥ 0.9, out-of-domain behaviour.
5. **Latency and cost**: by input length and question count, throughput, cold start, cost per decision.
6. **Caveats**.
7. **Findings for dgem**: one row per finding with its type and destination (section 5 below).
8. **Upstream feedback** to the competitor's authors (defects found in their server or docs).
9. **Cloud resources**: every resource created, and its final state.

---

## 5. Intake: where findings for dgem go

Every row of a report's "Findings for dgem" table gets exactly one type and one destination. Nothing stays only in a
report.

| Type | What it is | Destination |
|---|---|---|
| **Bug** | Reproducible wrong behaviour in dgem | GitHub issue → fix PR → `CHANGELOG.md` (Unreleased) → canary + T1 matrix |
| **Hypothesis** | An improvement we expect but have not measured | `PROP-XX` in the [Proposed Experiments Register](../experiments/proposed.md), with the comparison in its Source column; it gets an `EXP-XX` number when it starts running |
| **Coverage** | A failure no matrix suite would catch | A suite proposal, added in the next matrix version (`benchmarks/matrix/matrix_vN.json`) and its reference run |
| **Re-run** | An internal benchmark whose published number is affected | A new run in `benchmarks/runs/<date>-rerun-<exp>`; the old EXP page gets a dated note, never a rewrite |
| **Doc** | A wrong or outdated statement in our docs | A direct PR |

If the same finding has already come up in an earlier comparison, add the new comparison to the existing entry's Source
column instead of opening a second one.

---

## 6. Machine-readable contract (for the Decision Board and other readers)

- **A comparison run is never a reference run.** Every comparison run keeps both markers: `tier: "TC"` in
  `summary.json` and `role: "competitor"` on each competitor in `targets[]`. Readers exclude such runs from dgem's
  reference ranges, history and release gates. An E3 frozen-set run with `--competitor` has `role: "competitor"`
  targets too, so it is excluded the same way.
- **Competitor targets are self-describing.** `targets[]` entries for competitors carry `profile`, `model`,
  `evidence_level` and `org`, copied from the profile when the run starts.
- **Paired results in `summary.json` `competitors.<target>`:** per suite accuracy, paired counts and McNemar p, plus
  `confident_share` / `confident_accuracy` (answers at confidence ≥ 0.9, first run) for the competitor and the
  baseline (`ref_*`); `by_exposure` gives accuracy by training exposure on the same items. **Canonical definition:**
  every per-item matrix suite of the run, so study-only suites outside the matrix file are excluded. A competitor's
  own published harness run (evidence level E1) goes in `reproduction.json` in the run directory and appears in
  `summary.json` as `reproduction`.
- **Profiles are a stable schema, `dgem.competitor/v1`.** Fields readers rely on: `schema`, `name`, `org`, `id`,
  `evidence_level`, `identity.{code,weights,base_model}.repo`, `identity.size_class`, `identity.architecture`,
  `serving.{model_field,max_choice_options,context_tokens,deterministic,concurrency_safe,concurrency_note}`,
  `self_reported[]`, `reproduced`, `exposure`, `exposure_note`. Adding fields is fine; renaming or removing one needs
  `dgem.competitor/v2`.

## 7. Where things live

| What | Where |
|---|---|
| Competitor profiles (identity, E0 numbers, exposure map) | `benchmarks/competitors/<model>.json` |
| Comparison suites | `scripts/matrix/compare_cases.py`; tier TC in `benchmarks/matrix/matrix_v2.json` |
| Preflight, exposure split, report skeleton | `scripts/compare/` |
| Runs and receipts | `benchmarks/runs/<date>-compare-<model>/` (URLs redacted by the matrix) |
| Public write-up | `docs/experiments/exp-XX-<model>.md`, listed in the ledger |
| Internal drafts, endpoint values | gitignored `scratch/` |
