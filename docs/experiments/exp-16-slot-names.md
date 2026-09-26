---
title: "EXP-16: Slot Names Are Part of the Prompt (PROP-16)"
description: "Question ids are visible to the model. A single slot's id does not matter (decision, q1, random, mirror, check all within noise), but a loaded id on a second slot (__mirror_rev) cost 19 JevBench items even with no letter collision."
---

# EXP-16: Slot Names Are Part of the Prompt (`PROP-16`)

**Date:** 2026-09-26 · **Backend:** Vertex AI G4 endpoint `4423577720856772608` (g4-standard-48 + RTX PRO 6000, image `dgemma:ab208dd`) ·
**Run:** [`benchmarks/runs/20260926-prop16-slot-names`](../../benchmarks/runs/20260926-prop16-slot-names/manifest.json) · **Code:** commit `58cae3a` (`bench-jev --slot-id`)

## Question and pre-registered rule

The model server writes each question id into the prompt and answer template (`decision: B`). EXP-14 saw JevBench fall
from 163 to 145 when the mirror slot was named `__mirror_rev` instead of `__rev`, but that run also had letter
collision. **H16:** slot ids act as instructions; loaded words change answers more than neutral ids.

Rule (registered before data): an id "matters" if its correct count is more than 3 items outside the band of three
same-session baselines, or, for two-slot schemas, more than 3 items away from `copy` + `__rev`. Two-slot conditions use
`--mirror-mode copy`, so both slots list the same options with the same letters and **letter collision is ruled out**.

## Results (JevBench v1.3.1, 231 items, one session)

Baselines (id `decision`): **183, 189, 189** → band **[183, 189]**; baselines agree with each other on 215–218 answers.

| Condition | Correct | Verdict | Same answer as baseline majority |
| :--- | ---: | :--- | ---: |
| single slot, id `q1` | 184 | within band | 214 |
| single slot, id `x7k2q` (random) | 183 | within band | 212 |
| single slot, id `mirror` | 183 | within band | 209 |
| single slot, id `check` | 185 | within band | 214 |
| two slots, copy, second id `decision__rev` | 181 (second slot 185) | reference | 210 |
| two slots, copy, second id `decision_b` | 180 (second slot 177) | within 3 of reference | 213 |
| two slots, copy, second id `decision__mirror_rev` | **162** (second slot **135**) | **matters (−19)** | 186 |

## Verdict

**Partly supported.** For a single slot the id doesn't matter: neutral, random, and loaded words are all within noise.
For a **second** slot, a loaded name matters a lot: with identical options and letters, naming it `__mirror_rev` dropped
that slot to 135 and pulled the forward answer down to 163. The likely reading is that "mirror" next to a copy of the
same question invites the model to answer differently. This confirms that the EXP-14 collapse had two causes: letter
collision (EXP-15) and the slot name (this experiment).

Two-slot copy conditions (180–181) sit 2–3 items below the baseline band minimum. That is inside the pre-registered
tolerance but consistent with EXP-15's `copy` (184); a small cost of a second slot cannot be ruled out.

## Consequences

- **Template style rule:** in schemas with more than one question, use neutral ids that describe what is asked
  (`team`, `urgent`, `decision_b`). Avoid ids that suggest the answer should differ or be inverted (`mirror`,
  `reverse`, `opposite`, `alt`, `check_again`). Added to `docs/templates.md` and `AGENTS.md`.
- The dual-mirror default suffix `__rev` stays; `--mirror-slot-suffix __mirror_rev` is kept only to reproduce old runs.
- Next: `PROP-12` (separate-pass mirror).
