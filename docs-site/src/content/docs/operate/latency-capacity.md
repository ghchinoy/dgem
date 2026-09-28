---
title: "Latency and Capacity"
description: "What drives dgem latency and throughput: request design (samples, questions per request, thinking), engine settings (MAX_INFLIGHT, MAX_SEQS, KV cache), measured tables for Vertex G4 and Cloud Run, and how to measure your own deployment."
---

All numbers: serving image `504638d`, 1× RTX PRO 6000, single client in the same region, 30 sequential requests
per latency cell and 256 requests per concurrency cell
([receipts](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260927-image-parity/README.md)). GPU time is the server-reported denoise time;
end to end includes network and routing.

## Latency by request type

| Request (3 questions unless noted) | Vertex G4 GPU / e2e | Cloud Run GPU / e2e |
| :--- | :--- | :--- |
| `samples: 1` | **55 / 150 ms** | 61 / 153 ms |
| `samples: 4` (one parallel batch) | 91 / 190 ms | 101 / 182 ms |
| 3 questions + reversed-order check (6 slots), `samples: 1` | 58 / 152 ms | 66 / 152 ms |
| Image + 2 questions, `samples: 1` | 64 / 183 ms | 76 / 184 ms |
| 120-token reasoning trace (`think: 120`) + 2 questions | 364 / 463 ms | 376 / 462 ms |

Earlier measurements (image `ab208dd`) put `samples: "auto"` at ~150 ms GPU on G4, about 50 ms more than a fixed 4,
because it usually runs a second sequential batch; the L4 was ~3× slower on every row
([2026-09-25 run](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-serving-speed/README.md)).

## Request design: the biggest levers

- **`samples: 1` by default.** One read is the fastest path. Use `samples: 4` (one parallel batch, ~+35–40 ms)
  where you need agreement and standard error, for example to route to human review. Avoid `"auto"` on
  latency-sensitive paths. Schemas without `samples` use the server's `DEFAULT_SAMPLES` (1).
- **Ask everything about one input in one request.** All questions share one forward pass; three questions cost
  about the same as one.
- **Avoid `depends_on` / `ask_if` when latency matters.** Conditional questions run in a second stage (two
  sequential passes). Gate in your client instead if you need a single pass.
- **At most 26 options per `choice` question.** Wider taxonomies need a two-stage split (two passes).
- **Thinking costs ~300 ms per 120 tokens.** Use `think` only where it measurably helps (e.g. taxonomy discovery).

## Throughput and concurrency (one Vertex G4 replica)

| Concurrent clients | `samples: 1` decisions/s · p50 · p95 | `samples: 4` decisions/s · p50 · p95 | Errors |
| ---: | :--- | :--- | ---: |
| 1 | 5.4/s · 140 · 359 ms | 5.7/s · 176 · 182 ms | 0 |
| 4 | 20.9/s · 176 · 256 ms | 13.9/s · 208 · 1,106 ms | 0 |
| 8 | 38.2/s · 189 · 196 ms | 28.9/s · 256 · 266 ms | 0 |
| 16 | 54.3/s · 254 · 386 ms | 39.0/s · 349 · 1,164 ms | 0 / 256 |
| 32 | 72.3/s · 426 · 645 ms | 45.2/s · 685 · 717 ms | 0 / 256 |

Rows 1–8: 50 requests each on the same runtime before the final HTTP fixes; rows 16–32: final image, 256 requests
each. Beyond ~16 clients requests queue: latency rises while throughput grows slowly. **Add replicas for more
throughput** (Vertex autoscales on GPU duty cycle); Cloud Run runs one instance per service.

## Engine and server settings

| Setting | Default | Effect |
| :--- | :--- | :--- |
| `MAX_INFLIGHT` | 8 | Decisions processed at once per server; the rest wait up to `INFLIGHT_WAIT_S` (30 s), then `503`. Protects the engine: an L4 replica without a cap crashed under 8 concurrent 4-sample requests. |
| `DEFAULT_SAMPLES` | 1 | Samples for schemas that don't set `samples` |
| `MAX_SEQS` | 32 | vLLM sequences in one batch (4-sample requests use 4) |
| `KV_CACHE_GB` | 12 (Vertex G4), 2 (Cloud Run script) | KV cache memory; more allows more concurrent sequences |
| `DISABLE_MM` | 0 | `1` turns off the vision tower (required on L4 Cloud Run, saves memory) |
| `ENFORCE_EAGER` | 1 | Eager mode: shorter start-up, no graph compilation |

Keep-alive matters under load: the server speaks HTTP/1.1 with a 256-connection backlog. With HTTP/1.0 Vertex's
proxy returned `503 ... truncated headers` for 1–8% of requests at 16–32 concurrent clients.

## Measure your own deployment

```bash
# Per-mode latency (repeat --target to compare deployments side by side)
python3 scripts/serving_speed.py modes --target mine=<BASE_URL>/v1/chat/completions -n 30 -o modes.json

# Concurrency sweep; any error is a finding
python3 scripts/serving_speed.py sweep --target mine=<BASE_URL>/v1/chat/completions \
  --workers 1 4 8 16 32 --samples 1 4 --min-requests 256 -o sweep.json

# Cloud Run cold start for one configuration
python3 scripts/coldstart_probe.py --service <SERVICE> --project <PROJECT> --region <REGION> --label mine --vpc
```

`<BASE_URL>` is a Cloud Run URL or a Vertex invoke base (`https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke`).
Tokens come from Application Default Credentials. Measure from the region your clients run in, and don't run
benchmarks against the same endpoint at the same time (they contaminate each other's latency).
