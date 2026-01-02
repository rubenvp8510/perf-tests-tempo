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
		"cache",
		"resources",
		"query_performance",
		"query_latency",
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
					MetricNames: []string{"storage_request_duration_read_p99", "storage_request_duration_write_p99"},
					Title:       "Storage Latency P99",
					Description: "99th percentile latency of storage read and write operations",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds", ShowLegend: true},
				},
			},
		},
		"cache": {
			Title:       "Cache Performance",
			Description: "Cache hit ratios and efficiency",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"cache_hit_ratio"},
					Title:       "Cache Hit Ratio",
					Description: "Overall cache hit rate (0-1)",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "ratio", YAxisUnit: "percent"},
				},
				{
					MetricNames: []string{"cache_hits_by_type"},
					Title:       "Cache Hits by Type",
					Description: "Cache hit rate grouped by cache type",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "hits/sec", ShowLegend: true},
				},
				{
					MetricNames: []string{"cache_misses_by_type"},
					Title:       "Cache Misses by Type",
					Description: "Cache miss rate grouped by cache type",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "misses/sec", ShowLegend: true, ColorScheme: "red"},
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
					MetricNames: []string{"memory_usage_by_container"},
					Title:       "Memory by Container",
					Description: "Memory usage breakdown by container type",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes", ShowLegend: true, Stacked: true},
				},
				{
					MetricNames: []string{"cpu_usage_by_container"},
					Title:       "CPU by Container",
					Description: "CPU usage breakdown by container type",
					Type:        ChartTypeArea,
					Options:     ChartOptions{YAxisLabel: "cores", ShowLegend: true, Stacked: true},
				},
				{
					MetricNames: []string{"memory_usage_by_pod"},
					Title:       "Memory by Pod",
					Description: "Memory usage breakdown by individual pod instance",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "bytes", YAxisUnit: "bytes", ShowLegend: true},
				},
				{
					MetricNames: []string{"cpu_usage_by_pod"},
					Title:       "CPU by Pod",
					Description: "CPU usage breakdown by individual pod instance",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "cores", ShowLegend: true},
				},
			},
		},
		"query_performance": {
			Title:       "Query Performance",
			Description: "Query success rates and throughput",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"total_queries_rate"},
					Title:       "Query Rate",
					Description: "Total query rate (queries per second)",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "queries/sec"},
				},
				{
					MetricNames: []string{"query_failures_rate"},
					Title:       "Query Failure Rate",
					Description: "Rate of failed queries per second",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "failures/sec", ColorScheme: "red"},
				},
				{
					MetricNames: []string{"spans_returned_sum"},
					Title:       "Spans Returned Rate",
					Description: "Rate of total spans returned by queries",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "spans/sec"},
				},
				{
					MetricNames: []string{"spans_returned_count"},
					Title:       "Query Count Rate",
					Description: "Rate of query invocations returning spans",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "queries/sec"},
				},
			},
		},
		"query_latency": {
			Title:       "Query Latency",
			Description: "Query latency percentiles and distribution",
			Charts: []ChartDefinition{
				{
					MetricNames: []string{"query_latency_p90", "query_latency_p99"},
					Title:       "Query Latency Percentiles",
					Description: "90th and 99th percentile query latency",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "seconds", YAxisUnit: "seconds", ShowLegend: true},
				},
				{
					MetricNames: []string{"query_failures_timeseries"},
					Title:       "Query Failures Over Time",
					Description: "Query failure rate time series",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "failures/sec", ColorScheme: "red"},
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
		"memory_usage_total":                 "bytes",
		"memory_usage_by_container":          "bytes",
		"memory_usage_by_pod":                "bytes",
		"cpu_usage_total":                    "cores",
		"cpu_usage_by_container":             "cores",
		"cpu_usage_by_pod":                   "cores",
		"bytes_received_rate":                "bytes",
		"compactor_bytes_written":            "bytes",
		"query_frontend_bytes_inspected":     "bytes",
		"distributor_push_duration_p99":      "seconds",
		"storage_request_duration_read_p99":  "seconds",
		"storage_request_duration_write_p99": "seconds",
		"query_latency_p90":                  "seconds",
		"query_latency_p99":                  "seconds",
		"cache_hit_ratio":                    "percent",
	}

	if unit, ok := unitMap[metricName]; ok {
		return unit
	}
	return "count"
}
