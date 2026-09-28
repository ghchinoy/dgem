# Changelog

Releases of the `dgem` serving images (`dgem`, `dgem-weights`), gateway and CLI. One version covers all of them.
Versioning: `vMAJOR.MINOR.PATCH`. **Major:** breaking API or response changes. **Minor:** vLLM runtime or model
changes, new features. **Patch:** fixes. Images are published to
`us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/` as `:<version>` and `:<commit>`; `:latest` moves only after
a release is validated. Release process: [runbook](docs/operate/runbook.md#release-a-new-version).

## v0.1.0 (2026-09-28)

First versioned release (tag on `b8e1af1`; images were built from `f241b77`, which has the same `deploy/` tree
and differs only in docs and MCP help text). Serving is built on upstream vLLM nightly `a9eafde` (PRs #57250, #58216) plus
`deploy/cloudrun/server/dgem.patch`.

- Serving: upstream vLLM base (replaces the fork base), `/health` readiness and version fields, HTTP/1.1
  keep-alive with backlog 256 (fixes Vertex 503s at 16+ concurrent requests), `MAX_INFLIGHT` queueing,
  `DEFAULT_SAMPLES=1`, digit-string `samples`, `/predict` and `/rawPredict` aliases, atomic warm-up state,
  baked weights used when present.
- Validated side by side with the previous production image: API contract identical, JevBench and calibration
  within run-to-run noise with slightly better Brier, 6–10% faster GPU time, 0 errors at 32 concurrent clients
  ([image parity run](benchmarks/runs/20260927-image-parity/README.md)).
- Gateway and MCP: derived `available_backends`, port 8090, Gemini-safe MCP tool schemas, shared decide summary
  fields, `--vertex-url` needs only the project number.
- Images carry OCI version labels; `/health` (serving and gateway) and `dgem --version` report the version.
