---
title: "EXP-18: Judge Capabilities With and Without an Autoregressive Autorater (mizan Experiment 07)"
description: "Gold-scored, same-session comparison of DiffusionGemma, Gemini 3.5 Flash-Lite, Gemini 3.8 Flash, Vertex predefined metrics and no-model computation metrics across the Vertex Gen AI evaluation capabilities."
---

# EXP-18: Judge Capabilities With and Without an Autoregressive Autorater

**Date:** 2026-09-26 · **Harness:** `mizan eval compare-engines` (report schema v2), run from the mizan repository ·
**Backends:** Vertex AI G4 endpoint `<endpoint-id>` (direct `/invoke`) and Cloud Run RTX PRO 6000 (gateway, `X-DGem-Backend: cloudrun`) ·
**Full report:** [mizan `docs/experiments/07-judge-capability-rerun.md`](https://github.com/ghchinoy/mizan/blob/main/docs/experiments/07-judge-capability-rerun.md)

## Question

For each capability of the Vertex Gen AI Evaluation Service, this experiment asks which of three kinds of judge is enough:

- **no model** (computation metrics),
- **DiffusionGemma** as a non-autoregressive decision model,
- an **autoregressive Gemini autorater**.

Every verdict is scored against a human or gold label, and all engines ran in the same session. This replaces the agreement-only comparisons in mizan Experiments 04–06, which EXP-04/05 once cross-referenced.

## Results (n = 100 per suite, accuracy with 95% bootstrap CI; p = exact McNemar)

| Capability (dataset) | gemini-3.8-flash | dgem G4 | p | dgem Cloud Run | gemini-3.5-flash-lite |
| :--- | :---: | :---: | :---: | :---: | :---: |
| Response safety (BeaverTails) | 75 | 76 | 1.0 | 77 | 80 |
| Prompt toxicity (ToxicChat) | **89.8** | 79 | **0.003** | 77 | 83.7 |
| Faithfulness (HaluBench) | 76 | 82 | 0.36 | 85 | 79 |
| Pairwise, RewardBench | 90 | 86 (91 with mirror) | 0.42 | 86 | 90.8 |
| Pairwise, LLMBar instruction following | **93** | 79 | **0.003** | 77 | 85 |
| Pairwise, MT-Bench expert (turn 1) | **76** | 67 | **0.035** | 68 | 69 |
| Pairwise, MT-Bench expert (turn 2) | **71** | 61 | **0.021** | 64 | 65.7 |
| Likert, HelpSteer2 helpfulness (Spearman) | 0.681 | 0.652 | — | 0.651 | 0.663 |
| Likert, SummEval coherence (Spearman) | 0.777 | 0.719 | — | 0.711 | 0.787 |

- **No-model metrics.** mizan's new local `computation` kind agrees item for item with Vertex `EvaluateInstances` on exact match and on tool-call and trajectory metrics (350/350). It reproduces sacrebleu and rouge_score exactly. Vertex's own BLEU and ROUGE-Lsum differ from those libraries by up to 0.12 and 0.23, although the ranking is nearly the same (Spearman 0.99).
- **Entropy cascade.** At a fixed 35% hesitation threshold, the cascade (dgem, escalating to gemini-3.8-flash) reaches:
  - safety 80 with 10% of items escalated,
  - faithfulness 84 with 31% escalated, better than either judge alone,
  - toxicity 83.7 with 11% escalated.

  On the pairwise suites it needs 48–75% escalation to approach Gemini, so the saving there is small.
- **Latency.** Serial probe, p50: dgem on G4 **97 ms**, dgem on Cloud Run **178 ms**, gemini-3.5-flash-lite 998 ms, gemini-3.8-flash 3,050 ms. Vertex predefined metrics took 3.7–12 s in batch runs.
- **Backend parity.** G4 and Cloud Run give the same accuracy within noise on every suite. Every dgem answer came back as a structured-readout envelope.
- **Position bias.** On pairwise items, dgem picks a different winner when the two responses are swapped on 12–27% of items. A mirror read adds 0–4 points, which is not significant.

## Capability verdicts

| Capability | Verdict |
| :--- | :--- |
| Exact match, BLEU, ROUGE, tool-call and trajectory metrics | No model needed (mizan `kind: computation`, local, no credentials) |
| Binary safety / harm, groundedness / faithfulness | DiffusionGemma can replace the Gemini judge |
| Toxicity / jailbreak detection | DiffusionGemma as a front filter plus entropy cascade |
| Likert quality | Close to parity on rank correlation |
| General pairwise preference (RewardBench) | DiffusionGemma with a mirror read can replace the Gemini judge |
| Instruction-following, expert and multi-turn pairwise | Autoregressive judge required (−9 to −14 points, significant) |
| Rubric generation, explanations, audio/video | Autoregressive only |

## Notes

- **Prompt size.** Mizan's diffusion adapter used to send every field twice: once in the state JSON and again in a rendered `_prompt`. That pushed long HaluBench passages past the 4,096-token limit on Cloud Run. The adapter now sends each field once. Keep this in mind when running long-context policies on Cloud Run.
- **Gemini capacity.** gemini-3.8-flash queued requests for 80–150 s under modest concurrency, and first attempts at Vertex predefined metrics failed 20–34% of the time. Failed items were re-run on both engines rather than dropped. The re-run items were harder than average, so dropping them would have inflated accuracy.
