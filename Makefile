.PHONY: help run test format lint vet clean deps build install-tools coverage

# Variables
GO := go
GOFMT := gofmt
GOLINT := golangci-lint
TEST_PACKAGES := ./...
COVERAGE_FILE := coverage.out
BINARY_NAME := perf-tests

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: test ## Run tests (alias for test)

test: ## Run all tests
	$(GO) test -v $(TEST_PACKAGES)

test-verbose: ## Run tests with verbose output
	$(GO) test -v -race $(TEST_PACKAGES)

test-coverage: ## Run tests with coverage
	$(GO) test -v -coverprofile=$(COVERAGE_FILE) $(TEST_PACKAGES)
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-examples: ## Run example tests only
	$(GO) test -v ./examples/...

test-framework: ## Run framework tests only
	$(GO) test -v ./framework/...

format: ## Format Go code
	$(GOFMT) -s -w .
	@echo "Code formatted"

format-check: ## Check if code is formatted correctly
	@if [ $$($(GOFMT) -l . | wc -l) -ne 0 ]; then \
		echo "Code is not formatted. Run 'make format' to fix."; \
		$(GOFMT) -l .; \
		exit 1; \
	fi
	@echo "Code is properly formatted"

lint: ## Run linter
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run $(TEST_PACKAGES); \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

vet: ## Run go vet
	$(GO) vet $(TEST_PACKAGES)

mod-tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy
	@echo "Dependencies tidied"

mod-verify: ## Verify dependencies
	$(GO) mod verify
	@echo "Dependencies verified"

deps: mod-tidy ## Update and tidy dependencies
	@echo "Dependencies updated"

deps-update: ## Update all dependencies to latest versions
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated to latest versions"

build: ## Build the test binary
	$(GO) build -o $(BINARY_NAME) .

build-metrics-exporter: ## Build the metrics exporter CLI tool
	@mkdir -p bin
	$(GO) build -o $(METRICS_EXPORTER) ./cmd/metrics-exporter
	@echo "Metrics exporter built: $(METRICS_EXPORTER)"

build-all: build build-metrics-exporter ## Build all binaries

clean: ## Clean build artifacts
	$(GO) clean -cache -testcache
	rm -f $(BINARY_NAME) $(COVERAGE_FILE) coverage.html
	rm -rf bin/
	@echo "Clean complete"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

check: format-check vet ## Run all checks (format, vet)
	@echo "All checks passed"

ci: format-check vet test ## Run CI checks (format, vet, test)
	@echo "CI checks complete"

coverage: test-coverage ## Generate coverage report (alias for test-coverage)

benchmark: ## Run benchmarks
	$(GO) test -bench=. -benchmem $(TEST_PACKAGES)

test-race: ## Run tests with race detector
	$(GO) test -race $(TEST_PACKAGES)

test-short: ## Run tests in short mode
	$(GO) test -short $(TEST_PACKAGES)

test-timeout: ## Run tests with timeout (useful for long-running tests)
	$(GO) test -timeout 30m $(TEST_PACKAGES)

.PHONY: all
all: format-check vet test ## Run format check, vet, and tests

# K6 Performance Tests
# Requires xk6-tempo: https://github.com/rubenvp8510/xk6-tempo

K6_DIR := tests/k6
K6_SIZE ?= medium

.PHONY: k6-check k6-ingestion k6-query k6-combined k6-all

k6-check: ## Check if k6 with xk6-tempo is installed
	@k6 version || (echo "Error: k6 is not installed. Install xk6-tempo from https://github.com/rubenvp8510/xk6-tempo" && exit 1)

k6-ingestion: k6-check ## Run k6 ingestion test (SIZE=small|medium|large|xlarge)
	@echo "Running k6 ingestion test (size: $(K6_SIZE))..."
	SIZE=$(K6_SIZE) k6 run $(K6_DIR)/ingestion-test.js

k6-query: k6-check ## Run k6 query test (SIZE=small|medium|large|xlarge)
	@echo "Running k6 query test (size: $(K6_SIZE))..."
	SIZE=$(K6_SIZE) k6 run $(K6_DIR)/query-test.js

k6-combined: k6-check ## Run k6 combined test (SIZE=small|medium|large|xlarge)
	@echo "Running k6 combined test (size: $(K6_SIZE))..."
	SIZE=$(K6_SIZE) k6 run $(K6_DIR)/combined-test.js

k6-all: k6-ingestion k6-query k6-combined ## Run all k6 tests sequentially

