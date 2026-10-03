---
title: "Confidence beyond Shannon: hesitation-gated decisions"
description: "Why you can trust a dgem decision: a probability for every allowed answer from one pass, a hesitation score that flags uncertain answers, hand-off to a larger model, and the prompt, calibration and evaluation practices behind them. Every claim links to a measurement with its sample size."
---

# Confidence beyond Shannon: hesitation-gated decisions

`dgem` turns DiffusionGemma into a **decision model**: you describe a decision as a few typed questions, and for every
question you get **a probability for every allowed answer** from a single forward pass. This page explains how those
probabilities become decisions you can trust: a **hesitation score** that flags uncertain answers, a **gate** that hands
them to a larger model or a person, and the prompt, calibration and evaluation practices that keep both honest.

Every number below links to a committed measurement and gives its sample size. Where something is a limitation or a
negative result, it says so.

> **In one sentence:** `dgem` answers clear cases itself in about 0.1 s and knows when it is unsure, and that "knowing"
> is measured, tuned per domain, and re-checked on every release.

---

## 1. What every decision gives you

| You get | What it is | Where |
| :--- | :--- | :--- |
| **A probability for every option** | Not just a label: the full distribution over the answers you allowed | CLI JSON, HTTP gateway, MCP tools, Decision Studio |
| **Confidence** | Probability of the chosen answer | Same |
| **Hesitation** | How torn the model is, 0–100% (below) | Studio answer cards, `--stats`, JSON `entropy` |
| **Diagnostics** | Passes, layout used, latency, backend used | JSON `diagnostics`, `X-DGem-Backend-Used` |

All questions in a policy are answered **together in one pass**. Reading them jointly costs about the same as reading
one: on the v0.2.0 production endpoint the median server time is 54 ms for 1 question, 58 ms for 5 and 62 ms for 10
([latency reference](../operate/regression-matrix.md)). Reading each question in its own pass costs about 5× the
latency and, with the default prompt layout, gains only +1.6 points (not significant) on typed decisions
([EXP-19](../experiments/README.md)).

---

## 2. The first step: Shannon entropy, shown as hesitation

From the answer probabilities `dgem` computes Shannon entropy and divides it by its maximum for that question, so a
2-option and a 26-option question share one 0–100% scale:

$$\text{hesitation} = \frac{-\sum_k p_k \ln p_k}{\ln K}$$

0% means all the probability is on one answer; 100% means a perfect tie. Decision Studio labels it **Clear** (below
16%), **Somewhat unsure** (16–50%) and **Very unsure** (above 50%), with the raw value in nats on hover.

This is an approachable first step for anyone assessing probabilities: no labelled data, no extra passes, and it
works on every question type. It is also a strong one: on 231 JevBench items, hesitation alone separates wrong
answers from right ones with an AUROC of about **0.85** ([EXP-17](../experiments/exp-17-separate-pass-mirror.md)).

---

## 3. Hesitation-gating: the core technique

**Hesitation-gating** means: answer directly when hesitation is low, and hand the decision to a larger model (or a
person) when it is high, passing along `dgem`'s probabilities as a hint.

| Evidence | Result |
| :--- | :--- |
| **Live cascade, threshold fixed in advance** ([EXP-18](../experiments/exp-18-mizan-judge-capability.md), 13 judge suites, n=100 each) | A 35%-hesitation gate to `gemini-3.8-flash` reached **81/100 on safety and 86/100 on faithfulness** while handing off 8–32% of items. The live run matched the offline estimate within 1–2 items. |
| **Speed and cost of the fast path** (EXP-18) | Median 105 ms on the G4 endpoint vs 3,125 ms for Gemini; about 83 items/s per replica; about $0.02 per 1,000 judgements at full utilization vs $1.06–2.25 for Gemini on every item. |
| **JevBench, offline** ([EXP-14](../experiments/exp-14-idc-rerun.md), 231 items) | Handing off items at ≥16% hesitation (39% of items) gave **221/231**, against 187–189 for `dgem` alone and 225 for Gemini on everything. Threshold chosen on the same items, so treat it as an estimate. |

