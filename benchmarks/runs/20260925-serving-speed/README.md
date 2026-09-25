# Run 20260925-serving-speed

Serving speed review across three backends, measured from one client in the same session with
`scripts/serving_speed.py` (every request records HTTP status and error text; failures are never
counted as zero latency).

| Target | Endpoint | Hardware | Image | Engine settings |
| :--- | :--- | :--- | :--- | :--- |
| `l4` | Vertex `4217256562927861760` | `g2-standard-16` + 1× L4 | pre-`ab208dd` | `DISABLE_MM=1`, `KV_CACHE_GB=2`, no in-flight limiter |
| `g4` | Vertex `4423577720856772608` | `g4-standard-48` + 1× RTX PRO 6000 | `dgemma:ab208dd` | `DISABLE_MM=0`, `KV_CACHE_GB=12`, `MAX_SEQS=32`, `MAX_INFLIGHT=8`, replicas 1–2 |
| `cloudrun` | Cloud Run `dgemma` | 1× RTX PRO 6000 | pre-`ab208dd` | `DISABLE_MM=0`, `GPU_UTIL=0.40`, `KV_CACHE_GB=2`; Cloud Run may scale out instances |

Engine settings are not identical across targets (see table); single-request latency is dominated
by the GPU, while the concurrency results also reflect the limiter and Cloud Run's scale-out.

## Per-mode latency (50 sequential requests per cell, 3 warmup; ms, p50)

| Mode | L4 denoise | Cloud Run denoise | G4 denoise | L4 wall | Cloud Run wall | G4 wall |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1A samples=1 | 187.9 | 65.0 | **57.5** | 271.4 | 143.9 | **143.1** |
| 1B samples=4 | 322.5 | 107.7 | **97.9** | 405.2 | 186.8 | **181.4** |
| 1C samples=auto | 494.0 | 168.1 | **150.6** | 578.8 | 247.9 | **234.6** |
| 1D dual-mirror, samples=1 | 200.2 | 67.2 | **61.5** | 282.0 | 146.6 | 147.4 |
| 1E dual-mirror, samples=4 | 353.5 | 124.7 | **109.3** | 439.3 | 205.5 | **193.5** |
| 1F think=120, samples=2 | 1425.0 | 408.3 | **393.4** | 1506.7 | 493.0 | **475.6** |
| 1G image, samples=1 | no vision | 78.8 | **66.5** | — | 185.6 | **168.4** |

`modes_l4_cloudrun.json`, `modes_g4.json`, `modes_cloudrun_image.json`.

## Concurrency sweep (`sweep.json`; n = max(50, 4×workers) per cell)

| Target | samples | workers | ok/n | throughput/s | wall p50 | wall p95 |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| g4 | 1 | 8 / 16 / 32 | all ok | 35.3 / 60.4 / 62.6 | 199 / 239 / 480 | 210 / 336 / 595 |
| g4 | 4 | 8 / 16 / 32 | all ok | 23.6 / 34.8 / 35.7 | 302 / 438 / 867 | 419 / 531 / 964 |
| cloudrun | 1 | 8 / 16 / 32 | all ok | 34.7 / 63.1 / 77.0 | 210 / 218 / 372 | 221 / 394 / 527 |
| cloudrun | 4 | 16 / 32 | all ok | 38.2 / 48.0 | 383 / 604 | 521 / 836 |
| l4 | 1 | 8 | 50/50 | 16.0 | 460 | 495 |
| l4 | 4 | 4 | 50/50 | 5.0 | 592 | 2812 |
| l4 | 4 | 8 | **0/50** | — | — | — |

Notes:
- **L4 crashes under load.** At 4 samples × 8 workers (and earlier at 16 workers with mixed templates)
  every request failed: first `HTTP 502 upstream 500: EngineCore encountered an issue`, then
  connection resets, and the endpoint returned 503 "no healthy backend" for about 2 minutes while
  the container restarted. This is an engine crash, not Vertex admission control (no 429s). The
  G4 image adds an in-flight limiter (`MAX_INFLIGHT=8`), so excess requests queue: 0 errors at 32
  workers.
- G4 throughput plateaus around 60/s (1 sample) and 36/s (4 samples) on one replica because of the
  limiter; Cloud Run's higher top-end likely reflects instance scale-out.
- Two cells have single stalled requests (Cloud Run s4/w8: two at ~36 s, likely a new instance
  starting; L4 s1/w2: one at 69 s), which makes their throughput meaningless; percentiles are fine.

## G4 deployment timing
- `deployModel` call to operation done: ~9 min 50 s (22:06–22:16 UTC). The container was already
  `vllm_ready` and self-warmed (`warmup_s` 18.6) at the first health poll.
- No JIT cache archive exists in GCS, so there was nothing to unpack on Vertex.
