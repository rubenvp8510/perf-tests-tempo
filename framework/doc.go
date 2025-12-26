// Package framework provides a comprehensive testing framework for Tempo performance testing
// on Kubernetes/OpenShift clusters.
//
// The framework automates the deployment, testing, metrics collection, and cleanup of
// Tempo instances, providing a high-level API for performance testing workflows.
//
// # Quick Start
//
// Create a new framework instance and run a basic performance test:
//
//	ctx := context.Background()
//	fw, err := framework.New(ctx, "my-test-namespace")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer fw.Cleanup()
//
//	// Check prerequisites
//	prereqs, _ := fw.CheckPrerequisites()
//	if !prereqs.AllMet {
//	    log.Fatal("Prerequisites not met: ", prereqs.String())
//	}
//
//	// Deploy infrastructure
//	fw.SetupMinIO()
//	fw.SetupTempo("monolithic", &framework.ResourceConfig{Profile: "medium"})
//	fw.SetupOTelCollector()
//
//	// Run load test
//	result, _ := fw.RunK6IngestionTest(k6.SizeMedium)
//	fmt.Printf("Test completed: %v\n", result.Success)
//
// # Context Support
//
// The framework supports context-based cancellation for all operations:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
//	defer cancel()
//
//	fw, err := framework.New(ctx, "test-namespace")
//
// # Resource Configuration
//
// Tempo can be deployed with preset profiles or custom resources:
//
//	// Using preset profile
//	fw.SetupTempo("monolithic", &framework.ResourceConfig{Profile: "large"})
//
//	// Using custom resources
//	fw.SetupTempo("monolithic", &framework.ResourceConfig{
//	    Resources: &corev1.ResourceRequirements{
//	        Limits: corev1.ResourceList{
//	            corev1.ResourceMemory: resource.MustParse("8Gi"),
//	            corev1.ResourceCPU:    resource.MustParse("2"),
//	        },
//	    },
//	})
//
// # Metrics Collection
//
// The framework can collect and export Prometheus metrics:
//
//	testStart := time.Now()
//	// ... run tests ...
//	fw.CollectMetrics(testStart, "results/metrics.csv")
//
//	// Or use duration-based collection
//	fw.CollectMetricsWithDuration(30*time.Minute, "results/metrics.json")
//
// # Package Structure
//
// The framework is organized into subpackages:
//
//   - config: Centralized configuration with environment variable support
//   - concurrent: Concurrent execution helpers for parallel operations
//   - gvr: Centralized GroupVersionResource definitions
//   - k6: k6 load test execution
//   - metrics: Prometheus metrics collection and export
//   - minio: MinIO object storage deployment
//   - otel: OpenTelemetry Collector deployment
//   - retry: Retry logic with exponential backoff
//   - tempo: Tempo deployment (monolithic and stack)
//   - wait: Polling-based readiness checks
package framework
