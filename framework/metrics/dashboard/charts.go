package dashboard

// CategoryChartConfig defines chart configuration for a category
type CategoryChartConfig struct {
	Title       string
	Description string
	Charts      []ChartDefinition
}

// ChartDefinition defines which metrics go into a chart
type ChartDefinition struct {
	MetricNames []string
	Title       string
	Description string
	Type        ChartType
	Options     ChartOptions
}

// GetCategoryOrder returns the ordered list of category names
func GetCategoryOrder() []string {
	return []string{
		"ingestion",
		"compactor",
		"storage",
		"resources",
		"query_performance",
		"querier",
	}
}

// GetCategoryChartConfigs returns the chart configuration for all categories
func GetCategoryChartConfigs() map[string]CategoryChartConfig {
	return map[string]CategoryChartConfig{
		"ingestion": {
			Title:       "Ingestion Performance",
			Description: "Metrics related to trace ingestion through the distributor and ingester components",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"accepted_spans_rate", "refused_spans_rate"},
					Title:       "Spans Ingestion Rate",
					Description: "Rate of spans accepted and refused by Tempo's receiver",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "spans/sec", ShowLegend: true},
				},
				{
					MetricNames: []string{"bytes_received_rate"},
					Title:       "Bytes Received Rate",
					Description: "Rate of bytes received by the distributor",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes/sec", YAxisUnit: "bytes", ShowLegend: true},
				},
				{
					MetricNames: []string{"distributor_push_duration_p99"},
					Title:       "Push Latency P99",
					Description: "99th percentile latency of push operations to the distributor",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds"},
				},
				{
					MetricNames: []string{"ingester_append_failures", "discarded_spans"},
					Title:       "Ingestion Errors",
					Description: "Rate of append failures and discarded spans",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "errors/sec", ShowLegend: true, ColorScheme: "red"},
				},
				{
					MetricNames: []string{"ingester_live_traces"},
					Title:       "Live Traces per Ingester",
					Description: "Number of traces currently in memory on each ingester",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "traces", ShowLegend: true},
				},
				{
					MetricNames: []string{"ingester_blocks_flushed"},
					Title:       "Blocks Flushed Rate",
					Description: "Rate of blocks flushed from ingester to storage",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks/sec", ShowLegend: true},
				},
				{
					MetricNames: []string{"ingester_flush_queue_length"},
					Title:       "Flush Queue Length",
					Description: "Number of blocks waiting to be flushed",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks", ShowLegend: true},
				},
			},
		},
		"compactor": {
			Title:       "Compactor",
			Description: "Block compaction and storage optimization metrics",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"compactor_blocks_compacted"},
					Title:       "Blocks Compacted Rate",
					Description: "Rate of blocks successfully compacted",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks/sec"},
				},
				{
					MetricNames: []string{"compactor_bytes_written"},
					Title:       "Compactor Bytes Written",
					Description: "Rate of bytes written by compactor to storage",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes/sec", YAxisUnit: "bytes"},
				},
				{
					MetricNames: []string{"compactor_outstanding_blocks"},
					Title:       "Outstanding Blocks",
					Description: "Number of blocks waiting to be compacted",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks"},
				},
			},
		},
		"storage": {
			Title:       "Storage I/O",
			Description: "Storage read/write latencies and throughput",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"query_frontend_bytes_inspected"},
					Title:       "Bytes Inspected",
					Description: "Rate of bytes read from storage by query frontend",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes/sec", YAxisUnit: "bytes"},
				},
				{
					MetricNames: []string{"backend_read_latency_p99", "blocklist_poll_duration_p99"},
					Title:       "Storage Latency P99",
					Description: "P99 latency of backend operations and blocklist polling",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds", ShowLegend: true},
				},
				{
					MetricNames: []string{"blocklist_length"},
					Title:       "Blocklist Length",
					Description: "Number of blocks in the blocklist per tenant",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks", ShowLegend: true},
				},
				{
					MetricNames: []string{"compaction_bytes_written"},
					Title:       "Compaction Bytes Written",
					Description: "Rate of bytes written during compaction",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes/sec", YAxisUnit: "bytes"},
				},
				{
					MetricNames: []string{"bloom_filter_reads"},
					Title:       "Bloom Filter Reads",
					Description: "Rate of bloom filter reads (query optimization)",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "reads/sec"},
				},
			},
		},
		"resources": {
			Title:       "Resource Utilization",
			Description: "CPU and memory usage by Tempo components",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"memory_usage_total"},
					Title:       "Total Memory Usage",
					Description: "Total memory working set bytes used by all Tempo containers",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes"},
				},
				{
					MetricNames: []string{"cpu_usage_total"},
					Title:       "Total CPU Usage",
					Description: "Total CPU cores used by all Tempo containers",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "cores"},
				},
				{
					MetricNames: []string{"memory_usage_by_pod_container"},
					Title:       "Memory by Pod/Container",
					Description: "Memory usage for each container in each pod",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes", ShowLegend: true},
				},
				{
					MetricNames: []string{"cpu_usage_by_pod_container"},
					Title:       "CPU by Pod/Container",
					Description: "CPU usage for each container in each pod",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "cores", ShowLegend: true},
				},
				{
					MetricNames: []string{"memory_usage_by_component"},
					Title:       "Memory by Component",
					Description: "Memory usage by Tempo component (distributor, ingester, querier, etc.)",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes", ShowLegend: true, Stacked: true},
				},
				{
					MetricNames: []string{"cpu_usage_by_component"},
					Title:       "CPU by Component",
					Description: "CPU usage by Tempo component (distributor, ingester, querier, etc.)",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "cores", ShowLegend: true, Stacked: true},
				},
				{
					MetricNames: []string{"memory_max_total"},
					Title:       "Memory Max (Total)",
					Description: "Maximum total memory usage over 5-minute windows",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes"},
				},
				{
					MetricNames: []string{"cpu_max_total"},
					Title:       "CPU Max (Total)",
					Description: "Maximum total CPU usage over 5-minute windows",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "cores"},
				},
				{
					MetricNames: []string{"memory_max_by_component"},
					Title:       "Memory Max by Component",
					Description: "Maximum memory usage by Tempo component",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes", ShowLegend: true},
				},
				{
					MetricNames: []string{"cpu_max_by_component"},
					Title:       "CPU Max by Component",
					Description: "Maximum CPU usage by Tempo component",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "cores", ShowLegend: true},
				},
			},
		},
		"query_performance": {
			Title:       "Query Performance",
			Description: "Query throughput and latency metrics",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"queries_per_second"},
					Title:       "Queries Per Second",
					Description: "Total query throughput across all query frontends",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "queries/sec"},
				},
				{
					MetricNames: []string{"query_duration_p50", "query_duration_p99"},
					Title:       "Query Latency",
					Description: "End-to-end query latency (P50 and P99)",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds", ShowLegend: true},
				},
				{
					MetricNames: []string{"query_frontend_queue_duration_p99"},
					Title:       "Queue Wait Time P99",
					Description: "99th percentile queue wait time in query frontend",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds"},
				},
				{
					MetricNames: []string{"query_frontend_retries_rate"},
					Title:       "Query Retries Rate",
					Description: "Rate of query retries (indicates query issues)",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "retries/sec", ColorScheme: "red"},
				},
			},
		},
		"querier": {
			Title:       "Querier",
			Description: "Querier queue depth and job processing",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"querier_queue_length"},
					Title:       "Queue Length",
					Description: "Number of queries waiting in querier queue per pod",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "queries", ShowLegend: true},
				},
				{
					MetricNames: []string{"querier_jobs_in_progress"},
					Title:       "Jobs in Progress",
					Description: "Number of jobs currently being processed by querier",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "jobs", ShowLegend: true},
				},
			},
		},
	}
}

// GetMetricUnit returns the appropriate unit for a metric based on its name
func GetMetricUnit(metricName string) string {
	unitMap := map[string]string{
		"memory_usage_total":                "bytes",
		"memory_usage_by_pod_container":     "bytes",
		"memory_usage_by_component":         "bytes",
		"memory_max_total":                  "bytes",
		"memory_max_by_component":           "bytes",
		"cpu_usage_total":                   "cores",
		"cpu_usage_by_pod_container":        "cores",
		"cpu_usage_by_component":            "cores",
		"cpu_max_total":                     "cores",
		"cpu_max_by_component":              "cores",
		"bytes_received_rate":               "bytes",
		"compactor_bytes_written":           "bytes",
		"compaction_bytes_written":          "bytes",
		"query_frontend_bytes_inspected":    "bytes",
		"distributor_push_duration_p99":     "seconds",
		"backend_read_latency_p99":          "seconds",
		"blocklist_poll_duration_p99":       "seconds",
		"query_frontend_queue_duration_p99": "seconds",
		"query_duration_p99":                "seconds",
		"query_duration_p50":                "seconds",
	}

	if unit, ok := unitMap[metricName]; ok {
		return unit
	}
	return "count"
}
