---
title: "Monitoring and Alerts"
description: "How dgem is monitored: the alert policies and what to do when each fires, the scheduled health check, how to change thresholds, and how to deploy or remove the monitoring."
---

# Monitoring and Alerts

`dgem` monitoring has three parts:

| Part | What it does | Set up by |
| :--- | :--- | :--- |
| **Log-based metrics** | Turn gateway decision logs and health-check results into metrics (count, latency, backend) | `scripts/setup_alerts.py` (and `scripts/setup_cloud_monitoring.sh` for the dashboard) |
| **Scheduled health check** | Every hour, checks the Vertex endpoint and the gateway; once a day, the Cloud Run service | `scripts/deploy_probe.sh` |
| **Alert policies** | Email when something is broken or slow | `scripts/setup_alerts.py` |

Everything is created by idempotent scripts, so changing a threshold means editing one value and re-running a
script. Placeholders: `<PROJECT>`, `<REGION>`, `<ENDPOINT_ID>`, `<EMAIL>`.

## The scheduled health check

Two Cloud Run jobs triggered by Cloud Scheduler, running `scripts/probe.py` as a dedicated service account
(`dgem-probe-sa`, allowed only to call the Vertex endpoint and invoke the gateway and serving services).

| Job | Schedule (UTC) | Targets | What it checks |
| :--- | :--- | :--- | :--- |
| `dgem-probe-hourly` | `7 * * * *` | Vertex endpoint; gateway | Vertex: `/health`, the 14-case API contract (valid and error cases must return the expected status and answers), 20 latency requests. Gateway: `/health`, 3 decisions through the default route, which must be answered by Vertex |
| `dgem-probe-daily` | `17 6 * * *` | Cloud Run serving service | Same as Vertex; the first request wakes the scale-to-zero service (~2.5 min), then it idles back to zero |

Each target writes one JSON log line: `probe_event="dgem.probe"`, `probe_target`, `probe_ok`, `probe_failures`,
`probe_latency_p50_ms`, `probe_latency_p95_ms`, `probe_denoise_p50_ms`, `probe_version`. Find them in Logs Explorer
with `jsonPayload.probe_event="dgem.probe"`.

Cost: the hourly job uses the always-warm endpoint (a few seconds of CPU per run); the daily job keeps one Cloud Run
GPU up for about 20 minutes.

## Alerts: what they mean and what to do

All policies are named `dgem: …`, notify the email channel, and link back to this page.

| Alert | Fires when (default) | Likely causes | What to do |
| :--- | :--- | :--- | :--- |
| **Vertex endpoint has no available replica** (critical) | Available replicas sum < 1, or the replica metric is missing, for 5 min | Replica crashed or is restarting; endpoint scaled to 0 while idle; model undeployed | Check `availableReplicaCount` (`gcloud ai endpoints describe <ENDPOINT_ID> --region=<REGION>`). The gateway is serving from Cloud Run meanwhile. If it doesn't recover in ~15 min, redeploy or roll back ([runbook](runbook.md#roll-back)) |
| **Vertex 5xx/429 error rate** | 5xx + 429 responses > 1% of Vertex responses over 10 min | Overload (`503 server busy`: more than `MAX_INFLIGHT` queued), replica restart, bad image | Check load and replica count; raise max replicas or `MAX_INFLIGHT`; if it started with a new image, roll back |
| **Gateway failing over from Vertex to Cloud Run** | > 20% of `vertex_first` decisions answered by Cloud Run over 30 min (at least 5 decisions) | Vertex unhealthy or updating | Usually accompanies one of the Vertex alerts; users see slower answers (and a cold start if Cloud Run was idle). Fix Vertex |
| **Gateway decision latency p95 (Vertex)** (warning) | p95 of Vertex-answered decision wall time > 2,000 ms over 15 min | Queueing under load, very long prompts or many-question policies, reasoning (`think`) templates | Compare with normal (150–500 ms for typical policies). If it's load, add replicas; if it's one template, it may just be heavy |
| **Scheduled health check failed** | Any health-check run reported `probe_ok=false` in the last ~70 min | API contract changed (new image), endpoint down, auth or IAM change for the probe account, gateway fell back to Cloud Run | Read `jsonPayload.probe_failures` on the failing log line; it names the failing case or request |
| **Scheduled health check not running** (warning) | No Vertex health-check result for 2 h | Scheduler paused or deleted, job failing to start, image pull or IAM problem | `gcloud scheduler jobs describe dgem-probe-hourly --location=<REGION>`; run the job by hand (below) and read its logs |
| **Health-check latency to Vertex** (warning) | Health-check median end-to-end latency to Vertex > 400 ms | Slower serving image, GPU contention, network | Normal is ~80 ms from inside the region. Check `probe_denoise_p50_ms` (GPU time, normally ~55–60 ms): if GPU time rose, it's the model server; if only end-to-end rose, it's network or queueing |

