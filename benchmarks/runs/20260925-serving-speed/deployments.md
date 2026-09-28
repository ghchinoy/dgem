# Deployment log — 2026-09-25

| Time (UTC) | Service | Change | Result | Rollback |
| :--- | :--- | :--- | :--- | :--- |
| 22:00–22:04 | Artifact Registry | Built `dgem/dgemma:ab208dd` (digest `sha256:8405750c…`) via Cloud Build | OK | — |
| 22:06–22:16 | Vertex `<endpoint-id>` (`dgemma-dedicated-g4`) | New endpoint: `g4-standard-48` + 1× RTX PRO 6000, image `ab208dd`, replicas 1–2 (duty-cycle 70%), `DISABLE_MM=0`, `KV_CACHE_GB=12`, `MAX_SEQS=32`, `DEFAULT_SAMPLES=1`, `MAX_INFLIGHT=8` | Ready and self-warmed (18.6 s) at first poll; text, 4-sample, dual-mirror and image decisions verified | Undeploy model |
| 23:04–23:12 | Cloud Run `dgemma` → revision `dgemma-00030-qkl` | Image `ab208dd`; `DEFAULT_SAMPLES=1`, `MAX_INFLIGHT=8`; `HF_TOKEN` moved from plaintext env to Secret Manager `dgemma-hf-token:latest`; boot command without the old health patch | Healthy; weights staged in 403 s; self-warmup requests completed (first requests took 27–150 s while kernels compiled, then ~100 ms); text, 4-sample and image decisions verified | `gcloud run services update-traffic dgemma --to-revisions=dgemma-00029-ds9=100` |
| 23:14–23:36 | Cloud Run `dgemma-gateway` → revision `dgemma-gateway-00046-mcq` | Image `dgemma-gateway:6857cbe` (Studio text + G4 defaults); `DGEM_VERTEX_URL=<endpoint-id>` | `/api/decide` → `X-DGem-Backend-Used: vertex` (256–300 ms client wall); `X-DGem-Backend: cloudrun` override works; Studio loads; `/api/backend-config` reports G4 `deployed`, 1 replica | `gcloud run services update-traffic dgemma-gateway --to-revisions=dgemma-gateway-00045-fdz=100` |
| 23:40–23:55 | Cloud Run `dgemma-gateway` → revision `dgemma-gateway-00047-wvm` | Image `dgemma-gateway:a081cf8` (status display name fix) | `/api/backend-config` shows `dgemma-dedicated-g4` deployed; `/api/decide` → vertex | Revision `00046-mcq` |
| 23:37 | Vertex `<legacy-endpoint-id>` (L4) | Undeployed model `<legacy-model-id>`; empty endpoint and uploaded models retained | 0 deployed models (no GPU billing) | `VERTEX_PROFILE=l4 ./scripts/deploy_vertex_endpoint.sh` |
| 00:04–00:15 (09-26) | Cloud Run job `dgemma-dltest-*` (temporary, deleted) | Cloud Storage download throughput test, 4 GiB, 8 vCPU | Public egress 46–52 MiB/s; Direct VPC egress (default subnet, Private Google Access) 395–441 MiB/s | — |
| 00:16–00:19 (09-26) | Cloud Run `dgemma` → revision `dgemma-00031-47c` | Direct VPC egress (`--network default --subnet default --vpc-egress all-traffic`) | Weight copy 403 s → **83 s**; container start → warmed **~2.5 min** (was ~7.5); text/4-sample/image decisions and gateway→cloudrun verified | Revision `00030-qkl` |
| 00:10 (09-26) | Secret Manager `dgemma-hf-token` | Version 2 added (rotated token), version 1 disabled | Read via `:latest` on next revision/cold start (00031 uses v2) | — |

