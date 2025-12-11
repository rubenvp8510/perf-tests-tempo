# Tempo Performance Test Framework

A Ginkgo-based performance test framework for Tempo that provides structured deployment, testing, and cleanup capabilities.

## Features

- **Automated Deployment**: Deploy MinIO, Tempo (monolithic or stack), and OpenTelemetry Collector with proper readiness checks
- **Resource Configuration**: Configure Tempo resources using preset profiles (small/medium/large) or custom specifications
- **Comprehensive Cleanup**: Automatically clean up all resources including PVs/PVCs
- **Namespace Isolation**: Each test run uses a unique namespace for complete isolation
- **Ginkgo Integration**: Full integration with Ginkgo BDD testing framework

## Prerequisites

- Go 1.21 or later
- Kubernetes cluster access (kubeconfig or in-cluster)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Required CRDs:
  - `tempomonolithics.tempo.grafana.com`
  - `tempostacks.tempo.grafana.com`
  - `opentelemetrycollectors.opentelemetry.io`

## Installation

```bash
cd test
go mod download
```

## Usage

### Basic Example

```go
var _ = Describe("Performance Test", func() {
    var fw *framework.Framework

    BeforeEach(func() {
        var err error
        fw, err = framework.New()
        Expect(err).NotTo(HaveOccurred())

        // Deploy MinIO
        err = fw.SetupMinIO()
        Expect(err).NotTo(HaveOccurred())

        // Deploy Tempo with medium resources
        resourceConfig := &framework.ResourceConfig{
            Profile: "medium",
        }
        err = fw.SetupTempo("monolithic", resourceConfig)
        Expect(err).NotTo(HaveOccurred())

        // Deploy OpenTelemetry Collector
        err = fw.SetupOTelCollector()
        Expect(err).NotTo(HaveOccurred())
    })

    It("should handle medium load", func() {
        // Your performance test here
    })

    AfterEach(func() {
        if fw != nil {
            err := fw.Cleanup()
            Expect(err).NotTo(HaveOccurred())
        }
    })
})
```

### Resource Configuration

#### Using Preset Profiles

```go
resourceConfig := &framework.ResourceConfig{
    Profile: "small", // or "medium", "large"
}
fw.SetupTempo("monolithic", resourceConfig)
```

Preset profiles:
- **small**: 4Gi memory, 500m CPU
- **medium**: 8Gi memory, 1000m CPU
- **large**: 12Gi memory, 1500m CPU

#### Using Custom Resources

```go
resourceConfig := &framework.ResourceConfig{
    Resources: &corev1.ResourceRequirements{
        Limits: corev1.ResourceList{
            corev1.ResourceMemory: resource.MustParse("6Gi"),
            corev1.ResourceCPU:    resource.MustParse("750m"),
        },
        Requests: corev1.ResourceList{
            corev1.ResourceMemory: resource.MustParse("6Gi"),
            corev1.ResourceCPU:    resource.MustParse("750m"),
        },
    },
}
fw.SetupTempo("monolithic", resourceConfig)
```

#### Using Default Resources

```go
fw.SetupTempo("monolithic", nil) // Uses base deployment resources
```

### Tempo Variants

#### Monolithic

```go
fw.SetupTempo("monolithic", resourceConfig)
```

#### Stack

```go
fw.SetupTempo("stack", nil) // Resources not supported for stack
```

## Framework API

### Framework Methods

- `New()` - Create a new framework instance with unique namespace
- `NewWithNamespace(namespace)` - Create framework with specific namespace
- `GetNamespace()` - Get the namespace used by this framework
- `SetupMinIO()` - Deploy MinIO with PVC and wait for readiness
- `SetupTempo(variant, resources)` - Deploy Tempo (monolithic or stack) with optional resources
- `SetupOTelCollector()` - Deploy OpenTelemetry Collector with RBAC
- `Cleanup()` - Remove all resources including PVs/PVCs

### Resource Configuration

- `ResourceConfig.Profile` - Preset profile name ("small", "medium", "large")
- `ResourceConfig.Resources` - Custom resource requirements

## Running Tests

```bash
cd test
go test -v ./...
```

Or run specific test:

```bash
go test -v ./examples/...
```

## Cleanup

The framework automatically handles cleanup of:
- Deployments (MinIO, OTel Collector, query generators)
- Jobs (trace generators)
- CRs (TempoMonolithic, TempoStack, OpenTelemetryCollector)
- RBAC resources (ServiceAccounts, Roles, RoleBindings, ClusterRoles, ClusterRoleBindings)
- Services
- Secrets
- PVCs (with finalizer handling)
- PVs (Released/Available state)
- Namespace

## Examples

See [`examples/basic_perf_test.go`](examples/basic_perf_test.go) for complete examples including:
- Basic performance test with preset resources
- Custom resource configuration
- Tempo Stack deployment
- Default resource usage

## Notes

- Resource configuration only applies to Tempo Monolithic, not Tempo Stack
- Each test run creates a unique namespace for isolation
- Cleanup waits for all pods to terminate before deleting PVCs
- The framework handles stuck PVC finalizers automatically
- Orphaned PVs are automatically cleaned up

