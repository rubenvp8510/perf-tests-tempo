package framework

import (
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/k6"
	"github.com/redhat/perf-tests-tempo/test/framework/metrics"
	"github.com/redhat/perf-tests-tempo/test/framework/minio"
	"github.com/redhat/perf-tests-tempo/test/framework/otel"
	"github.com/redhat/perf-tests-tempo/test/framework/tempo"
	"github.com/redhat/perf-tests-tempo/test/framework/wait"

	"k8s.io/apimachinery/pkg/labels"
)

// SetupMinIO deploys MinIO with PVC and waits for it to be ready
func (f *Framework) SetupMinIO() error {
	if err := f.EnsureNamespace(); err != nil {
		return err
	}
	return minio.Setup(f)
}

// SetupTempo deploys Tempo (monolithic or stack) with optional resource configuration
// variant: "monolithic" or "stack"
// resources: optional resource configuration (only applies to monolithic)
func (f *Framework) SetupTempo(variant string, resources *ResourceConfig) error {
	// ResourceConfig types are structurally identical, cast directly
	return tempo.Setup(f, variant, (*tempo.ResourceConfig)(resources))
}

// SetupOTelCollector deploys OpenTelemetry Collector with RBAC
func (f *Framework) SetupOTelCollector() error {
	return otel.SetupCollector(f)
}

// RunK6Test deploys and runs a k6 test as a Kubernetes Job
func (f *Framework) RunK6Test(testType k6.TestType, config *k6.Config) (*k6.Result, error) {
	return k6.RunTest(f, testType, config)
}

// RunK6IngestionTest runs the ingestion performance test
func (f *Framework) RunK6IngestionTest(size k6.Size) (*k6.Result, error) {
	return k6.RunIngestionTest(f, size)
}

// RunK6QueryTest runs the query performance test
func (f *Framework) RunK6QueryTest(size k6.Size) (*k6.Result, error) {
	return k6.RunQueryTest(f, size)
}

// RunK6CombinedTest runs the combined ingestion+query performance test
func (f *Framework) RunK6CombinedTest(size k6.Size) (*k6.Result, error) {
	return k6.RunCombinedTest(f, size)
}

// RunK6ParallelTests runs ingestion and query tests as separate parallel Kubernetes Jobs
func (f *Framework) RunK6ParallelTests(config *k6.Config) (*k6.ParallelResult, error) {
	return k6.RunParallelTests(f, config)
}

// CollectMetrics collects performance metrics for the test namespace and exports to CSV
func (f *Framework) CollectMetrics(testStart time.Time, outputPath string) error {
	return metrics.CollectMetrics(f, testStart, outputPath)
}

// CollectMetricsWithDuration collects metrics for a specific duration (counting back from now)
func (f *Framework) CollectMetricsWithDuration(duration time.Duration, outputPath string) error {
	return metrics.CollectMetricsWithDuration(f, duration, outputPath)
}

// WaitForPodsReady waits for pods matching the selector to be ready
func (f *Framework) WaitForPodsReady(selector labels.Selector, timeout time.Duration, minReady int) error {
	return wait.ForPodsReady(f, selector, timeout, minReady)
}

// WaitForDeploymentReady waits for a deployment to be ready
func (f *Framework) WaitForDeploymentReady(name string, timeout time.Duration) error {
	return wait.ForDeploymentReady(f, name, timeout)
}

// WaitForPodsTerminated waits for pods matching the selector to be fully terminated
func (f *Framework) WaitForPodsTerminated(selector labels.Selector, timeout time.Duration) error {
	return wait.ForPodsTerminated(f, selector, timeout)
}

// WaitForTempoPodsReady waits for Tempo pods using multiple label selectors
func (f *Framework) WaitForTempoPodsReady(timeout time.Duration) error {
	return wait.ForTempoPodsReady(f, timeout)
}
