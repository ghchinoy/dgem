#!/usr/bin/env python3
"""Create or update dgem log-based metrics, an email notification channel, and alert policies.

Idempotent: metrics and policies are matched by name / display name and updated in place, so re-run it after
changing a threshold. Every threshold has a default and an environment-variable / flag override (below).
Guide: docs/operate/monitoring.md.

  ALERT_EMAIL=you@example.com GCP_PROJECT=<project> VERTEX_ENDPOINT_ID=<id> python3 scripts/setup_alerts.py
  python3 scripts/setup_alerts.py --dry-run          # print what would be applied

Policies (display names are prefixed "dgem: "):
  vertex-no-replicas       Vertex endpoint has no available replica for VERTEX_NO_REPLICA_MIN minutes (default 5)
  vertex-error-rate        Vertex 5xx+429 share > VERTEX_ERROR_RATE (default 0.01) over 10 min
  gateway-failover         Share of vertex_first decisions answered by Cloud Run > FAILOVER_SHARE (0.2) over 30 min
                           (with at least FAILOVER_MIN_DECISIONS=5 vertex_first decisions)
  gateway-decision-p95     p95 of Vertex-answered decision wall time > DECISION_P95_MS (2000) over 15 min
  probe-failed             A scheduled health check reported a failure (any target)
  probe-missing            No health check result for PROBE_MISSING_HOURS (2) hours (target=vertex)
  probe-latency            Health-check end-to-end median to Vertex > PROBE_P50_MS (400) ms

Auth: gcloud access token (CLOUDSDK_AUTH_ACCESS_TOKEN_FILE or `gcloud auth print-access-token`).
"""
import argparse
import json
import os
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

PREFIX = "dgem: "


def env(name, default):
    return os.environ.get(name, default)


def token():
    f = os.environ.get("CLOUDSDK_AUTH_ACCESS_TOKEN_FILE")
    if f and os.path.exists(f):
        return open(f).read().strip()
    return subprocess.check_output(["gcloud", "auth", "print-access-token"], text=True).strip()


