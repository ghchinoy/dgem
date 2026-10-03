# EXP-21: attributing the ForecastBench / PhishNChips drops (2026-10-03)

Production v0.2.0 through local `dgem systemone serve --prompt-layout {schema_first,document_first}` adapters
(`run.py`, `run_fbd.py`). Yes/no descriptions "dropped" = criteria removed from noul questions (the pre-#53 adapter
behaviour). No Decision Index rows were used.

- `forecast_dev.json`: 182 single market questions (manifold, metaculus, polymarket) from ForecastBench question sets
  2025-03-16 and 2025-05-25, forecast dates absent from the index; question texts present in the index removed.
- `forecast_dataset_dev.json`: 320 single dataset-source questions (FRED, yfinance, ACLED, DBnomics, Wikipedia) from
  the same dates, templates filled, one resolved date each. The index's ForecastBench is ~92% dataset-source.
- phishing: 200 balanced emails from `ealvaradob/phishing-dataset` (texts.json, apache-2.0) with a URL, rendered in the
  index's PhishNChips question schema (body-only states).
- `results_*.jsonl`: answers (choice, noul, probabilities) per item and arm.

Result: neither factor changes either family beyond noise. Combination forecasting questions are untested.
