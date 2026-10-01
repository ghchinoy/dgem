# Changelog

Releases of the `dgem` serving images (`dgem`, `dgem-weights`), gateway and CLI. One version covers all of them.
Versioning: `vMAJOR.MINOR.PATCH`. **Major:** breaking API or response changes. **Minor:** vLLM runtime or model
changes, new features. **Patch:** fixes. Images are published to
`us-central1-docker.pkg.dev/dgem-diffusiongemma/dgem/` as `:<version>` and `:<commit>`; `:latest` moves only after
a release is validated. Release process: [runbook](docs/operate/runbook.md#release-a-new-version). Gateway/CLI-only releases
(`make release-gateway`) reuse the previous serving images.

## Unreleased

## v0.1.3 (2026-10-01)

Serving images rebuilt (structured server fix). Gateway, MCP and CLI from the same commit.

- **Fix (#42):** `/v1/systemone` escaped every non-ASCII character in a JSON-object `state` to `\uXXXX`
  (`json.dumps` defaults, inherited from vLLM's example), so Russian, Thai or Hindi input reached the model as hex
  codes. Now `ensure_ascii=False`; ASCII-only states produce byte-identical prompts. MASSIVE (51 languages, 20 options)
  macro accuracy 0.444 → 0.821 on Vertex G4. `chat/completions` (`dgem decide`, bench commands), the gateway's default
  systemone adapter and string states were not affected.
- `/v1/systemone` noul answers include `probabilities` (`true`/`false`) and `confidence`, like choice and score.
- Decision responses send `Server-Timing: decide;dur=<ms>` and `X-Inference-Time-Ms`.
- `scripts/contract_diff.py`: six multilingual systemone cases with known answers, reported as `WRONG` on any target.
- `bench-intents`: more than 26 candidate intents run as a 2-stage bracket (groups of ≤ 20, top 5 to a final round)
  instead of being rejected by the server (`--dataset banking77` sends 30).
- Docs: self-hosted evaluation guide; EXP-18 rerun on v0.1.0, throughput and cost per 1k; CI check for docs/ and
  docs-site/ drift.

## v0.1.2 (2026-09-28)

Gateway and monitoring; serving images unchanged (v0.1.0).

- Monitoring: scheduled health check (`scripts/probe.py`, `scripts/deploy_probe.sh`: Cloud Run jobs + Scheduler),
  alert policies and log-based metrics (`scripts/setup_alerts.py`), guide `docs/operate/monitoring.md`.
- Gateway decision logs record the requested backend (`dgem_backend_requested`) next to the one that answered, and
  failed requests are logged with backend `none` instead of `cloudrun`, so failover can be measured.
- `scripts/template_sweep.py`: run every policy template through a gateway with its sample variables.

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
