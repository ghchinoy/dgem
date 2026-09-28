# Run 20260927-image-parity

Side-by-side validation of the upstream-based serving image against the previous production image,
before promotion. Same hardware, same client, same session.

| Image | Built from | vLLM | Structured server |
| :--- | :--- | :--- | :--- |
| `ab208dd` (previous production) | fork base + patches | nightly `dee37d8` | original dgem server |
| `56baadf` (public image at the time) | `deploy/` on `main` after PR #5 | upstream nightly `a9eafde` | vLLM example + `dgem.patch` |
| `504638d` (promoted) | `56baadf` + fixes found here | upstream nightly `a9eafde` | as above + 4 fixes (below) |

Targets: Vertex AI dedicated endpoint (`g4-standard-48` + 1× RTX PRO 6000) and Cloud Run
(1× RTX PRO 6000, Direct VPC egress). Canaries ran next to production and were deleted afterwards.

## Regressions found and fixed before promotion

| # | Symptom on `56baadf` | Cause | Fix |
| :--- | :--- | :--- | :--- |
| 1 | `/health` lost `vllm_ready`, `phase`, `bytes_staged_gb`, `warmed` (MCP health tool and Studio cold-start view go blind) | upstream server answers `/health` with `{status, server}` only | merge `warmup_state.json` again (`c23b491`) |
| 2 | 1–8% HTTP 503 `Model server early terminated the request (truncated headers)` at 16–32 concurrent requests on Vertex (the old image also showed 0.2–0.8%) | HTTP/1.0 server closes every connection; listen backlog 5 | HTTP/1.1 keep-alive + backlog 256 (`78b1b86`, `504638d`) |
| 3 | `warmed` missing even after warmup completed | tmpfs reclaim step overwrote the state file | atomic merge in both writers (`504638d`) |
| 4 | `dgem-weights` (baked) image copied 17.53 GiB from Cloud Storage anyway and ignored its baked weights | staging block ran whenever the bucket was mounted | skip staging when weights are baked (`504638d`) |

Also restored `/predict` and `/rawPredict` aliases (`4676127`). Harness fixes found along the way:
`--vertex-url` silently fell back to localhost when the endpoint ID could not be expanded (0% runs,
discarded and re-run); `bench_runs.py` lost manifest entries under parallel `exec` (now locked).

## Contract (`contract_*.json`, `scripts/contract_diff.py`)

13 cases: valid decisions (samples 1/4/default, dual-mirror, think=120, image, 26 options, `depends_on`
→ 2 reads, `/v1/systemone`) and error cases (27 options, unknown type, empty questions, invalid JSON).
Status codes, error messages, envelope keys, diagnostics keys and read counts are identical on every
image and platform. Answer differences are limited to cases that also flip between repeated calls on
the same image (`choice_26`, the dual-mirror score slot, `think120`: wrong answer 4/40 on `ab208dd`
vs 6/40 on `504638d`, Fisher p ≈ 0.7).

## Accuracy and calibration (`jevbench__*.json`, `calibration__*.json`)

JevBench 231 items and the 50-item calibration suite, `samples=1`, 4 workers. "upstream" pools
`56baadf` (3 runs) and `504638d` (2 runs); they share the model runtime (the fixes are HTTP-layer only).

| Platform | Suite | `ab208dd` correct per run | upstream correct per run | Mean Brier old → new |
| :--- | :--- | :--- | :--- | :--- |
| G4 | JevBench | 187, 187, 188 | 186, 186, 183, 186, 186 | 0.269 → 0.261 |
| Cloud Run | JevBench | 187, 187, 185 | 186, 187, 185, 192, 191 | 0.270 → 0.265 |
| G4 | Calibration | 45, 45, 44 | 44, 45, 44, 44, 43 | 0.186 → 0.177 |
| Cloud Run | Calibration | 44, 44, 45 | 44, 44, 45, 45, 44 | 0.181 → 0.168 |

All runs fall inside the established JevBench noise band (182–192). Pooled across platforms the mean is
186.8 correct for both images. Brier is slightly better on the new runtime in all four cells.

Per-item agreement (`scripts/receipt_agreement.py`):

| Platform / suite | Within `ab208dd` | Within upstream | Across images |
| :--- | ---: | ---: | ---: |
| G4 JevBench | 96.8% | 95.3% | 90.3% |
| Cloud Run JevBench | 97.4% | 92.9% | 91.5% |
| G4 calibration | 98.7% | 94.8% | 92.8% |
| Cloud Run calibration | 97.3% | 94.4% | 89.9% |

Accuracy is unchanged, but roughly 5% of items change answer because of the runtime itself (beyond
run-to-run noise), and repeat runs agree slightly less with each other on the new runtime. Experiment
results measured on `ab208dd` (EXP-14 to EXP-17) remain valid as aggregates; do not mix images in
per-item comparisons.

