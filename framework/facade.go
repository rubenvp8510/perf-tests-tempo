package framework

import (
	"fmt"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/k6"
	"github.com/redhat/perf-tests-tempo/test/framework/metrics"
	"github.com/redhat/perf-tests-tempo/test/framework/metrics/dashboard"
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
// resources: optional resource configuration
func (f *Framework) SetupTempo(variant string, resources *ResourceConfig) error {
	// Convert framework.ResourceConfig to tempo.ResourceConfig
	var tempoConfig *tempo.ResourceConfig
	if resources != nil {
		tempoConfig = &tempo.ResourceConfig{
			Profile:           resources.Profile,
			Resources:         resources.Resources,
			ReplicationFactor: resources.ReplicationFactor,
			NodeSelector:      resources.NodeSelector,
		}
		if resources.Overrides != nil {
			tempoConfig.Overrides = &tempo.TempoOverrides{
				MaxTracesPerUser: resources.Overrides.MaxTracesPerUser,
			}
		}
		if resources.Storage != nil {
			tempoConfig.Storage = &tempo.StorageConfig{
				Type:            resources.Storage.Type,
				SecretName:      resources.Storage.SecretName,
				Endpoint:        resources.Storage.Endpoint,
				Bucket:          resources.Storage.Bucket,
				Region:          resources.Storage.Region,
				AccessKeyID:     resources.Storage.AccessKeyID,
				SecretAccessKey: resources.Storage.SecretAccessKey,
				Insecure:        resources.Storage.Insecure,
			}
		}
	}
	return tempo.Setup(f, variant, tempoConfig)
}

// SetupOTelCollector deploys OpenTelemetry Collector with RBAC
// tempoVariant should be "monolithic" or "stack" to configure the correct Tempo gateway endpoint
func (f *Framework) SetupOTelCollector(tempoVariant string) error {
	return otel.SetupCollector(f, tempoVariant)
}

// SetupTempoMonitoring verifies ServiceMonitors and creates PodMonitor fallback if needed
func (f *Framework) SetupTempoMonitoring(variant string) error {
	return tempo.SetupTempoMonitoring(f, variant)
}

// SetupK6PrometheusMetrics enables k6 to export metrics to Prometheus
// Returns the remote write URL to configure in k6.Config.PrometheusRWURL
func (f *Framework) SetupK6PrometheusMetrics() (string, error) {
	url, err := k6.SetupK6PrometheusMetrics(f.ctx, f.client)
	if err != nil {
		return "", fmt.Errorf("failed to setup k6 Prometheus metrics: %w", err)
	}
	return url, nil
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

// ExportK6Metrics exports k6 metrics to a JSON file
func (f *Framework) ExportK6Metrics(k6Metrics *k6.K6Metrics, outputPath string, testType string) error {
	return metrics.ExportK6Metrics(k6Metrics, outputPath, testType)
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

// GenerateDashboard generates an HTML dashboard from a metrics CSV file
func (f *Framework) GenerateDashboard(csvPath, outputPath, profileName string) error {
	config := dashboard.DashboardConfig{
		Title:       "Tempo Performance Test Report",
		ProfileName: profileName,
		TestType:    "combined",
		GeneratedAt: time.Now(),
	}
	return dashboard.Generate(csvPath, outputPath, config)
}

// CheckMetricAvailability checks which metrics are available in Prometheus
func (f *Framework) CheckMetricAvailability(duration time.Duration) (*metrics.AvailabilityReport, error) {
	return metrics.CheckMetricAvailability(f, duration)
}

// PrintMetricAvailabilityReport prints a human-readable availability report
func (f *Framework) PrintMetricAvailabilityReport(report *metrics.AvailabilityReport) {
	metrics.PrintAvailabilityReport(report)
}

// DiagnoseMetricIssues provides diagnostic information about missing metrics
func (f *Framework) DiagnoseMetricIssues(report *metrics.AvailabilityReport) []string {
	return metrics.DiagnoseMetricIssues(report)
}
