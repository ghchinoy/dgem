---
title: "Proposed Experiments Register (Hypotheses & Designs)"
description: "Register of experiments we intend to run, each with a motivation, a falsifiable pre-registered hypothesis, a design, metrics, decision criteria, and dependencies. Seeded with the open questions from Invariant Decision Calibration (IDC)."
---

The [Experiment Ledger](/dgem/experiments/) records experiments we have **run**. This page records experiments we **intend to run**, written down *before* any results exist, so that the hypothesis, the metric, and the pass/fail line can't quietly move after the data comes in.

> **Naming note (2026-10-03):** entries written before this date say "IDC" for the in-pass order corrections (null-prior, dual-mirror). Those corrections are retired; the production approach is [hesitation-gating](/dgem/confidence/overview/), with order bias monitored per release.

**Lifecycle:** `Proposed` → `Ready` (dependencies met, design reviewed) → `Running` → `Done` (or `Closed` / `Parked` when superseded). When an entry starts running, give it the next free `EXP-XX` number, add it to the ledger, and link it back here. Keep the `PROP-XX` entry, but mark it `Done → EXP-XX` and note any change to the design along with the reason for it.

**Numbering:** claim the `EXP-XX` number by adding the ledger row (status `🧪 Running`) **when the run starts**, in the same change that marks the entry `Running`, so two teams never pick the same number. "Free" means unused on `main` **and** in every open pull request (`gh pr list`, then check each branch's ledger and this page). The same applies to `PROP-XX`.

**Where entries come from:** besides our own open questions, every [model comparison](/dgem/operate/model-comparison/#5-intake-where-findings-for-dgem-go) ends with a "Findings for dgem" table; each hypothesis in it becomes an entry here. The **Source** column records every study that raised it (`Order` = the order-bias research behind EXP-13–EXP-17, formerly called IDC; `Laya`, `Strands`, `Clef` = model comparisons; `DI` = Decision Index runs). When a later study raises the same point, add it to Source instead of opening a second entry.

---

## 1. Ground Rules for Every Entry

1. **Pre-register.** State the hypothesis, the primary metric, and the decision threshold before running. If you change them afterwards, say so explicitly.
2. **Report the sample size next to every number.** Include a 95% interval. Use a bootstrap over items for Brier/ECE/accuracy, and Wilson intervals for proportions.
3. **Hold data out.** Any parameter fitted on labeled data ($T^*$, cascade thresholds, `--prior-alpha`) must be evaluated on items it was not fitted on, either a held-out split or k-fold cross-validation.
4. **Compare like with like.** Run the baseline and the treatment in the **same session, against the same endpoint revision and model build**, over the same items. Record the endpoint URL, model path, and git SHA in the receipt.
5. **Measure the noise floor.** Where possible, repeat the baseline at least 3 times to see how much results move from run to run (see `PROP-10`).
6. **Leave a receipt.** Every run writes a `benchmarks/results_*.json` receipt with per-item probabilities, so figures can be regenerated from data (as `scripts/generate_idc_diagrams.py` does).

**Small-sample guidance:** 10-bin ECE is unreliable below roughly 200 items. Zero errors in *n* high-confidence items is still consistent with an error rate of up to about 3/*n* (the "rule of three"). Prefer paired comparisons: the same items under two conditions.

---

## 2. Register at a Glance

| ID | Title | Question it answers | Depends on | Source | Priority | Status |
| :--- | :--- | :--- | :--- | :--- | :---: | :--- |
| [`PROP-00`](#prop-00-expose-mirror-tvd-on-all-four-surfaces-enabler) | Expose Mirror TVD on all four surfaces (*enabler*) | Can production use the mirror signal at all? | — | Order | **P0** | Closed: not worth it after [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/) |
| [`PROP-10`](#prop-10-run-to-run-and-revision-noise-floor) | Run-to-run & revision noise floor | How big a difference is real? | — | Order | **P0** | Done → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (noise floor: ±1–2 items, Brier ±0.02) |
| [`PROP-01`](#prop-01-held-out-temperature-scaling) | Held-out temperature scaling | Does the ECE gain from $T^*$ survive out of sample? | — | Order | **P0** | Done → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (met on 231 JevBench items, not on 50) |
| [`PROP-02`](#prop-02-end-to-end-idc-cascade) | End-to-end IDC cascade | Does IDC + a mirror-aware gate beat the entropy-only cascade? | `PROP-00`, `PROP-01` | Order | **P0** | Partly → EXP-14 (entropy gates); mirror gate moves to `PROP-13` |
| [`PROP-03`](#prop-03-same-canvas-vs-separate-pass-mirror) | Same-canvas vs separate-pass mirror | Does sharing a canvas hide disagreement? | — | Order | P1 | **Closed**: replaced by `PROP-12` (done → [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/)) |
| [`PROP-04`](#prop-04-order-sensitivity-on-real-chaosnli) | Order sensitivity on real `ChaosNLI` | Do entropy / Mirror TVD / cyclic JSD track real human disagreement? | — | Order, Laya | P1 | Proposed |
| [`PROP-05`](#prop-05-mirror-merge-rule) | Mirror merge rule | Is the 70/30 forward-priority merge the right one? | `PROP-03` (optional) | Order | P1 | **Closed** → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (50/50 beats 70/30; neither beats single slot); mirror not used in production |
| [`PROP-06`](#prop-06-null-prior-strength-and-label-aware-priors) | Null-prior strength & label-aware priors | Can de-biasing improve calibration *without* raising order flips? | — | Order | P1 | **Closed**: in-pass order corrections retired after EXP-13–EXP-17; order bias is monitored per release instead ([overview §4.3](/dgem/confidence/overview/#43-order-bias-measured-and-monitored)) |
| [`PROP-07`](#prop-07-wording-invariance) | Wording invariance | How often does rephrasing an option change the decision? | — | Order | P2 | Proposed |
| [`PROP-08`](#prop-08-missing-context-detection) | Missing-context detection | Does uncertainty rise when a decisive fact is removed? | — | Order | P2 | Proposed |
| [`PROP-09`](#prop-09-base-rate-label-shift-adaptation) | Base-rate (label-shift) adaptation | Can unlabeled target traffic correct for different class frequencies? | — | Order | P2 | Proposed |
| [`PROP-11`](#prop-11-letter-collision-in-the-mirror) | Letter collision in the mirror | Is the mirror's damage caused by shared letters rather than by a second slot? | — | Order | **P0** | Done → [EXP-15](/dgem/experiments/exp-15-letter-collision/) (supported) |
| [`PROP-12`](#prop-12-separate-pass-mirror) | Separate-pass mirror (replaces `PROP-03`) | Does a mirror read in its own pass give a clean order signal? | `PROP-11` | Order | P1 | Done → [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/) (supported, small effect) |
| [`PROP-13`](#prop-13-mirror-aware-cascade) | Mirror-aware cascade (unblocks `PROP-02`) | Does an uncoupled mirror improve the hand-off gate? | `PROP-12` | Order | P1 | Closed: not worth it after [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/) |
| [`PROP-14`](#prop-14-in-context-vs-blank-question-prior) | In-context vs blank-question prior | Does a prior estimated from real items fix null-prior's overcorrection? | — | Order | P1 | Parked (research only): production monitors order rather than correcting it ([overview §4.3](/dgem/confidence/overview/#43-order-bias-measured-and-monitored)) |
| [`PROP-15`](#prop-15-correction-strength-by-question-type) | Correction strength by question type (extends `PROP-06`) | Do yes/no and lettered choices need different correction strengths? | `PROP-14` | Order | P2 | Parked (research only, depends on `PROP-14`); see [overview §4.3](/dgem/confidence/overview/#43-order-bias-measured-and-monitored) |
| [`PROP-16`](#prop-16-slot-names-are-part-of-the-prompt) | Slot names are part of the prompt | How much do slot ids change answers? | — | Order | P1 | Done → [EXP-16](/dgem/experiments/exp-16-slot-names/) (single slot: no; second slot: yes) |
| [`PROP-17`](#prop-17-calibrated-agent-context-pre-compiler-internal-pilot) | Calibrated agent context pre-compiler (internal pilot) | Can one multi-slot pass decide which context blocks an agent turn needs, dropping little that matters? | — | Pilot | P2 | Running (internal pilot) |
| [`PROP-18`](#prop-18-validate-image-metrics-beyond-exp-09) | Validate image metrics beyond EXP-09 | Which image aspects (location, presence/count, attributes/relations, quality/occlusion, domain checks) does one pass read reliably, judged against ground truth, Gemini 3.x and classical detectors? | — | DI | P1 | Phases 1–4 (pilot) done → [EXP-09 re-baseline](/dgem/experiments/exp-09-spatial-grounding/), [EXP-22](/dgem/experiments/exp-22-image-readouts/); real-image bench-vision, matrix image gate and live image cascade done; guided boxes and masks done → [EXP-24](/dgem/experiments/exp-24-guided-cascade/) |
| [`PROP-19`](#prop-19-cross-slot-coupling-on-mixed-polarity-yesno-questions) | Cross-slot coupling on mixed-polarity yes/no questions | Why do two yes/no answers in one request collapse to one label, and which serving change removes it? | — | Strands | **P0** | Proposed (bug [#68](https://github.com/ghchinoy/dgem/issues/68)) |
| [`PROP-20`](#prop-20-shipped-temperature-per-question-type-and-domain) | Shipped temperature per question type and domain | Does a held-out temperature per type (and per policy) as a served default fix dgem's raw overconfidence without hurting error detection? | `PROP-01` | Laya, Strands, DI | **P0** | **Closed (per-policy only)**: leave-one-suite-out on 8 dev suites, a global T and a per-type T each made ECE worse on 4 of 8; no served default. Per-policy `temperature` on all four surfaces instead |
| [`PROP-21`](#prop-21-fine-tuning-diffusiongemma-lora-on-the-letter-readout) | Fine-tuning DiffusionGemma: LoRA on the letter readout | Does a LoRA trained with a masked-denoising loss on answer slots raise in-domain accuracy and fix coupling without losing zero-shot breadth? | `PROP-19`, `PROP-20` | Laya, Strands, Clef | P1 | Proposed |
| [`PROP-22`](#prop-22-distil-dgem-into-a-small-student) | Distil dgem into a small student | Do dgem's soft labels train a better 2B decision model than hard labels? | `PROP-20` | Strands | P2 | Proposed |
| [`PROP-23`](#prop-23-nondeterminism-at-samples1) | Nondeterminism at `samples=1` | Where do the 5–7% answer changes between identical requests come from, and can serving remove them? | — | Laya, Strands | P1 | Proposed ([#82](https://github.com/ghchinoy/dgem/issues/82)) |
| [`PROP-24`](#prop-24-wide-option-accuracy-626-options-and-beyond) | Wide-option accuracy (6–26 options and beyond) | Why does accuracy fall with option count even in a single read, and which readout change recovers it? | — | DI, Clef, Strands | P1 | Proposed |
| [`PROP-25`](#prop-25-nli-neutral-pull) | NLI "neutral" pull | Why does dgem over-predict "neutral" on XNLI, and does wording or a per-label prior fix it on the validation split? | — | Laya, Strands | P2 | Proposed |
| [`PROP-26`](#prop-26-first-shown-bias-on-small-option-sets) | First-shown bias on small option sets (extends `PROP-14`) | Does the first listed option win too often at K ≤ 6, and does an in-context prior remove it? | `PROP-14` | Laya, Strands | P2 | Parked (research only, with `PROP-14`): order bias is monitored per release, not corrected |

### Queued re-runs

Re-runs re-measure a published number after a harness or serving change; they need no hypothesis, but they are
listed here so they are not lost. Each lands as a new run in `benchmarks/runs/` and a dated note on the affected EXP
page (history is annotated, not rewritten).

| Re-run | Why | Affects | Source | Status |
| :--- | :--- | :--- | :--- | :--- |
| JevBench through `bench-jev` with ordinal `score` items kept as `score` | `bench-jev` flattens score items to choice; the `/v1/systemone` path keeps them ordinal and scores about 4–5 more of the 18 score items | EXP-11, EXP-14 to EXP-17 headline counts | Laya | **Done** (2026-10-05): totals within noise; score items 14–15 vs 9–14 of 18 (`20261005-rerun-jev-score`) |
| `--auto-temperature` fitted by k-fold, not in-sample | In-sample T is optimistic on 50 items | EXP-04, EXP-05, EXP-11 calibration figures | Laya | **Done** (2026-10-05): held-out ECE no better than raw on every 50-item receipt; note on EXP-11 |
| Matrix v2 reference run (T1 + TC baseline only) | Sets reference ranges for `calib_systemone`, `intents_systemone`, `gate_mixed_noul` before v2 becomes the default | [Regression matrix](/dgem/operate/regression-matrix/) | Strands | **Done** (#87, 2026-10-03; re-referenced for v0.3.x) |
| EXP-23 on matrix v2 tier TC | The EXP-23 numbers came from the study harness that TC replaces; one TC run confirms the tooling reproduces them | [EXP-23](/dgem/experiments/exp-23-strands-decider/) | Strands | Queued (needs a competitor deployment): [#83](https://github.com/ghchinoy/dgem/issues/83) |


---

## 3. Entries

### `PROP-00`: Expose Mirror TVD on all four surfaces (*enabler*)

* **Closed (2026-09-28), not run.** EXP-15–EXP-17 showed the mirror signal adds little over hesitation: a
  separate-pass mirror improves cross-validated error detection by at most 0.016 AUROC at twice the cost, and the
  same-canvas mirror costs accuracy unless labels are digits. Exposing it on every surface (PROP-00) or gating the
  cascade on it (PROP-13) is not worth the complexity; hesitation remains the production signal. Reopen only if a
  cheaper, uncoupled order signal appears.
* **Motivation:** `--dual-mirror` computes Mirror TVD/JSD in `pkg/permutation.PostProcessDecisionResponse`, but `cmd/decide.go` discards the return values. `dgem serve`, MCP, and the Studio don't offer `dual_mirror` / `null_prior_debias` at all, which violates the four-surface parity rule in `AGENTS.md`. A mirror-aware gate (`PROP-02`) can't run in production until this is fixed.
* **Deliverable (not a hypothesis):**
  * Add `diagnostics.mirror_tvd` / `mirror_jsd` per slot to the `dgem decide` output.
  * Add `dual_mirror`, `null_prior_debias`, and `prior_alpha` to `GatewayDecideRequest`, `DecidePolicyToolInput` / `DecideCustomToolInput`, and the Studio.
  * Add a `cascade_mode` option that escalates on `mirror_tvd ≥ τ` (with `τ` configurable).
  * Wrap the post-processing in an OTel child span (`dgem.idc.postprocess`).
* **Acceptance:** The same request returns identical `mirror_tvd` on all four surfaces. Unit tests cover the merge and TVD computation.

### `PROP-01`: Held-out temperature scaling

* **Motivation:** The headline ECE gain (`0.075 → 0.033`, $T^* = 1.35$) was fitted and scored on the same 50 items (`--auto-temperature` on `results_calibration_cloudrun.json`).
* **Hypothesis (H1):** Under 5-fold cross-validation, temperature scaling still lowers ECE relative to $T = 1$, but by **less** than the in-sample figure. Pre-registered success line: the cross-validated ECE reduction is **≥ 25%** and its 95% bootstrap interval excludes 0.
* **Design:**
  * 5-fold CV on the 50-item calibration suite, fitting $T^*$ on 4 folds and scoring the 5th.
  * Repeat on the 231-item `JevBench v1.3.1` public split (`benchmarks/jevbench/`), which is large enough for a meaningful 10-bin ECE.
  * Also fit on JevBench and test on the calibration suite, to check transfer across suites.
* **Metrics:** 10-bin ECE, Brier score, NLL. Report the fitted $T^*$ for each fold to show how stable it is.
* **Decision:** If H1 fails, stop reporting in-sample ECE as a headline result and keep $T = 1$ as the default.
* **Cost:** No GPU needed. It re-scores existing receipts (`--from-receipt`), plus a small code change to add a `--cv-folds` option.

### `PROP-02`: End-to-end IDC cascade

* **Motivation:** The 98% (49/50) cascade result is entropy-only (`results_calibration_cascade_normalized.json`). The IDC page describes a combined gate (entropy **or** mirror disagreement), but it has never been run end to end.
* **Hypothesis (H2):** At a **matched escalation rate** (±3 points), a gate using null-prior + Dual-Mirror + (held-out) temperature, escalating on $\tilde{H} \ge \tau_H$ **or** $\text{TVD}_{\text{mirror}} \ge \tau_{\text{TVD}}$, achieves higher cascade accuracy than the entropy-only gate. Alternatively, at matched accuracy it escalates **fewer** items.
* **Design:**
  * Choose $\tau_H$ and $\tau_{\text{TVD}}$ on a tuning split; report on a held-out split.
  * Stage 2 is `gemini-3.8-flash` with prior forwarding, as in `EXP-05b`.
  * Use a suite of ≥ 200 items: the 50-item suite plus JevBench, plus the `PROP-04` items once available.
  * Compare: (a) no cascade, (b) entropy-only, (c) IDC + entropy gate, (d) IDC + entropy-or-TVD gate.
* **Metrics:** Accuracy, escalation rate, the accuracy-vs-escalation curve (area under the curve), p50/p95 latency, USD per 1,000 decisions, and wrong answers that exited early.
* **Decision:** If (d) doesn't beat (b) on the held-out split, describe Dual-Mirror as a diagnostic tool rather than a production gate.

### `PROP-03`: Same-canvas vs separate-pass mirror

* **Motivation:** On one canvas, the forward and reversed slots can see each other. In `perm_08`, adding the reversed slot moved the forward reading from 99.9% to 71.3%. The two readings may therefore be coupled, which could either hide disagreement (the slots copy each other) or manufacture it.
* **Hypothesis (H3):** Mirror TVD measured on a shared canvas is **systematically lower** than TVD between two separate single-slot passes (forward and reversed orders). Pre-registered: a paired Wilcoxon test over items gives p < 0.05, and the median ratio is < 0.8.
* **Design:**
  * For each item, run: (i) forward-only single slot, (ii) reversed-only single slot, (iii) the shared mirror canvas. Compare TVD(i, ii) with TVD(fwd, rev) from (iii).
  * Add a condition where the reversed slot's instructions forbid looking at the other slot. This tests whether an instruction reduces coupling.
  * Suite: the 16 `permutation_suite.jsonl` items, plus the `PROP-04` items.
* **Metrics:** Paired TVD difference, agreement between each gate's decisions (Cohen's κ), and latency.
* **Decision:** If the shared canvas masks disagreement, `--dual-mirror` should document the coupling or offer a two-pass mode for high-stakes slots.

### `PROP-04`: Order sensitivity on real `ChaosNLI`

* **Motivation:** The four "ambiguous" items in `EXP-13` are authored ("ChaosNLI-style"). The `EXP-04` 8× entropy figure rests on 3 items per group. Neither is enough to claim that uncertainty signals track human disagreement.
* **Hypothesis (H4):** Across ≥ 300 real `ChaosNLI` items (SNLI, MNLI, and αNLI subsets, each with ~100 annotator labels), the entropy of the human label distribution correlates positively with (a) single-pass $\tilde{H}$, (b) Mirror TVD, and (c) cyclic JSD. Pre-registered: Spearman ρ ≥ 0.3 for each, with a 95% interval excluding 0. **Secondary:** adding (b) or (c) to (a) improves the prediction of human entropy (partial ρ > 0).
* **Design:**
  * Sample items stratified by human entropy.
  * Run `bench-permutation`-style evaluation: every cyclic order, the null prior, and the Dual-Mirror canvas.
  * Add the items to a new `benchmarks/chaosnli_permutation_suite.jsonl`, recording the source dataset and version.
* **Metrics:** Spearman ρ, soft-label TVD against the human distribution, order-flip rate by human-entropy tercile, and the fraction of items with $\tilde{H} < 0.16$ that still flip under reordering (the "hidden toss-up rate").
* **Decision:** Replaces the `EXP-04` / `EXP-13` n = 3–4 figures in the docs. If ρ is weak, soften the claim that entropy "tracks human disagreement".

### `PROP-05`: Mirror merge rule

* **Motivation:** Production merges the forward and reversed readings as 70% / 30% with a forward-priority winner rule. On the 50-item suite, Dual-Mirror gave worse Brier (0.221 vs 0.186) and worse ECE (0.081 vs 0.075) than the baseline.
* **Hypothesis (H5):** A symmetric merge, either the arithmetic mean $\tfrac12(p^{\text{fwd}} + p^{\text{rev}})$ or the normalized geometric mean, gives a Brier score **no worse** than the single-slot baseline (non-inferiority margin 0.01) and better than the 70/30 rule.
* **Design:**
  * Re-score existing Dual-Mirror receipts offline. The raw forward and reversed probabilities must be stored in the receipt, so add them if they're missing.
  * Compare merges: 70/30 + forward priority, 50/50 arithmetic, geometric, and "forward only" (the mirror used for gating only).
* **Metrics:** Accuracy, Brier score, ECE (cross-validated where anything is fitted), and flip rate against the single-slot answer.
* **Cost:** No GPU needed once the receipts store both readings.

### `PROP-06`: Null-prior strength and label-aware priors

* **Motivation:** Null-prior de-biasing is the strongest IDC component on the 50-item suite (Brier 0.186 → 0.149). But on the ordering suite, at α = 0.75, it **doubled** the cyclic flip rate (12.5% → 25%). The prior is also measured with neutral `Option 1…K` labels rather than the real option text.
* **Hypotheses:**
  * **H6a:** An intermediate α (chosen by cross-validation from {0, 0.25, 0.5, 0.75, 1.0}) improves Brier over α = 0 **without** increasing the cyclic flip rate by more than 5 points.
  * **H6b:** A *label-aware* null prior (content-free input, but the template's real option labels) lowers the flip rate compared with the neutral-label prior at the same α.
* **Design:** Sweep α on the calibration suite and the ordering suite (plus `PROP-04` items), and compute a label-aware prior for each template.
* **Metrics:** Brier score, ECE, accuracy, cyclic flip rate, and cyclic JSD.

### `PROP-07`: Wording invariance

* **Motivation:** IDC only addresses option **order**. Rephrasing an option label or description ("benign" vs "authorized business activity") may shift decisions just as much.
* **Hypothesis (H7):** On ambiguous items, the paraphrase flip rate (answers changing under a meaning-preserving rewording of the options) is **at least as large** as the order flip rate. On clear-cut items, both are below 2%.
* **Design:**
  * Generate 3 paraphrases per option set (Gemini, human-checked), covering the 16 ordering items plus a 100-item sample of the calibration suite and `PROP-04` items.
  * Start from `bench-jev`'s existing paraphrase-consistency support.
  * Optionally test a "paraphrase mirror": a second slot on the same canvas using the paraphrased labels.
* **Metrics:** Paraphrase flip rate, paraphrase TVD, and their correlation with order TVD.
* **Decision:** If wording sensitivity is large, broaden what "invariant" covers in IDC (or say clearly that it doesn't).

### `PROP-08`: Missing-context detection

* **Motivation:** The docs use the example of a phishing message whose sender domain is omitted. It's unknown whether `dgem`'s uncertainty signals actually rise when a decisive field is removed, or whether the model stays confidently wrong in both orders.
* **Hypothesis (H8):** When a decisive field is removed, $\tilde{H}$ **or** Mirror TVD rises above the gate threshold on **≥ 50%** of items that were answered correctly with the field present.
* **Design:**
  * Build ~60 paired items where one field determines the label (phishing with/without `sender_domain`, refunds with/without order status, agent steps with/without the task goal).
  * The gold label for the "field removed" version is "cannot determine" or a known human split.
  * Evaluate with and without IDC.
* **Metrics:** Detection rate, false-certainty rate (> 90% confidence on items with the field removed), and change in uncertainty between each pair.
* **Decision:** If detection is poor, recommend explicit "is required information present?" slots in templates instead of relying on uncertainty.

### `PROP-09`: Base-rate (label-shift) adaptation

* **Motivation:** A score calibrated on a benchmark can be wrong for a deployment where class frequencies differ (1% fraud vs 50% fraud). There's currently no base-rate option.
* **Hypothesis (H9):** Using **unlabeled** target traffic, EM-based prior re-estimation (Saerens et al., 2002) lowers Brier on a skewed target split compared with no adjustment. Pre-registered: ≥ 10% relative Brier reduction on at least 2 of 3 skewed splits.
* **Design:**
  * Build skewed resamples of `banking77_26.jsonl` / `clinc150_26.jsonl` / the guardrail items (e.g., 5%, 50%, and 90% positive class).
  * Run once, then re-weight probabilities offline.
  * If it works, add a `--class-prior` option (explicit base rates) alongside the EM estimate.
* **Metrics:** Brier score, ECE, accuracy, and error in the estimated prior.

### `PROP-10`: Run-to-run and revision noise floor

* **Motivation:** The IDC comparisons in the docs mix runs from different days and Cloud Run revisions (`/mnt/gcs/dgemma` on 2026-09-20 vs `diffgemma-26b-a4b-it-q4` on 2026-09-23). Without a noise floor, a 2-point accuracy difference on 50 items can't be interpreted.
* **Hypothesis (H10):** Repeating the 50-item calibration suite 3× against one revision, and 1× against each of two revisions, gives accuracy variation of ≤ 2 points and Brier variation of ≤ 0.01 within a revision. Differences between revisions are reported separately.
* **Design:** Run the same items, same flags, and `-w 1` for determinism where possible, on Vertex AI (`<legacy-l4-endpoint>`) and Cloud Run.
* **Metrics:** Standard deviation within a revision and differences between revisions for accuracy, Brier score, ECE, and per-item answer agreement.
* **Decision:** Sets the minimum effect size that later entries (`PROP-01`–`PROP-09`) must exceed to count as real.

---

### `PROP-11`: Letter collision in the mirror

* **Motivation:** In EXP-14 the forward answer changed 65% of the time when the forward and reversed slots chose the same letter (which then means different options), versus 9% otherwise. See [EXP-14, "Why the mirror degrades"](/dgem/experiments/exp-14-idc-rerun/#why-the-mirror-degrades-and-null-prior-overcorrects-analysis).
* **Hypothesis (H11):** The mirror's damage to the forward reading comes from the two slots sharing letters with different meanings, not from the presence of a second slot.
* **Design:** JevBench (231), Vertex G4, baselines at the start, middle and end of the session. Conditions: (a) single slot; (b) two identical copies in the same order; (c) forward + reversed with letters (current); (d′) forward + reversed where the reversed slot is sent as a `score` question so the server labels it 1, 2, 3… (client-only way to remove shared letters); (e) reversed slot placed before the forward slot. Server-side condition (d), explicit labels that keep each option's original letter, is deferred until the server accepts caller-supplied labels.
* **Metrics:** forward-slot accuracy vs. (a); same-letter rate; how often the forward answer changes vs. (a); breakdown by question type and number of options.
* **Decision (pre-registered):** H11 is supported if (b) and (d′) are within ±2 items of (a) while (c) stays more than 3 items below (a). If (b) is also degraded, a second slot hurts by itself and the same-canvas mirror should be dropped in favour of `PROP-12`.
* **Cost:** about 1,200 requests. Client-only (`--mirror-mode`).
* **Amendment (2026-09-25, before any PROP-11 data):** the parity probe measured a JevBench noise floor of about ±3 items (baselines 187–192), so the ±2 band would fail on noise alone. Revised rule: run at least 3 baselines in the session; "within noise" means inside their min–max range; "degraded" means more than 3 items below the range minimum. Added a per-item test that is insensitive to run-to-run drift: among items where the two slots pick the same *letter* for different options, the forward answer should change (vs. the baseline majority answer) significantly more often than on other items, and this gap should vanish in `copy` and `reversed-digits`. Questions with more than 9 options fall back to letters in `reversed-digits` (0 such items in JevBench).

### `PROP-12`: Separate-pass mirror

* **Motivation:** Replaces `PROP-03`. A reversed reading in its own request cannot copy the forward slot.
* **Hypothesis (H12):** Forward vs. reversed disagreement from two separate passes adds error detection beyond hesitation (entropy).
* **Design:** JevBench and the 50-item suite: one pass in original order, one pass with options reversed. Record per-item disagreement (TVD) and argmax agreement.
* **Metrics:** partial correlation of disagreement with errors, controlling for hesitation; AUROC for error detection of hesitation alone vs. hesitation + disagreement.
* **Decision:** Supported if disagreement adds detection (partial correlation > 0 with a 95% bootstrap interval excluding 0). Forward accuracy equals baseline by construction; the cost is a second pass.
* **Pre-registered design (2026-09-26, before data):** JevBench 231 only (the 50-item suite has too few errors for detection metrics, and `bench-calibration` has no option-reversal flag). Vertex G4, one session, order: forward F1, reversed R1 (`bench-jev --flip-options`), forward F2, reversed R2. Primary: partial Spearman correlation between forward/reversed TVD and forward-pass error, controlling for forward hesitation, averaged over the four F×R pairings, 95% bootstrap CI (2,000 resamples). **Noise control:** the same statistic for F1 vs F2 (two forward passes), since Vertex is not fully deterministic; order disagreement must beat it. Secondary: 5-fold CV AUROC of logistic error models (hesitation vs. hesitation + TVD), and accuracy of the averaged forward+reversed distribution vs. forward alone. About 924 requests.
* **Cost:** about 460 requests.

### `PROP-13`: Mirror-aware cascade

* **Closed (2026-09-28), not run.** EXP-15–EXP-17 showed the mirror signal adds little over hesitation: a
  separate-pass mirror improves cross-validated error detection by at most 0.016 AUROC at twice the cost, and the
  same-canvas mirror costs accuracy unless labels are digits. Exposing it on every surface (PROP-00) or gating the
  cascade on it (PROP-13) is not worth the complexity; hesitation remains the production signal. Reopen only if a
  cheaper, uncoupled order signal appears.
* **Motivation:** Unblocks `PROP-02` once an uncoupled mirror exists.
* **Hypothesis (H13):** A gate of hesitation ≥ τ **or** disagreement ≥ τ′ beats entropy alone at the same hand-off rate.
* **Design:** Offline, using `PROP-12` readings and the existing Gemini-on-every-item receipt; choose τ, τ′ on half the items and report on the other half.
* **Decision:** Supported if accuracy is higher at a matched hand-off rate (±3 points), or hand-offs are fewer at matched accuracy.
* **Cost:** offline.

### `PROP-14`: In-context vs blank-question prior

* **Motivation:** On JevBench the baseline picks position A less often than the correct answer is there (62 vs 70), so dividing out a blank-question slot-A habit overcorrects.
* **Hypothesis (H14):** A position prior estimated from real items with rotated options (PriDe-style, Zheng et al. 2023) avoids null-prior's overcorrection.
* **Design:** Rotate options on held-out JevBench items (60) and the 50-item suite; estimate per-position pick rates independent of content; compare blank-question prior, in-context prior, and none, with cross-validation.
* **Decision:** Supported if the in-context prior is no worse than baseline on JevBench and at least as good as the blank-question prior on the 50-item suite.
* **Cost:** about 900 requests.

### `PROP-15`: Correction strength by question type

* **Motivation:** Extends `PROP-06`. Three yes/no items broke under null-prior on JevBench.
* **Hypothesis (H15):** Yes/no and lettered choices need different correction strengths (possibly none for yes/no).
* **Design:** Sweep α ∈ {0, 0.25, 0.5, 0.75, 1} separately by question type and number of options on both suites, with cross-validation (offline re-scoring from receipts' raw distributions).
* **Decision:** Supported if a per-type setting is never worse than baseline on either suite.
* **Cost:** offline.

### `PROP-16`: Slot names are part of the prompt

* **Motivation:** Renaming the mirror slot from `__mirror_rev` to `__rev` changed JevBench from 145 to 163 correct.
* **Hypothesis (H16):** Slot ids act as instructions; meaningful or loaded words change answers more than neutral ids.
* **Design:** Single-slot and two-slot schemas with ids: `decision`, `q1`, random strings, and loaded words (e.g. "mirror", "check").
* **Decision:** If the effect exceeds 2 items on JevBench, add a template style rule for slot ids.
* **Cost:** about 1,200 requests.
* **Pre-registered design (2026-09-26, before data):** JevBench 231, Vertex G4, one session. Single slot: 3 baselines with id `decision` (noise band = min–max), then ids `q1`, `x7k2q` (random), `mirror`, `check`. Two slots (`--dual-mirror --mirror-mode copy`, so no letter collision): suffixes `__rev`, `__mirror_rev`, `_b`. An id "matters" if its correct count is more than 3 items outside the baseline band, or (two-slot) more than 3 items away from `copy`+`__rev`. Secondary: per-item agreement with the baseline majority answer. About 2,300 requests (revised up from 1,200).

### `PROP-17`: Calibrated agent context pre-compiler (internal pilot)

* **Motivation:** Agents re-send large context windows (tool schemas, history, retrieved documents) every turn. A
  single `dgem` pass could grade every candidate block at once and use hesitation to downgrade uncertain blocks to a
  one-line stub instead of dropping them. The pilot's design and data are internal for now; this entry holds the
  number and the decision rule.
* **Hypothesis (H17):** On labelled agent turns, one pass keeps at least 95% of the blocks the gold labels mark as
  required (`full` or `summary`), removes at least 50% of candidate tokens, and flags planted prompt-injection
  blocks with recall ≥ 0.9; hesitation-driven "stub instead of drop" reduces required-block misses compared with
  taking the most likely action.
* **Design:** ~40 labelled turns across several domains, 8 candidate blocks each (tool schemas, history turns,
  retrieved documents, notes), gold action per block from one annotator with a second model labelling independently
  to measure agreement; production serving image, `samples: 1`, neutral slot ids (EXP-16).
* **Metrics:** required-block recall (primary), token savings, quarantine precision/recall, rescue rate
  (required blocks kept as stubs only because of hesitation), agreement with the gold labels.
* **Decision:** If H17 holds, build a larger labelled set from real agent traces and consider publishing the
  templates; if recall < 95%, test a two-stage variant (coarse keep/drop, then action) before continuing.
* **Cost / Dependencies:** ~200 requests plus labelling; none.

### `PROP-18`: Validate image metrics beyond EXP-09

* **Motivation:** The [EXP-09 re-baseline](/dgem/experiments/exp-09-spatial-grounding/) (phase 1) found three things:
  - The model localizes on 11 synthetic UI fixtures (mIoU 0.60 vs 0.33 for a fixed box).
  - The sub-bin and occlusion-entropy claims do not hold.
  - Boxes are unstable under flips and padding (consistency IoU 0.34–0.53).

  The suite is too small and too synthetic to say which image readouts can be trusted.
* **Hypothesis (H18):** For each aspect:
  - Accuracy beats the best image-free baseline, with a paired 95% interval above 0.
  - Per-slot hesitation flags errors with AUROC ≥ 0.75.
  - Results hold within the measured noise floor under transforms that should not change the answer.
* **Design (phased; each phase is analyzed before the next is fixed):**
  1. *Done.* Harness fixes, image-free baselines, repeats, probe variants (EXP-09 re-baseline).
  2. **Gemini 3.x reference and judge** (`gemini-3.8-flash`, `gemini-3.7-flash`). Gemini gives its own answers on the
     same items; it also judges dgem's annotated overlays edge by edge. First measure the judge against ground truth
     (agreement, Cohen's κ) on the synthetic and RefCOCO subsets; only then use it to label images without ground
     truth.
  3. **Data and references.**
     - A generated synthetic set of several hundred images, varying: aspect ratio (native, not just padded),
       horizontal position, size, occluder coverage 0–100% per edge, distractors, blur/noise/JPEG.
     - RefCOCO/COCO and ScreenSpot/RICO subsets: committed manifests; images fetched by script.
     - OWLv2, Grounding DINO and SAM boxes as references, computed on a short-lived GCE GPU VM that the script tears
       down when it finishes.
     - Test client-side preprocessing (pad or stretch to square) against the aspect-ratio failure.
  4. **`bench-vision` multi-aspect harness.** Typed aspects: location (box and 3×3/5×5 grid), presence and count,
     attributes and relations, quality/occlusion/legibility, and the domain templates (`ui_design_review`,
     `pcb_defect_triage`, `kyc_document_audit`). Each aspect has its own ground-truth source, metric and Gemini
     comparison. A small image gate goes into the regression matrix.
* **Metrics:**
  - Primary: per-aspect accuracy (or IoU) lift over image-free baselines.
  - Then: hesitation→error AUROC, transform consistency, Gemini agreement and κ, latency and cost against Gemini.
* **Decision:**
  - Aspects that pass get documented as supported, with their measured limits.
  - Aspects that fail are documented as unsupported.
  - If padding/flip instability persists on real images, the localization template gets a preprocessing
    recommendation or a coarse grid output.
* **Cost / Dependencies:**
  - About 5k dgem requests and 2k Gemini calls per full pass.
  - About 1 GPU-hour of GCE for the detector references.
  - Phase 2 needs Vertex Gemini access.
### `PROP-19`: Cross-slot coupling on mixed-polarity yes/no questions

* **Source:** Strands ([EXP-23](/dgem/experiments/exp-23-strands-decider/), section 5). Also filed as a bug.
* **Motivation:** On 51 tool-call review cases with two yes/no questions per request, dgem's two answers carried the
  same label in 34–40 cases (gold: 7) and it scored 58% (Strands Decider 86%). Asked one question per request, dgem
  scored 100 of 102; through `dgem systemone serve --noul-mode choice`, 95 of 102. typed-decisions (200 yes/no pairs
  in one request) shows no coupling, so the trigger is related wording with opposite answers.
* **Hypothesis (H19):** Coupling comes from rendering both nouls as bare `yes`/`no` labels on one canvas. Rendering
  them as two-option choices whose labels carry the true/false descriptions brings equal-answer cases on
  `gate_mixed_noul` to ≤ 12 of 51 without moving JevBench or typed-decisions beyond the noise floor.
* **Design:** Matrix v2 T1 plus `gate_mixed_noul_single`, on a canary next to production, 3 runs. Arms: current
  rendering; noul-as-choice server-side; noul-as-choice plus neutral distinct letters per slot; one read per noul
  (latency reference). Develop on a second, independently written gate suite; score `gate_mixed_noul` once.
* **Metrics:** equal-answer cases (primary); gate accuracy; JevBench and typed-decisions agreement vs production;
  latency.
* **Decision:** Ship the best arm that passes T1 as the server default; add the coupling count to the matrix gates.
* **Cost / Dependencies:** about 5k requests per arm; a server change in `structured_server.py`.

### `PROP-20`: Shipped temperature per question type and domain

* **Source:** Laya (held-out T ≈ 1.5 global, 2–3 for yes/no; T ≈ 3.3 on typed-decisions), Strands (EXP-23: one T ≈ 1.5
  brings pooled ECE from 0.069 to 0.028; yes/no wants 1.6–3.5; raw dgem is 74% correct at ≥ 0.9 on XNLI), DI
  (improvement plan track T1: overconfidence grows with option count).
* **Motivation:** dgem's confidence ranks errors well (higher AUROC than both comparison models on most suites) but its
  absolute level is too high off the development suites, so a "≥ 0.9 means act" policy over-automates.
* **Hypothesis (H20):** Temperatures per question type and option-count bucket, fitted by k-fold on the matrix
  development suites and served as defaults, cut ECE by ≥ 30% on every T1 suite and on the frozen sets (scored once),
  with AUROC unchanged (± 0.01).
* **Design:** Fit on T1 receipts (5-fold by case); evaluate on held-out folds, then on T2 frozen sets after shipping.
  Compare global T, per-type T, per-type × bucket T. Per-policy override via template field (four-surface parity).
* **Metrics:** ECE10 (primary), Brier, NLL, AUROC, coverage and accuracy at ≥ 0.9.
* **Decision:** Ship the simplest variant within 0.005 ECE of the best.
* **Cost / Dependencies:** offline on existing receipts, then one T1 + one T2 run.

### `PROP-21`: Fine-tuning DiffusionGemma: LoRA on the letter readout

* **Source:** Laya (a 421M encoder fine-tuned on 6,000 typed-decisions labels beat zero-shot dgem on that domain),
  Strands (EXP-23: a 1.9B LoRA model matches dgem on its training tasks, 85.9% vs 85.2%, and trails by 10 points
  elsewhere), Clef (a trained routing head leads on intent and taxonomy benchmarks).
* **Motivation:** Fine-tuning is the main accuracy lever of every competitor; dgem's lead is its base model's breadth.
  The diffusion objective matches inference exactly (unmask the answer slots), and a training canvas can hold several
  questions with independent labels, which is the direct fix for `PROP-19`.
* **Hypothesis (H21):** A rank-16 LoRA on attention and the shared expert (router frozen), trained with the masked
  denoising loss on answer slots plus a proper scoring rule and a KL term to the frozen model, on Strands Decider's
  public training corpus plus synthetic mixed-polarity canvases: (1) raises accuracy on the `train` exposure pool from
  85% to ≥ 90%; (2) keeps JevBench ≥ 190 of 231; (3) brings `gate_mixed_noul` equal answers to ≤ 12 of 51; (4) lowers the
  fitted yes/no temperature below 1.5.
* **Design:** Same corpus as the competitor so the base model is the only difference. BF16 training (about 52 GB of
  weights, 80 GB-class GPUs), merge, re-quantize to NVFP4, refit T. Spike first: vLLM structured mode with a merged
  checkpoint on the G4 profile. Never train on JevBench or frozen sets; differences under 10 JevBench items are
  unresolved.
* **Metrics:** the four predictions; full T1 against production; frozen sets once, after the decision.
* **Decision:** Ship only if (2) and T1 pass; otherwise publish the result and keep zero-shot as default.
* **Cost / Dependencies:** training compute (hours on 8× H100-class), one canary; `PROP-19` and `PROP-20` define the
  baselines.

### `PROP-22`: Distil dgem into a small student

* **Source:** Strands (its recipe distils from a frozen 4B teacher; about 1 hour on 8× H100).
* **Hypothesis (H22):** Replacing the teacher in Strands Decider's recipe with dgem's temperature-scaled soft labels
  gives a 2B student that beats v19 on the `unseen` exposure pool by ≥ 3 points at equal latency.
* **Design:** One recipe change, same data and seeds; compare with v19 and with dgem on matrix TC.
* **Decision:** If it holds, publish the recipe and offer the student as a CPU-capable option.
* **Cost / Dependencies:** dgem labelling of the corpus (one batch job), training compute; `PROP-20` first.

### `PROP-23`: Nondeterminism at `samples=1`

* **Source:** Laya (5–7% answer changes between identical requests at `seed=42`), Strands (6.4%, EXP-23). Both
  competitors are bit-identical across runs.
* **Hypothesis (H23):** The changes come from batch composition in vLLM (requests batched with different neighbours),
  not from sampling; serving with batch-invariant kernels or a fixed per-request seed path removes most of them at ≤ 10%
  throughput cost.
* **Design:** Identity runs at 1 vs 16 concurrent workers; then the candidate fixes on a canary.
* **Metrics:** answer agreement between identical runs (primary), throughput.
* **Decision:** Ship if agreement ≥ 99% and throughput ≥ 90% of current.
* **Cost / Dependencies:** about 3k requests per arm; vLLM options.

### `PROP-24`: Wide-option accuracy (6–26 options and beyond)

* **Source:** DI (improvement plan findings F1 and F8: 6–26 options in one read already score 36% on the index
  benchmarks), Clef (leads intent and taxonomy benchmarks by 20–55 points), Strands (EXP-23: banking77 with 30 options,
  dgem 80% through the adapter vs 87% natively).
* **Motivation:** Letter readout caps a slot at 26 options and loses accuracy well before that.
* **Hypothesis (H24):** On `di_wide` and `intents_systemone`, hierarchical routing with calibrated finalists and option
  descriptions in the prompt recovers ≥ 5 points over the current bracket routing. A pointer readout (`PROP-21` route B)
  is the longer-term fix.
* **Design:** Owned with the Decision Index improvement plan; develop on validation data only; T1 plus Decision Index
  dev set.
* **Decision:** Ship in the adapter if T1 passes.
* **Cost / Dependencies:** adapter changes; no training.

### `PROP-25`: NLI "neutral" pull

* **Source:** Laya (XNLI: dgem predicted "neutral" for 659 of 1,500 contradictions and 714 of 1,500 entailments),
  Strands (EXP-23: XNLI is dgem's smallest lead, 70.0% vs 63.7%, with ECE 0.227).
* **Hypothesis (H25):** Clearer option descriptions or a per-label prior estimated in context raise XNLI validation
  accuracy by ≥ 3 points without changing the calibration suite's NLI items.
* **Design:** XNLI validation only for development; one scored test run after shipping.
* **Cost / Dependencies:** about 5k requests.

### `PROP-26`: First-shown bias on small option sets

* **Source:** Laya (emotion, K=6: dgem picks the first-shown option 41.5% of the time vs 30.5% gold), Strands (EXP-23:
  45.0% vs 30.5% on the same suite; Strands Decider 47%).
* **Hypothesis (H26):** The in-context prior of `PROP-14` brings first-shown picks within 5 points of the gold rate
  at K ≤ 6 without raising order flips.
* **Design:** Option-order suite plus `PROP-14`'s items; extends `PROP-14`.
* **Status:** Parked with `PROP-14`. Under hesitation-gating, order bias is monitored per release (the matrix's
  option-order suite) rather than corrected; reopen only if the monitored first-shown rate rises.
* **Cost / Dependencies:** `PROP-14`.


---

## 4. How to Add an Entry

Copy this block, choose the next `PROP-XX`, and add a row (with its Source) to the table in §2:

```markdown
### `PROP-XX`: <title>

* **Source:** <studies that raised it: Order, a comparison (e.g. Strands), DI; add to this list instead of duplicating>
* **Motivation:** <which doc claim or open question this tests; link receipts>
* **Hypothesis (HXX):** <falsifiable statement + pre-registered metric and threshold>
* **Design:** <items, conditions, splits/CV, endpoint + revision>
* **Metrics:** <primary first; secondary after>
* **Decision:** <what we change in code/docs if it passes or fails>
* **Cost / Dependencies:** <GPU hours, Gemini calls, code changes, blocking PROPs>
```