**Choosing a threshold.** Start at 16–35% and fit it on a labelled sample of your own decisions, choosing the
threshold on one half and checking it on the other ([Calibrate your policy](calibrate-your-policy.md)). The gate is
available on every surface (`cascade_mode`, `cascade_threshold`, `cascade_model`; see
[authoring and Stage 2 cascades](../policies/authoring.md)). Note that the gateway's `cascade_threshold` is raw entropy
in nats (default 0.35), not hesitation %; multiply hesitation by ln K to convert (16% hesitation ≈ 0.18 nats for 3
options).

---

## 4. What makes the answers trustworthy

Hesitation is only useful if the probabilities behind it are well formed. These practices keep them that way.

### 4.1 The answer template is part of the prompt

The model reads the whole answer template, including question ids and option labels. Two measured effects follow:

- **Shared letters cause copying.** When two questions label different options with the same letters (A, B, C…),
  the model tends to copy the letter across questions. On JevBench, a second question listing the options in reverse
  with letters dropped accuracy from a baseline band of 182–189 to **154**; the answer changed on 67% of items where
  the letters matched versus 6% elsewhere (p ≈ 10⁻¹⁹). The same question labelled with digits caused no measurable
  harm ([EXP-15](../experiments/exp-15-letter-collision.md)).
- **Question ids act as instructions.** A single question's id made no difference, but naming a second question
  `…__mirror_rev` cost 19 of 231 items even with identical options ([EXP-16](../experiments/exp-16-slot-names.md)).

**What you do:** use neutral, descriptive ids (`team`, `urgent`, `severity`) and avoid ids that hint the answer
should differ (`mirror`, `reverse`, `opposite`). This is a template rule ([templates](../policies/templates.md)).

### 4.2 Prompt layout

Since serving v0.2.0 the default layout puts your input first and the questions after it (`document_first`). On
frozen held-out sets scored after the change shipped ([prompt layout](../policies/prompt-layout.md)):

| Data | Items | Previous layout | `document_first` |
| :--- | ---: | ---: | ---: |
| typed decisions, 5 questions per decision | 2,000 | 0.674 | **0.728** |
| XNLI, 15 languages | 4,500 | 0.662 | **0.699** |
| CLINC150 validation, 151 options (development split, release gate) | 100 × 3 | 0.740 | **0.863** |
| RAGTruth train, unsupported content (development split, release gate) | 198 × 3 | 0.712 | **0.771** |
| JevBench, single questions | 231 × 3 | 0.840 | 0.848 (within noise) |

**Trade-off:** with a catch-all "out of scope" option, out-of-scope recall fell from 100% to 82.5% (n=40), and
calibration error rose on MASSIVE. Layout can be set per template, per request, or per deployment on every surface.

### 4.3 Order bias: measured and monitored

When unsure, the model leans toward the first option listed. On blank questions with meaningless options it picks
the first one 88% (2 options), 78% (3) and 49% (4) of the time ([EXP-13](../experiments/exp-13-permutation-invariance.md)).
Every T2 release run shuffles option order on four suites and reports the extra flips over an identical repeat:
+0.035 to +0.090 on v0.2.0, with the first option of 6-option emotion chosen 45.5% of the time against a gold rate of
30.5% ([v0.2.0 reference](../../benchmarks/runs/20261002-v020-reference-t2/report.md)).

We tested in-pass corrections (dividing out the blank-question prior, and reading a reversed copy of the options in
the same or a separate pass). None improved decisions reliably (EXP-14 to EXP-17), so order bias is **tracked as a
release gate rather than corrected**. Hesitation already catches most of the cases it affects.

### 4.4 Calibration by domain

Raw probabilities are somewhat over-confident, and by how much depends on the domain. Fitted on held-out folds with
v0.2.0, the temperature that best calibrates confidence is about 1.2–1.9 on JevBench, MASSIVE, wide-option suites and
the calibration suite, but **2.6 on typed decisions, 3.5 on RAGTruth and 3.6 on XNLI**
([v0.2.0 reference](../../benchmarks/runs/20261002-v020-reference-t2/report.md)). On XNLI, held-out temperature cuts
calibration error from 0.227 to 0.036 (n=4,493).

