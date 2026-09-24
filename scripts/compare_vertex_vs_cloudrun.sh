#!/usr/bin/env bash
set -euo pipefail

VERTEX_ID="${VERTEX_ID:-4217256562927861760}"
CLOUDRUN_URL="${CLOUDRUN_URL:-https://dgemma-tkb3aiuiea-uc.a.run.app/v1}"

echo "========================================================================"
echo "  Head-to-Head Benchmark Matrix: Vertex AI Dedicated Endpoint vs Cloud Run"
echo "  Vertex Endpoint: ${VERTEX_ID} (g2-standard-16 · 1x NVIDIA L4 · 64GB RAM)"
echo "  Cloud Run URL:   ${CLOUDRUN_URL}"
echo "========================================================================"

# 1. Concurrency Scaling Sweep on Vertex AI Dedicated Endpoint (w=1, 4, 8, 16)
for W in 1 4 8 16; do
  echo ""
  echo ">>> [Phase 1] Running 30-case dgem bench on Vertex AI L4 (workers=${W})..."
  ./bin/dgem bench \
    --vertex-url "${VERTEX_ID}" \
    --gcp-auth \
    -w "${W}" \
    -d benchmarks/eval_dataset.jsonl \
    -o "/tmp/bench_vertex_w${W}.json"
done

# 2. 50-Case Calibration & Guardrail Suite (EXP-04) on Vertex AI L4
echo ""
echo ">>> [Phase 2] Running 50-case dgem bench-calibration on Vertex AI L4 (workers=8)..."
./bin/dgem bench-calibration \
  --vertex-url "${VERTEX_ID}" \
  --gcp-auth \
  -w 8 \
  -o benchmarks/results_calibration_vertex_l4.json

# 3. 30-Query (300-Passage) Listwise Reranking Suite (EXP-10) on Vertex AI L4
echo ""
echo ">>> [Phase 3] Running 30-query dgem bench-rerank on Vertex AI L4 (workers=8)..."
./bin/dgem bench-rerank \
  --vertex-url "${VERTEX_ID}" \
  --gcp-auth \
  -w 8 \
  -o benchmarks/results_rerank_vertex_l4.json

# 4. 12-Case Multimodal SigLIP Bounding-Box Suite (EXP-09) on Vertex AI L4 (g2-standard-16 64GB RAM)
echo ""
echo ">>> [Phase 4] Running 12-case dgem bench-bbox on Vertex AI L4 (workers=4)..."
./bin/dgem bench-bbox \
  --vertex-url "${VERTEX_ID}" \
  --gcp-auth \
  -w 4 \
  -o benchmarks/results_bbox_vertex_l4.json

echo ""
echo ">>> Generating Head-to-Head Comparison Summary..."
python3 - << 'PYEOF'
import json, statistics, os

def analyze_eval(path):
    with open(path) as f:
        d = json.load(f)
    cases = d.get("cases", [])
    gpu_ms = [c["server_denoise_ms"] for c in cases if c.get("server_denoise_ms", 0) > 0]
    wall_ms = [c["wall_time_ms"] for c in cases if c.get("wall_time_ms", 0) > 0]
    gpu_sorted = sorted(gpu_ms)
    wall_sorted = sorted(wall_ms)
    def pct(arr, p):
        if not arr: return 0
        k = int(round((p/100.0)*(len(arr)-1)))
        return arr[max(0, min(k, len(arr)-1))]
    return {
        "accuracy_pct": d.get("overall_accuracy_pct", 0),
        "total_sec": d.get("total_elapsed_sec", 0),
        "throughput_dps": round(len(cases) / max(0.001, d.get("total_elapsed_sec", 1)), 2),
        "gpu_mean_ms": round(statistics.mean(gpu_ms), 1) if gpu_ms else 0,
        "gpu_p50_ms": round(pct(gpu_sorted, 50), 1),
        "gpu_p90_ms": round(pct(gpu_sorted, 90), 1),
        "gpu_p99_ms": round(pct(gpu_sorted, 99), 1),
        "wall_mean_ms": round(statistics.mean(wall_ms), 1) if wall_ms else 0,
        "wall_p50_ms": round(pct(wall_sorted, 50), 1),
        "wall_p90_ms": round(pct(wall_sorted, 90), 1),
    }

