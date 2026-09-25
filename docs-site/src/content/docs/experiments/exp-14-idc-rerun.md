---
title: "EXP-14: Same-Session IDC Re-run on Vertex AI (versioned runs)"
description: "Pre-registered follow-ups PROP-01, PROP-02 (offline), PROP-05, PROP-10 and an EXP-13 reproduction, run in one session against the Vertex AI endpoint with versioned receipts. Null-prior helps on the 50-item suite but not on JevBench; the dual-mirror slot name degraded readings; held-out temperature scaling works on 231 items but not 50."
---

# EXP-14: Same-Session IDC Re-run on Vertex AI

**Date:** 2026-09-25 · **Backend:** Vertex AI Dedicated Endpoint `4217256562927861760` (1× L4) · **Stage 2:** `gemini-3.8-flash`
**Runs:** [`benchmarks/runs/20260925-vertex-idc`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-vertex-idc/manifest.json),
[`20260925-vertex-idc-mirrorfix`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-vertex-idc-mirrorfix/manifest.json),
[`20260925-offline-prop01`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-offline-prop01/cv_temperature.json) ·
**Reproduce:** `RUN_ID=<new id> ./scripts/run_idc_rerun.sh`, then `python3 scripts/bench_runs.py compare`

This experiment runs the follow-ups from the [Proposed Experiments Register](/dgem/experiments/proposed/) that could be done
with existing harnesses: `PROP-01` (held-out temperature), `PROP-10` (noise floor), `PROP-05` (merge rules),
`PROP-02` (cascade gates, offline), and a reproduction of [EXP-13](/dgem/experiments/exp-13-permutation-invariance/) on Vertex.
It is also the first time `dgem` itself (rather than the upstream reference) was run on all 231 JevBench items.
All receipts are versioned (see [`benchmarks/runs/README.md`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/README.md)); legacy receipts are
registered as run `20260920-legacy` for comparison.

## Headline findings

1. **Noise floor (`PROP-10`).** Three interleaved baselines on the 50-item suite gave identical answers (44/50 each)
   but Brier 0.175–0.193 and ECE 0.046–0.062. A fresh baseline an hour later scored 45/50. On JevBench, two
   baselines scored 187 and 189/231. **Differences of ±1 item (50) or ±2 items (231), or ±0.01 Brier, are noise.**
2. **Null-prior de-biasing is suite-dependent.** On the 50-item suite it reproduced: Brier 0.147 and ECE 0.026 vs
   baseline 0.175–0.193 / 0.046–0.062, accuracy 45 vs 44. **On JevBench (231) it did not help:** 186 vs 187 correct,
   Brier 0.293 vs 0.264, ECE 0.104 vs 0.081. It should not be a default.
3. **The dual-mirror slot name was a bug.** The reversed slot was named `<id>__mirror_rev`. Slot ids are visible to
   the model, and the word "mirror" degraded *both* readings. On an easy intent item the forward reading fell from
   0.99 to 0.62–0.77 and the reversed slot picked the wrong answer; neutral ids (`__rev`, `_reversed`, `_b`) were
   unaffected. With the old name, JevBench fell from 187 to **145/231**. Fixed in commit `2f731b0` (default suffix
   `__rev`; `--mirror-slot-suffix=__mirror_rev` reproduces the old behaviour). This likely explains the earlier
   dual-mirror Brier/ECE regressions reported in EXP-13 and IDC §6.
4. **Even with the fix, a second slot on the same canvas hurts the forward reading on JevBench** (forward-only
   161–170 vs baseline 189). The best recombination, a 50/50 average with real option names, reaches 185/231, close to
   but not better than the single-slot baseline. On the 50-item suite dual-mirror is within noise of baseline.
   **Recommendation: do not enable `--dual-mirror` in production;** keep it as a research diagnostic. This supports
   the coupling concern in `PROP-03`.
5. **Held-out temperature scaling (`PROP-01`) works on 231 items, not on 50.** On the 50-item suite the in-sample
   56% ECE reduction shrinks to 5–7% held out (legacy receipt) or becomes worse than T=1 (new baseline). On `dgem`'s
   JevBench baseline, held-out ECE falls from 0.081 to 0.054–0.061 (24–33%, 19–20 of 20 repeats better than T=1),
   with T* ≈ 1.5. Transfer across suites is unreliable.