**What you do:** fit one temperature per policy on held-out decisions; a single global value does not fit every
domain ([Calibrate your policy](calibrate-your-policy.md)). Temperature never changes which answer wins, only how
confident it reports.

### 4.5 Results judged against measured noise

The model is not perfectly deterministic: two identical runs agree on 94–99% of answers depending on the suite.
Every serving change goes through a [regression matrix](../operate/regression-matrix.md) that measures this
agreement in the same session and judges the candidate against it (paired McNemar tests), with pinned datasets,
frozen held-out sets scored only after a change ships, and versioned receipts for every run
([benchmarks/runs](../../benchmarks/runs/README.md)). A result that is within noise is reported as such.

---

## 5. Strong, flexible, explainable, transparent

| Property | What provides it |
| :--- | :--- |
| **Strong** | Competitive accuracy on public suites at about 0.1 s per decision; hesitation-gating recovers most of a larger model's accuracy while handing off a minority of items (§3). |
| **Flexible** | Decisions are templates you edit, not models you retrain; any mix of yes/no, choice and score questions; per-template layout and temperature; four surfaces (CLI, HTTP, MCP, Studio). |
| **Explainable** | A probability for every option and a hesitation score on every answer; the reason an item was handed off is visible (its hesitation); diagnostics report layout and passes. |
| **Transparent** | Every claim cites a receipt with its sample size; pre-registered experiments, including the ones that failed ([experiment ledger](../experiments/README.md), [proposed experiments](../experiments/proposed.md)); release gates against measured noise. |

---

## 6. What it does not do (yet)

- **Missing context:** if a decisive fact is absent, the model can be confidently wrong. List required facts as their
  own question (proposed test: `PROP-08`).
- **Rewording:** rephrasing options can shift answers; not yet measured at scale (`PROP-07`).
- **Base rates:** confidence is not adjusted for how common each class is in your traffic (`PROP-09`).
- **Out-of-scope recall** is lower under the default layout (§4.2).
- **Bounding boxes:** locating objects in images is far behind Gemini and not recommended; use categorical image
  questions instead.
- **Multi-step reasoning:** one pass cannot do arithmetic or long chains of inference; these items show up as hesitant
  and are handed off.

---

## 7. Prior art and what is new

Entropy-based abstention and confidence-gated cascades (selective prediction, Geifman & El-Yaniv 2017; FrugalGPT,
Chen et al. 2023) and temperature scaling (Guo et al. 2017) are established techniques; `dgem` applies them to a
single-pass slot readout. What is specific to `dgem`:

- **Joint reads at single-question cost:** many typed questions answered together, each with a full probability
  distribution, in one pass of about 0.1 s.
- **Measured template effects in slot readouts:** shared labels and loaded question ids change answers (EXP-15,
  EXP-16), turned into template rules.
- **Noise-judged release gates:** accuracy, calibration, coverage and option order compared against agreement
  measured in the same session, on every serving change.

How we got here, including the order-check techniques we tried and retired, is in the historical write-up kept in
the repository (`docs/history/`).

---

## 8. Quick reference

```bash
# One decision, with hesitation and timing
./bin/dgem decide -t templates/support_triage.json.tmpl -v ticket="..." --stats

# Choose a prompt layout for a request (default: document_first)
./bin/dgem decide -t templates/support_triage.json.tmpl -v ticket="..." --layout schema_first

# Hesitation-gated hand-off to Gemini (HTTP gateway)
curl -s http://localhost:8090/api/decide/support_triage -H 'Content-Type: application/json' \
  -d '{"variables":{"ticket":"..."},"cascade_mode":"entropy","cascade_threshold":0.35}'   # threshold in nats

# Check a serving change against production, judged against measured noise
python3 scripts/bench_matrix.py run --tier T1 --target prod=<url> --target new=<url> --baseline prod
```

Related: [Confidence and calibration](index.md) (how to use the signals day to day) ·
[Calibrate your policy](calibrate-your-policy.md) · [Prompt layout](../policies/prompt-layout.md) ·
[Regression matrix](../operate/regression-matrix.md) · [Glossary](../glossary.md).
