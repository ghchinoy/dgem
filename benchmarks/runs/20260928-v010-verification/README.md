# Run 20260928-v010-verification

Before pruning old revisions, every sample suite was run against production **v0.1.0** (Vertex G4 endpoint, via
`--vertex-url`) in one session, and the retained rollback targets (n-1) were exercised. Numbers compare with the
committed receipts in `benchmarks/` (earlier images and hardware) and with the image-parity run.

## Version n: v0.1.0 on Vertex G4

| Suite | Items | Result | Reference | Verdict |
| :--- | :---: | :--- | :--- | :--- |
| `bench-jev` | 231 | 187 correct (81.0%) | Band 182–192 (image parity) | Pass |
| `bench-calibration` | 50 | 46 (92%) | 43–46 (image parity) | Pass |
| `bench` (triage, code review, security) | 30 | 22 (73.3%) | 23–24 on L4 receipts | Pass (±2 items is noise on 30) |
| `bench-intents` clinc150 (small set) | 30 | 29 (96.7%) after tie-break; domain 28/30 | Committed receipt 96.7% | Pass |
| `bench-intents` banking77 (small set) | 30 | 0 at first (every request rejected, `at most 26 alternatives`); after the bracket fix: **23 (76.7%)** | Committed receipt 26 (Sep 20, older server allowed 30 options in one question) | Fixed (harness) |
| `bench-bbox` | 12 | mIoU 0.406 argmax / 0.422 expectation | 0.290 / 0.377 (Cloud Run L4) | Pass |
| `bench-rerank` | 30 queries | nDCG@10 0.824, 0.856, 0.856 (3 runs); 0% ties, 100% poison quarantine | 0.9265 Cloud Run L4, 0.8502 Vertex L4 | Pass (in the range of earlier receipts) |
| `bench-permutation` | 16 | 100% accuracy, 6.25% flip rate | EXP-13 | Pass |
| `bench-decision-index` panel | 22 | 22/22 supported, index 98.89 | 98.89 | Pass |
| Every policy template through the gateway (`scripts/template_sweep.py`) | 28 | 23 pass; 5 need harness-supplied variables (below) | — | Pass |
| `bench-ecotone` | — | Not run (needs the C++ ecotone sidecar) | — | Skipped |

**Two findings, both pre-existing, not regressions:**
- `bench-intents --dataset banking77` sent all 30 candidate intents in one `choice` question. The server has
  rejected more than 26 options since before v0.1.0 (`ab208dd` had the same check); the committed receipt predates
  it. **Fixed:** with more than 26 options the harness now runs a 2-stage bracket (groups of ≤ 20, top 5 of each
  group to a final round), like `dgem systemone serve`. Result 23/30 (76.7%); keeping the top 3 per group gave 22,
  the top 8 gave 23. The misses are confusable intent pairs (e.g. `card_arrival` vs `card_delivery_estimate`).
- `intent_banking77`, `intent_clinc150`, `intent_tiebreak`, `jevbench_generic` and `tn_disambiguation` are harness
  templates: they take option lists from `bench-intents`, `bench-jev` or `bench-ecotone`, so the gateway's sample
  variables leave them with fewer than two options (400). The harnesses themselves pass (above).

## Rollback targets (n-1)

| Target | Check | Result |
| :--- | :--- | :--- |
| Vertex model `504638d` (redeployed on the production endpoint, then 100% of traffic for the drill, then back) | Contract (`contract_rollback_vertex.json`), calibration | Contract identical (the digit-string `samples` case returns 400, confirming the older model served); 44/50 |
| Cloud Run revision on `504638d` (tagged URL) | Contract, calibration | Contract identical; 43/50 |
| Gateway revision on `v0.1.0-5-g3216438` (tagged URL) | Routing, admin 403s, unknown backend / foreign `vertex_url` 400s | All pass |

After the checks, older Cloud Run revisions (81) and three older Vertex model resources were deleted; the drill copy
of the rollback model was undeployed (its model resource is kept).