6. **Cascade (`PROP-02`, offline).** On JevBench, sending items with hesitation ≥ 16% (39%) to Gemini gives
   **221/231 (95.7%)**; ≥ 35% (27%) gives 214 (92.6%); Gemini on everything gives 225. On the 50-item suite,
   26–34% escalation gives 48/50 (96%), equal to Gemini alone (48/50 in this session). The mirror-aware gate could
   not be evaluated fairly because of findings 3–4.
7. **EXP-13 reproduction on Vertex.** The content-free slot-A habit reproduced (87% / 76% / 57% for K = 2 / 3 / 4,
   vs 88% / 78% / 49% on Cloud Run). Cyclic reordering flipped 3/16 items (3 of 4 ambiguous) vs 2/16 before.
   `perm_08`, the IDC worked example, behaved differently: single-slot hesitation was 0.196 (already escalated by
   entropy) and Mirror TVD only 0.064. `perm_06` again was not caught by the mirror (TVD 0.001).

## Results tables

### 50-item public calibration suite (T = 1)

| Run / config | Correct | Brier | ECE-10 | >0.9 conf correct | mean Mirror TVD |
| :--- | ---: | ---: | ---: | ---: | ---: |
| legacy baseline (Cloud Run, Sep 20) | 44 | 0.186 | 0.075 | 34/36 | — |
| legacy null-prior (Cloud Run, Sep 23) | 45 | 0.149 | 0.061 | 34/34 | — |
| vertex-idc baseline r1 / r2 / r3 | 44 / 44 / 44 | 0.185 / 0.193 / 0.175 | 0.061 / 0.062 / 0.046 | 34/36 each | — |
| vertex-idc null-prior | 45 | **0.147** | **0.026** | 34/35 | — |
| vertex-idc dual-mirror (`__mirror_rev`) | 45 | 0.186 | 0.106 | 25/25 | 0.224 |
| vertex-idc dual-mirror, real names (`__mirror_rev`) | 45 | 0.169 | 0.062 | 24/25 | 0.186 |
| vertex-idc null-prior + dual-mirror (`__mirror_rev`) | 45 | 0.180 | 0.074 | 20/21 | 0.283 |
| mirrorfix baseline | 45 | 0.179 | 0.069 | 34/37 | — |
| mirrorfix dual-mirror (`__rev`) | 46 | 0.177 | 0.100 | 30/31 | 0.156 |
| mirrorfix dual-mirror, real names (`__rev`) | 45 | 0.194 | 0.098 | 31/34 | 0.126 |
| mirrorfix null-prior + dual-mirror, real names | 46 | 0.178 | 0.086 | 27/29 | 0.151 |
| Gemini 3.8 Flash alone (this session) | 48 | 0.053 | 0.041 | 39/39 | — |

### JevBench v1.3.1 (231 items, T = 1)

| Run / config | Correct | Easy / Std / Hard | Brier | ECE-10 | mean Mirror TVD |
| :--- | ---: | :--- | ---: | ---: | ---: |
| legacy upstream djev reference | 194 | 48 / 71 / 75 | 0.243 | 0.077 | — |
| vertex-idc `dgem` baseline | 187 | 48 / 67 / 72 | 0.264 | 0.081 | — |
| vertex-idc null-prior | 186 | 48 / 65 / 73 | 0.293 | 0.104 | — |
| vertex-idc dual-mirror (`__mirror_rev`) | 145 | 34 / 47 / 64 | 0.469 | 0.052 | 0.441 |
| vertex-idc dual-mirror, real names (`__mirror_rev`) | 146 | 27 / 51 / 68 | 0.475 | 0.088 | 0.503 |
| mirrorfix baseline | 189 | 48 / 65 / 76 | 0.264 | 0.089 | — |
| mirrorfix dual-mirror (`__rev`) | 163 | 46 / 50 / 67 | 0.364 | 0.079 | 0.239 |
| mirrorfix dual-mirror, real names (`__rev`) | 169 | 43 / 62 / 64 | 0.318 | 0.089 | 0.277 |
| Gemini 3.8 Flash on all items (prior-guided by baseline) | 225 | — | 0.047 | 0.052 | — |

Merge rules on the mirrorfix dual-mirror receipts (offline, `scripts/analyze_idc.py merge-rules`, exact-match items):

