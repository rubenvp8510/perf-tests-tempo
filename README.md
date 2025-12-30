# Tempo Performance Test Framework

A Go framework for performance testing Tempo on Kubernetes/OpenShift clusters.

## Features

- **YAML Profile Configuration**: Define test profiles in YAML files for easy customization
- **Automated Deployment**: Deploy MinIO, Tempo (monolithic or stack), and OpenTelemetry Collector
- **Parallel Test Execution**: Run ingestion and query tests as separate parallel Kubernetes Jobs
- **Load Testing**: Integrated k6 load testing with MB/s-based throughput targeting
- **Metrics Collection**: Export performance metrics to CSV from Prometheus/Thanos
- **Comprehensive Cleanup**: Automatic cleanup of all resources including finalizer handling

## Prerequisites

- Go 1.21+
- Kubernetes/OpenShift cluster access (kubeconfig configured)
- Tempo Operator installed
- OpenTelemetry Operator installed
- k6 with xk6-tempo extension (for standalone k6 tests)

## Installation

```bash
go mod download
```

## Quick Start

```bash
# Run all profiles with default settings (combined ingestion + query tests)
make perf-test

# Run specific profiles
make perf-test PROFILES=small,medium

# Dry run to see what would be executed
make perf-test-dry-run

# Run only ingestion tests
make perf-test TEST_TYPE=ingestion PROFILES=medium

# Run only query tests
make perf-test TEST_TYPE=query PROFILES=large
```

## CLI Runner

The CLI runner (`cmd/perf-runner`) executes performance tests based on YAML profile configurations.

### Usage

```bash
go run ./cmd/perf-runner [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--profiles` | (all) | Comma-separated list of profiles to run (e.g., `small,medium`) |
| `--profiles-dir` | `profiles` | Directory containing profile YAML files |
| `--output` | `results` | Output directory for metrics CSV files |
| `--test-type` | `combined` | Test type: `ingestion`, `query`, or `combined` |
| `--dry-run` | `false` | Print what would be executed without running |
| `--skip-cleanup` | `false` | Skip cleanup after tests (useful for debugging) |

### Examples

```bash
# Run all profiles in profiles/ directory
go run ./cmd/perf-runner

# Run specific profiles
go run ./cmd/perf-runner --profiles=small,medium

# Run only ingestion tests
go run ./cmd/perf-runner --profiles=medium --test-type=ingestion

# Dry run to preview execution
go run ./cmd/perf-runner --profiles=large --dry-run

# Skip cleanup for debugging
go run ./cmd/perf-runner --profiles=small --skip-cleanup

# Custom output directory
go run ./cmd/perf-runner --profiles=medium --output=/tmp/results
```

## Profile Configuration

Profiles are defined in YAML files in the `profiles/` directory.

### Profile Schema

```yaml
name: medium
description: "Moderate load - typical production baseline"

tempo:
  variant: stack           # "monolithic" or "stack"
  resources:               # Optional - omit to use operator defaults
    memory: "8Gi"
    cpu: "1000m"

k6:
  duration: "5m"
  vus:
    min: 10
    max: 50
  ingestion:
    mbPerSecond: 1         # Target ingestion rate in MB/s
    traceProfile: medium   # Trace complexity: small, medium, large, xlarge
  query:
    queriesPerSecond: 25
```

### Built-in Profiles

| Profile | MB/s | Queries/s | VUs | Description |
|---------|------|-----------|-----|-------------|
| small | 0.1 | 5 | 5-20 | Light load for CI/development |
| medium | 1 | 25 | 10-50 | Typical production baseline |
| large | 5 | 50 | 20-100 | Stress testing |
| xlarge | 20 | 100 | 50-200 | Capacity testing |

### Creating Custom Profiles

Create a new YAML file in `profiles/`:

```yaml
# profiles/custom.yaml
name: custom
description: "Custom high-throughput profile"

tempo:
  variant: stack

k6:
  duration: "10m"
  vus:
    min: 30
    max: 150
  ingestion:
    mbPerSecond: 10
    traceProfile: large
  query:
    queriesPerSecond: 75
```

Then run it:

```bash
go run ./cmd/perf-runner --profiles=custom
```

## Test Execution Flow

When you run a profile, the following happens:

1. **Create Namespace**: `tempo-perf-{profile-name}`
2. **Check Prerequisites**: Verify Tempo and OpenTelemetry operators are installed
3. **Deploy MinIO**: Object storage backend for Tempo
4. **Deploy Tempo**: Using the configured variant (monolithic/stack) and resources
5. **Deploy OTel Collector**: For trace ingestion
6. **Run k6 Tests**:
   - `combined`: Runs ingestion and query as parallel Kubernetes Jobs
   - `ingestion`: Runs only ingestion test
   - `query`: Runs only query test
