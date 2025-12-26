# Tempo Performance Test Framework

A Go framework for performance testing Tempo on Kubernetes/OpenShift clusters.

## Features

- **Automated Deployment**: Deploy MinIO, Tempo (monolithic or stack), and OpenTelemetry Collector
- **Resource Configuration**: Configure Tempo resources using preset profiles (small/medium/large) or custom specs
- **Load Testing**: Integrated k6 load testing with configurable test profiles
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

## Running Tests

### Using Make (Recommended)

```bash
# Show all available targets
make help

# Run all Go tests
make test

# Run example tests only
make test-examples

# Run with race detector
make test-race
```

### Using Ginkgo CLI

```bash
# Install ginkgo
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Run all tests
ginkgo -v ./...

# Run specific test file
ginkgo -v --focus "medium profile" ./examples/...

# Run with parallel execution
ginkgo -v -p ./examples/...
```

### Standalone k6 Tests

Run k6 load tests directly against an existing Tempo instance:

```bash
# Run individual tests (SIZE: small|medium|large|xlarge)
make k6-ingestion              # Trace ingestion test
make k6-query                  # Query test
make k6-combined               # Combined test

# Run with different size
make k6-ingestion K6_SIZE=large

# Run all k6 tests
make k6-all
```

## Creating New Tests

### 1. Create a Test File

Create a new file in `examples/` or your test directory:

```go
// examples/my_perf_test.go
package examples

import (
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/redhat/perf-tests-tempo/test/framework"
    "github.com/redhat/perf-tests-tempo/test/framework/k6"
)

var _ = Describe("My Performance Test", func() {
    var (
        fw        *framework.Framework
        testStart time.Time
    )

    BeforeEach(func() {
        var err error
        // Create framework with a unique namespace
        fw, err = framework.New("my-perf-test")
        Expect(err).NotTo(HaveOccurred())

        // Check prerequisites (Tempo and OpenTelemetry operators)
        prereqs, err := fw.CheckPrerequisites()
        Expect(err).NotTo(HaveOccurred())
        Expect(prereqs.AllMet).To(BeTrue(), prereqs.String())

        // Enable user workload monitoring (OpenShift)
        Expect(fw.EnableUserWorkloadMonitoring()).To(Succeed())

        // Deploy infrastructure
        Expect(fw.SetupMinIO()).To(Succeed())
        Expect(fw.SetupTempo("monolithic", &framework.ResourceConfig{
            Profile: "medium",
        })).To(Succeed())
        Expect(fw.SetupOTelCollector()).To(Succeed())

        testStart = time.Now()
    })

    AfterEach(func() {
        if fw != nil {
            // Optional: collect metrics before cleanup
            _ = fw.CollectMetrics(testStart, "results/my-test-metrics.csv")

            // Always cleanup resources
            Expect(fw.Cleanup()).To(Succeed())
        }
    })

    It("should handle my workload", func() {
        result, err := fw.RunK6IngestionTest(k6.SizeMedium)
        Expect(err).NotTo(HaveOccurred())
        Expect(result.Success).To(BeTrue())
    })
})
```

### 2. Test Structure

A typical test follows this pattern:

```
BeforeEach:
  1. Create Framework with namespace
  2. Check prerequisites (operators installed)
  3. Enable user workload monitoring (OpenShift)
  4. Deploy MinIO (storage backend)
  5. Deploy Tempo (monolithic or stack)
  6. Deploy OpenTelemetry Collector
  7. Record test start time

It (test case):
  1. Run load test (k6 ingestion/query/combined)
  2. Assert results

AfterEach:
  1. Collect metrics (optional)
  2. Cleanup all resources
```

### 3. Resource Configuration Options

**Using preset profiles:**

```go
fw.SetupTempo("monolithic", &framework.ResourceConfig{
    Profile: "small",   // 4Gi memory, 500m CPU
    Profile: "medium",  // 8Gi memory, 1000m CPU
    Profile: "large",   // 12Gi memory, 1500m CPU
})
```

**Using custom resources:**

