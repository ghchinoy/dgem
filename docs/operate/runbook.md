---
title: "Operations Runbook"
description: "Operating dgem in production: health signals, routine checks, promoting a new serving image with side-by-side validation, rollback, teardown, alerting and incident playbooks."
---

# Operations Runbook

For SREs running the Vertex endpoint, the Cloud Run GPU service and the gateway described in
[From Laptop to Production](../deploy/index.md). Placeholders: `<PROJECT>`, `<REGION>`, `<ENDPOINT_ID>`,
`<your-dgem-gateway>`.

## Health signals

`GET /health` on the serving container (`/invoke/health` on Vertex) returns HTTP 200 once the container is up, with
readiness details:

| Field | Meaning |
| :--- | :--- |
| `phase` | `mounting_gcs` → `staging_tmpfs` → `loading_vllm_siglip` → `ready` |
| `bytes_staged_gb` | Weight copy progress (17.53 when done) |
| `vllm_ready` | The engine answers; decisions will succeed |
| `warmed`, `warmup_s` | The self-warmup (1-sample, 4-sample and image decisions) finished |

The gateway exposes `GET /health`, `GET /api/status?backend=vertex|cloudrun|local` and `GET /api/backend-config`
(live Vertex state: `deployed`, replica count, machine type), and the MCP tool `get_health_and_gpu_status`.
Each backend's readiness is tracked separately, so a Vertex probe never marks Cloud Run as warm.

## Routine checks

- Vertex: `availableReplicaCount ≥ 1` on the deployed model (`gcloud ai endpoints describe <ENDPOINT_ID>`), and a
  periodic `/invoke/health` (the gateway sends one every 4 minutes; keep the gateway at min 1 instance).
- Cloud Run GPU: scales to zero by design; verify it can wake with an occasional decision.
- Gateway: one decision per backend with `X-DGem-Backend` and check `X-DGem-Backend-Used`.
- Costs: only intended Vertex deployed models and GCE VMs exist (`gcloud ai endpoints list`,
  `gcloud compute instances list`).

## Promote a new serving image

A serving image bundles the vLLM runtime and the structured server; small changes can alter answers or break the
API. Validate every new image **next to production, in the same session**, before moving traffic. The procedure we
used for the current image is in [the image-parity run](../../benchmarks/runs/20260927-image-parity/README.md); it
found four regressions that single-image testing would have missed.

1. **Build** with an immutable tag: `GCP_PROJECT=<PROJECT> IMAGE_TAG=$(git rev-parse --short HEAD) ./scripts/build_cloudrun_image.sh`.
2. **Canaries:** a second Vertex endpoint (`VERTEX_ENDPOINT_NAME=dgemma-canary-<TAG> VERTEX_MAX_REPLICAS=1`) and/or
   a Cloud Run service (`CLOUDRUN_SERVICE_NAME=dgemma-canary`) on the new image.
3. **Compare** (all against production and the canary):

   | Check | Tool | Pass |
   | :--- | :--- | :--- |
   | API contract | `scripts/contract_diff.py --target old=<URL> --target new=<URL>` | Same status codes, errors, response keys, read counts; answer changes only where repeats also flip |
   | Accuracy | 3× `dgem bench-jev` and `dgem bench-calibration` per image, then `scripts/receipt_agreement.py` | Inside run-to-run noise (JevBench 182–192 of 231) |
   | Latency | `scripts/serving_speed.py modes` | No regression beyond noise |
   | Load | `scripts/serving_speed.py sweep --workers 16 32 --min-requests 256` | 0 errors |
   | Readiness | `/health` shows `vllm_ready` and `warmed` | Present |
   | Cold start (Cloud Run) | `scripts/coldstart_probe.py` | Comparable to the current image |

