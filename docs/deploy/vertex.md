---
title: "Step 4: Production on Vertex AI"
description: "Run DiffusionGemma on an always-warm Vertex AI dedicated endpoint (g4-standard-48 + RTX PRO 6000): deploy, verify, scale, swap images with zero downtime, and tear down."
---

# Step 4: Production on Vertex AI

A Vertex AI **dedicated endpoint** is the **run** stage: always warm, IAM-protected, autoscaling replicas and
Cloud Monitoring, on the same container image as Cloud Run. The container is deployed with arbitrary custom routes
(`invokeRoutePrefix: "/*"`), so every server route is available under `/invoke/*`
([why](../reference/vertex-vs-cloud-run.md)).

## What you get

Measured on one `g4-standard-48` + 1× RTX PRO 6000 replica, serving image `504638d`
([receipts](../../benchmarks/runs/20260927-image-parity/README.md)):

| Request (3 questions unless noted) | GPU (p50) | End to end (p50) |
| :--- | ---: | ---: |
| 1 sample | 55 ms | 150 ms |
| 4 samples (one batch) | 91 ms | 190 ms |
| Image + 2 questions | 64 ms | 183 ms |
| 120-token reasoning trace + 2 questions | 364 ms | 463 ms |
| Throughput per replica, 32 concurrent clients | 72 decisions/s (1 sample), 45/s (4 samples), 0 errors | |

Cold start: none while minimum replicas ≥ 1. Deploying a new model takes ~10–15 minutes.

## 1. Prerequisites

- APIs: `aiplatform.googleapis.com`, `artifactregistry.googleapis.com`, `storage.googleapis.com`.
- **Vertex AI serving quota** for RTX PRO 6000 in your region:

  ```bash
  gcloud beta quotas info describe CustomModelServingRTXPRO6000GPUsPerProjectPerRegion \
    --service=aiplatform.googleapis.com --project=<PROJECT> --format=json \
    | jq -r '.dimensionsInfos[] | select(.dimensions.region == "<REGION>") | .details.value'
  ```

- Weights in `gs://dgem-weights-<PROJECT>/dgemma/` (see [Cloud Run §2](cloud-run.md#2-stage-the-weights-in-cloud-storage));
  the endpoint mounts them as the model artifact.
- A service account for the endpoint (default `dgemma-gpu-sa@<PROJECT>.iam.gserviceaccount.com`) with read access
  to the bucket.
- Vertex AI requires an Artifact Registry or Container Registry image. Pin by digest; the public image works:
  `us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem@sha256:5fa4a866163169aaf91c3bdb727020ff3c86e26e84ffadad86bc859053d45b26` ([current digests](public-images.md)).

## 2. Deploy

```bash
GCP_PROJECT=<PROJECT> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> GCP_REGION=<REGION> \
IMAGE_URI=us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/dgem@sha256:5fa4a866163169aaf91c3bdb727020ff3c86e26e84ffadad86bc859053d45b26 \
./scripts/deploy_vertex_endpoint.sh
```

The default profile (`VERTEX_PROFILE=g4-rtxpro6000`) configures:

| Setting | Value |
| :--- | :--- |
| Machine | `g4-standard-48` + 1× `NVIDIA_RTX_PRO_6000` |
| Replicas | min 1 (`VERTEX_MIN_REPLICAS`), max 2 (`VERTEX_MAX_REPLICAS`), autoscale at 70% GPU duty cycle |
| Engine | `KV_CACHE_GB=12`, `MAX_SEQS=32`, `MAX_INFLIGHT=8`, `DEFAULT_SAMPLES=1`, vision on |
| Endpoint | dedicated DNS, display name `dgemma-dedicated-g4` (`VERTEX_ENDPOINT_NAME`; reused if it exists) |
| Startup | long startup probe (weights load + self-warmup before the replica takes traffic) |

The script prints the endpoint ID and the invoke URLs:

```text
https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke/...
  /v1/chat/completions      structured decisions (what dgem uses)
  /v1/systemone             Jev-style question sets
  /health                   readiness
```

## 3. Verify

```bash
INVOKE=https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke
curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" $INVOKE/health
#   -> {"status": "ok", "vllm_ready": true, "phase": "ready", "warmed": true, ...}

export GCP_PROJECT=<PROJECT> GCP_PROJECT_NUMBER=<PROJECT_NUMBER>
./bin/dgem decide --vertex-url <ENDPOINT_ID> --gcp-auth -t templates/support_triage.json.tmpl \
  -v ticket="Charged twice" --stats
./bin/dgem decide --vertex-url <ENDPOINT_ID> --gcp-auth -t templates/multimodal/bbox_localization.json.tmpl \
  -I fixtures/bbox/bbox-t1-03-offgrid-card.png -v target=checkout_summary_card
```

## 4. Size it

- One replica sustains ~70 one-sample or ~45 four-sample decisions/s; beyond ~16 concurrent clients requests queue
  (latency rises, throughput plateaus). Set `VERTEX_MAX_REPLICAS` for your peak; the endpoint adds replicas on GPU
  duty cycle. Details: [Latency and capacity](../operate/latency-capacity.md).
- Keep minimum replicas ≥ 1 and a periodic `/invoke/health` probe (the [gateway](gateway.md) runs one every 4
  minutes). Idle dedicated endpoints have been observed at 0 available replicas, returning `429` until they scale
  back up.

## 5. Swap images with zero downtime

Deploy the new image next to the current one at 0% traffic, verify it on the same endpoint, then move traffic:

```bash
# 1. New model on the same endpoint, 0% traffic (the current split is kept)
VERTEX_NEW_TRAFFIC=0 GCP_PROJECT=<PROJECT> GCP_PROJECT_NUMBER=<PROJECT_NUMBER> \
IMAGE_URI=<NEW_IMAGE> VERTEX_MODEL_NAME=dgemma-g4-<TAG> ./scripts/deploy_vertex_endpoint.sh

# 2. Move traffic (deployed model IDs from `gcloud ai endpoints describe <ENDPOINT_ID> --region=<REGION>`)
curl -X PATCH -H "Authorization: Bearer $(gcloud auth print-access-token)" -H "Content-Type: application/json" \
  "https://<REGION>-aiplatform.googleapis.com/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>?updateMask=trafficSplit" \
  -d '{"trafficSplit": {"<NEW_DEPLOYED_MODEL_ID>": 100, "<OLD_DEPLOYED_MODEL_ID>": 0}}'

# 3. After a soak period, undeploy the old model (its model resource stays uploaded for rollback)
gcloud ai endpoints undeploy-model <ENDPOINT_ID> --region=<REGION> --deployed-model-id=<OLD_DEPLOYED_MODEL_ID>
```

Rollback while both are deployed is instant (move the split back); after undeploying, redeploy the old model
resource (~15 minutes). Before step 2 on a new serving image, run the comparison in
[Promote a new serving image](../operate/runbook.md#promote-a-new-serving-image).

## 6. Tear down

```bash
GCP_PROJECT=<PROJECT> make vertex-teardown      # undeploys models and deletes the dgemma-dedicated-g4 endpoint
```

A deployed replica bills per hour whether or not it serves traffic; tear down endpoints you don't need, and keep a
scale-to-zero [Cloud Run service](cloud-run.md) for occasional use.

## Next

[Gateway and routing](gateway.md): one URL in front of Vertex and Cloud Run, with automatic failover.