```go
import (
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

fw.SetupTempo("monolithic", &framework.ResourceConfig{
    Resources: &corev1.ResourceRequirements{
        Limits: corev1.ResourceList{
            corev1.ResourceMemory: resource.MustParse("16Gi"),
            corev1.ResourceCPU:    resource.MustParse("2000m"),
        },
        Requests: corev1.ResourceList{
            corev1.ResourceMemory: resource.MustParse("16Gi"),
            corev1.ResourceCPU:    resource.MustParse("2000m"),
        },
    },
})
```

**Using Tempo Stack (distributed deployment):**

```go
fw.SetupTempo("stack", nil)
```

### 4. Load Test Sizes

| Size   | Description |
|--------|-------------|
| small  | Light load for quick validation |
| medium | Moderate load for standard testing |
| large  | Heavy load for stress testing |
| xlarge | Maximum load for limit testing |

### 5. Collecting Metrics

```go
// Collect from specific start time
err := fw.CollectMetrics(testStart, "results/metrics.csv")

// Collect last N minutes
err := fw.CollectMetricsWithDuration(30*time.Minute, "results/metrics.csv")
```

## Framework API Reference

### Creating a Framework

```go
// Namespace is required
fw, err := framework.New("my-namespace")
```

### Checking Prerequisites

```go
// Check if Tempo and OpenTelemetry operators are installed
prereqs, err := fw.CheckPrerequisites()
if !prereqs.AllMet {
    fmt.Println(prereqs.String())
    // Handle missing operators
}

// Check individual operators
if !prereqs.TempoOperator.Installed {
    fmt.Println("Tempo Operator missing:", prereqs.TempoOperator.Message)
}
if !prereqs.OpenTelemetryOperator.Installed {
    fmt.Println("OpenTelemetry Operator missing:", prereqs.OpenTelemetryOperator.Message)
}
```

### Enabling User Workload Monitoring (OpenShift)

```go
// Enable user workload monitoring for metrics collection
err := fw.EnableUserWorkloadMonitoring()

// Check if already enabled
enabled, err := fw.IsUserWorkloadMonitoringEnabled()
```

### Deploying Components

```go
fw.SetupMinIO()                                    // Storage backend
fw.SetupTempo("monolithic", &framework.ResourceConfig{...})  // Tempo
fw.SetupTempo("stack", nil)                        // Tempo Stack variant
fw.SetupOTelCollector()                            // OTel Collector
```

### Running Load Tests

```go
// Generic method
result, err := fw.RunK6Test(testType, config)

// Convenience methods
fw.RunK6IngestionTest(k6.SizeMedium)
fw.RunK6QueryTest(k6.SizeLarge)
fw.RunK6CombinedTest(k6.SizeSmall)
```

### Cleanup

```go
// Cleanup all resources (CRs, RBAC, PVs, namespace)
err := fw.Cleanup()
```

## Package Structure

```
framework/
├── framework.go      # Core Framework, New()
├── types.go          # Interfaces, TrackedResource, ResourceConfig
├── facade.go         # API methods (SetupMinIO, SetupTempo, etc.)
├── prerequisites.go  # CheckPrerequisites() - operator verification
├── monitoring.go     # EnableUserWorkloadMonitoring() - OpenShift monitoring
├── namespace.go      # Namespace lifecycle
├── cleanup.go        # Resource cleanup with finalizer handling
├── tempo/            # Tempo deployment (monolithic + stack)
├── minio/            # MinIO storage backend
├── otel/             # OpenTelemetry Collector
├── k6/               # Load testing runner
├── wait/             # Readiness utilities
└── metrics/          # Metrics collection and export

tests/
└── k6/               # Standalone k6 test scripts
    ├── ingestion-test.js
    ├── query-test.js
    ├── combined-test.js
    └── lib/          # Shared k6 utilities

examples/             # Example Ginkgo tests
├── basic_perf_test.go
└── metrics_export_test.go
```

## Examples

See [`examples/`](examples/) for complete examples:

- `basic_perf_test.go` - Basic deployment with different resource configurations
- `metrics_export_test.go` - Metrics collection examples
