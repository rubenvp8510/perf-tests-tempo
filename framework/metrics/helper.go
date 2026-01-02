package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/k6"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NamespaceProvider provides the namespace for metrics collection
type NamespaceProvider interface {
	Namespace() string
}

// ConfigProvider optionally provides a REST config for auto-discovery
type ConfigProvider interface {
	Config() *rest.Config
}

// CollectMetrics collects performance metrics for the test namespace and exports to CSV
// This should be called at the end of your test, before cleanup
//
// Example:
//
//	testStart := time.Now()
//	// ... run your test ...
//	err := metrics.CollectMetrics(fw, testStart, "results/my-test.csv")
func CollectMetrics(np NamespaceProvider, testStart time.Time, outputPath string) error {
	ctx := context.Background()
	namespace := np.Namespace()

	// Calculate duration
	duration := time.Since(testStart)

	fmt.Printf("\n📊 Collecting metrics for namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Second))
	fmt.Printf("   Output: %s\n\n", outputPath)

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get KubeConfig - try interface first, then fall back to discovery
	var kubeConfig *rest.Config
	if cp, ok := np.(ConfigProvider); ok {
		kubeConfig = cp.Config()
	} else {
		// Fall back to standard config discovery
		var err error
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
			if err != nil {
				return fmt.Errorf("failed to get kube config: %w", err)
			}
		}
	}

	// Create metrics client with auto-discovery
	config := &ClientConfig{
		Namespace:           namespace,
		AutoDiscover:        true,
		MonitoringNamespace: "openshift-monitoring",
		ServiceAccountName:  "prometheus-k8s",
		KubeConfig:          kubeConfig,
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create metrics client: %w", err)
	}

	// Collect all metrics from test start to now
	endTime := time.Now()
	results, err := client.CollectAllMetrics(ctx, testStart, endTime)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Collect summary metrics (P99/max/avg over full test duration)
	summaryResults, err := client.CollectSummaryMetrics(ctx, endTime)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to collect summary metrics: %v\n", err)
		// Continue without summary metrics
	}

	// Export to CSV
	exporter := NewCSVExporter(outputPath)
	if err := exporter.Export(results); err != nil {
		return fmt.Errorf("failed to export metrics: %w", err)
	}

	// Export summary metrics to JSON
	if len(summaryResults) > 0 {
		summaryPath := outputPath[:len(outputPath)-len(filepath.Ext(outputPath))] + "-summary.json"
		if err := exportSummaryMetrics(summaryResults, summaryPath); err != nil {
			fmt.Printf("⚠️  Warning: failed to export summary metrics: %v\n", err)
		} else {
			fmt.Printf("📊 Summary metrics exported to %s\n", summaryPath)
		}
	}

	fmt.Printf("✅ Metrics collection complete: %d data series exported\n\n", len(results))
	return nil
}

// SummaryMetricsExport represents the JSON export of summary metrics
type SummaryMetricsExport struct {
	ExportedAt string               `json:"exported_at"`
	Duration   string               `json:"duration"`
	Metrics    []SummaryMetricValue `json:"metrics"`
}

// SummaryMetricValue represents a single summary metric value
type SummaryMetricValue struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// exportSummaryMetrics exports summary metrics to a JSON file
func exportSummaryMetrics(results []MetricResult, outputPath string) error {
	duration := os.Getenv("DURATION")
	if duration == "" {
		duration = "5m"
	}

	export := SummaryMetricsExport{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Duration:   duration,
		Metrics:    make([]SummaryMetricValue, 0, len(results)),
	}

	for _, result := range results {
		if result.Error != nil || len(result.DataPoints) == 0 {
			continue
		}

		export.Metrics = append(export.Metrics, SummaryMetricValue{
			Name:        result.MetricName,
			Description: result.Description,
			Value:       result.DataPoints[0].Value,
			Labels:      result.Labels,
		})
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode summary metrics: %w", err)
	}

	return nil
}

// CollectMetricsWithDuration collects metrics for a specific duration (counting back from now)
// Useful if you don't have the exact start time
//
// Example:
//
//	err := metrics.CollectMetricsWithDuration(fw, 30*time.Minute, "results/my-test.csv")
func CollectMetricsWithDuration(np NamespaceProvider, duration time.Duration, outputPath string) error {
	testStart := time.Now().Add(-duration)
	return CollectMetrics(np, testStart, outputPath)
}

// K6MetricsExport is the JSON structure for k6 metrics export
type K6MetricsExport struct {
	ExportedAt string `json:"exported_at"`
	TestType   string `json:"test_type,omitempty"`

	// Query metrics
	QueryRequestsTotal   float64         `json:"query_requests_total,omitempty"`
	QueryFailuresTotal   float64         `json:"query_failures_total,omitempty"`
	QuerySpansReturned   *k6.MetricStats `json:"query_spans_returned,omitempty"`
	QueryDurationSeconds *k6.MetricStats `json:"query_duration_seconds,omitempty"`

	// Ingestion metrics
	IngestionBytesTotal  float64         `json:"ingestion_bytes_total,omitempty"`
	IngestionTracesTotal float64         `json:"ingestion_traces_total,omitempty"`
	IngestionRateBPS     float64         `json:"ingestion_rate_bps,omitempty"`
	IngestionDuration    *k6.MetricStats `json:"ingestion_duration,omitempty"`
}

// ExportK6Metrics exports k6 metrics to a JSON file
func ExportK6Metrics(metrics *k6.K6Metrics, outputPath string, testType string) error {
	if metrics == nil {
		return nil // Nothing to export
	}

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	export := K6MetricsExport{
		ExportedAt:           time.Now().UTC().Format(time.RFC3339),
		TestType:             testType,
		QueryRequestsTotal:   metrics.QueryRequestsTotal,
		QueryFailuresTotal:   metrics.QueryFailuresTotal,
		IngestionBytesTotal:  metrics.IngestionBytesTotal,
		IngestionTracesTotal: metrics.IngestionTracesTotal,
		IngestionRateBPS:     metrics.IngestionRateBPS,
	}

	// Only include non-empty stats
	if metrics.QuerySpansReturned.Avg > 0 || metrics.QuerySpansReturned.Max > 0 {
		export.QuerySpansReturned = &metrics.QuerySpansReturned
	}
	if metrics.QueryDurationSeconds.Avg > 0 || metrics.QueryDurationSeconds.Max > 0 {
		export.QueryDurationSeconds = &metrics.QueryDurationSeconds
	}
	if metrics.IngestionDuration.Avg > 0 || metrics.IngestionDuration.Max > 0 {
		export.IngestionDuration = &metrics.IngestionDuration
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode k6 metrics: %w", err)
	}

	fmt.Printf("📊 Exported k6 metrics to %s\n", outputPath)
	return nil
}
