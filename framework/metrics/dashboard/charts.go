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
				{
					MetricNames: []string{"ingester_traces_created"},
					Title:       "Total Traces Created",
					Description: "Cumulative count of traces created in ingester",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "traces"},
				},
				{
					MetricNames: []string{"distributor_spans_received"},
					Title:       "Total Spans Received",
					Description: "Cumulative count of spans received by distributor",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "spans"},
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
				{
					MetricNames: []string{"retention_deleted_total", "retention_marked_for_deletion"},
					Title:       "Retention Activity",
					Description: "Blocks deleted and marked for deletion by retention policy",
					Type:        ChartTypeLine,
					Options:     ChartOptions{YAxisLabel: "blocks", ShowLegend: true},
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

// GetMetricQuery returns the PromQL query template for a metric
// The {namespace} placeholder should be replaced with the actual namespace
func GetMetricQuery(metricName string) string {
	queryMap := map[string]string{
		// Ingestion metrics
		"accepted_spans_rate":         `sum(rate(tempo_receiver_accepted_spans{namespace="{namespace}"}[1m]))`,
		"refused_spans_rate":          `sum(rate(tempo_receiver_refused_spans{namespace="{namespace}"}[1m]))`,
		"bytes_received_rate":         `sum(rate(tempo_distributor_bytes_received_total{namespace="{namespace}"}[1m])) by (status)`,
		"distributor_push_duration_p99": `histogram_quantile(0.99, sum(rate(tempo_distributor_push_duration_seconds_bucket{namespace="{namespace}"}[1m])) by (le))`,
		"ingester_append_failures":    `sum(rate(tempo_ingester_failed_flushes_total{namespace="{namespace}"}[1m]))`,
		"discarded_spans":             `sum(rate(tempo_discarded_spans_total{namespace="{namespace}"}[1m])) by (reason)`,
		"ingester_live_traces":        `sum(tempo_ingester_live_traces{namespace="{namespace}"}) by (pod)`,
		"ingester_blocks_flushed":     `sum(rate(tempo_ingester_blocks_flushed_total{namespace="{namespace}"}[1m])) by (pod)`,
		"ingester_flush_queue_length": `sum(tempo_ingester_flush_queue_length{namespace="{namespace}"}) by (pod)`,
		"ingester_traces_created":     `sum(tempo_ingester_traces_created_total{namespace="{namespace}"})`,
		"distributor_spans_received":  `sum(tempo_distributor_spans_received_total{namespace="{namespace}"})`,

		// Compactor metrics
		"compactor_blocks_compacted":       `sum(rate(tempodb_compaction_blocks_total{namespace="{namespace}"}[1m]))`,
		"compactor_bytes_written":          `sum(rate(tempodb_compaction_bytes_written_total{namespace="{namespace}"}[1m]))`,
		"compactor_outstanding_blocks":     `sum(tempodb_compaction_outstanding_blocks{namespace="{namespace}"})`,
		"retention_deleted_total":          `sum(tempodb_retention_deleted_total{namespace="{namespace}"})`,
		"retention_marked_for_deletion":    `sum(tempodb_retention_marked_for_deletion_total{namespace="{namespace}"})`,

		// Storage metrics
		"query_frontend_bytes_inspected": `sum(rate(tempo_query_frontend_bytes_inspected_total{namespace="{namespace}"}[1m]))`,
		"backend_read_latency_p99":       `histogram_quantile(0.99, sum(rate(tempodb_backend_request_duration_seconds_bucket{namespace="{namespace}"}[1m])) by (le))`,
		"blocklist_poll_duration_p99":   `histogram_quantile(0.99, sum(rate(tempodb_blocklist_poll_duration_seconds_bucket{namespace="{namespace}"}[1m])) by (le))`,
		"blocklist_length":              `sum(tempodb_blocklist_length{namespace="{namespace}"}) by (tenant)`,

		// Resource metrics
		"memory_usage_total":           `sum(container_memory_working_set_bytes{namespace="{namespace}", container=~"tempo.*"})`,
		"cpu_usage_total":              `sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}", container=~"tempo.*"}[5m]))`,
		"memory_usage_by_pod_container": `sum(container_memory_working_set_bytes{namespace="{namespace}", container=~"tempo.*"}) by (pod, container)`,
		"cpu_usage_by_pod_container":   `sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}", container=~"tempo.*"}[5m])) by (pod, container)`,
		"memory_usage_by_component":    `sum by (component) (label_replace(...container_memory_working_set_bytes...))`,
		"cpu_usage_by_component":       `sum by (component) (label_replace(...container_cpu_usage_seconds_total...))`,
		"memory_max_total":             `max_over_time(sum(container_memory_working_set_bytes{namespace="{namespace}", container=~"tempo.*"})[5m:])`,
		"cpu_max_total":                `max_over_time(sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}", container=~"tempo.*"}[1m]))[5m:])`,
		"memory_max_by_component":      `max by (component) (max_over_time(...container_memory_working_set_bytes...)[5m:])`,
		"cpu_max_by_component":         `max by (component) (max_over_time(...container_cpu_usage_seconds_total...)[5m:])`,

		// Query performance metrics
		"queries_per_second":              `sum(rate(tempo_query_frontend_queries_total{namespace="{namespace}"}[1m]))`,
		"query_duration_p99":              `histogram_quantile(0.99, sum(rate(tempo_request_duration_seconds_bucket{namespace="{namespace}", route=~".*search.*|.*Search.*"}[5m])) by (le))`,
		"query_duration_p50":              `histogram_quantile(0.50, sum(rate(tempo_request_duration_seconds_bucket{namespace="{namespace}", route=~".*search.*|.*Search.*"}[5m])) by (le))`,
		"query_frontend_queue_duration_p99": `histogram_quantile(0.99, sum(rate(tempo_query_frontend_queue_duration_seconds_bucket{namespace="{namespace}"}[1m])) by (le))`,
		"query_frontend_retries_rate":    `sum(rate(tempo_query_frontend_retries_count{namespace="{namespace}"}[1m]))`,

		// Querier metrics
		"querier_queue_length":      `sum(tempo_query_frontend_queue_length{namespace="{namespace}"}) by (pod)`,
		"querier_jobs_in_progress":  `sum(rate(tempo_query_frontend_queries_total{namespace="{namespace}"}[1m])) by (pod)`,
	}

	if query, ok := queryMap[metricName]; ok {
		return query
	}
	return ""
}