7. **Save Results**: Export k6 logs and Prometheus metrics
8. **Cleanup**: Delete all resources and namespace

## Output Files

All output files are saved to the `--output` directory (default: `results/`):

| File | Description |
|------|-------------|
| `{profile}-k6-ingestion.log` | k6 ingestion test output (summary, metrics) |
| `{profile}-k6-query.log` | k6 query test output (summary, metrics) |
| `{profile}-metrics.csv` | Prometheus metrics collected during test |

For single test types (`--test-type=ingestion` or `--test-type=query`), only the corresponding log file is created.

Example output structure:
```
results/
├── small-k6-ingestion.log
├── small-k6-query.log
├── small-metrics.csv
├── medium-k6-ingestion.log
├── medium-k6-query.log
└── medium-metrics.csv
```

## Makefile Targets

```bash
make help              # Show all available targets

# Profile-based tests
make perf-test                           # Run all profiles
make perf-test PROFILES=small,medium     # Run specific profiles
make perf-test TEST_TYPE=ingestion       # Run only ingestion
make perf-test-dry-run                   # Preview without executing
make validate-profiles                   # Validate YAML files

# Standalone k6 tests (against existing Tempo)
make k6-ingestion K6_SIZE=medium         # Run ingestion test
make k6-query K6_SIZE=large              # Run query test
make k6-combined                         # Run combined test

# Development
make test              # Run unit tests
make check             # Run format-check + vet
make format            # Format Go code
```

## Standalone k6 Tests

Run k6 tests directly against an existing Tempo instance:

```bash
# Set environment variables
export TEMPO_ENDPOINT="tempo-distributor.tempo.svc.cluster.local:4317"
export TEMPO_QUERY_ENDPOINT="http://tempo-query-frontend.tempo.svc.cluster.local:3200"

# Run tests
make k6-ingestion K6_SIZE=medium
make k6-query K6_SIZE=large
make k6-combined K6_SIZE=small
```

## Framework API

The framework can also be used programmatically:

```go
package main

import (
    "context"
    "github.com/redhat/perf-tests-tempo/test/framework"
    "github.com/redhat/perf-tests-tempo/test/framework/k6"
)

func main() {
    ctx := context.Background()

    // Create framework
    fw, _ := framework.New(ctx, "my-perf-test")
    defer fw.Cleanup()

    // Check prerequisites
    prereqs, _ := fw.CheckPrerequisites()
    if !prereqs.AllMet {
        panic("Prerequisites not met")
    }

    // Deploy stack
    fw.SetupMinIO()
    fw.SetupTempo("stack", nil)
    fw.SetupOTelCollector()

    // Run parallel tests
    result, _ := fw.RunK6ParallelTests(&k6.Config{
        MBPerSecond:      1.0,
        QueriesPerSecond: 25,
        Duration:         "5m",
        VUsMin:           10,
        VUsMax:           50,
        TraceProfile:     "medium",
    })

    if result.Success() {
        println("Tests passed!")
    }
}
```

## Package Structure

```
cmd/
└── perf-runner/       # CLI entry point
    └── main.go

profiles/              # YAML profile configurations
├── small.yaml
├── medium.yaml
├── large.yaml
└── xlarge.yaml

framework/
├── framework.go       # Core Framework, New()
├── types.go           # Interfaces, TrackedResource, ResourceConfig
├── facade.go          # API methods (SetupMinIO, SetupTempo, etc.)
├── prerequisites.go   # CheckPrerequisites() - operator verification
├── monitoring.go      # EnableUserWorkloadMonitoring() - OpenShift monitoring
├── namespace.go       # Namespace lifecycle
├── cleanup.go         # Resource cleanup with finalizer handling
├── profile/           # YAML profile loading
├── tempo/             # Tempo deployment (monolithic + stack)
├── minio/             # MinIO storage backend
├── otel/              # OpenTelemetry Collector
├── k6/                # Load testing runner (parallel jobs)
├── wait/              # Readiness utilities
└── metrics/           # Metrics collection and export

tests/
└── k6/                # k6 test scripts
    ├── ingestion-test.js
    ├── query-test.js
    ├── combined-test.js
    └── lib/           # Shared k6 utilities
        ├── config.js
        └── trace-profiles.js
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPO_PERF_CR_DELETION_TIMEOUT` | `120s` | CR deletion timeout |
| `TEMPO_PERF_POD_READY_TIMEOUT` | `120s` | Pod readiness timeout |
| `TEMPO_PERF_JOB_TIMEOUT` | `30m` | k6 job timeout |
| `TEMPO_PERF_MAX_CONCURRENT_QUERIES` | `5` | Prometheus query concurrency |
