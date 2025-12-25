package metrics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NamespaceProvider provides the namespace for metrics collection
type NamespaceProvider interface {
	Namespace() string
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

	// Create metrics client with auto-discovery
	config := &ClientConfig{
		Namespace:           namespace,
		AutoDiscover:        true,
		MonitoringNamespace: "openshift-monitoring",
		ServiceAccountName:  "monitoring-sa",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create metrics client: %w", err)
	}

	// Collect all metrics from test start to now
	results, err := client.CollectAllMetrics(ctx, testStart, time.Now())
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Export to CSV
	exporter := NewCSVExporter(outputPath)
	if err := exporter.Export(results); err != nil {
		return fmt.Errorf("failed to export metrics: %w", err)
	}

	fmt.Printf("✅ Metrics collection complete: %d data series exported\n\n", len(results))
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
