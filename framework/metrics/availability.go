package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// MetricAvailability represents the availability status of a metric
type MetricAvailability struct {
	QueryID     string
	Name        string
	Category    string
	Available   bool
	SeriesCount int
	Error       string
}

// AvailabilityReport summarizes metric availability
type AvailabilityReport struct {
	TotalMetrics     int
	AvailableMetrics int
	MissingMetrics   int
	Metrics          []MetricAvailability
	ByCategory       map[string]CategoryAvailability
}

// CategoryAvailability summarizes availability for a category
type CategoryAvailability struct {
	Total     int
	Available int
	Missing   int
}

// CheckMetricAvailability checks which metrics are available in Prometheus
func CheckMetricAvailability(np NamespaceProvider, duration time.Duration) (*AvailabilityReport, error) {
	ctx := context.Background()
	namespace := np.Namespace()

	// Get KubeConfig
	var kubeConfig *rest.Config
	if cp, ok := np.(ConfigProvider); ok {
		kubeConfig = cp.Config()
	} else {
		var err error
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
			if err != nil {
				return nil, fmt.Errorf("failed to get kube config: %w", err)
			}
		}
	}

	// Create client
	config := &ClientConfig{
		Namespace:           namespace,
		AutoDiscover:        true,
		MonitoringNamespace: "openshift-monitoring",
		ServiceAccountName:  "prometheus-k8s",
		KubeConfig:          kubeConfig,
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	// Get all queries
	queries := GetAllQueries(namespace)

	// Calculate time range
	end := time.Now()
	start := end.Add(-duration)

	report := &AvailabilityReport{
		TotalMetrics: len(queries),
		Metrics:      make([]MetricAvailability, 0, len(queries)),
		ByCategory:   make(map[string]CategoryAvailability),
	}

	fmt.Println("\n📊 Checking metric availability...")
	fmt.Printf("   Time range: %s to %s\n\n", start.Format("15:04:05"), end.Format("15:04:05"))

	for _, query := range queries {
		avail := MetricAvailability{
			QueryID:  query.ID,
			Name:     query.Name,
			Category: query.Category,
		}

		// Execute query to check if data exists
		result, err := client.QueryRange(ctx, query.Query, start, end, 60*time.Second)
		if err != nil {
			avail.Error = err.Error()
		} else if len(result.Data.Result) == 0 {
			avail.Error = "no data"
		} else {
			avail.Available = true
			avail.SeriesCount = len(result.Data.Result)
			report.AvailableMetrics++
		}

		if !avail.Available {
			report.MissingMetrics++
		}

		report.Metrics = append(report.Metrics, avail)

		// Update category stats
		cat := report.ByCategory[query.Category]
		cat.Total++
		if avail.Available {
			cat.Available++
		} else {
			cat.Missing++
		}
		report.ByCategory[query.Category] = cat
	}

	return report, nil
}

// PrintAvailabilityReport prints a human-readable availability report
func PrintAvailabilityReport(report *AvailabilityReport) {
	separator := strings.Repeat("=", 60)

	fmt.Println("\n" + separator)
	fmt.Println("METRIC AVAILABILITY REPORT")
	fmt.Println(separator)

	// Summary
	fmt.Printf("\nSummary: %d/%d metrics available (%.1f%%)\n",
		report.AvailableMetrics, report.TotalMetrics,
		float64(report.AvailableMetrics)/float64(report.TotalMetrics)*100)

	// By category
	fmt.Println("\nBy Category:")
	categoryOrder := []string{
		"ingestion", "compactor", "storage", "cache",
		"resources", "query_performance", "querier",
	}

	for _, cat := range categoryOrder {
		if stats, ok := report.ByCategory[cat]; ok {
			status := "✅"
			if stats.Available == 0 {
				status = "❌"
			} else if stats.Available < stats.Total {
				status = "⚠️"
			}
			fmt.Printf("  %s %-20s %d/%d available\n", status, cat, stats.Available, stats.Total)
		}
	}

	// Missing metrics details
	if report.MissingMetrics > 0 {
		fmt.Println("\nMissing Metrics:")
		for _, m := range report.Metrics {
			if !m.Available {
				fmt.Printf("  ❌ %s (%s): %s\n", m.Name, m.Category, m.Error)
			}
		}
	}

	// Available metrics details
	if report.AvailableMetrics > 0 {
		fmt.Println("\nAvailable Metrics:")
		for _, m := range report.Metrics {
			if m.Available {
				fmt.Printf("  ✅ %s (%s): %d series\n", m.Name, m.Category, m.SeriesCount)
			}
		}
	}

	fmt.Println()
}

// DiagnoseMetricIssues provides diagnostic information about why metrics might be missing
func DiagnoseMetricIssues(report *AvailabilityReport) []string {
	var issues []string

	// Check for common issues based on patterns

	// Issue 1: All Tempo metrics missing (ServiceMonitor issue)
	tempoMetricsAvailable := false
	for _, m := range report.Metrics {
		if m.Category == "ingestion" || m.Category == "compactor" || m.Category == "storage" || m.Category == "cache" || m.Category == "querier" {
			if m.Available {
				tempoMetricsAvailable = true
				break
			}
		}
	}
	if !tempoMetricsAvailable {
		issues = append(issues, "All Tempo-internal metrics are missing. Possible causes:\n"+
			"  - ServiceMonitor not created by Tempo operator\n"+
			"  - User workload monitoring not enabled\n"+
			"  - Prometheus not scraping Tempo pods\n"+
			"  Solution: Ensure user workload monitoring is enabled and check for PodMonitor/ServiceMonitor")
	}

	// Issue 2: Resource metrics missing (kubelet/cadvisor issue)
	resourcesAvailable := false
	for _, m := range report.Metrics {
		if m.Category == "resources" && m.Available {
			resourcesAvailable = true
			break
		}
	}
	if !resourcesAvailable {
		issues = append(issues, "Resource metrics (CPU/memory) are missing. Possible causes:\n"+
			"  - Cluster monitoring not functioning\n"+
			"  - Thanos querier not accessible\n"+
			"  Solution: Check openshift-monitoring namespace and Thanos route")
	}

	// Issue 3: k6 metrics missing
	k6MetricsAvailable := false
	for _, m := range report.Metrics {
		if m.Category == "query_performance" || m.Category == "query_latency" {
			if m.Available {
				k6MetricsAvailable = true
				break
			}
		}
	}
	if !k6MetricsAvailable {
		issues = append(issues, "k6 custom metrics are missing. Possible causes:\n"+
			"  - k6 not configured to export metrics to Prometheus\n"+
			"  - Prometheus remote write receiver not enabled\n"+
			"  Solution: Enable k6 Prometheus remote write with K6_PROMETHEUS_RW_SERVER_URL")
	}

	return issues
}