4. **Switch** without downtime: Vertex with `VERTEX_NEW_TRAFFIC=0` then a traffic-split update
   ([steps](../deploy/vertex.md#5-swap-images-with-zero-downtime)); Cloud Run and the gateway with a `--no-traffic`
   tagged revision then `update-traffic`.
5. **Clean up:** delete canaries; after a soak period, undeploy the old Vertex model (keep its model resource for
   rollback).

Per-item answers can shift a few percent between vLLM versions even when accuracy is unchanged: don't compare
individual answers across images in experiments.

When changing `deploy/cloudrun/server/structured_server.py`, keep it as vLLM's upstream example plus
`dgem.patch`: edit the file, regenerate the patch with `diff -u` against `structured_server.upstream.py`, and check it
re-applies. The patch must keep the `/health` readiness fields, HTTP/1.1 keep-alive with backlog 256,
`MAX_INFLIGHT`, `DEFAULT_SAMPLES` and the `/predict` / `/rawPredict` aliases.

## Release a new version

Serving images, the gateway and the CLI share one version (`vMAJOR.MINOR.PATCH`, see
[CHANGELOG.md](../../CHANGELOG.md)). A running service reports it: `/health` returns `version` and `revision`,
images carry `org.opencontainers.image.version`, and `dgem --version` prints it.

1. Add a `## vX.Y.Z` section to `CHANGELOG.md` and commit it on `main`.
2. `make release VERSION=vX.Y.Z`: checks a clean `main`, runs the tests, tags git, builds `dgem` and
   `dgem-weights` with the version baked in, and adds the `vX.Y.Z` image tags. It prints the digests.
3. Validate the release image next to production ([Promote a new serving image](#promote-a-new-serving-image)).
4. `make publish-latest VERSION=vX.Y.Z` to move `:latest`, `git push origin vX.Y.Z`, and pin the new digests in
   [Public container images](../deploy/public-images.md).
5. Deploy to your own endpoints by digest; record the version in your deployment log.

Builds between releases report `git describe` versions such as `v0.1.0-3-gabc1234`.

## Roll back

| Component | How | Time |
| :--- | :--- | :--- |
| Vertex (old model still deployed) | Move the traffic split back | Seconds |
| Vertex (old model undeployed) | Redeploy the old model resource with `VERTEX_NEW_TRAFFIC=0`, then move traffic | ~15 min |
| Cloud Run GPU service / gateway | `gcloud run services update-traffic <SERVICE> --to-revisions=<PREVIOUS>=100` | Seconds (GPU service then cold-starts) |

## Tear down

- Vertex: `GCP_PROJECT=<PROJECT> make vertex-teardown` (or undeploy the model to keep the endpoint and its URL).
- Cloud Run: idle services cost nothing; `gcloud run services delete <SERVICE>` removes them.
- GCE benchmark VMs: `make gce-teardown` immediately after use.

## Alerting

`scripts/setup_cloud_monitoring.sh` creates log-based metrics and a dashboard for the gateway. Useful alerts:

- Vertex deployed model with 0 available replicas for > 5 minutes.
- Gateway `X-DGem-Backend-Used: cloudrun` share rising while the default is `vertex_first` (Vertex unhealthy).
- `503 server busy` rate > 1% (raise capacity or `MAX_INFLIGHT`).
- p95 decision latency above your SLO on `dgem.gateway.decide` traces ([Observability](observability.md)).

## Incident playbooks

| Symptom | Likely cause | Action |
| :--- | :--- | :--- |
| Vertex `429 ... scale-up from zero` / "not yet ready" | Replica scaled to 0 or still starting | Check `availableReplicaCount`; the gateway fails over to Cloud Run meanwhile; confirm min replicas ≥ 1 and keepalive running |
| Many `503 server busy` | Load above `MAX_INFLIGHT` for 30 s | Add replicas / raise max replicas; ask clients to back off |
| `503 Model server early terminated the request` | Serving image without HTTP keep-alive | Upgrade to an image with the keep-alive fix |
| All traffic on Cloud Run with multi-minute latencies | Vertex down and Cloud Run cold | Restore Vertex; Cloud Run warms in ~2.5 min |
| `/health` stuck in `staging_tmpfs` | Slow or blocked Cloud Storage access | Check Direct VPC egress and Private Google Access ([Cloud Run §4](../deploy/cloud-run.md#4-networking-direct-vpc-egress-the-biggest-cold-start-win)) |
| Replica restarts during boot | Out of host memory | Larger machine / RTX PRO 6000 instance, or `DISABLE_MM=1` |
| IAP 401/403 for a user | Not in the allowed group | Add to `<GROUP>` |
