# Changelog

Releases of the `dgem` serving images (`dgem`, `dgem-weights`), gateway and CLI. One version covers all of them.
Versioning: `vMAJOR.MINOR.PATCH`. **Major:** breaking API or response changes. **Minor:** vLLM runtime or model
changes, new features. **Patch:** fixes. Images are published to
`us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/` as `:<version>` and `:<commit>`; `:latest` moves only after
a release is validated. Release process: [runbook](docs/operate/runbook.md#release-a-new-version). Gateway/CLI-only releases
(`make release-gateway`) reuse the previous serving images.

## v0.1.1 (2026-09-28)

Gateway, MCP and CLI only. **Serving images are unchanged**: the latest serving image is still `v0.1.0`
(`dgem@sha256:5fa4a866…`), and production Vertex and Cloud Run keep running it.

- Security (#18): `POST /api/backend-config` and `/api/vertex/deploy|teardown` require `--enable-admin-api`
  (off by default); request-supplied `vertex_url` limited to the configured endpoint or
  `--allowed-vertex-endpoints`. Previously any signed-in user could change routing for everyone or deploy/tear down
  the Vertex endpoint.
- Backend allow-list (#18): `--backends` / `DGEM_BACKENDS`, validated at startup; unknown or disabled backends
  return 400 on every surface. Studio backend choice is per browser.
- MCP: `dgem mcp` over stdio honours `DGEM_VERTEX_URL` for the allow-list (#19), health and routing (#22); tool
  schemas list only the configured backends (#23).

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
