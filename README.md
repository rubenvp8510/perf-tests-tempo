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
- Kubernetes/OpenShift cluster access
- Tempo Operator installed
- OpenTelemetry Operator installed

## Installation

```bash
go mod download
```

## Quick Start

```go
package mytest

import (
    "github.com/redhat/perf-tests-tempo/test/framework"
    "github.com/redhat/perf-tests-tempo/test/framework/k6"
)

func Example() {
    // Create framework with explicit namespace
    fw, err := framework.New("my-perf-test")
    if err != nil {
        panic(err)
    }
    defer fw.Cleanup()

    // Deploy infrastructure
    fw.SetupMinIO()
    fw.SetupTempo("monolithic", &framework.ResourceConfig{Profile: "medium"})
    fw.SetupOTelCollector()

    // Run load test
    result, err := fw.RunK6Test(k6.TestIngestion, &k6.Config{Size: k6.SizeMedium})
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Output)
}
```

## Framework API

### Creating a Framework

```go
// Create with explicit namespace (required)
fw, err := framework.New("my-namespace")
```

### Deploying Components

```go
// Deploy MinIO storage backend
fw.SetupMinIO()

// Deploy Tempo with preset profile
fw.SetupTempo("monolithic", &framework.ResourceConfig{Profile: "medium"})

// Deploy Tempo with custom resources
fw.SetupTempo("monolithic", &framework.ResourceConfig{
    Resources: &corev1.ResourceRequirements{
        Limits: corev1.ResourceList{
            corev1.ResourceMemory: resource.MustParse("8Gi"),
            corev1.ResourceCPU:    resource.MustParse("1000m"),
        },
    },
})

// Deploy Tempo Stack variant
fw.SetupTempo("stack", nil)

// Deploy OpenTelemetry Collector
fw.SetupOTelCollector()
```

### Resource Profiles

| Profile | Memory | CPU |
|---------|--------|-----|
| small   | 4Gi    | 500m |
| medium  | 8Gi    | 1000m |
| large   | 12Gi   | 1500m |

### Running Load Tests

```go
import "github.com/redhat/perf-tests-tempo/test/framework/k6"

// Run ingestion test
result, err := fw.RunK6Test(k6.TestIngestion, &k6.Config{
    Size: k6.SizeMedium,
})

// Convenience methods
fw.RunK6IngestionTest(k6.SizeMedium)
fw.RunK6QueryTest(k6.SizeLarge)
fw.RunK6CombinedTest(k6.SizeSmall)
```

### Collecting Metrics

```go
import "time"

testStart := time.Now()

// ... run your test ...

// Collect metrics from test start to now
err := fw.CollectMetrics(testStart, "results/metrics.csv")

// Or collect last N minutes
err := fw.CollectMetricsWithDuration(30*time.Minute, "results/metrics.csv")
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
├── facade.go         # Backward-compatible API methods
├── namespace.go      # Namespace lifecycle
├── cleanup.go        # Resource cleanup
├── tempo/            # Tempo deployment
├── minio/            # MinIO storage backend
├── otel/             # OpenTelemetry Collector
├── k6/               # Load testing
├── wait/             # Readiness utilities
└── metrics/          # Metrics collection
```

## Using with Ginkgo

```go
var _ = Describe("Performance Test", func() {
    var fw *framework.Framework

    BeforeEach(func() {
        var err error
        fw, err = framework.New("tempo-perf-test")
        Expect(err).NotTo(HaveOccurred())

        Expect(fw.SetupMinIO()).To(Succeed())
        Expect(fw.SetupTempo("monolithic", &framework.ResourceConfig{
            Profile: "medium",
        })).To(Succeed())
        Expect(fw.SetupOTelCollector()).To(Succeed())
    })

    It("should handle load", func() {
        result, err := fw.RunK6IngestionTest(k6.SizeMedium)
        Expect(err).NotTo(HaveOccurred())
        Expect(result.Success).To(BeTrue())
    })

    AfterEach(func() {
        if fw != nil {
            Expect(fw.Cleanup()).To(Succeed())
        }
    })
})
```

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run examples
go test -v ./examples/...
```

## Examples

See [`examples/`](examples/) for complete examples:
- `basic_perf_test.go` - Basic deployment and resource configuration
- `metrics_export_test.go` - Metrics collection examples
