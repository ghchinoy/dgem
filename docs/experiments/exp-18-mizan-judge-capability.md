---
title: "EXP-18: Judge Capabilities With and Without an Autoregressive Autorater (mizan Experiment 07)"
description: "Gold-scored, same-session comparison of DiffusionGemma (v0.1.0), Gemini 3.5 Flash-Lite, Gemini 3.8 Flash, Vertex predefined metrics and no-model computation metrics across the Vertex Gen AI evaluation capabilities, replicated on two dates."
---

# EXP-18: Judge Capabilities With and Without an Autoregressive Autorater

**Dates:**
- 2026-09-29: primary session, serving image **v0.1.0** (`/health`: `version v0.1.0`, `revision f241b77`).
- 2026-09-26: replication on the pre-v0.1.0 images.

**Harness:** `mizan eval compare-engines` (report schema v2), run from the mizan repository.

**Backends:**
- Vertex AI G4 dedicated endpoint `<endpoint-id>`, called directly on `/invoke`.
- Cloud Run RTX PRO 6000 through the gateway, with `X-DGem-Backend: cloudrun`.
- The gateway's `vertex_first` routing, for latency only.

**Full report:** [mizan `docs/experiments/07-judge-capability-rerun.md`](https://github.com/ghchinoy/mizan/blob/main/docs/experiments/07-judge-capability-rerun.md) (landing in mizan PR #110).

## Question

For each capability of the Vertex Gen AI Evaluation Service, which judge is enough?

- **no model** (computation metrics)
- **DiffusionGemma**, as a non-autoregressive decision model
- an **autoregressive Gemini autorater**

Every verdict is scored against a human or gold label. Both engines in each comparison ran on the same items in the same run. This replaces the agreement-only comparisons in mizan Experiments 04–06, which EXP-04/05 once cross-referenced.

## Results

v0.1.0, 2026-09-29. n = 100 per suite; accuracy in %. The 09-26 value is in parentheses. p is the exact McNemar test between gemini-3.8-flash and dgem on G4.

| Capability (dataset) | gemini-3.8-flash | dgem G4 | p | dgem Cloud Run | gemini-3.5-flash-lite |
| :--- | :---: | :---: | :---: | :---: | :---: |
| Response safety (BeaverTails) | 76 (75) | 80 (76) | 0.34 | 79 (77) | 79 (80) |
| Prompt toxicity (ToxicChat) | 87.9 (89.8) | 81 (79) | 0.065 (0.003) | 81 (77) | 85.7 (83.7) |
| Faithfulness (HaluBench) | 78 (76) | 81 (82) | 0.72 | 84 (85) | 81 (79) |
| Pairwise, RewardBench | 90 (90) | 87 (86); **90 with mirror** | 0.51 | 87 (86) | 91 (90.8) |
| Pairwise, LLMBar instruction following | **94** (93) | 79 (79) | **0.0015** (0.003) | 79 (77) | 82 (85) |
| Pairwise, MT-Bench expert (turn 1) | 78 (76) | 73 (67) | 0.36 (0.035) | 72 (68) | 68 (69) |
| Pairwise, MT-Bench expert (turn 2) | 71 (71) | 65 (61) | 0.21 (0.021) | 65 (64) | 66 (65.7) |
| Likert, HelpSteer2 helpfulness (Spearman) | 0.695 | 0.652 | — | 0.652 | 0.665 |
| Likert, SummEval coherence (Spearman) | 0.778 | 0.742 | — | 0.739 | 0.773 |

- **No regression on v0.1.0.**
  - Paired on identical items across all suites, dgem on G4 moved from 67.0% to 67.9% (1,720 verdicts, p = 0.23). dgem on Cloud Run moved from 68.6% to 69.8% (p = 0.21).
  - No single dgem suite changed significantly.
  - The Gemini judges were just as stable across the two dates.
- **No-model metrics.** mizan's local `computation` kind agrees item for item with Vertex `EvaluateInstances` on exact match, tool-call and trajectory metrics (350/350). It also reproduces sacrebleu and rouge_score exactly. Vertex's own BLEU and ROUGE-Lsum differ from those libraries by up to 0.12 and 0.23, although they rank items almost identically (Spearman 0.99).
- **Entropy cascade** (dgem G4 escalating to gemini-3.8-flash at a fixed 35% hesitation threshold). Each figure is cascade accuracy, followed by the share of items escalated:

  | Suite | Cascade accuracy | Items escalated |
  | :--- | :---: | :---: |
  | Safety | 81 | 8% |
  | Faithfulness | 86 | 32% |
  | RewardBench | 91 | 31% |
  | Toxicity | 83.8 | 12% |

  On LLMBar and MT-Bench it needs 41–73% escalation to approach Gemini.
- **Latency** (serial probe, p50):
  - dgem, G4 direct: **105 ms**
  - dgem through the gateway (`vertex_first`, served by Vertex): 169 ms
  - dgem, Cloud Run: 182 ms
  - gemini-3.5-flash-lite: 1,429 ms
  - gemini-3.8-flash: 3,125 ms
  - Vertex predefined metrics: 3.9–12.6 s
- **Throughput** (2026-09-29, production G4 as-is, one engine per run):
  - One G4 replica saturates at **~83 items/s** on short inputs and **~68 items/s** on 1–3k-token passages, reached by 16 concurrent requests (`MAX_INFLIGHT=8`). Past that point, extra concurrency only adds queueing (p50 about 0.4 s at 32 concurrent).
  - A 3.5-minute sustained run at 32 concurrent held 85 items/s with 0 errors. The GPU-duty-cycle autoscaler did not add the second replica in that window.
  - Cloud Run reached 78 items/s on short inputs but only 39 on long ones. It runs with `KV_CACHE_GB=2` and a 4,096-token context, which limits batching of long prompts.
  - At 32 concurrent, gemini-3.5-flash-lite reached 29 items/s and gemini-3.8-flash 7.
- **Live cascade** (served, not simulated): accuracy and escalation rate match the offline estimate within 1–2 items. p50 stays at dgem's 94–133 ms; p95/p99 is Gemini's latency (3.5–15 s) on the escalated items.
- **Gemini thinking budget 0:** saves 0.4–1.4 s at p50 on gemini-3.8-flash, with no significant accuracy change on 8 paired suites. The model still emitted 27–116 thinking tokens per item. It remains about 20× slower than dgem on G4.
- **Cost per 1,000 judgements** at list prices:

  | Judge | USD per 1k |
  | :--- | :---: |
  | dgem, fully utilized (G4 at $5.85/h, Cloud Run at $3.19/h) | ~0.01–0.02 |
  | gemini-3.5-flash-lite | 0.27–0.51 |
  | gemini-3.8-flash (introductory price; double from 2027) | 1.06–2.25 |
  | Cascade | 0.10–0.75 |
  | Vertex predefined metrics | ≥ 10–31 |

  An always-on G4 replica costs $140/day, so it beats 3.8-flash on cost only above roughly 60k–130k judgements per day. Cloud Run has no idle cost but pays a cold start.
- **Position bias.** dgem picks a different pairwise winner after the two responses are swapped on 12–27% of items, on both dates. The mirror read changes accuracy by −4 to +5 points, which is not significant.
- **Predefined-safety mapping.** Mapping the Vertex predefined safety metric onto dgem, which sees only the response, scores 68. A purpose-written template scores 80. Use the template.

## Capability verdicts

| Capability | Verdict |
| :--- | :--- |
| Exact match, BLEU, ROUGE, tool-call and trajectory metrics | No model needed (mizan `kind: computation`, local, no credentials) |
| Binary safety / harm, groundedness / faithfulness | DiffusionGemma can replace the Gemini judge; the cascade beats either alone |
| Toxicity / jailbreak detection | DiffusionGemma as a front filter plus entropy cascade |
| Likert quality | Close to parity on rank correlation |
| General pairwise preference (RewardBench) | DiffusionGemma plus mirror, or the cascade |
| Expert and multi-turn pairwise (MT-Bench) | Prefer an autoregressive judge (Gemini ahead by 5–10 points on both dates; significant on one) |
| Instruction-following pairwise (LLMBar) | Autoregressive judge required (−15 points, significant on both dates) |
| High-volume or latency-sensitive judging | DiffusionGemma (about 80 items/s per G4 replica, about $0.02 per 1k at full use) |
| Rubric generation, explanations, audio/video | Autoregressive only |

## Notes

- **Prompt size.** mizan's diffusion adapter now sends each field once. It previously also sent a rendered copy, which pushed long HaluBench passages past the 4,096-token context of the Cloud Run deployment. Keep this limit in mind for long-context policies on Cloud Run.
- **Predefined metrics.** They use the Vertex service's own judge, documented as `gemini-2.5-flash`, and the API rejects any judge override. Gemini 2.5 retires 2026-10-20, so predefined-metric numbers may change when the service switches judges.
- **Failed calls.** On 09-26, gemini-3.8-flash queued requests for 80–150 s and predefined metrics failed 20–34% of first attempts. Failed items were re-run on both engines rather than dropped. On 09-29 the full rerun took 24 minutes. The only unrecovered failures are deterministic: 3 empty Gemini responses and 3 unparseable native pairwise replies.
