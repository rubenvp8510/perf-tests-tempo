# Tempo Performance Test Framework

A Go framework for performance testing [Tempo](https://grafana.com/oss/tempo/) distributed tracing backend on Kubernetes/OpenShift clusters.

## Overview

This tool automates the entire performance testing lifecycle:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Performance Test Flow                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. SETUP                    2. TEST                    3. COLLECT          │
│  ┌──────────────┐           ┌──────────────┐           ┌──────────────┐     │
│  │ Create NS    │           │ k6 Ingestion │           │ k6 Logs      │     │
│  │ Deploy MinIO │    ──►    │ Job (OTLP)   │    ──►    │ Prometheus   │     │
│  │ Deploy Tempo │           │              │           │ Metrics      │     │
│  │ Deploy OTel  │           │ k6 Query     │           │              │     │
│  └──────────────┘           │ Job (HTTP)   │           └──────────────┘     │
│                             └──────────────┘                                 │
│                                    │                                         │
│                                    ▼                                         │
│                            4. CLEANUP                                        │
│                            ┌──────────────┐                                  │
│                            │ Delete CRs   │                                  │
│                            │ Delete NS    │                                  │
│                            └──────────────┘                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **YAML Profile Configuration**: Define test profiles in YAML files for easy customization
- **Automated Deployment**: Deploy MinIO, Tempo (monolithic or stack), and OpenTelemetry Collector
- **Parallel Test Execution**: Run ingestion and query tests as separate parallel Kubernetes Jobs
- **MB/s-based Throughput**: Specify ingestion rate in MB/s, automatically calculated to traces/sec
- **Metrics Collection**: Export performance metrics to CSV from Prometheus/Thanos
- **Comprehensive Cleanup**: Automatic cleanup of all resources including finalizer handling

## Architecture

### Component Interaction

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Namespace: tempo-perf-{profile}                   │    │
│  │                                                                      │    │
│  │   ┌─────────────┐      ┌─────────────────┐      ┌──────────────┐    │    │
│  │   │  k6 Job     │      │  OTel Collector │      │    Tempo     │    │    │
│  │   │ (Ingestion) │─────►│                 │─────►│  (receives   │    │    │
│  │   │             │ OTLP │  (receives and  │ OTLP │   traces)    │    │    │
│  │   └─────────────┘ gRPC │   forwards)     │ gRPC │              │    │    │
│  │                        └─────────────────┘      └──────┬───────┘    │    │
│  │   ┌─────────────┐                                      │            │    │
│  │   │  k6 Job     │                                      │            │    │
│  │   │  (Query)    │──────────────────────────────────────┘            │    │
│  │   │             │  HTTP (TraceQL queries)                           │    │
│  │   └─────────────┘                                                   │    │
│  │                                                                      │    │
│  │   ┌─────────────┐      ┌─────────────────┐                          │    │
│  │   │   MinIO     │◄─────│     Tempo       │                          │    │
│  │   │  (storage)  │      │  (stores traces)│                          │    │
│  │   └─────────────┘      └─────────────────┘                          │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### k6 with xk6-tempo Extension

The k6 tests use the [xk6-tempo](https://github.com/grafana/xk6-tempo) extension which provides:

- **Trace Generation**: `tempo.generateTrace(profile)` creates realistic traces
- **Throughput Calculation**: `tempo.calculateThroughput(profile, bytesPerSec, vus)` converts MB/s to traces/sec
- **OTLP Ingestion**: Native gRPC client for pushing traces
- **TraceQL Queries**: HTTP client for querying traces

```javascript
// How throughput is calculated in k6 scripts
const throughput = tempo.calculateThroughput(
    traceProfile,           // Defines spans/trace complexity
    config.bytesPerSecond,  // Target MB/s converted to bytes
    config.vus.min          // Number of virtual users
);
const tracesPerSecond = Math.ceil(throughput.totalTracesPerSec);
```

## Prerequisites

- Go 1.21+
- Kubernetes/OpenShift cluster access (kubeconfig configured)
- [Tempo Operator](https://github.com/grafana/tempo-operator) installed
- [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator) installed
- k6 with xk6-tempo extension (for standalone k6 tests only)

## Installation

```bash
git clone https://github.com/redhat/perf-tests-tempo.git
cd perf-tests-tempo
go mod download
```

## Quick Start

```bash
# Dry run to see what would be executed
make perf-test-dry-run

# Run all profiles with default settings (combined ingestion + query tests)
make perf-test

# Run LokiStack-style profiles (recommended)
make perf-test PROFILES=1x-extra-small,1x-small

# Run specific legacy profiles
make perf-test PROFILES=small,medium

# Run only ingestion tests
make perf-test TEST_TYPE=ingestion PROFILES=1x-small

# Run only query tests
make perf-test TEST_TYPE=query PROFILES=1x-medium
```

## CLI Runner

The CLI runner (`cmd/perf-runner`) is the main entry point for running performance tests.

### Usage

```bash
go run ./cmd/perf-runner [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--profiles` | (all) | Comma-separated list of profiles to run (e.g., `small,medium`) |
| `--profiles-dir` | `profiles` | Directory containing profile YAML files |
| `--output` | `results` | Output directory for logs and metrics |
| `--test-type` | `combined` | Test type: `ingestion`, `query`, or `combined` |
| `--dry-run` | `false` | Print what would be executed without running |
| `--skip-cleanup` | `false` | Skip cleanup after tests (useful for debugging) |
| `--check-metrics` | `false` | Check and report metric availability after collection |
| `--generate-dashboard` | `true` | Generate HTML dashboard after metrics collection |
| `--collect-logs` | `true` | Collect logs from all components (Tempo, MinIO, OTel, k6) after test |
| `--node-selector` | (none) | Node selector for Tempo pods (e.g., `node-role.kubernetes.io/infra=`) |

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

# Run on infrastructure nodes (node selector)
go run ./cmd/perf-runner --profiles=medium --node-selector="node-role.kubernetes.io/infra="

# Multiple node selectors
go run ./cmd/perf-runner --profiles=large --node-selector="node-role.kubernetes.io/infra=,kubernetes.io/os=linux"

# Check metric availability after test
go run ./cmd/perf-runner --profiles=small --check-metrics
```

## Profile Configuration

Profiles define the test parameters in YAML files located in the `profiles/` directory.

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
  vus:
    min: 10                # Minimum virtual users
    max: 50                # Maximum virtual users
  ingestion:
    mbPerSecond: 1         # Target ingestion rate in MB/s
    traceProfile: medium   # Trace complexity: small, medium, large, xlarge
  query:
    queriesPerSecond: 25   # Target query rate
```

**Note:** Test duration is controlled via the `DURATION` environment variable (default: `5m`).

### Profile Fields Explained

| Field | Description |
|-------|-------------|
| `tempo.variant` | `monolithic` (single pod) or `stack` (distributed components) |
| `tempo.resources` | Optional CPU/memory limits; omit to use operator defaults |
| `k6.vus.min/max` | Virtual user range for k6 executor |
| `k6.ingestion.mbPerSecond` | Target throughput in megabytes per second |
| `k6.ingestion.traceProfile` | Trace complexity affecting spans per trace |
| `k6.query.queriesPerSecond` | TraceQL queries per second |

### Trace Profiles

The `traceProfile` setting controls trace complexity:

| Profile | Spans/Trace | Use Case |
|---------|-------------|----------|
| `small` | 8-15 | Simple microservice calls |
| `medium` | 25-40 | Typical production traces |
| `large` | 50-80 | Complex distributed transactions |
| `xlarge` | 100-150 | Heavy batch processing |

### Built-in Profiles

#### LokiStack-Style Profiles (Recommended)

These profiles follow the [LokiStack sizing conventions](https://docs.redhat.com/en/documentation/red_hat_openshift_logging/6.2/html/configuring_logging/configuring-lokistack-storage) for consistent sizing across observability components:

| Profile | MB/s | GB/day | Queries/s | VUs | Tempo Variant | Description |
|---------|------|--------|-----------|-----|---------------|-------------|
| 1x-demo | 0.05 | ~4 | 2 | 2-5 | monolithic | Demo environment, no HA |
| 1x-extra-small | 1.2 | ~100 | 5 | 5-20 | stack | Small clusters, limited workloads |
| 1x-small | 5.8 | ~500 | 25 | 20-80 | stack | Production, moderate workloads |
| 1x-medium | 23 | ~2000 | 100 | 50-200 | stack | Production, high workloads |

#### Legacy Profiles

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
  resources:
    memory: "16Gi"
    cpu: "2000m"

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

Run it:

```bash
go run ./cmd/perf-runner --profiles=custom
```

## Test Execution Flow

When you run a profile, the following steps execute:

### 1. Create Namespace
Creates an isolated namespace `tempo-perf-{profile-name}` for all resources.

### 2. Check Prerequisites
Verifies that Tempo Operator and OpenTelemetry Operator are installed by checking for their CRDs.

### 3. Deploy MinIO
Deploys MinIO as the object storage backend for Tempo trace data:
- Creates PVC for persistent storage
- Deploys MinIO StatefulSet
- Creates Service and Secret with credentials

### 4. Deploy Tempo
Deploys Tempo using the configured variant:
- **Monolithic**: Single-pod deployment (TempoMonolithic CR)
- **Stack**: Distributed deployment with separate components (TempoStack CR)

### 5. Deploy OTel Collector
Deploys OpenTelemetry Collector configured to:
- Receive traces via OTLP gRPC (port 4317)
- Forward traces to Tempo distributor

### 6. Run k6 Tests
Executes k6 load tests as Kubernetes Jobs:

| Test Type | Jobs Created | Description |
|-----------|--------------|-------------|
| `combined` | 2 parallel jobs | Ingestion + Query run simultaneously |
| `ingestion` | 1 job | Only trace ingestion |
| `query` | 1 job | Only TraceQL queries |

### 7. Save Results
Exports test results to the output directory:
- k6 job logs (stdout with metrics summary)
- Prometheus metrics (CSV format)

### 8. Cleanup
Deletes all resources in reverse order:
- Custom Resources (TempoMonolithic/TempoStack, OpenTelemetryCollector)
- Wait for finalizers
- Cluster-scoped resources (ClusterRoleBindings)
- Namespace and all remaining resources

## Output Files

All output files are saved to the `--output` directory (default: `results/`):

| File | Description |
|------|-------------|
| `{profile}-k6-ingestion.log` | k6 ingestion test output with metrics summary |
| `{profile}-k6-query.log` | k6 query test output with metrics summary |
| `{profile}-k6-ingestion-metrics.json` | Parsed k6 ingestion metrics (JSON) |
| `{profile}-k6-query-metrics.json` | Parsed k6 query metrics (JSON) |
| `{profile}-metrics.csv` | Prometheus metrics collected during test |
| `{profile}-dashboard.html` | Interactive HTML dashboard with charts |

Example output structure:
```
results/
├── small-k6-ingestion.log
├── small-k6-query.log
├── small-k6-ingestion-metrics.json
├── small-k6-query-metrics.json
├── small-metrics.csv
├── small-dashboard.html
├── medium-k6-ingestion.log
├── medium-k6-query.log
├── medium-k6-ingestion-metrics.json
├── medium-k6-query-metrics.json
├── medium-metrics.csv
└── medium-dashboard.html
```

### k6 Log Contents

The k6 logs contain the full test output including:
- Test configuration summary
- Real-time progress (iterations, data transfer)
- Final metrics summary (requests/sec, latency percentiles, error rates)
- Custom xk6-tempo metrics (traces sent, bytes ingested, query latencies)

## Makefile Targets

```bash
make help                            # Show all available targets

# Profile-based tests
make perf-test                       # Run all profiles
make perf-test PROFILES=small,medium # Run specific profiles
make perf-test TEST_TYPE=ingestion   # Run only ingestion tests
make perf-test-dry-run               # Preview without executing
make validate-profiles               # Validate all YAML files

# Standalone k6 tests (requires existing Tempo instance)
make k6-ingestion K6_SIZE=medium     # Run ingestion test
make k6-query K6_SIZE=large          # Run query test
make k6-combined                     # Run combined test

# Development
make test                            # Run unit tests
make check                           # Run format-check + vet
make format                          # Format Go code
make deps                            # Tidy dependencies
```

## Standalone k6 Tests

Run k6 tests directly against an existing Tempo instance (without deploying infrastructure):

```bash
# Set environment variables
export TEMPO_ENDPOINT="tempo-distributor.tempo.svc.cluster.local:4317"
export TEMPO_QUERY_ENDPOINT="http://tempo-query-frontend.tempo.svc.cluster.local:3200"

# Run tests with different sizes
make k6-ingestion K6_SIZE=small
make k6-query K6_SIZE=medium
make k6-combined K6_SIZE=large
```

This requires k6 with the xk6-tempo extension installed locally.

## Framework API

The framework can be used programmatically in Go:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/redhat/perf-tests-tempo/test/framework"
    "github.com/redhat/perf-tests-tempo/test/framework/k6"
)

func main() {
    ctx := context.Background()

    // Create framework with namespace
    fw, err := framework.New(ctx, "my-perf-test")
    if err != nil {
        panic(err)
    }
    defer fw.Cleanup()

    // Check prerequisites
    prereqs, _ := fw.CheckPrerequisites()
    if !prereqs.AllMet {
        fmt.Println(prereqs.String())
        return
    }

    // Deploy infrastructure
    fw.SetupMinIO()
    fw.SetupTempo("stack", nil)  // or "monolithic"
    fw.SetupOTelCollector()

    // Run tests
    testStart := time.Now()
    result, _ := fw.RunK6ParallelTests(&k6.Config{
        MBPerSecond:      1.0,
        QueriesPerSecond: 25,
        Duration:         "5m",
        VUsMin:           10,
        VUsMax:           50,
        TraceProfile:     "medium",
    })

    if result.Success() {
        fmt.Println("Tests passed!")
        // Collect metrics
        fw.CollectMetrics(testStart, "results/metrics.csv")
    }
}
```

### Key Framework Methods

| Method | Description |
|--------|-------------|
| `New(ctx, namespace)` | Create framework instance |
| `CheckPrerequisites()` | Verify operators are installed |
| `SetupMinIO()` | Deploy MinIO storage |
| `SetupTempo(variant, resources)` | Deploy Tempo (monolithic/stack) |
| `SetupOTelCollector()` | Deploy OTel Collector |
| `RunK6Test(type, config)` | Run single k6 test |
| `RunK6ParallelTests(config)` | Run ingestion + query in parallel |
| `CollectMetrics(start, path)` | Export Prometheus metrics |
| `Cleanup()` | Delete all resources |

## Project Structure

```
.
├── cmd/
│   └── perf-runner/           # CLI entry point
│       └── main.go            # Main program, profile execution loop
│
├── profiles/                  # YAML profile configurations
│   ├── 1x-demo.yaml           # LokiStack-style: demo (no HA)
│   ├── 1x-extra-small.yaml    # LokiStack-style: ~100GB/day
│   ├── 1x-small.yaml          # LokiStack-style: ~500GB/day
│   ├── 1x-medium.yaml         # LokiStack-style: ~2TB/day
│   ├── small.yaml             # Legacy: light load
│   ├── medium.yaml            # Legacy: moderate load
│   ├── large.yaml             # Legacy: stress testing
│   └── xlarge.yaml            # Legacy: capacity testing
│
├── framework/                 # Go framework packages
│   ├── framework.go           # Core Framework struct, New()
│   ├── types.go               # Interfaces, ResourceConfig
│   ├── facade.go              # Public API methods
│   ├── prerequisites.go       # Operator verification
│   ├── monitoring.go          # OpenShift user workload monitoring
│   ├── namespace.go           # Namespace lifecycle
│   ├── cleanup.go             # Resource cleanup with finalizers
│   │
│   ├── profile/               # YAML profile loading
│   │   ├── types.go           # Profile struct definitions
│   │   └── loader.go          # Load, validate YAML files
│   │
│   ├── tempo/                 # Tempo deployment
│   │   ├── monolithic.go      # TempoMonolithic CR
│   │   └── stack.go           # TempoStack CR
│   │
│   ├── minio/                 # MinIO deployment
│   │   └── minio.go           # PVC, StatefulSet, Service, Secret
│   │
│   ├── otel/                  # OpenTelemetry Collector
│   │   └── collector.go       # OpenTelemetryCollector CR
│   │
│   ├── k6/                    # k6 test runner
│   │   ├── types.go           # Config, Result, TestType
│   │   └── runner.go          # Job creation, log collection
│   │
│   ├── metrics/               # Metrics collection
│   │   ├── collector.go       # Prometheus queries
│   │   └── exporter.go        # CSV export
│   │
│   └── wait/                  # Wait utilities
│       └── wait.go            # Pod ready, deployment ready
│
├── tests/
│   └── k6/                    # k6 JavaScript test scripts
│       ├── ingestion-test.js  # Trace ingestion test
│       ├── query-test.js      # TraceQL query test
│       ├── combined-test.js   # Both scenarios
│       └── lib/
│           ├── config.js      # Size configurations, env vars
│           └── trace-profiles.js  # Trace complexity profiles
│
├── Makefile                   # Build and test targets
├── go.mod                     # Go module definition
└── README.md                  # This file
```

## Environment Variables

### Framework Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPO_PERF_CR_DELETION_TIMEOUT` | `120s` | Timeout for CR deletion |
| `TEMPO_PERF_POD_READY_TIMEOUT` | `120s` | Timeout for pod readiness |
| `TEMPO_PERF_JOB_TIMEOUT` | `30m` | Timeout for k6 job completion |
| `TEMPO_PERF_MAX_CONCURRENT_QUERIES` | `5` | Prometheus query concurrency |

### k6 Test Configuration

These environment variables control test execution:

| Variable | Default | Description |
|----------|---------|-------------|
| `DURATION` | `5m` | **Test duration** (e.g., `10m`, `1h`) |
| `SIZE` | - | Test size: small, medium, large, xlarge |
| `MB_PER_SECOND` | - | Override ingestion rate |
| `QUERIES_PER_SECOND` | - | Override query rate |
| `VUS_MIN` | - | Override minimum VUs |
| `VUS_MAX` | - | Override maximum VUs |
| `TRACE_PROFILE` | - | Override trace profile |
| `TEMPO_ENDPOINT` | - | OTLP gRPC endpoint |
| `TEMPO_QUERY_ENDPOINT` | - | HTTP query endpoint |

Example:
```bash
# Run a 30-minute test
DURATION=30m go run ./cmd/perf-runner --profiles=1x-small
```

## Advanced Configuration

### Node Selector

Use the `--node-selector` flag to schedule Tempo pods on specific nodes. This is useful for:
- Running on dedicated infrastructure nodes
- Isolating performance tests from other workloads
- Testing on nodes with specific hardware characteristics

```bash
# Schedule on OpenShift infrastructure nodes
go run ./cmd/perf-runner --profiles=medium --node-selector="node-role.kubernetes.io/infra="

# Schedule on nodes with specific labels
go run ./cmd/perf-runner --profiles=large --node-selector="workload=performance,tier=dedicated"
```

The node selector is applied to:
- **TempoMonolithic**: The single Tempo pod
- **TempoStack**: All components (distributor, ingester, querier, compactor, query-frontend, gateway)

**Generator Pod Isolation**: When a node selector is configured, the framework automatically applies anti-affinity to ensure generator pods (MinIO, OTel Collector, k6 jobs) are scheduled on *different* nodes than Tempo. This provides:
- Clean performance isolation between the system under test (Tempo) and load generators
- More accurate performance measurements without resource contention
- Realistic distributed deployment scenarios

### External S3 Storage

By default, the framework deploys MinIO as in-cluster S3 storage. For testing with external AWS S3, use the framework API:

```go
import (
    "github.com/redhat/perf-tests-tempo/test/framework"
)

// Configure external S3 storage
resourceConfig := &framework.ResourceConfig{
    Storage: &framework.StorageConfig{
        Type:            "s3",              // Use external S3
        Bucket:          "my-tempo-bucket",
        Region:          "us-east-2",
        AccessKeyID:     "AKIAXXXXXXXX",
        SecretAccessKey: "your-secret-key",
        // Endpoint: "",                    // Leave empty for AWS S3
    },
    NodeSelector: map[string]string{
        "node-role.kubernetes.io/infra": "",
    },
}

// Deploy Tempo with external S3
fw.SetupTempo("stack", resourceConfig)
```

Storage configuration options:

| Field | Description |
|-------|-------------|
| `Type` | `"minio"` (default, in-cluster) or `"s3"` (external AWS S3) |
| `Bucket` | S3 bucket name |
| `Region` | AWS region (required for AWS S3) |
| `Endpoint` | S3 endpoint URL (required for MinIO, optional for AWS S3) |
| `AccessKeyID` | AWS access key ID |
| `SecretAccessKey` | AWS secret access key |
| `SecretName` | Custom secret name (default: `"minio"` or `"tempo-s3"`) |

## Troubleshooting

### Common Issues

**Prerequisites not met**
```
Error: prerequisites not met: Tempo=false, OTel=false
```
Install the required operators:
- [Tempo Operator](https://github.com/grafana/tempo-operator)
- [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator)

**k6 job timeout**
```
Error: k6 test failed: context deadline exceeded
```
Increase the job timeout:
```bash
export TEMPO_PERF_JOB_TIMEOUT=60m
```

**Cleanup failures**
```
Warning: cleanup failed: ...
```
Use `--skip-cleanup` to preserve resources for debugging:
```bash
go run ./cmd/perf-runner --profiles=small --skip-cleanup
kubectl get all -n tempo-perf-small
```

**No metrics collected**
Ensure user workload monitoring is enabled (OpenShift) or Prometheus is accessible.

## License

Apache License 2.0