| Receipt | forward only | reversed only | 50/50 mean | geometric | 70/30 (production) |
| :--- | ---: | ---: | ---: | ---: | ---: |
| JevBench, aliased names | 161 | 177 | 176 | 175 | 163 |
| JevBench, real names | 170 | 154 | **185** | 183 | 177 |
| 50-item, aliased names | 36 | 32 | 35 | 34 | 36 |

### Held-out temperature scaling (`PROP-01`)

5-fold cross-validation repeated 20×, T grid 0.50–3.50, ECE pooled over out-of-fold predictions
(`scripts/analyze_idc.py cv-temperature`).

| Receipt | n | ECE T=1 | ECE in-sample (T*) | ECE held-out, mean [range] | Held-out reduction | Repeats better than T=1 |
| :--- | ---: | ---: | :--- | :--- | ---: | ---: |
| legacy 50-item baseline, fit by ECE | 50 | 0.075 | 0.033 (1.35) | 0.071 [0.032–0.121] | 5% | 12/20 |
| legacy 50-item baseline, fit by NLL | 50 | 0.075 | 0.088 (1.45) | 0.069 [0.038–0.096] | 7% | 11/20 |
| vertex-idc 50-item baseline, fit by ECE | 50 | 0.061 | 0.045 (1.35) | 0.070 [0.032–0.128] | −15% | 11/20 |
| upstream djev JevBench, fit by ECE | 231 | 0.077 | 0.077 (1.25) | 0.075 [0.064–0.111] | 3% | 16/20 |
| **`dgem` JevBench baseline, fit by ECE** | 231 | 0.081 | 0.040 (1.50) | **0.054** [0.040–0.081] | **33%** | 19/20 |
| **`dgem` JevBench baseline, fit by NLL** | 231 | 0.081 | 0.063 (1.65) | **0.061** [0.049–0.071] | **24%** | 20/20 |

### Cascade gates (offline, `PROP-02`)

Escalated items take Gemini's answer from a run on every item (JevBench: prior-guided by the baseline; 50-item suite:
standalone Gemini). Three JevBench Gemini calls failed with HTTP 429 after retries and count as their Stage-1 answer.

| Suite | Gate (hesitation ≥) | Escalated | Correct |
| :--- | :--- | ---: | ---: |
| JevBench, `dgem` baseline | 0.10 | 115 (50%) | 222 / 231 |
| | **0.16** | **90 (39%)** | **221 / 231** |
| | 0.25 | 73 (32%) | 218 / 231 |
| | 0.35 | 62 (27%) | 214 / 231 |
| | 0.50 | 46 (20%) | 210 / 231 |
| 50-item, baseline r1 | 0.16 | 17 (34%) | 48 / 50 |
| | 0.25 | 13 (26%) | 48 / 50 |
| | 0.35 | 11 (22%) | 47 / 50 |

## Addendum: replication on Vertex G4 (run `20260925-g4-idc`)

Same four configurations re-run on the new G4 endpoint (`4423577720856772608`, `g4-standard-48` + RTX PRO 6000,
image `dgemma:ab208dd`, mirror suffix `__rev`), all in one session. Results agree with the findings above.

| Suite | Config | Correct | Brier (T=1) | ECE-10 (T=1) | Mean Mirror TVD |
| :--- | :--- | ---: | ---: | ---: | ---: |
| 50-item | baseline | 44/50 | 0.176 | 0.034 | — |
| 50-item | null-prior | 46/50 | 0.148 | 0.037 | — |
| 50-item | dual-mirror | 46/50 | 0.197 | 0.136 | 0.133 |
| 50-item | null-prior + dual-mirror | 46/50 | 0.188 | 0.114 | 0.190 |
| JevBench | baseline | 187/231 | 0.286 | 0.097 | — |
| JevBench | null-prior | 183/231 | 0.304 | 0.104 | — |
| JevBench | dual-mirror | 156/231 | 0.349 | 0.083 | 0.223 |
| JevBench | null-prior + dual-mirror | 155/231 | 0.353 | 0.126 | 0.290 |

- Null-prior again helps on the 50-item suite (Brier 0.148 vs 0.176) and not on JevBench (183 vs 187).
- Dual-mirror again costs accuracy on JevBench. Offline re-scoring (`analysis/merge_jevbench_dual_mirror.json`)
  shows the **forward slot on the mirror canvas** gets only 156/230 right (it agrees with the single-slot answer on
  178/231 items), while symmetric merges (50/50 mean or geometric) recover **185/230**, close to baseline. The
  coupling between slots on one canvas, not the merge alone, is the main cost, which strengthens the case for
  `PROP-03`.
