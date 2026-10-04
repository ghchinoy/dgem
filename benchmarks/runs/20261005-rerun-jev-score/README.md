# Re-run: JevBench through `bench-jev` with ordinal score items (#84, 2026-10-05)

Production v0.3.1 (Vertex G4, 8k). `dgem bench-jev` ×3 with the new default (score items sent as ordinal `score`
questions with their rubric levels) and ×3 with `--flatten-score` (the behaviour before 2026-10-04: a choice over bare
level indices, no rubric), interleaved in one session.

| | run 1 | run 2 | run 3 |
|---|---|---|---|
| ordinal: total / 231 | 190 | 189 | 192 |
| ordinal: score items / 18 | 15 | 14 | 15 |
| flatten: total / 231 | 186 | 191 | 192 |
| flatten: score items / 18 | 9 | 14 | 14 |

Score items: mean accuracy 0.815 (ordinal) vs 0.685 (flatten), +2.3 items of 18 (2 SE 4.0, not significant on 18
items); mean absolute error in levels 0.19 vs 0.32. Totals are within the JevBench noise floor (about ±4 items between
identical runs). The `/v1/systemone` path (matrix `jev_systemone`) always sent the levels.
