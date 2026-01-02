package metrics

import (
	"fmt"
	"os"
)

// MetricQuery represents a single PromQL query with metadata
type MetricQuery struct {
	ID          string
	Name        string
	Description string
	Query       string
	Category    string
	Type        string // "instant" or "range"
}

// GetAllQueries returns all metric queries defined in promql-queries.md
func GetAllQueries(namespace string) []MetricQuery {
	queries := []MetricQuery{
		// Ingestion Metrics (Tempo Receiver/Distributor)
		{
			ID:          "1",
			Name:        "accepted_spans_rate",
			Description: "Rate of spans successfully accepted by Tempo's receiver per second",
			Query:       fmt.Sprintf(`sum(rate(tempo_receiver_accepted_spans{namespace="%s"}[1m]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "2",
			Name:        "refused_spans_rate",
			Description: "Rate of spans refused/rejected by Tempo's receiver per second",
			Query:       fmt.Sprintf(`sum(rate(tempo_receiver_refused_spans{namespace="%s"}[1m]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "3",
			Name:        "bytes_received_rate",
			Description: "Rate of bytes received by the distributor per second, grouped by status",
			Query:       fmt.Sprintf(`sum(rate(tempo_distributor_bytes_received_total{namespace="%s"}[1m])) by (status)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "4",
			Name:        "distributor_push_duration_p99",
			Description: "P99 latency of push operations to the distributor",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_distributor_push_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "5",
			Name:        "ingester_append_failures",
			Description: "Rate of failed ingester flushes",
			Query:       fmt.Sprintf(`sum(rate(tempo_ingester_failed_flushes_total{namespace="%s"}[1m]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "6",
			Name:        "discarded_spans",
			Description: "Rate of discarded spans per second, grouped by discard reason",
			Query:       fmt.Sprintf(`sum(rate(tempo_discarded_spans_total{namespace="%s"}[1m])) by (reason)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "7",
			Name:        "ingester_live_traces",
			Description: "Number of live (in-memory) traces in each ingester",
			Query:       fmt.Sprintf(`sum(tempo_ingester_live_traces{namespace="%s"}) by (pod)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "8",
			Name:        "ingester_blocks_flushed",
			Description: "Rate of blocks flushed from ingester to storage",
			Query:       fmt.Sprintf(`sum(rate(tempo_ingester_blocks_flushed_total{namespace="%s"}[1m])) by (pod)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "9",
			Name:        "ingester_flush_queue_length",
			Description: "Number of blocks waiting to be flushed",
			Query:       fmt.Sprintf(`sum(tempo_ingester_flush_queue_length{namespace="%s"}) by (pod)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},

		// Compactor Metrics
		{
			ID:          "10",
			Name:        "compactor_blocks_compacted",
			Description: "Rate of compaction errors (inverse indicator of successful compaction)",
			Query:       fmt.Sprintf(`sum(rate(tempodb_compaction_errors_total{namespace="%s"}[1m]))`, namespace),
			Category:    "compactor",
			Type:        "range",
		},
		{
			ID:          "11",
			Name:        "compactor_bytes_written",
			Description: "Rate of bytes deleted by retention",
			Query:       fmt.Sprintf(`sum(rate(tempodb_retention_deleted_total{namespace="%s"}[1m]))`, namespace),
			Category:    "compactor",
			Type:        "range",
		},
		{
			ID:          "12",
			Name:        "compactor_outstanding_blocks",
			Description: "Number of items in the work queue",
			Query:       fmt.Sprintf(`sum(tempodb_work_queue_length{namespace="%s"})`, namespace),
			Category:    "compactor",
			Type:        "range",
		},

		// Storage and I/O Metrics
		{
			ID:          "13",
			Name:        "query_frontend_bytes_inspected",
			Description: "Rate of bytes read from storage by query frontend",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_bytes_inspected_total{namespace="%s"}[1m]))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "14",
			Name:        "backend_read_latency_p99",
			Description: "P99 latency of backend read operations (all operations)",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempodb_backend_request_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "15",
			Name:        "blocklist_poll_duration_p99",
			Description: "P99 blocklist poll duration (storage access patterns)",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempodb_blocklist_poll_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},

		// Storage Block Metrics
		{
			ID:          "16",
			Name:        "blocklist_length",
			Description: "Number of blocks in the blocklist per tenant",
			Query:       fmt.Sprintf(`sum(tempodb_blocklist_length{namespace="%s"}) by (tenant)`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "17",
			Name:        "compaction_bytes_written",
			Description: "Rate of bytes written during compaction",
			Query:       fmt.Sprintf(`sum(rate(tempodb_compaction_bytes_written_total{namespace="%s"}[1m]))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "18",
			Name:        "bloom_filter_reads",
			Description: "Rate of bloom filter reads (query optimization)",
			Query:       fmt.Sprintf(`sum(rate(tempodb_bloom_filter_reads_total{namespace="%s"}[1m]))`, namespace),
			Category:    "storage",
			Type:        "range",
		},

		// Resource Utilization Metrics
		{
			ID:          "19",
			Name:        "memory_usage_total",
			Description: "Total memory working set bytes used by all Tempo containers",
			Query:       fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"})`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "20",
			Name:        "cpu_usage_total",
			Description: "Total CPU cores used by all Tempo containers",
			Query:       fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[5m]))`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "21",
			Name:        "memory_usage_by_pod_container",
			Description: "Memory usage for each container in each pod",
			Query:       fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"}) by (pod, container)`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "22",
			Name:        "cpu_usage_by_pod_container",
			Description: "CPU usage for each container in each pod",
			Query:       fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[5m])) by (pod, container)`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "23",
			Name:        "memory_usage_by_component",
			Description: "Memory usage grouped by Tempo component (distributor, ingester, etc.)",
			Query: fmt.Sprintf(`sum by (component) (
  label_replace(
    label_replace(
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              container_memory_working_set_bytes{namespace="%s", container=~"tempo.*", container!=""},
              "component", "distributor", "pod", ".*-distributor-.*"
            ),
            "component", "ingester", "pod", ".*-ingester-.*"
          ),
          "component", "querier", "pod", ".*-querier-.*"
        ),
        "component", "compactor", "pod", ".*-compactor-.*"
      ),
      "component", "gateway", "pod", ".*-gateway-.*"
    ),
    "component", "query-frontend", "pod", ".*-query-frontend-.*"
  )
)`, namespace),
			Category: "resources",
			Type:     "range",
		},
		{
			ID:          "24",
			Name:        "cpu_usage_by_component",
			Description: "CPU usage grouped by Tempo component (distributor, ingester, etc.)",
			Query: fmt.Sprintf(`sum by (component) (
  label_replace(
    label_replace(
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[5m]),
              "component", "distributor", "pod", ".*-distributor-.*"
            ),
            "component", "ingester", "pod", ".*-ingester-.*"
          ),
          "component", "querier", "pod", ".*-querier-.*"
        ),
        "component", "compactor", "pod", ".*-compactor-.*"
      ),
      "component", "gateway", "pod", ".*-gateway-.*"
    ),
    "component", "query-frontend", "pod", ".*-query-frontend-.*"
  )
)`, namespace),
			Category: "resources",
			Type:     "range",
		},

		// Max Resource Metrics (simpler than P99, always works)
		{
			ID:          "25",
			Name:        "memory_max_by_component",
			Description: "Max memory usage by Tempo component over 5-minute windows",
			Query: fmt.Sprintf(`max by (component) (
  max_over_time(
    sum by (component) (
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              label_replace(
                label_replace(
                  container_memory_working_set_bytes{namespace="%s", container=~"tempo.*", container!=""},
                  "component", "distributor", "pod", ".*-distributor-.*"
                ),
                "component", "ingester", "pod", ".*-ingester-.*"
              ),
              "component", "querier", "pod", ".*-querier-.*"
            ),
            "component", "compactor", "pod", ".*-compactor-.*"
          ),
          "component", "gateway", "pod", ".*-gateway-.*"
        ),
        "component", "query-frontend", "pod", ".*-query-frontend-.*"
      )
    )[5m:]
  )
)`, namespace),
			Category: "resources",
			Type:     "range",
		},
		{
			ID:          "26",
			Name:        "cpu_max_by_component",
			Description: "Max CPU usage by Tempo component over 5-minute windows",
			Query: fmt.Sprintf(`max by (component) (
  max_over_time(
    sum by (component) (
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              label_replace(
                label_replace(
                  rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]),
                  "component", "distributor", "pod", ".*-distributor-.*"
                ),
                "component", "ingester", "pod", ".*-ingester-.*"
              ),
              "component", "querier", "pod", ".*-querier-.*"
            ),
            "component", "compactor", "pod", ".*-compactor-.*"
          ),
          "component", "gateway", "pod", ".*-gateway-.*"
        ),
        "component", "query-frontend", "pod", ".*-query-frontend-.*"
      )
    )[5m:]
  )
)`, namespace),
			Category: "resources",
			Type:     "range",
		},
		{
			ID:          "27",
			Name:        "memory_max_total",
			Description: "Max total memory usage over 5-minute windows",
			Query:       fmt.Sprintf(`max_over_time(sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"})[5m:])`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "28",
			Name:        "cpu_max_total",
			Description: "Max total CPU usage over 5-minute windows",
			Query:       fmt.Sprintf(`max_over_time(sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]))[5m:])`, namespace),
			Category:    "resources",
			Type:        "range",
		},

		// Query Performance Metrics (Tempo-internal)
		// Note: k6 metrics (query_failures_rate, total_queries_rate, spans_returned_sum, query_latency_p90/p99)
		// are exported to separate JSON files since OpenShift doesn't support Prometheus remote write receiver
		{
			ID:          "29",
			Name:        "query_frontend_queue_duration_p99",
			Description: "Query frontend queue wait time p99",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_query_frontend_queue_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "30",
			Name:        "query_frontend_retries_rate",
			Description: "Query frontend retries rate (indicates query issues)",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_retries_count{namespace="%s"}[1m]))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},

		// Querier Specific Metrics
		{
			ID:          "31",
			Name:        "querier_queue_length",
			Description: "Number of queries waiting in query frontend queue",
			Query:       fmt.Sprintf(`sum(tempo_query_frontend_queue_length{namespace="%s"}) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
		{
			ID:          "32",
			Name:        "querier_jobs_in_progress",
			Description: "Total queries processed by query frontend",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_queries_total{namespace="%s"}[1m])) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
	}

	return queries
}

// GetSummaryQueries returns instant queries for summary metrics (P99 over full test duration)
// These are executed once at the end of the test to get aggregate values
func GetSummaryQueries(namespace string) []MetricQuery {
	// Get duration from env var, default to 5m
	duration := os.Getenv("DURATION")
	if duration == "" {
		duration = "5m"
	}

	return []MetricQuery{
		{
			ID:          "summary_1",
			Name:        "summary_memory_p99_total",
			Description: fmt.Sprintf("P99 total memory usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`quantile_over_time(0.99, sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"})[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
		{
			ID:          "summary_2",
			Name:        "summary_cpu_p99_total",
			Description: fmt.Sprintf("P99 total CPU usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`quantile_over_time(0.99, sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]))[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
		{
			ID:          "summary_3",
			Name:        "summary_memory_p99_by_component",
			Description: fmt.Sprintf("P99 memory by component over the entire test (%s)", duration),
			Query: fmt.Sprintf(`quantile_over_time(0.99,
  sum by (component) (
    label_replace(
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              label_replace(
                container_memory_working_set_bytes{namespace="%s", container=~"tempo.*", container!=""},
                "component", "distributor", "pod", ".*-distributor-.*"
              ),
              "component", "ingester", "pod", ".*-ingester-.*"
            ),
            "component", "querier", "pod", ".*-querier-.*"
          ),
          "component", "compactor", "pod", ".*-compactor-.*"
        ),
        "component", "gateway", "pod", ".*-gateway-.*"
      ),
      "component", "query-frontend", "pod", ".*-query-frontend-.*"
    )
  )[%s:])`, namespace, duration),
			Category: "summary",
			Type:     "instant",
		},
		{
			ID:          "summary_4",
			Name:        "summary_cpu_p99_by_component",
			Description: fmt.Sprintf("P99 CPU by component over the entire test (%s)", duration),
			Query: fmt.Sprintf(`quantile_over_time(0.99,
  sum by (component) (
    label_replace(
      label_replace(
        label_replace(
          label_replace(
            label_replace(
              label_replace(
                rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]),
                "component", "distributor", "pod", ".*-distributor-.*"
              ),
              "component", "ingester", "pod", ".*-ingester-.*"
            ),
            "component", "querier", "pod", ".*-querier-.*"
          ),
          "component", "compactor", "pod", ".*-compactor-.*"
        ),
        "component", "gateway", "pod", ".*-gateway-.*"
      ),
      "component", "query-frontend", "pod", ".*-query-frontend-.*"
    )
  )[%s:])`, namespace, duration),
			Category: "summary",
			Type:     "instant",
		},
		{
			ID:          "summary_5",
			Name:        "summary_memory_max_total",
			Description: fmt.Sprintf("Max total memory usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`max_over_time(sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"})[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
		{
			ID:          "summary_6",
			Name:        "summary_cpu_max_total",
			Description: fmt.Sprintf("Max total CPU usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`max_over_time(sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]))[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
		{
			ID:          "summary_7",
			Name:        "summary_memory_avg_total",
			Description: fmt.Sprintf("Average total memory usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`avg_over_time(sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"})[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
		{
			ID:          "summary_8",
			Name:        "summary_cpu_avg_total",
			Description: fmt.Sprintf("Average total CPU usage over the entire test (%s)", duration),
			Query:       fmt.Sprintf(`avg_over_time(sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[1m]))[%s:])`, namespace, duration),
			Category:    "summary",
			Type:        "instant",
		},
	}
}
