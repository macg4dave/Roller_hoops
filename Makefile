# Roller_hoops — Docker-only Makefile
#
# All targets run via Docker so no local Go, Node, or Postgres is needed.
# Prerequisites: Docker + Docker Compose (and GNU make on the host).
#
# Windows note: install make via  choco install make  or  scoop install make
#               or run inside WSL where make is available by default.

SHELL := /bin/sh
.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Compose
# ---------------------------------------------------------------------------

.PHONY: up
up: ## Start the full stack (default profile)
	docker compose up --build

.PHONY: dev
dev: ## Start the stack with dev-seed data
	docker compose --profile dev up --build

.PHONY: down
down: ## Stop the stack
	docker compose down

.PHONY: reset
reset: ## Stop the stack and destroy volumes (wipes DB)
	docker compose --profile dev down -v

.PHONY: logs
logs: ## Tail all container logs
	docker compose logs -f --tail=200

.PHONY: ps
ps: ## Show running containers
	docker compose ps

# ---------------------------------------------------------------------------
# Validation — Go
# ---------------------------------------------------------------------------

.PHONY: go-fmt
go-fmt: ## Check Go formatting
	docker build -f docker/validate/core-go.Dockerfile --target fmtcheck .

.PHONY: go-vet
go-vet: ## Run go vet
	docker build -f docker/validate/core-go.Dockerfile --target vet .

.PHONY: go-test
go-test: ## Run Go tests
	docker build -f docker/validate/core-go.Dockerfile --target test .

.PHONY: go-validate
go-validate: go-fmt go-vet go-test ## Run all Go checks

# ---------------------------------------------------------------------------
# Validation — UI (Node/Next.js)
# ---------------------------------------------------------------------------

.PHONY: ui-deps
ui-deps: ## Install / verify UI dependencies
	docker build -f docker/validate/ui-node.Dockerfile --target deps .

.PHONY: ui-test
ui-test: ## Run UI tests
	docker build -f docker/validate/ui-node.Dockerfile --target test .

.PHONY: ui-build
ui-build: ## Run UI build / type-check
	docker build -f docker/validate/ui-node.Dockerfile --target build .

.PHONY: ui-validate
ui-validate: ui-test ui-build ## Run all UI checks

# ---------------------------------------------------------------------------
# OpenAPI type generation
# ---------------------------------------------------------------------------

.PHONY: gen-types
gen-types: ## Regenerate ui-node/lib/api-types.ts from api/openapi.yaml
	@IMAGE=$$(docker build -q -f docker/validate/ui-node.Dockerfile --target generated-types .); \
	CID=$$(docker create $$IMAGE); \
	docker cp "$$CID:/workspace/ui-node/lib/api-types.ts" ui-node/lib/api-types.ts; \
	docker rm -f $$CID >/dev/null

# ---------------------------------------------------------------------------
# Full validation
# ---------------------------------------------------------------------------

.PHONY: validate
validate: go-validate ui-validate ## Run all validation checks (Go + UI)

# ---------------------------------------------------------------------------
# Dev tools container
# ---------------------------------------------------------------------------

.PHONY: devtools-build
devtools-build: ## Build the devtools image
	docker build -f docker/devtools.Dockerfile -t roller-devtools .

.PHONY: devtools
devtools: devtools-build ## Open a shell with Go + Node + project tools
	docker run --rm -it \
		-v "$$(pwd)":/workspace \
		-w /workspace \
		roller-devtools sh

# ---------------------------------------------------------------------------
# Smoke test
# ---------------------------------------------------------------------------

.PHONY: smoke
smoke: ## Full stack smoke (prod-readiness profile)
	docker compose --profile prod up --build --abort-on-container-exit

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

.PHONY: clean
clean: ## Remove dangling images and build cache
	docker image prune -f
	docker builder prune -f

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
