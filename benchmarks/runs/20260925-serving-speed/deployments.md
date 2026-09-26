# Deployment log — 2026-09-25

| Time (UTC) | Service | Change | Result | Rollback |
| :--- | :--- | :--- | :--- | :--- |
| 22:00–22:04 | Artifact Registry | Built `dgem/dgemma:ab208dd` (digest `sha256:8405750c…`) via Cloud Build | OK | — |
| 22:06–22:16 | Vertex `4423577720856772608` (`dgemma-dedicated-g4`) | New endpoint: `g4-standard-48` + 1× RTX PRO 6000, image `ab208dd`, replicas 1–2 (duty-cycle 70%), `DISABLE_MM=0`, `KV_CACHE_GB=12`, `MAX_SEQS=32`, `DEFAULT_SAMPLES=1`, `MAX_INFLIGHT=8` | Ready and self-warmed (18.6 s) at first poll; text, 4-sample, dual-mirror and image decisions verified | Undeploy model |
| 23:04–23:12 | Cloud Run `dgemma` → revision `dgemma-00030-qkl` | Image `ab208dd`; `DEFAULT_SAMPLES=1`, `MAX_INFLIGHT=8`; `HF_TOKEN` moved from plaintext env to Secret Manager `dgemma-hf-token:latest`; boot command without the old health patch | Healthy; weights staged in 403 s; self-warmup requests completed (first requests took 27–150 s while kernels compiled, then ~100 ms); text, 4-sample and image decisions verified | `gcloud run services update-traffic dgemma --to-revisions=dgemma-00029-ds9=100` |
| 23:14–23:36 | Cloud Run `dgemma-gateway` → revision `dgemma-gateway-00046-mcq` | Image `dgemma-gateway:6857cbe` (Studio text + G4 defaults); `DGEM_VERTEX_URL=4423577720856772608` | `/api/decide` → `X-DGem-Backend-Used: vertex` (256–300 ms client wall); `X-DGem-Backend: cloudrun` override works; Studio loads; `/api/backend-config` reports G4 `deployed`, 1 replica | `gcloud run services update-traffic dgemma-gateway --to-revisions=dgemma-gateway-00045-fdz=100` |
| 23:40–23:55 | Cloud Run `dgemma-gateway` → revision `dgemma-gateway-00047-wvm` | Image `dgemma-gateway:a081cf8` (status display name fix) | `/api/backend-config` shows `dgemma-dedicated-g4` deployed; `/api/decide` → vertex | Revision `00046-mcq` |
| 23:37 | Vertex `4217256562927861760` (L4) | Undeployed model `9093397118067933184`; empty endpoint and uploaded models retained | 0 deployed models (no GPU billing) | `VERTEX_PROFILE=l4 ./scripts/deploy_vertex_endpoint.sh` |
| 00:04–00:15 (09-26) | Cloud Run job `dgemma-dltest-*` (temporary, deleted) | Cloud Storage download throughput test, 4 GiB, 8 vCPU | Public egress 46–52 MiB/s; Direct VPC egress (default subnet, Private Google Access) 395–441 MiB/s | — |
| 00:16–00:19 (09-26) | Cloud Run `dgemma` → revision `dgemma-00031-47c` | Direct VPC egress (`--network default --subnet default --vpc-egress all-traffic`) | Weight copy 403 s → **83 s**; container start → warmed **~2.5 min** (was ~7.5); text/4-sample/image decisions and gateway→cloudrun verified | Revision `00030-qkl` |
| 00:10 (09-26) | Secret Manager `dgemma-hf-token` | Version 2 added (rotated token), version 1 disabled | Read via `:latest` on next revision/cold start (00031 uses v2) | — |

Follow-ups:
- ~~Rotate the Hugging Face token~~ done (version 2).
- ~~Cloud Run health lacks `warmed`~~ not a bug: the earlier check ran before warmup finished; later checks show
  `"warmed": true`.