summary = {"concurrency_sweep_vertex_l4": {}}
for w in [1, 4, 8, 16]:
    p = f"/tmp/bench_vertex_w{w}.json"
    if os.path.exists(p):
        summary["concurrency_sweep_vertex_l4"][f"w={w}"] = analyze_eval(p)

if os.path.exists("benchmarks/results_cloudrun.json"):
    summary["cloudrun_baseline_w4"] = analyze_eval("benchmarks/results_cloudrun.json")

if os.path.exists("benchmarks/results_calibration_vertex_l4.json"):
    with open("benchmarks/results_calibration_vertex_l4.json") as f:
        cal_v = json.load(f)
    summary["calibration_vertex_l4"] = {
        "accuracy_pct": cal_v.get("overall_accuracy_pct"),
        "ece_10bin": cal_v.get("ece_10bin"),
        "brier_score": cal_v.get("mean_brier_score"),
        "avg_latency_ms": cal_v.get("avg_latency_ms"),
        "total_elapsed_sec": cal_v.get("total_elapsed_sec"),
    }

if os.path.exists("benchmarks/results_calibration_cloudrun.json"):
    with open("benchmarks/results_calibration_cloudrun.json") as f:
        cal_c = json.load(f)
    summary["calibration_cloudrun_l4"] = {
        "accuracy_pct": cal_c.get("overall_accuracy_pct"),
        "ece_10bin": cal_c.get("ece_10bin"),
        "brier_score": cal_c.get("mean_brier_score"),
        "avg_latency_ms": cal_c.get("avg_latency_ms"),
        "total_elapsed_sec": cal_c.get("total_elapsed_sec"),
    }

if os.path.exists("benchmarks/results_rerank_vertex_l4.json"):
    with open("benchmarks/results_rerank_vertex_l4.json") as f:
        rr_v = json.load(f)
    summary["rerank_vertex_l4"] = {
        "ndcg_at_10": rr_v.get("summary", {}).get("exp_ndcg_at_10"),
        "mrr_at_10": rr_v.get("summary", {}).get("exp_mrr_at_10"),
        "tie_rate_pct": rr_v.get("summary", {}).get("exp_tie_rate_pct"),
        "avg_latency_ms": rr_v.get("summary", {}).get("avg_wall_time_ms"),
        "total_elapsed_sec": rr_v.get("total_elapsed_sec"),
    }

if os.path.exists("benchmarks/results_rerank_cloudrun.json"):
    with open("benchmarks/results_rerank_cloudrun.json") as f:
        rr_c = json.load(f)
    summary["rerank_cloudrun_l4"] = {
        "ndcg_at_10": rr_c.get("summary", {}).get("exp_ndcg_at_10"),
        "mrr_at_10": rr_c.get("summary", {}).get("exp_mrr_at_10"),
        "tie_rate_pct": rr_c.get("summary", {}).get("exp_tie_rate_pct"),
        "avg_latency_ms": rr_c.get("summary", {}).get("avg_wall_time_ms"),
        "total_elapsed_sec": rr_c.get("total_elapsed_sec"),
    }

if os.path.exists("benchmarks/results_bbox_vertex_l4.json"):
    with open("benchmarks/results_bbox_vertex_l4.json") as f:
        bb_v = json.load(f)
    summary["bbox_vertex_l4"] = {
        "mean_iou_expectation": bb_v.get("mean_iou_expectation"),
        "iou_at_50_pct": bb_v.get("iou_at_50_expectation_pct"),
        "presence_accuracy_pct": bb_v.get("presence_accuracy_pct"),
        "avg_latency_ms": bb_v.get("avg_wall_time_ms"),
    }

with open("benchmarks/results_head_to_head_vertex_vs_cloudrun.json", "w") as out:
    json.dump(summary, out, indent=2)

print(json.dumps(summary, indent=2))
PYEOF