| 09-27 20:00 – 09-28 14:55 | Canaries (Vertex G4 endpoints + Cloud Run `dgemma-canary`, all deleted) | Side-by-side validation of the upstream-based image; four serving regressions fixed | See [`20260927-image-parity`](../20260927-image-parity/README.md) | — |
| 09-28 14:40–14:51 | Vertex `<endpoint-id>` | Image `504638d` deployed next to `ab208dd` at 0%, switched to 100%, old model undeployed | Contract re-checked on the prod endpoint | Redeploy the previous model resource (~15 min) |
| 09-28 14:20 | Cloud Run `dgemma` → revision `00032` | Image `504638d` (no-traffic tag, verified, then 100%) | Contract identical | Revision `00031` |
| 09-28 14:41–14:52 | Cloud Run `dgemma-gateway` → revision `00050` | Rebuilt from `main` (PRs #5–#11), `DGEM_GATEWAY_HOSTS` set, no-traffic tag then 100% | Backend config, all backends, MCP, IAP custom domain | Revision `00047` |

| 09-28 15:39–16:35 | Public images `dgem:4b1b809`, `dgem-weights:4b1b809` (dgem-diffusiongemma) | First build with all serving fixes + digit-string samples; canary-validated | Contract identical to prod (plus samples "4"), 0 errors at 16/32 workers; baked image uses its weights | — |
| 09-28 17:06–17:34 | Release **v0.1.0**: `dgem@sha256:5fa4a866…`, `dgem-weights@sha256:cbbb53c2…` | First versioned release (`make release`); images report `version` in `/health` | — | — |
| 09-28 17:27–17:40 | Vertex `<endpoint-id>` | v0.1.0 deployed next to `504638d` at 0% (`VERTEX_NEW_TRAFFIC=0`), switched to 100% | Contract identical; GPU 55.2 / 91.0 / 63.2 ms (1 sample / 4 samples / image); 0 errors in 1,024 requests at 16/32 workers, up to 73.9 req/s | Move traffic back to the `504638d` deployed model (kept deployed) |
| 09-28 17:20 | Cloud Run `dgemma` → revision `00034` (tag `v010`) | v0.1.0 via `deploy_cloudrun_vllm.sh CLOUDRUN_TAG=v010` (refreshes the embedded entrypoint), verified, 100% | `/health` version v0.1.0; contract identical | Revision `00032` |
| 09-28 17:45 | Cloud Run `dgemma-gateway` → revision `00053` | Rebuilt from `main` (`v0.1.0-2-gbbcbc45`): PRs #13, #15, #17, MCP health text, client samples fix | Backend config, all three backends, MCP tools/list, decide summary fields, health tool, IAP custom domain, `dgem mcp --remote` | Revision `00050` |

| 09-28 ~19:00 | Cloud Run `dgemma-gateway` → revision `00055` | Rebuilt from `main` `3216438` (`v0.1.0-5-g3216438`, PR #18: backend allow-list, admin API **off**, `vertex_url` restricted); `DGEM_VERTEX_MODEL_ID` = v0.1.0 model | All backends route correctly; admin POSTs 403; unknown backend and foreign `vertex_url` 400 (HTTP and MCP); default unchanged; MCP via `--remote`; IAP custom domain | Revision `00053` |
| 09-28 | Vertex `<endpoint-id>` | `504638d` deployed model undeployed (model resource kept) | Endpoint serves only v0.1.0 | Redeploy the kept model (~15 min) |
| 09-28 | Public registry | Deleted `dgem` / `dgem-weights` tags `56baadf` (pre-fix, buggy) and `4b1b809` (superseded) | Registry holds `v0.1.0` = `latest` | — |

| 09-28 ~21:00 | Cloud Run `dgemma-gateway` → revision `00057` | **v0.1.1** (gateway/CLI-only release; serving stays v0.1.0): PRs #19, #22, #23 on top of #18; admin API off | `/health` version v0.1.1; all backends route; admin POSTs 403; unknown backend / foreign `vertex_url` 400; MCP tool schemas list `[vertex_first, vertex, cloudrun]` and reject unknown backends at validation; IAP custom domain; `dgem mcp --remote` | Revision `00055` |

Follow-ups:
- ~~Rotate the Hugging Face token~~ done (version 2).
- ~~Cloud Run health lacks `warmed`~~ not a bug: the earlier check ran before warmup finished; later checks show
  `"warmed": true`.
