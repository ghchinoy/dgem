.PHONY: help build run test fmt clean setup download serve stop bench bench-ecotone install docs-build docs-dev cloudrun-deploy gce-deploy gce-teardown

.DEFAULT_GOAL := help

# Version and build metadata
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

help: ## Show this help message
	@echo "dgem - DiffusionGemma Assistant & Structured Decision Engine"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

build: ## Build the dgem binary into bin/
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o bin/dgem .

run: build ## Build and run dgem
	./bin/dgem

test: ## Run Go unit tests
	go test -v ./pkg/...

fmt: ## Format Go source code
	go fmt ./...

clean: ## Remove compiled binaries and temporary artifacts
	rm -rf bin/ dgem diffgemma.pid

setup: ## Verify prerequisites and install diffgemma engine
	./scripts/setup_diffgemma.sh

download: ## Download the 4-bit DiffusionGemma model pack (mmastrac/diffgemma-26b-a4b-it-q4)
	./scripts/download_model.sh

serve: ## Start the diffgemma server in background (32k context on 127.0.0.1:8080)
	./scripts/serve.sh

stop: ## Stop the background diffgemma server
	./scripts/stop_server.sh

bench: build ## Run the local Jev vs autoregressive benchmark suite
	./bin/dgem bench

bench-ecotone: build ## Run Ecotone (Sparrowhawk WFST) vs DiffusionGemma semiotics benchmark
	./bin/dgem bench-ecotone

bench-calibration: build ## Run the public dataset calibration, guardrail, and human-entropy suite
	./bin/dgem bench-calibration

docs-build: ## Build the Astro Starlight documentation site
	pnpm run --dir docs-site build

docs-dev: ## Launch local development server for the documentation site
	pnpm run --dir docs-site dev

cloudrun-build: ## Build and push the self-contained Cloud Run container image to Artifact Registry
	./scripts/build_cloudrun_image.sh

cloudrun-stage: ## Pre-stage model weights in GCS for Cloud Run GCS FUSE volume mounting
	./scripts/stage_model_gcs.sh

cloudrun-deploy: ## Deploy DiffusionGemma to Google Cloud Run with GPU (L4 or RTX Pro 6000)
	./scripts/deploy_cloudrun_vllm.sh

cloudrun-teardown: ## Delete Cloud Run dgemma GPU service to eliminate any cloud resource footprint
	gcloud run services delete dgemma --region=$${GCP_REGION:-us-central1} --quiet

gce-deploy: ## Deploy vLLM with DiffusionGemma structured reads to GCE with L4 GPU
	./scripts/deploy_gce_vllm.sh

gce-teardown: ## Delete GCE GPU benchmark instance and firewall to eliminate idle cost
	./scripts/teardown_gce_vllm.sh

install: ## Install dgem binary to GOBIN
	go install -ldflags="$(LDFLAGS)" .
