---
title: "Path to Production: Serving dgem at Scale"
description: "Crawl, walk, run: how to take dgem from a laptop to a production Vertex AI endpoint, with measured latency and concurrency scaling, hardware recommendations, and a deploy/verify/rollback checklist."
---

This track takes `dgem` from your laptop to a production endpoint and keeps it running. Every number on these
pages is measured (sample sizes and receipts are linked); identifiers such as `<PROJECT>`, `<REGION>` and
`<ENDPOINT_ID>` are placeholders.

## The journey

| Step | Page | What you get |
| :--- | :--- | :--- |
| 1 | [Run on your laptop](/dgem/deploy/laptop/) | A first decision in minutes on Apple Silicon or a local NVIDIA GPU |
| 2 | [Use a remote GPU](/dgem/deploy/remote-gpu/) | Point the CLI, Studio and MCP at any hosted endpoint |
| 3 | [Deploy on Cloud Run](/dgem/deploy/cloud-run/) | Your own scale-to-zero GPU service: $0 when idle, ~2.5 min cold start |
| 4 | [Production on Vertex AI](/dgem/deploy/vertex/) | An always-warm endpoint with autoscaling replicas and zero cold start |
| 5 | [Gateway and routing](/dgem/deploy/gateway/) | One URL for users, automatic failover from Vertex to Cloud Run, IAP |
| 6 | [Latency and capacity](/dgem/operate/latency-capacity/) | Request design, engine settings and how to measure your own |
| 7 | [Operations runbook](/dgem/operate/runbook/) | Health, promotion, rollback, teardown and troubleshooting |
| 8 | [Monitoring and alerts](/dgem/operate/monitoring/) | Scheduled health check, alert policies, what to do when they fire |

Deciding a whole dataset rather than one request at a time? [To batch or not to batch](/dgem/deploy/batch/) compares a warm
endpoint, Cloud Run Jobs, Dataflow and Vertex AI batch prediction, with measured cost per 1,000 decisions.

Hosting it yourself? [Evaluate dgem on your own GPU](/dgem/deploy/evaluate/) covers GPU requirements, checking the install
against known benchmark ranges, and a custom evaluation on your own labelled data.

## Crawl → walk → run

| Stage | Platform | Use it for | Cost model | Cold start | Warm latency, 1 sample (p50) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Crawl** | Apple Silicon (`diffgemma`, Metal, q4) or a local NVIDIA GPU | Development, writing policies, offline and private data | Your machine | None | ~0.9 s on Metal (different engine; numbers don't transfer to GPU serving) |
| **Walk** | Cloud Run GPU, 1× RTX PRO 6000 | Batch evaluation, research, bursty internal use, failover | Per instance-second; **$0 when idle** | 2.3–2.7 min (lean image + Direct VPC egress) | 61 ms GPU / 153 ms end to end |
| **Run** | Vertex AI dedicated endpoint, `g4-standard-48` + 1× RTX PRO 6000 | Production: always warm, IAM, autoscaling, monitoring, images | Per replica-hour while deployed | **None** (min replicas ≥ 1) | **55 ms GPU / 150 ms end to end** |

Latency: 3-question decision, 30 sequential requests, same-region client, serving image `504638d`
([receipts](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260927-image-parity/README.md)). The same image, policies, CLI, HTTP
API and MCP server work on every stage.

## TL;DR recommendations

- **GPU:** use a Blackwell GPU (RTX PRO 6000). The checkpoint is NVFP4 (4-bit); Blackwell runs FP4 natively, and
  the L4 was about 3× slower on every request type we measured. Other GPUs are untested
  ([requirements](/dgem/deploy/evaluate/#0-hardware)).
- **Production:** a Vertex AI dedicated endpoint with minimum replicas ≥ 1, autoscaling on GPU duty cycle.
- **Failover:** a scale-to-zero Cloud Run GPU service using the lean image and Direct VPC egress.
- **Bulk decisions:** Cloud Run Jobs with the batch server settings above ~10,000 rows, or your warm endpoint if it
  has spare capacity ([to batch or not to batch](/dgem/deploy/batch/)).
- **One gateway** (`dgem serve`) in front of both, routing `vertex_first` with automatic failover.
- **Requests:** `"samples": 1` by default; opt in to `4` only when you need agreement/standard error; never
  `"auto"` on latency-sensitive paths. Put all questions about one input in one request.
- **Change safely:** validate every new serving image next to production before switching traffic
  ([runbook](/dgem/operate/runbook/#promote-a-new-serving-image)).

## Method

- Tools: [`scripts/serving_speed.py`](https://github.com/ghchinoy/dgem/blob/main/scripts/serving_speed.py) (latency and concurrency),
  [`scripts/coldstart_probe.py`](https://github.com/ghchinoy/dgem/blob/main/scripts/coldstart_probe.py) (Cloud Run cold start),
  [`scripts/contract_diff.py`](https://github.com/ghchinoy/dgem/blob/main/scripts/contract_diff.py) (API contract). Failed requests are counted as
  errors, never as zero latency.
- Single client in the same region: these are service-side characteristics, not end-user internet latencies.
- Receipts: [`benchmarks/runs/20260927-image-parity`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260927-image-parity/README.md)
  (current image) and [`benchmarks/runs/20260925-serving-speed`](https://github.com/ghchinoy/dgem/blob/main/benchmarks/runs/20260925-serving-speed/README.md)
  (L4 comparison, earlier image).