- The GPU change (L4 → RTX PRO 6000) did not change accuracy: JevBench baseline 187/231 on both.

## Why the mirror degrades and null-prior overcorrects (analysis)

Per-item analysis of the `20260925-vertex-idc*` receipts (read-only, `scripts/analyze_idc.py` plus ad-hoc scripts).

**1. The two slots copy each other's letter.** `structured_server.py` labels every `choice` option **A, B, C…** and
the model fills lines such as `decision: X` and `decision__rev: Y`. In the reversed slot the same letters stand for
different options (forward A = first option; reversed A = last option). Because both blanks are filled together and
see each other, the model tends to write the **same letter** in both, which means **contradictory options**:

| Run (JevBench, 230 items with mirror data) | Slots agree on the option | Same **letter**, different option | Forward answer changed vs. baseline when same-letter | Forward changed otherwise |
| :--- | ---: | ---: | ---: | ---: |
| Old slot name `__mirror_rev`, aliased names | 91 | **114** | 32/114 | 24/116 |
| New slot name `__rev`, aliased names | 167 | 48 | **31/48 (65%)** | 17/182 (9%) |
| New slot name `__rev`, real option names | 141 | 76 | 23/76 (30%) | 13/154 (8%) |

This fits every observation so far: the word "mirror" roughly doubled letter-copying (114 vs 48); the damage is
worst on 3–4-option choices and 4-level scores (0.18–0.32 of probability lost on the correct answer) and small at
6+ options; and the 50-item suite was barely affected because its yes/no questions are labelled with the words
`yes`/`no` in the forward slot, so there is no shared letter to copy. JevBench converts yes/no into lettered choices,
so both slots collide. The mechanism is a hypothesis until `PROP-11` tests it directly.

**2. Null-prior corrects for a habit the model doesn't show on JevBench.** The correction divides out a slot-A
preference measured on **blank** questions (87% / 76% / 57% for 2 / 3 / 4 options). On real JevBench items the
correct answer is in position A on 70 items, and the baseline picks position A on only 62 (and position B on 93), so
dividing out the blank-question habit pushes answers away from A. Three yes/no items whose correct answer "yes" is
listed first broke under the correction. On the 50-item suite the habit does appear in context, which is why the
correction helped there. Tested by `PROP-14` and `PROP-15`.

Follow-ups are pre-registered as `PROP-11`–`PROP-16` in the [Proposed Experiments Register](/dgem/experiments/proposed/) and will be run
one at a time, in that order, on the Vertex G4 endpoint.

## Verdicts on pre-registered hypotheses

| ID | Pre-registered criterion | Verdict |
| :--- | :--- | :--- |
| `PROP-01` | Held-out ECE reduction ≥ 25% with CI excluding 0 | **Met on `dgem` JevBench (231)** by ECE-fit (33%), borderline by NLL-fit (24%, 20/20 repeats better). **Not met** on the 50-item suite or on the upstream receipt. Do not cite the 50-item in-sample 56%. |
| `PROP-10` | Within-revision accuracy variation ≤ 2 pts, Brier ≤ 0.01 | **Met for accuracy** (0–2 pts); **Brier varied by up to 0.018** across three baselines. Minimum meaningful effect: > 2 items (50) / > 3 items (231), > 0.02 Brier. |
| `PROP-05` | Symmetric merge non-inferior (≤ 0.01 Brier) to single-slot baseline and better than 70/30 | **Better than 70/30** on JevBench with real names (185 vs 177); **not non-inferior** to the single-slot baseline (Brier 0.317 vs 0.264). |
| `PROP-02` | IDC + mirror gate beats entropy-only at matched escalation | **Not testable yet**: the mirror readings were degraded by the slot-name bug and slot coupling. Entropy-only cascade results recorded above. |

## Consequences

- IDC docs, README and Studio text updated: null-prior is suite-dependent; `--dual-mirror` is a diagnostic, not a
  production default; temperature scaling can help on larger sets but must be fit on held-out data.
- Code: dual-mirror slot suffix renamed to `__rev`; receipts now carry per-item raw forward/reversed distributions
  (`idc` block); Gemini Stage-2 calls retry transient errors.
- Next: `PROP-11` → `PROP-16`, one at a time (see the analysis above).