def api(method, url, body=None):
    req = urllib.request.Request(url, json.dumps(body).encode() if body is not None else None, method=method,
                                 headers={"Authorization": f"Bearer {token()}", "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raise SystemExit(f"{method} {url}: HTTP {e.code} {e.read()[:600].decode(errors='replace')}")


# ---------------------------------------------------------------- log-based metrics
def log_metrics():
    decide = 'resource.type="cloud_run_revision" AND jsonPayload.span_name="dgem.gateway.decide"'
    probe = 'jsonPayload.probe_event="dgem.probe"'
    return {
        # Replaces the older definition from setup_cloud_monitoring.sh, adding backend labels (labels can only be added).
        "dgem_decisions_total": {
            "description": "dgem gateway decisions (Studio, REST), by surface, template, backend used and requested",
            "filter": decide,
            "metricDescriptor": {"metricKind": "DELTA", "valueType": "INT64", "unit": "1", "labels": [
                {"key": k, "valueType": "STRING"} for k in
                ["surface", "template", "user", "multimodal", "reads", "backend", "backend_requested"]]},
            "labelExtractors": {
                "surface": "EXTRACT(jsonPayload.dgem_surface)", "template": "EXTRACT(jsonPayload.dgem_template)",
                "user": "EXTRACT(jsonPayload.dgem_user)", "multimodal": "EXTRACT(jsonPayload.dgem_multimodal)",
                "reads": "EXTRACT(jsonPayload.dgem_gpu_reads)", "backend": "EXTRACT(jsonPayload.dgem_backend)",
                "backend_requested": "EXTRACT(jsonPayload.dgem_backend_requested)"},
        },
        "dgem_decision_wall_ms": {
            "description": "dgem gateway decision wall time (ms), by backend used",
            "filter": decide + " AND jsonPayload.dgem_total_wall_ms > 0",
            "valueExtractor": "EXTRACT(jsonPayload.dgem_total_wall_ms)",
            "bucketOptions": {"exponentialBuckets": {"numFiniteBuckets": 40, "growthFactor": 1.25, "scale": 20}},
            "metricDescriptor": {"metricKind": "DELTA", "valueType": "DISTRIBUTION", "unit": "ms",
                                 "labels": [{"key": "backend", "valueType": "STRING"}]},
            "labelExtractors": {"backend": "EXTRACT(jsonPayload.dgem_backend)"},
        },
        "dgem_probe_runs": {
            "description": "dgem scheduled health-check results, by target and outcome",
            "filter": probe,
            "metricDescriptor": {"metricKind": "DELTA", "valueType": "INT64", "unit": "1", "labels": [
                {"key": "target", "valueType": "STRING"}, {"key": "ok", "valueType": "STRING"}]},
            "labelExtractors": {"target": "EXTRACT(jsonPayload.probe_target)", "ok": "EXTRACT(jsonPayload.probe_ok)"},
        },
        "dgem_probe_latency_ms": {
            "description": "dgem health-check end-to-end median latency per run (ms), by target",
            "filter": probe + " AND jsonPayload.probe_latency_p50_ms > 0",
            "valueExtractor": "EXTRACT(jsonPayload.probe_latency_p50_ms)",
            "bucketOptions": {"exponentialBuckets": {"numFiniteBuckets": 40, "growthFactor": 1.2, "scale": 20}},
            "metricDescriptor": {"metricKind": "DELTA", "valueType": "DISTRIBUTION", "unit": "ms",
                                 "labels": [{"key": "target", "valueType": "STRING"}]},
            "labelExtractors": {"target": "EXTRACT(jsonPayload.probe_target)"},
        },
    }


def upsert_metric(project, name, spec, dry):
    base = f"https://logging.googleapis.com/v2/projects/{project}/metrics"
    body = dict(spec, name=name)
    if dry:
        print(f"  metric {name}")
        return
    try:
        api("GET", f"{base}/{name}")
        api("PUT", f"{base}/{name}", body)
        print(f"  updated metric {name}")
    except SystemExit as e:
        if "404" not in str(e):
            raise
        api("POST", base, body)
        print(f"  created metric {name}")


# ---------------------------------------------------------------- notification channel
def email_channel(project, email, dry):
    base = f"https://monitoring.googleapis.com/v3/projects/{project}/notificationChannels"
    name = f"{PREFIX}{email}"
    if dry:
        print(f"  channel {name}")
        return "projects/-/notificationChannels/DRYRUN"
    for ch in api("GET", base).get("notificationChannels", []):
        if ch.get("type") == "email" and ch.get("labels", {}).get("email_address") == email:
            print(f"  using channel {ch['name']} ({email})")
            return ch["name"]
    ch = api("POST", base, {"type": "email", "displayName": name, "labels": {"email_address": email}})
    print(f"  created channel {ch['name']} ({email})")
    return ch["name"]


# ---------------------------------------------------------------- policies
def promql(query, duration_s, eval_s=60):
    return {"conditionPrometheusQueryLanguage": {"query": query, "duration": f"{duration_s}s",
                                                 "evaluationInterval": f"{eval_s}s"}}


def policies(project, ep, cfg):
    lm = "logging_googleapis_com:user_"
    vx = f'monitored_resource="aiplatform.googleapis.com/Endpoint",endpoint_id="{ep}"'
    # Log-based metrics can come from several resource types (gateway = Cloud Run service, probe = Cloud Run job);
    # alerting PromQL requires naming one.
    rev = 'monitored_resource="cloud_run_revision"'
    job = 'monitored_resource="cloud_run_job"'
    doc = ("See docs/operate/monitoring.md (alert '{key}') for what this means and what to do. "
           "Thresholds: scripts/setup_alerts.py.")

    def pol(key, title, cond, duration_note, severity="ERROR", auto_close="1800s"):
        return {"displayName": f"{PREFIX}{title}", "combiner": "OR", "severity": severity,
                "documentation": {"content": doc.format(key=key) + f" {duration_note}", "mimeType": "text/markdown"},
                "conditions": [dict(cond, displayName=title)],
                "alertStrategy": {"autoClose": auto_close}, "userLabels": {"app": "dgem", "alert": key}}

    return [
        pol("vertex-no-replicas", "Vertex endpoint has no available replica",
            promql(f'sum(avg_over_time(aiplatform_googleapis_com:prediction_online_replicas{{{vx}}}[5m])) < 1 '
                   f'or absent(aiplatform_googleapis_com:prediction_online_replicas{{{vx}}})',
                   int(cfg["VERTEX_NO_REPLICA_MIN"]) * 60),
            "Traffic fails over to Cloud Run (slower; cold start if idle).", "CRITICAL"),
        pol("vertex-error-rate", "Vertex 5xx/429 error rate",
            promql(f'(sum(rate(aiplatform_googleapis_com:prediction_online_response_count{{{vx},response_code=~"5..|429"}}[10m])) '
                   f'/ sum(rate(aiplatform_googleapis_com:prediction_online_response_count{{{vx}}}[10m]))) > {cfg["VERTEX_ERROR_RATE"]}',
                   300),
            f"Threshold {float(cfg['VERTEX_ERROR_RATE']):.0%} over 10 min."),
        pol("gateway-failover", "Gateway failing over from Vertex to Cloud Run",
            promql(f'(sum(increase({lm}dgem_decisions_total{{{rev},backend_requested="vertex_first",backend="cloudrun"}}[30m])) '
                   f'/ sum(increase({lm}dgem_decisions_total{{{rev},backend_requested="vertex_first"}}[30m])) > {cfg["FAILOVER_SHARE"]}) '
                   f'and sum(increase({lm}dgem_decisions_total{{{rev},backend_requested="vertex_first"}}[30m])) >= {cfg["FAILOVER_MIN_DECISIONS"]}',
                   300),
            f"More than {float(cfg['FAILOVER_SHARE']):.0%} of vertex_first decisions answered by Cloud Run in 30 min."),
        pol("gateway-decision-p95", "Gateway decision latency p95 (Vertex)",
            promql(f'histogram_quantile(0.95, sum by (le) (rate({lm}dgem_decision_wall_ms_bucket{{{rev},backend="vertex"}}[15m]))) > {cfg["DECISION_P95_MS"]}',
                   300, 120),
            f"Threshold {cfg['DECISION_P95_MS']} ms over 15 min.", "WARNING"),
        pol("probe-failed", "Scheduled health check failed",
            promql(f'sum by (target) (increase({lm}dgem_probe_runs{{{job},ok="false"}}[70m])) > 0', 0, 300),
            "The log line (jsonPayload.probe_failures) lists what failed.", "ERROR", "7200s"),
        pol("probe-missing", "Scheduled health check not running",
            promql(f'absent_over_time({lm}dgem_probe_runs{{{job},target="vertex"}}[{int(cfg["PROBE_MISSING_HOURS"])}h])', 0, 600),
            f"No health-check result for {cfg['PROBE_MISSING_HOURS']} h: check the Cloud Scheduler trigger and job.", "WARNING", "7200s"),
        pol("probe-latency", "Health-check latency to Vertex",
            promql(f'histogram_quantile(0.5, sum by (le) (rate({lm}dgem_probe_latency_ms_bucket{{{job},target="vertex"}}[3h]))) > {cfg["PROBE_P50_MS"]}',
                   0, 600),
            f"Median end-to-end latency above {cfg['PROBE_P50_MS']} ms (normal ~150-200 ms).", "WARNING", "7200s"),
    ]


def upsert_policy(project, p, channel, dry, only):
    key = p["userLabels"]["alert"]
    if only and key not in only:
        return
    p["notificationChannels"] = [channel]
    base = f"https://monitoring.googleapis.com/v3/projects/{project}/alertPolicies"
    if dry:
        print(f"  policy {p['displayName']}\n    {json.dumps(p['conditions'][0])[:400]}")
        return
    flt = urllib.parse.quote(f'display_name="{p["displayName"]}"')
    existing = api("GET", f"{base}?filter={flt}").get("alertPolicies", [])
    if existing:
        name = existing[0]["name"]
        api("PATCH", f"https://monitoring.googleapis.com/v3/{name}", dict(p, name=name))
        print(f"  updated policy {p['displayName']} ({name.split('/')[-1]})")
    else:
        r = api("POST", base, p)
        print(f"  created policy {p['displayName']} ({r['name'].split('/')[-1]})")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--only", nargs="*", help="only these policy keys (e.g. probe-latency)")
    ap.add_argument("--skip-metrics", action="store_true")
    a = ap.parse_args()
    project = env("GCP_PROJECT", "")
    email = env("ALERT_EMAIL", "")
    ep = env("VERTEX_ENDPOINT_ID", "")
    if not (project and email and ep):
        sys.exit("set GCP_PROJECT, ALERT_EMAIL and VERTEX_ENDPOINT_ID")
    cfg = {k: env(k, d) for k, d in {
        "VERTEX_NO_REPLICA_MIN": "5", "VERTEX_ERROR_RATE": "0.01", "FAILOVER_SHARE": "0.2",
        "FAILOVER_MIN_DECISIONS": "5", "DECISION_P95_MS": "2000", "PROBE_MISSING_HOURS": "2", "PROBE_P50_MS": "400"}.items()}
    print(f"Project {project}, endpoint {ep}, alerts to {email}")
    print("Thresholds: " + ", ".join(f"{k}={v}" for k, v in cfg.items()))
    if not a.skip_metrics:
        print("Log-based metrics:")
        for name, spec in log_metrics().items():
            upsert_metric(project, name, spec, a.dry_run)
    print("Notification channel:")
    ch = email_channel(project, email, a.dry_run)
    print("Alert policies:")
    for p in policies(project, ep, cfg):
        upsert_policy(project, p, ch, a.dry_run, set(a.only or []))


if __name__ == "__main__":
    main()
