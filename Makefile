.DEFAULT_GOAL := help

# Variables
GO := go
GOFMT := gofmt
GOLINT := golangci-lint

##@ General

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Development

.PHONY: format
format: ## Format Go code
	$(GOFMT) -s -w .

.PHONY: format-check
format-check: ## Check code formatting
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make format'" && $(GOFMT) -l . && exit 1)

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@command -v $(GOLINT) >/dev/null || (echo "Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	$(GOLINT) run ./...

.PHONY: check
check: format-check vet ## Run all checks

##@ Testing

.PHONY: test
test: ## Run all tests
	$(GO) test -v ./...

.PHONY: test-examples
test-examples: ## Run example tests
	$(GO) test -v ./examples/...

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GO) test -race ./...

##@ Dependencies

.PHONY: deps
deps: ## Tidy dependencies
	$(GO) mod tidy

.PHONY: deps-update
deps-update: ## Update dependencies to latest
	$(GO) get -u ./...
	$(GO) mod tidy

##@ k6 Load Tests
# Set test size: K6_SIZE=small|medium|large|xlarge (default: medium)

K6_SIZE ?= medium

.PHONY: k6-all
k6-all: k6-ingestion k6-query k6-combined ## Run all k6 tests

.PHONY: k6-ingestion
k6-ingestion: ## Run k6 ingestion test
	SIZE=$(K6_SIZE) k6 run tests/k6/ingestion-test.js

.PHONY: k6-query
k6-query: ## Run k6 query test
	SIZE=$(K6_SIZE) k6 run tests/k6/query-test.js

.PHONY: k6-combined
k6-combined: ## Run k6 combined test
	SIZE=$(K6_SIZE) k6 run tests/k6/combined-test.js

##@ Cleanup

.PHONY: clean
clean: ## Clean test cache
	$(GO) clean -testcache

##@ CI

.PHONY: ci
ci: check test ## Run CI pipeline (check + test)