## Latency (`speed_modes.json`, `speed_modes_504638d.json`; 30 sequential requests, p50 ms)

Clean run, all four targets back to back, `504638d` vs `ab208dd`:

| Mode | G4 old denoise | G4 new denoise | CR old denoise | CR new denoise | G4 old wall | G4 new wall | CR old wall | CR new wall |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| samples=1 | 60.2 | **55.3** | 63.0 | **60.8** | 157.5 | **149.7** | 160.3 | **152.8** |
| samples=4 | 100.8 | **91.3** | 107.4 | **101.2** | 195.1 | **190.0** | 209.0 | **181.8** |
| dual-mirror, samples=1 | 62.6 | **57.9** | 66.4 | **65.8** | 157.3 | **152.4** | 165.1 | **151.6** |
| think=120 | 388.7 | **364.2** | 424.2 | **376.3** | 507.1 | **462.9** | 536.0 | **462.0** |
| image, samples=1 | 69.3 | **63.8** | 76.1 | **75.7** | 191.7 | **182.9** | 201.2 | **184.1** |

(`speed_modes.json` repeats the comparison for `56baadf` with the same result.)

## Concurrency on G4 (`speed_sweep*.json`)

256 requests per cell at 16 and 32 concurrent workers:

| Image | samples | 16 workers: ok, req/s, p50 ms | 32 workers: ok, req/s, p50 ms |
| :--- | :--- | :--- | :--- |
| `ab208dd` | 1 | 255/256, 59.3, 252 | 256/256, 63.1, 492 |
| `504638d` | 1 | **256/256**, 54.3, 254 | **256/256, 72.3, 426** |
| `ab208dd` | 4 | 256/256, 35.9, 439 | 256/256, 35.1, 889 |
| `504638d` | 4 | **256/256, 39.0, 349** | **256/256, 45.2, 685** |

Before the keep-alive fix, `56baadf`/`4676127` dropped 4–20 of 256 requests per cell at samples=1.

## Cloud Run cold start (`coldstart.jsonl`, `scripts/coldstart_probe.py`)

Seconds from revision creation. Same service, RTX PRO 6000, image `56baadf` variants.

| Configuration | Runs | First log line (image pull) | Weight copy | Warmed |
| :--- | ---: | ---: | ---: | ---: |
| Lean image, Direct VPC egress | 3 | 4 | 64–76 | **135–159** |
| Lean image, public egress | 1 | 3 | 386 | 461 |
| Baked-weights image (first pull of the image) | 1 | **742** | 64 (bug #4) | 878 |
| Baked-weights image (image already cached) | 1 | 24 | 74 (bug #4) | 179 |
| Baked-weights image `4b1b809` (fix #4, fresh instance) | 1 | 474 | none (baked weights used) | 626 |

The baked-weights image does not help on Cloud Run: its first pull of ~26 GB took over 12 minutes, and
even with the image cached it was slower than the lean image. Bug #4 meant these runs also copied weights
from Cloud Storage, so the cached-image number is an upper bound; the first-pull penalty applies to every
new image regardless. With fix #4 the baked image skips the copy, but its ~8-minute image pull still makes it slower than the lean image. Recommendation: lean image + Direct VPC egress.

## Promotion (2026-09-28)

- Vertex endpoint: `504638d` deployed next to `ab208dd` at 0% traffic, switched to 100%, contract
  re-checked, old model undeployed (its model resource remains uploaded for rollback).
- Cloud Run `dgemma`: `504638d` as a no-traffic tagged revision, contract checked on the tag URL,
  traffic migrated. Rollback: previous revision.
- Gateway: rebuilt from `main` (PRs #5–#11), deployed with no traffic, checked (backend config, all three
  backends, MCP `tools/list` and health tool), traffic migrated, then re-checked through the IAP-fronted
  custom domain.

## Public images published (2026-09-28)

Built in `dgem-diffusiongemma` from `4b1b809` (includes the four fixes above plus digit-string `samples`):
`dgem:4b1b809@sha256:edc06728…abcfc` and `dgem-weights:4b1b809@sha256:7cfbbb92…3c56`. The lean image passed the
contract probe against production (identical except `samples: "4"`, which it now accepts), latency matched
(61.5 ms / 101 ms / 75 ms GPU for 1 sample / 4 samples / image), and 256 requests at 16 and 32 workers had 0
errors. The baked image loaded its own weights (no Cloud Storage copy).

## v0.1.0 in production (2026-09-28)

`v0.1.0` (`dgem@sha256:5fa4a866…`, same serving tree as `4b1b809`) now serves both production backends.
On the production Vertex endpoint (`speed_modes_v010_g4.json`, `speed_sweep_v010_g4.json`): GPU p50 55.2 ms
(1 sample), 91.0 ms (4 samples), 63.2 ms (image); 0 errors in 1,024 requests at 16 and 32 workers
(73.9 decisions/s at 1 sample, 45.8/s at 4 samples).
