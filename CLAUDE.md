# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Tempo Performance Test Framework - A Go-based framework for performance testing Tempo (distributed tracing system) on Kubernetes/OpenShift clusters. Automates deployment, load testing with k6, metrics collection, and cleanup.

## Build and Development Commands

```bash
# Format and lint
make format           # Format Go code with gofmt
make check            # Run format-check + vet

# Run tests
make test             # Run all tests: go test -v ./...
make test-examples    # Run example tests only
make test-race        # Run with race detector

# Run specific test with Ginkgo
ginkgo -v --focus "test name pattern" ./examples/...

# Dependencies
make deps             # go mod tidy

# k6 load tests (standalone, requires existing Tempo instance)
make k6-ingestion K6_SIZE=medium   # Sizes: small|medium|large|xlarge
make k6-query
make k6-combined
```

## Architecture

### Core Framework Pattern

The framework uses a facade pattern with `framework.Framework` as the entry point:

1. **Initialization**: `framework.New(ctx, namespace)` creates Kubernetes clients, sets up namespace
2. **Prerequisites**: `CheckPrerequisites()` verifies Tempo and OpenTelemetry operators are installed
3. **Deployment**: `SetupMinIO()` → `SetupTempo("monolithic"|"stack", config)` → `SetupOTelCollector()`
4. **Testing**: `RunK6IngestionTest(size)`, `RunK6QueryTest(size)`, `RunK6CombinedTest(size)`
5. **Metrics**: `CollectMetrics(startTime, outputPath)` exports to CSV/JSON
6. **Cleanup**: `Cleanup()` deletes all tracked resources (5-phase: CRs → wait → cluster-scoped → namespace → orphaned PVs)

### Key Subpackages

- `framework/tempo/` - Tempo deployment (monolithic.go, stack.go)
- `framework/k6/` - k6 job runner and test types
- `framework/metrics/` - Prometheus metrics collection/export
- `framework/gvr/` - All Kubernetes GroupVersionResource definitions
- `framework/config/` - Centralized config with env var overrides

### Resource Tracking

All created resources are tracked via `TrackedResource` and automatically cleaned up. Resources are labeled with `managed-by` and `instance` for identification.

### Tempo Deployment Variants

- `monolithic`: Single-pod deployment (simpler, for smaller workloads)
- `stack`: Distributed deployment (distributor, ingester, querier components)

## Test Structure Pattern

Tests follow Ginkgo BDD style:
```
BeforeEach: Create Framework → CheckPrerequisites → EnableUserWorkloadMonitoring → Deploy (MinIO, Tempo, OTel)
It: Run k6 load test → Assert results
AfterEach: CollectMetrics (optional) → Cleanup
```

## Configuration

### Resource Profiles (Tempo)
- `small`: 4Gi memory, 500m CPU
- `medium`: 8Gi memory, 1000m CPU
- `large`: 12Gi memory, 1500m CPU

### k6 Load Test Sizes
| Size   | Traces/sec | Spans/trace | VUs     |
|--------|------------|-------------|---------|
| small  | 10         | 8-15        | 5-20    |
| medium | 50         | 25-40       | 10-50   |
| large  | 100        | 50-80       | 20-100  |
| xlarge | 500        | 100-150     | 50-200  |

### Environment Variables
```bash
TEMPO_PERF_CR_DELETION_TIMEOUT=120s   # CR deletion timeout
TEMPO_PERF_POD_READY_TIMEOUT=120s     # Pod readiness timeout
TEMPO_PERF_JOB_TIMEOUT=30m            # k6 job timeout
TEMPO_PERF_MAX_CONCURRENT_QUERIES=5   # Prometheus query concurrency
```

## Prerequisites

- Tempo Operator installed (provides TempoMonolithic/TempoStack CRDs)
- OpenTelemetry Operator installed (provides OpenTelemetryCollector CRD)
- k6 with xk6-tempo extension (for standalone k6 tests)
