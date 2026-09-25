---
title: "Proposed Experiments Register (Hypotheses & Designs)"
description: "Register of experiments we intend to run, each with a motivation, a falsifiable pre-registered hypothesis, a design, metrics, decision criteria, and dependencies. Seeded with the open questions from Invariant Decision Calibration (IDC)."
---

# Proposed Experiments Register

The [Experiment Ledger](/dgem/experiments/) records experiments we have **run**. This page records experiments we **intend to run**, written down *before* any results exist, so that the hypothesis, the metric, and the pass/fail line can't quietly move after the data comes in.

**Lifecycle:** `Proposed` → `Ready` (dependencies met, design reviewed) → `Running` → `Done`. When an entry starts running, give it the next free `EXP-XX` number, add it to the ledger, and link it back here. Keep the `PROP-XX` entry, but mark it `Done → EXP-XX` and note any change to the design along with the reason for it.

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

| ID | Title | Question it answers | Depends on | Priority | Status |
| :--- | :--- | :--- | :--- | :---: | :--- |
| [`PROP-00`](#prop-00-expose-mirror-tvd-on-all-four-surfaces-enabler) | Expose Mirror TVD on all four surfaces (*enabler*) | Can production use the mirror signal at all? | — | **P0** | Proposed |
| [`PROP-10`](#prop-10-run-to-run-and-revision-noise-floor) | Run-to-run & revision noise floor | How big a difference is real? | — | **P0** | Done → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (noise floor: ±1–2 items, Brier ±0.02) |
| [`PROP-01`](#prop-01-held-out-temperature-scaling) | Held-out temperature scaling | Does the ECE gain from $T^*$ survive out of sample? | — | **P0** | Done → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (met on 231 JevBench items, not on 50) |
| [`PROP-02`](#prop-02-end-to-end-idc-cascade) | End-to-end IDC cascade | Does IDC + a mirror-aware gate beat the entropy-only cascade? | `PROP-00`, `PROP-01` | **P0** | Partly → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (entropy gates measured; mirror gate blocked by PROP-03) |
| [`PROP-03`](#prop-03-same-canvas-vs-separate-pass-mirror) | Same-canvas vs separate-pass mirror | Does sharing a canvas hide disagreement? | — | P1 | **Next** (EXP-14 found same-canvas coupling lowers the forward reading) |
| [`PROP-04`](#prop-04-order-sensitivity-on-real-chaosnli) | Order sensitivity on real `ChaosNLI` | Do entropy / Mirror TVD / cyclic JSD track real human disagreement? | — | P1 | Proposed |
| [`PROP-05`](#prop-05-mirror-merge-rule) | Mirror merge rule | Is the 70/30 forward-priority merge the right one? | `PROP-03` (optional) | P1 | Done → [EXP-14](/dgem/experiments/exp-14-idc-rerun/) (50/50 beats 70/30; neither beats single slot) |
| [`PROP-06`](#prop-06-null-prior-strength-and-label-aware-priors) | Null-prior strength & label-aware priors | Can de-biasing improve calibration *without* raising order flips? | — | P1 | Proposed |
| [`PROP-07`](#prop-07-wording-invariance) | Wording invariance | How often does rephrasing an option change the decision? | — | P2 | Proposed |
| [`PROP-08`](#prop-08-missing-context-detection) | Missing-context detection | Does uncertainty rise when a decisive fact is removed? | — | P2 | Proposed |
| [`PROP-09`](#prop-09-base-rate-label-shift-adaptation) | Base-rate (label-shift) adaptation | Can unlabeled target traffic correct for different class frequencies? | — | P2 | Proposed |

---

## 3. Entries

### `PROP-00`: Expose Mirror TVD on all four surfaces (*enabler*)

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
* **Design:** Run the same items, same flags, and `-w 1` for determinism where possible, on Vertex AI (`4217256562927861760`) and Cloud Run.
* **Metrics:** Standard deviation within a revision and differences between revisions for accuracy, Brier score, ECE, and per-item answer agreement.
* **Decision:** Sets the minimum effect size that later entries (`PROP-01`–`PROP-09`) must exceed to count as real.

---

## 4. How to Add an Entry

Copy this block, choose the next `PROP-XX`, and add a row to the table in §2:

```markdown
### `PROP-XX`: <title>

* **Motivation:** <which doc claim or open question this tests; link receipts>
* **Hypothesis (HXX):** <falsifiable statement + pre-registered metric and threshold>
* **Design:** <items, conditions, splits/CV, endpoint + revision>
* **Metrics:** <primary first; secondary after>
* **Decision:** <what we change in code/docs if it passes or fails>
* **Cost / Dependencies:** <GPU hours, Gemini calls, code changes, blocking PROPs>
```
