# EXP-20: production context length (2026-10-03)

v0.2.0 image (`658ad8ee`) on identical temporary Vertex G4 canaries (g4-standard-48, 1× RTX PRO 6000, KV_CACHE_GB=12,
MAX_SEQS=32, MAX_INFLIGHT=8), differing only in MAX_MODEL_LEN. The decision rule was written before the run (see the
experiments ledger).

- `report.md`, `summary.json`: T1 with three targets (4096 baseline, 8192, 32768). The 32k latency FAIL is a replica
  that died after `long_q1`; container logs were off.
- `latency-rerun-*`: the latency suite on fresh 8k and 32k replicas (both PASS, same server times).
- `long_slice_results.jsonl` (`long_slice.py run`): 19 JevBench long_policy and 60 RAGTruth-train items, padded with
  unrelated RAGTruth-train documents to roughly +0/+5k/+10k/+20k tokens; the 4k target is production (same image).
- `load_*.json` (`long_slice.py load`), `stress.py`: long-prompt load (~8k tokens at 8 concurrent; ~24k tokens at
  8 and 16 concurrent on 32k: 128/128, 0 errors).

Result: 8k is the deploy default for RTX PRO 6000; 32k is pending a repeat T1 with container logging on.