Alerts close automatically when the condition clears (after 30 min for Vertex alerts, 2 h for health-check alerts).

## Changing thresholds

Every threshold is an environment variable read by `scripts/setup_alerts.py`:

| Variable | Default | Alert |
| :--- | :--- | :--- |
| `VERTEX_NO_REPLICA_MIN` | `5` | Minutes with no replica before alerting |
| `VERTEX_ERROR_RATE` | `0.01` | Vertex 5xx+429 share |
| `FAILOVER_SHARE` / `FAILOVER_MIN_DECISIONS` | `0.2` / `5` | Gateway failover share and minimum volume |
| `DECISION_P95_MS` | `2000` | Gateway decision p95 (ms) |
| `PROBE_MISSING_HOURS` | `2` | Hours without a health-check result |
| `PROBE_P50_MS` | `400` | Health-check median latency (ms) |

```bash
# Preview, then apply (updates policies in place)
GCP_PROJECT=<PROJECT> ALERT_EMAIL=<EMAIL> VERTEX_ENDPOINT_ID=<ENDPOINT_ID> DECISION_P95_MS=3000 \
  python3 scripts/setup_alerts.py --dry-run
GCP_PROJECT=<PROJECT> ALERT_EMAIL=<EMAIL> VERTEX_ENDPOINT_ID=<ENDPOINT_ID> DECISION_P95_MS=3000 \
  python3 scripts/setup_alerts.py --only gateway-decision-p95
```

Health-check schedules and targets are set in `scripts/deploy_probe.sh` (`HOURLY_SCHEDULE`, `DAILY_SCHEDULE`,
`PROBE_LATENCY_REQUESTS`, `PROBE_DAILY=off`); re-run it to apply. To pause an alert without deleting it, disable the
policy in the Cloud Monitoring console; to pause the health check, `gcloud scheduler jobs pause dgem-probe-hourly
--location=<REGION>`.

## Deploying the monitoring

```bash
# 1. Health check (service account, image, jobs, schedules)
GCP_PROJECT=<PROJECT> \
PROBE_VERTEX_URL=https://<ENDPOINT_ID>.<REGION>-<PROJECT_NUMBER>.prediction.vertexai.goog/v1/projects/<PROJECT>/locations/<REGION>/endpoints/<ENDPOINT_ID>/invoke \
PROBE_GATEWAY_URL=https://<gateway-service>.run.app \
PROBE_CLOUDRUN_URL=https://<CLOUD_RUN_URL> \
PROBE_GATEWAY_AUDIENCE=<IAP programmatic client ID, if the gateway uses IAP> \
./scripts/deploy_probe.sh
gcloud run jobs execute dgem-probe-hourly --region=<REGION> --wait     # run once now

# 2. Metrics, email channel and alert policies
GCP_PROJECT=<PROJECT> ALERT_EMAIL=<EMAIL> VERTEX_ENDPOINT_ID=<ENDPOINT_ID> python3 scripts/setup_alerts.py
```

- **Gateway with IAP:** IAP accepts a service account's ID token only if its audience is one of the gateway's
  programmatic OAuth clients (`gcloud iap settings get --resource-type=cloud-run --service=<gateway> --region=<REGION>`,
  `programmaticClients`), and the account needs the IAP-secured Web App User role; the script grants it when
  `PROBE_GATEWAY_AUDIENCE` is set. Use the gateway's `run.app` URL, not a custom domain.
- **After changing the Vertex endpoint** (new endpoint ID), re-run both scripts with the new values.
- **Failover and latency alerts need gateway logs with `dgem_backend_requested`** (gateway v0.1.2 or later).
- **Removing everything:** delete the `dgem:` alert policies, the scheduler jobs and Cloud Run jobs
  (`dgem-probe-hourly`, `dgem-probe-daily`), and the `dgem-probe-sa` account.

## Manual checks

The health check runs a subset of the manual tools used when promoting a serving image
([runbook](runbook.md#promote-a-new-serving-image)): `scripts/contract_diff.py`, `scripts/serving_speed.py`,
`scripts/coldstart_probe.py`, and `scripts/template_sweep.py` (every policy template through a gateway). Run those
by hand before and after changes; the scheduled check catches regressions in between.
