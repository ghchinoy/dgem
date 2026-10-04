# Issue #75: Stage-2 prior (2026-10-04)
Production v0.3.0 Stage 1, gemini-3.8-flash Stage 2. Decision rule in the PR.
- `vision_<prior>_t<threshold>.json`: `bench-vision --only-scored --cascade-threshold T --cascade-prior P` on the 230-item
  seeded sweep (items trimmed to scoring fields).
- `text_t<threshold>.jsonl`: JevBench public (231) through a local `dgem serve` `/api/decide` with `cascade_mode=entropy`
  and `stage2_prior` full/soft/none.
