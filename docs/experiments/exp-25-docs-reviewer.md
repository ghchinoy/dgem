---
title: "EXP-25: dgem as a Documentation Reviewer"
description: "Pre-registered test of three dgem policies that review documentation sections: Diátaxis type, six house-style problems and whether measured results cite evidence. Compared with docstats (deterministic rules), gemini-3.8-flash and a hesitation-gated cascade; development results and the frozen human-labelled test plan."
---

# EXP-25: dgem as a Documentation Reviewer

**Status:** 🧪 Running. The method below is fixed. The development results in section 5 were measured before the test
set was labelled. The frozen test set (section 6) has not been scored.

- **Date:** 2026-10-03 (pre-registration and development runs).
- **dgem:** serving image v0.2.1 on the Vertex AI G4 dedicated endpoint (1× RTX PRO 6000), `/v1/systemone`,
  `samples=1`, default layout (`document_first`).
- **Gemini:** `gemini-3.8-flash` on Vertex AI (global), default thinking, JSON response schema with one enum per question.
- **docstats:** [`ghchinoy/docstats`](https://github.com/ghchinoy/docstats) `ai_patterns.detect_ai_patterns` (regex rules).
- **Code:** `scripts/docs_eval/`. **Policies:** `templates/docs/`. **Data:** `benchmarks/docs_eval/`.
- **Development receipts:** [`benchmarks/runs/20261003-exp25-dev`](../../benchmarks/runs/20261003-exp25-dev/report__dev_injected.md).

## 1. Question

Can dgem review documentation fast enough to run on every page and well enough to be useful? Specifically:

1. Does it identify a section's [Diátaxis](https://diataxis.fr/) type (tutorial, how-to, reference, explanation),
   plus two types this repository needs (results report, navigation)?
2. Does it flag six house-style problems from the `technical-post-editorial` guidelines: announcing instead of stating,
   contrast frames, hidden actors, staccato fragments, hand-holding and promotional tone?
3. Does it tell whether the measured results in a section cite a sample size or a source, a rule this repository
   applies to every number?
4. Does hesitation-gating to Gemini close the gap at a fraction of Gemini's latency and cost?

Em dashes, filler adverbs and readability grades are left out on purpose: deterministic tools such as docstats already
check them exactly.

## 2. Policies

All three policies read a section in one request, use only `choice` and `score` questions (no yes/no questions,
because of the [`PROP-19`](proposed.md) coupling defect on related yes/no pairs), and give every option a distinct,
descriptive name. Question ids are neutral ([EXP-16](exp-16-slot-names.md)).

| Policy | Questions | Options |
| :--- | :--- | :--- |
| `templates/docs/docs_diataxis.json.tmpl` | `doc_type`, `purity` | 6 types; `single_purpose` / `mostly_one_type` / `clearly_mixed` |
| `templates/docs/docs_style_audit.json.tmpl` | `openers`, `framing`, `actors`, `sentences`, `reader`, `tone`, `verdict` | clean option first, problem option second; verdict `ship` / `light_edit` / `rewrite` |
| `templates/docs/docs_claim_evidence.json.tmpl` | `evidence` | `no_measured_results` / `all_sourced` / `some_unsourced` / `illustrative_labelled` |

Each policy also runs from the CLI:
`dgem decide -t templates/docs/docs_style_audit.json.tmpl -v text="<section>"`.

**Unit:** one H2 section of a page in `docs/` (long H2 sections are split at H3 headings), with frontmatter removed,
code blocks cut to 4 lines and tables to 6 rows, at most 3,500 characters, and at least 40 words of prose. That gives
355 sections from 51 pages (`docs/history/` excluded).

## 3. Data

| Set | Size | Gold | Use |
| :--- | ---: | :--- | :--- |
| `label_set` (frozen test) | 150 sections + 15 repeats | one human labeller ([instructions](../../benchmarks/docs_eval/LABELING.md)) | scored once, after this page is merged |
| `dev_injected` | 25 clean sections × (original + 6 single-problem variants) = 175 | the injected problem | method choices, thresholds |
| `dev_pairs` | 10 sections before and after a docs rewrite commit | the rewrite is the weak "better" label | descriptive only |
| `diataxis_pages.jsonl` | 51 pages | page-level type labels, draft to be confirmed by the labeller | page scorecard |

- The test set is a stratified sample (seed 2025) by docs area: policies 34, deploy 26, operate 26, reference 19,
  confidence 15, root pages 15, experiments 15. It takes at most 5 sections per page. The 15 repeats are copies under
  new ids and measure the labeller's self-agreement.
- Development sections never appear in the test set. Seeds for `dev_injected` were drawn from the rest and filtered to
  docstats-clean prose (`ai_tell_score` ≥ 8).
- `gemini-3.8-flash` wrote the injected variants ("add exactly this problem, change nothing else").

**Known biases in the development set.**
- It favours docstats: the seeds are docstats-clean, and the injected phrases are the canonical ones its rules target.
- It favours Gemini: Gemini wrote the injections.
- It labels only the injected problem. An original section that already had a subtle problem counts as clean.

The frozen human-labelled set has none of these biases, which is why the decision rule uses it.

## 4. Pre-registered analysis (frozen test set)

**Configurations scored:**
- `dgem` with the frozen per-question thresholds: joint read, original order, run 1, thresholds from section 5.3.
- `dgem` raw (argmax).
- `gemini-3.8-flash`.
- `docstats`, on the four style rules it can see: `openers`, `framing`, `actors` (passive hints ≥ 2) and `sentences`.
- The hesitation-gated cascade at hesitation ≥ 0.35. Each question goes to Gemini when dgem's normalized entropy on
  it is at or above 0.35; otherwise dgem's thresholded answer is kept. 0.16, 0.25 and 0.50 are reported as
  sensitivity, not used for the decision.
- `dgem` with options shuffled and an identical repeat, to measure order effects and noise.

**Metrics:**
- Per question: accuracy with a bootstrap 95% interval, and for each style problem F1 on the problem class and AUROC of
  p(problem).
- Diátaxis type: accuracy, macro-F1 and Cohen's κ against the human.
- Hesitation AUROC for dgem's own errors.
- Flip rates; p50 latency per request and per section.
- The labeller's self-agreement on the 15 repeats, as the ceiling.
- Items the labeller marked unsure are scored, and also reported separately.

**Decision rule (fixed before scoring):**

| ID | Use | Passes if |
| :--- | :--- | :--- |
| D1 | dgem triages page type | `doc_type` accuracy ≥ Gemini − 0.10 and κ ≥ 0.40 |
| D2 | dgem as an advisory style linter on its own | mean F1 on the 4 shared rules ≥ docstats, and mean F1 on all 6 rules ≥ Gemini − 0.15 |
| D3 | hesitation-gated cascade | mean accuracy over all questions ≥ Gemini − 0.02, with ≤ 30% of answers handed to Gemini |
| D4 | evidence check | `evidence` accuracy ≥ 0.70 and ≥ the majority-class baseline + 0.10 |

- A question where the labeller's self-agreement is below 0.70 is reported but left out of D1–D4.
- If D2 or D3 passes, a follow-up PR proposes an advisory `make docs-lint` (report only, never blocking), using the
  configurations that passed. If D1 also passes, the lint includes page type.
- If nothing passes, the write-up says so and the policies stay as examples.

## 5. Development results (`dev_injected`, n=175 sections, 6 style questions)

These numbers set the method. They are not the test result. Receipts:
[`report__dev_injected.md`](../../benchmarks/runs/20261003-exp25-dev/report__dev_injected.md),
[`summary__dev_injected.json`](../../benchmarks/runs/20261003-exp25-dev/summary__dev_injected.json).

### 5.1 Raw dgem ranks well but over-flags

| Configuration | Mean accuracy (1,050 answers) | Hesitation AUROC for its errors | p50 per section |
| :--- | ---: | ---: | ---: |
| dgem, joint, original order (run 1 / run 2) | 0.813 / 0.815 | 0.75 / 0.74 | 1.1–1.2 s |
| dgem, one question per request | 0.796 | 0.64 | 3.2 s |
| dgem, options shuffled | 0.884 | 0.76 | 1.2 s |
| docstats (4 rules, 700 answers) | 0.917 | — | < 1 ms |
| gemini-3.8-flash | 0.958 | — | 16.6 s |

- Per-section latency covers the three policies, sent one after another from a client outside the endpoint's region.
- dgem's p(problem) ranks problem sections well: AUROC 0.96 for `openers`, 0.99 for `sentences` and `reader`, and
  1.00 for `tone`. It is weaker on `framing` (0.78) and `actors` (0.72).
- At the default argmax, dgem flags `openers`, `reader` and `sentences` on 70–75 of 175 sections. That includes almost
  every edited variant, whichever single problem was injected. dgem notices that a section has been made worse, but at
  argmax it does not say which rule was broken.
- Reading each question in its own request (0.796) does not remove the over-flagging, so it is not the cross-question
  coupling of `PROP-19`.
- Shuffling the options changed 13.1% of answers against 7.8% for an identical repeat (1,750 answers each). With two
  options, a shuffle always puts the problem option first, so this is the first-option habit at work.

### 5.2 dgem against docstats and Gemini, per rule (accuracy, F1 on the problem class)

| Rule | dgem raw | docstats | gemini-3.8-flash |
| :--- | :--- | :--- | :--- |
| `openers` | 0.714, F1 0.50 | 1.000, F1 1.00 | 0.897, F1 0.74 |
| `framing` | 0.880, F1 0.55 | 0.937, F1 0.72 | 0.977, F1 0.93 |
| `actors` | 0.874, F1 0.27 | 0.783, F1 0.54 | 0.931, F1 0.77 |
| `sentences` | 0.743, F1 0.53 | 0.949, F1 0.78 | 0.994, F1 0.98 |
| `reader` | 0.737, F1 0.52 | — | 0.954, F1 0.86 |
| `tone` | 0.931, F1 0.81 | — | 0.994, F1 0.98 |

### 5.3 Method choice: per-question thresholds

Because the ranking is good and the argmax is poor, the pre-registered dgem configuration flags a problem when
p(problem) reaches a per-question threshold. Each threshold maximizes F1 on `dev_injected`. Under 5-fold
cross-validation grouped by seed section (an original and its six variants stay in one fold):

| Rule | Frozen threshold | CV F1 | CV accuracy |
| :--- | ---: | ---: | ---: |
| `openers` | 0.9991 | 0.66 | 0.89 |
| `framing` | 0.0686 | 0.49 | 0.86 |
| `actors` | 0.0439 | 0.31 | 0.74 |
| `sentences` | 0.9455 | 0.90 | 0.97 |
| `reader` | 0.9864 | 0.87 | 0.96 |
| `tone` | 0.9937 | 0.98 | 0.99 |

The thresholds are frozen in [`benchmarks/docs_eval/thresholds.json`](../../benchmarks/docs_eval/thresholds.json)
(rubric hash `cea4c82b5ad6`). In-sample on development data, thresholded dgem reaches 0.92 mean accuracy. The
cross-validated figures above are the honest estimate.

### 5.4 Hesitation-gated cascade (offline, development data)

| Hesitation ≥ | Mean accuracy | 95% CI | Answers handed to Gemini |
| ---: | ---: | :--- | ---: |
| 0.16 | 0.950 | 0.936–0.962 | 30% |
| 0.25 | 0.949 | 0.935–0.962 | 22% |
| **0.35 (pre-registered)** | **0.945** | 0.931–0.958 | **17%** |
| 0.50 | 0.938 | 0.924–0.952 | 12% |

Gemini on every answer scored 0.958. The cascade's kept answers use the in-sample thresholds, so this table is
optimistic. The frozen test decides.

### 5.5 Before/after rewrites (`dev_pairs`, n=10)

- dgem's style verdict preferred the rewritten section in 5 of 10 pairs and the original in the other 5.
- Gemini rated 9 of 10 pairs the same and preferred the rewrite once.
- These rewrites were mostly content and structure changes, not style fixes, so the set says little about style. It
  is reported for completeness and not used.

## 6. Frozen test: status

- **Labelling:** 165 sections in `benchmarks/docs_eval/label_set.jsonl`, labelled with `scripts/docs_eval/labeler.html`
  following [`LABELING.md`](../../benchmarks/docs_eval/LABELING.md). Pending.
- **Page scorecard:** `benchmarks/docs_eval/diataxis_pages.jsonl` holds draft page-level labels from the 2026-10-04
  inventory, marked `"confirmed": false` until the labeller reviews them.
- **Scoring:** one run of each configuration in section 4 with `--frozen-ok`, then
  `python3 -m scripts.docs_eval.report --set label_set --labels benchmarks/docs_eval/labels.jsonl`. The results replace
  this section; any later change to labels or method is listed with its reason.

## 7. Reproduce

```bash
python3 -m scripts.docs_eval.build_sets label-set          # frozen test set and the dev pool
python3 -m scripts.docs_eval.build_sets inject -n 25       # dev_injected (needs GOOGLE_CLOUD_PROJECT)
python3 -m scripts.docs_eval.run --run-dir benchmarks/runs/<id> --set dev_injected --engine dgem --target <URL>
python3 -m scripts.docs_eval.run --run-dir benchmarks/runs/<id> --set dev_injected --engine gemini
python3 -m scripts.docs_eval.run --run-dir benchmarks/runs/<id> --set dev_injected --engine docstats  # DOCSTATS_DIR
python3 -m scripts.docs_eval.report --run-dir benchmarks/runs/<id> --set dev_injected
python3 -m scripts.docs_eval.report thresholds --run-dir benchmarks/runs/<id> --write
```

`<URL>` is a dgem serving base URL (`/invoke` base for a Vertex dedicated endpoint, `https://<CLOUD_RUN_URL>` or
`http://<GPU_HOST>:8080`). Run directories record only the kind of target, never the URL.
