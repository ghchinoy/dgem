---
title: "Confidence and Calibration"
description: "How dgem assures confidence in its decisions: per-answer probabilities from one forward pass, hesitation, calibration checks, escalation, and what the experiments do and don't support."
---

For teams who need to know when a `dgem` decision can be trusted, and how we check that. Every claim below links
to a measurement with its sample size; simulated or illustrative numbers are labelled as such. For the full picture of
why `dgem` decisions can be trusted, start with [Confidence beyond Shannon: hesitation-gated decisions](/dgem/confidence/overview/).

## What you get with every answer

DiffusionGemma fills in all answer slots of a policy at once and returns, for each question, a **probability for
every allowed answer** (not just a label). From those probabilities `dgem` derives:

| Signal | What it is | Where you see it |
| :--- | :--- | :--- |
| Confidence | Probability of the chosen answer | CLI table, JSON `confidence` |
| **Hesitation** | Entropy of the answer distribution scaled to 0–100% by the number of options (0% = certain, 100% = coin toss) | Decision Studio answer cards; `--stats` |
| Standard error, agreement | Spread across samples (with `samples: 4`) | CLI table, JSON |

Hesitation bands used in the Studio: **below 16% clear**, **16–50% somewhat unsure**, **above 50% very unsure**.
Why a model can read its own confidence in one pass, in plain language:
[The journey to decision models](/dgem/decision-models-primer/) and the [glossary](/dgem/glossary/).

## How we assure it

1. **Measure calibration on labelled data.** Confidence should match accuracy. Public suites: 231-item JevBench
   and a 50-item calibration suite across 11 datasets ([benchmarks](/dgem/benchmarks/)); your own data:
   [Calibrate your policy](/dgem/confidence/calibrate-your-policy/).
2. **Gate on hesitation.** Answers above a hesitation threshold go to a person or a larger model. Hesitation
   alone detects most errors (AUROC ≈ 0.85 on JevBench, [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/)).
3. **Escalate only what needs it.** A Stage 2 cascade to Gemini on answers above 16% hesitation reached 221/231 on
   JevBench while escalating 39% (Gemini on everything: 225/231), and 48/50 on the calibration suite at 34%
   escalated ([EXP-14](/dgem/experiments/exp-14-idc-rerun/)).
4. **Monitor known biases.** The model has a measurable preference for the first option when unsure; it is measured
   on every release rather than corrected, because the corrections we tested did not help reliably
   ([Confidence beyond Shannon](/dgem/confidence/overview/#43-order-bias-measured-and-monitored)).
5. **Re-validate on every serving change.** A new serving image is compared with production in the
   [regression matrix](/dgem/operate/regression-matrix/), judged against the run-to-run agreement measured in the same
   session, before it takes traffic.

## What is recommended today

| Technique | Status | Evidence |
| :--- | :--- | :--- |
| Hesitation gate (~16%) | **Recommended** | Catches most errors at low cost ([EXP-14](/dgem/experiments/exp-14-idc-rerun/), [EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/)) |
| Stage 2 cascade to Gemini on hesitant answers | **Recommended** where a larger model is acceptable | EXP-05, EXP-14 |
| `samples: 4` for an error bar | Use where agreement matters | Adds ~35–40 ms |
| Temperature scaling | **Recommended per policy**, fit on held-out data | Best temperature ranges from 1.2 to 3.6 by domain ([v0.2.0 reference](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20261002-v020-reference-t2/report.md)) |
| `document_first` prompt layout | **Default** from serving v0.2.0 | Gains of 3–12 points on multi-question and wide-option sets ([prompt layout](/dgem/policies/prompt-layout/)) |
| Null-prior de-biasing | Research: helps one suite, hurts another | EXP-13, EXP-14 |
| Reversed-order ("mirror") check | Research: little gain over hesitation for double the cost | [EXP-15](/dgem/experiments/exp-15-letter-collision/)–[EXP-17](/dgem/experiments/exp-17-separate-pass-mirror/) |

## Caveats

- **Small suites are noisy.** 50 and 231 items give coarse calibration estimates; two identical runs agree on 94–99% of
  answers depending on the suite ([regression matrix](/dgem/operate/regression-matrix/)).
- **Individual answers depend on the serving image.** Accuracy was unchanged across the last runtime upgrade, but
  about 5% of items changed answer; don't compare per-item results across images.
- **Calibration is policy- and data-specific.** Numbers on public suites don't transfer automatically; check your
  own ([how](/dgem/confidence/calibrate-your-policy/)).

## Read more

- [Confidence beyond Shannon: hesitation-gated decisions](/dgem/confidence/overview/): the techniques behind trustworthy decisions and their evidence.
- [How the model decides in one pass](/dgem/confidence/architecture/): discrete diffusion vs autoregression.
- [Benchmark report](/dgem/benchmarks/), [Ecotone comparison](/dgem/ecotone-comparison/), and the
  [experiment ledger](/dgem/experiments/) with pre-registered [proposed experiments](/dgem/experiments/proposed/).
