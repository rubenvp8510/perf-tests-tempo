package metrics

import (
	"fmt"
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
			Name:        "storage_request_duration_read_p99",
			Description: "P99 latency of storage read operations",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempodb_backend_request_duration_seconds_bucket{namespace="%s",operation="GET"}[1m])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "15",
			Name:        "storage_request_duration_write_p99",
			Description: "P99 latency of retention operations",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempodb_retention_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},

		// Cache Metrics (using blocklist poll as proxy when cache not configured)
		{
			ID:          "16",
			Name:        "cache_hit_ratio",
			Description: "Blocklist poll duration p99 (proxy for storage access patterns)",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempodb_blocklist_poll_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "cache",
			Type:        "range",
		},
		{
			ID:          "17",
			Name:        "cache_hits_by_type",
			Description: "Backend hedged roundtrips rate (storage access optimization)",
			Query:       fmt.Sprintf(`sum(rate(tempodb_backend_hedged_roundtrips_total{namespace="%s"}[1m])) by (pod)`, namespace),
			Category:    "cache",
			Type:        "range",
		},
		{
			ID:          "18",
			Name:        "cache_misses_by_type",
			Description: "Blocklist poll rate by pod",
			Query:       fmt.Sprintf(`sum(rate(tempodb_blocklist_poll_duration_seconds_count{namespace="%s"}[1m])) by (pod)`, namespace),
			Category:    "cache",
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
			Name:        "cpu_usage_by_container",
			Description: "CPU cores used by each individual Tempo container",
			Query:       fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[5m])) by (container)`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "22",
			Name:        "memory_usage_by_container",
			Description: "Memory working set bytes used by each individual Tempo container",
			Query:       fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"}) by (container)`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "23",
			Name:        "memory_usage_by_pod",
			Description: "Memory working set bytes for each Tempo container instance",
			Query:       fmt.Sprintf(`container_memory_working_set_bytes{namespace="%s", container=~"tempo.*"}`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "24",
			Name:        "cpu_usage_by_pod",
			Description: "CPU cores used by each Tempo container instance",
			Query:       fmt.Sprintf(`rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*", container!=""}[5m])`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "34",
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
			ID:          "35",
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

		// Query Performance Metrics (from k6 xk6-tempo extension - requires k6 test with Prometheus export)
		{
			ID:          "25",
			Name:        "query_failures_rate",
			Description: "Rate of failed k6 queries per second (requires k6 test)",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_failures_total{namespace="%s"}[1m]))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "26",
			Name:        "total_queries_rate",
			Description: "Total k6 query rate per second (requires k6 test)",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_requests_total{namespace="%s"}[1m]))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "27",
			Name:        "spans_returned_sum",
			Description: "Traces returned by k6 queries (requires k6 test)",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_traces_returned_total{namespace="%s"}[1m]))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "28",
			Name:        "spans_returned_count",
			Description: "Query frontend queue wait time p99 (proxy for query load)",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_query_frontend_queue_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "query_performance",
			Type:        "range",
		},

		// Query Latency Time-Series (from k6 xk6-tempo extension - requires k6 test with Prometheus export)
		{
			ID:          "29",
			Name:        "query_latency_p90",
			Description: "90th percentile k6 query latency (requires k6 test)",
			Query:       fmt.Sprintf(`histogram_quantile(0.90, sum(rate(tempo_query_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "query_latency",
			Type:        "range",
		},
		{
			ID:          "30",
			Name:        "query_latency_p99",
			Description: "99th percentile k6 query latency (requires k6 test)",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_query_duration_seconds_bucket{namespace="%s"}[1m])) by (le))`, namespace),
			Category:    "query_latency",
			Type:        "range",
		},
		{
			ID:          "31",
			Name:        "query_failures_timeseries",
			Description: "Query frontend retries rate (indicates query issues)",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_retries_count{namespace="%s"}[1m]))`, namespace),
			Category:    "query_latency",
			Type:        "range",
		},

		// Querier Specific Metrics
		{
			ID:          "32",
			Name:        "querier_queue_length",
			Description: "Number of queries waiting in query frontend queue",
			Query:       fmt.Sprintf(`sum(tempo_query_frontend_queue_length{namespace="%s"}) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
		{
			ID:          "33",
			Name:        "querier_jobs_in_progress",
			Description: "Total queries processed by query frontend",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_queries_total{namespace="%s"}[1m])) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
	}

	return queries
}
