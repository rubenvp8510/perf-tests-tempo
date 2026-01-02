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

##@ Profile-based Performance Tests
# Set profiles: PROFILES=small,medium,large (comma-separated)
# Set profiles directory: PROFILES_DIR=profiles
# Set test type: TEST_TYPE=ingestion|query|combined

PROFILES ?=
PROFILES_DIR ?= profiles
TEST_TYPE ?= combined
OUTPUT_DIR ?= results

.PHONY: perf-test
perf-test: ## Run performance tests with specified profiles
	@if [ -z "$(PROFILES)" ]; then \
		$(GO) run ./cmd/perf-runner --profiles-dir=$(PROFILES_DIR) --test-type=$(TEST_TYPE) --output=$(OUTPUT_DIR); \
	else \
		$(GO) run ./cmd/perf-runner --profiles=$(PROFILES) --profiles-dir=$(PROFILES_DIR) --test-type=$(TEST_TYPE) --output=$(OUTPUT_DIR); \
	fi

.PHONY: perf-test-dry-run
perf-test-dry-run: ## Dry run - show what would be executed
	@if [ -z "$(PROFILES)" ]; then \
		$(GO) run ./cmd/perf-runner --profiles-dir=$(PROFILES_DIR) --test-type=$(TEST_TYPE) --dry-run; \
	else \
		$(GO) run ./cmd/perf-runner --profiles=$(PROFILES) --profiles-dir=$(PROFILES_DIR) --test-type=$(TEST_TYPE) --dry-run; \
	fi

.PHONY: validate-profiles
validate-profiles: ## Validate all profile YAML files
	$(GO) run ./cmd/perf-runner --profiles-dir=$(PROFILES_DIR) --dry-run

##@ k6 Load Tests (Standalone)
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

##@ Dashboard

.PHONY: dashboard
dashboard: ## Generate HTML dashboard from CSV: make dashboard CSV=results/small-metrics.csv
	@if [ -z "$(CSV)" ]; then \
		echo "Usage: make dashboard CSV=results/small-metrics.csv"; \
		exit 1; \
	fi
	$(GO) run ./cmd/dashboard --input=$(CSV)

.PHONY: dashboards
dashboards: ## Generate dashboards for all CSV files in results/
	@for csv in results/*-metrics.csv; do \
		if [ -f "$$csv" ]; then \
			echo "Generating dashboard for $$csv..."; \
			$(GO) run ./cmd/dashboard --input=$$csv; \
		fi \
	done

.PHONY: compare
compare: ## Compare runs: make compare FILES="results/small-metrics.csv,results/medium-metrics.csv"
	@if [ -z "$(FILES)" ]; then \
		echo "Usage: make compare FILES=\"results/small-metrics.csv,results/medium-metrics.csv\""; \
		exit 1; \
	fi
	$(GO) run ./cmd/dashboard --compare=$(FILES)

##@ Cleanup

.PHONY: clean
clean: ## Clean test cache
	$(GO) clean -testcache

##@ CI

.PHONY: ci
ci: check test ## Run CI pipeline (check + test)
