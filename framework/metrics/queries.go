package metrics

import (
	"fmt"
	"strings"
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
	// Replace namespace hyphens with underscores for metric names
	nsMetric := strings.ReplaceAll(namespace, "-", "_")

	queries := []MetricQuery{
		// Ingestion Metrics (Tempo Receiver/Distributor)
		{
			ID:          "1",
			Name:        "accepted_spans_rate",
			Description: "Rate of spans successfully accepted by Tempo's receiver per second",
			Query:       fmt.Sprintf(`sum(rate(tempo_receiver_accepted_spans{namespace="%s"}[30s]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "2",
			Name:        "refused_spans_rate",
			Description: "Rate of spans refused/rejected by Tempo's receiver per second",
			Query:       fmt.Sprintf(`sum(rate(tempo_receiver_refused_spans{namespace="%s"}[30s]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "3",
			Name:        "bytes_received_rate",
			Description: "Rate of bytes received by the distributor per second, grouped by status",
			Query:       fmt.Sprintf(`sum(rate(tempo_distributor_bytes_received_total{namespace="%s"}[30s])) by (status)`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "4",
			Name:        "distributor_push_duration_p99",
			Description: "P99 latency of push operations to the distributor",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_distributor_push_duration_seconds_bucket{namespace="%s"}[30s])) by (le))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "5",
			Name:        "ingester_append_failures",
			Description: "Rate of failures when the distributor attempts to append to ingesters",
			Query:       fmt.Sprintf(`sum(rate(tempo_distributor_ingester_append_failures_total{namespace="%s"}[30s]))`, namespace),
			Category:    "ingestion",
			Type:        "range",
		},
		{
			ID:          "6",
			Name:        "discarded_spans",
			Description: "Rate of discarded spans per second, grouped by discard reason",
			Query:       fmt.Sprintf(`sum(rate(tempo_discarded_spans_total{namespace="%s"}[30s])) by (reason)`, namespace),
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
			Query:       fmt.Sprintf(`sum(rate(tempo_ingester_blocks_flushed_total{namespace="%s"}[30s])) by (pod)`, namespace),
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
			Description: "Rate of blocks successfully compacted",
			Query:       fmt.Sprintf(`sum(rate(tempo_compactor_blocks_compacted_total{namespace="%s"}[30s]))`, namespace),
			Category:    "compactor",
			Type:        "range",
		},
		{
			ID:          "11",
			Name:        "compactor_bytes_written",
			Description: "Rate of bytes written by compactor to storage",
			Query:       fmt.Sprintf(`sum(rate(tempo_compactor_bytes_written_total{namespace="%s"}[30s]))`, namespace),
			Category:    "compactor",
			Type:        "range",
		},
		{
			ID:          "12",
			Name:        "compactor_outstanding_blocks",
			Description: "Number of blocks waiting to be compacted",
			Query:       fmt.Sprintf(`sum(tempo_compactor_outstanding_blocks{namespace="%s"})`, namespace),
			Category:    "compactor",
			Type:        "range",
		},

		// Storage and I/O Metrics
		{
			ID:          "13",
			Name:        "query_frontend_bytes_inspected",
			Description: "Rate of bytes read from storage by query frontend",
			Query:       fmt.Sprintf(`sum(rate(tempo_query_frontend_bytes_inspected_total{namespace="%s"}[30s]))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "14",
			Name:        "storage_request_duration_read_p99",
			Description: "P99 latency of storage read operations",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_storage_request_duration_seconds_bucket{namespace="%s",operation="read"}[30s])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},
		{
			ID:          "15",
			Name:        "storage_request_duration_write_p99",
			Description: "P99 latency of storage write operations",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tempo_storage_request_duration_seconds_bucket{namespace="%s",operation="write"}[30s])) by (le))`, namespace),
			Category:    "storage",
			Type:        "range",
		},

		// Cache Metrics
		{
			ID:          "16",
			Name:        "cache_hit_ratio",
			Description: "Cache hit rate (0-1)",
			Query:       fmt.Sprintf(`sum(rate(tempo_cache_hits_total{namespace="%s"}[30s])) / (sum(rate(tempo_cache_hits_total{namespace="%s"}[30s])) + sum(rate(tempo_cache_misses_total{namespace="%s"}[30s])))`, namespace, namespace, namespace),
			Category:    "cache",
			Type:        "range",
		},
		{
			ID:          "17",
			Name:        "cache_hits_by_type",
			Description: "Cache hit rate by cache type",
			Query:       fmt.Sprintf(`sum(rate(tempo_cache_hits_total{namespace="%s"}[30s])) by (cache_type)`, namespace),
			Category:    "cache",
			Type:        "range",
		},
		{
			ID:          "18",
			Name:        "cache_misses_by_type",
			Description: "Cache miss rate by cache type",
			Query:       fmt.Sprintf(`sum(rate(tempo_cache_misses_total{namespace="%s"}[30s])) by (cache_type)`, namespace),
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
			Query:       fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*"}[30s]))`, namespace),
			Category:    "resources",
			Type:        "range",
		},
		{
			ID:          "21",
			Name:        "cpu_usage_by_container",
			Description: "CPU cores used by each individual Tempo container",
			Query:       fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*"}[30s])) by (container)`, namespace),
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
			Query:       fmt.Sprintf(`rate(container_cpu_usage_seconds_total{namespace="%s", container=~"tempo.*"}[30s])`, namespace),
			Category:    "resources",
			Type:        "range",
		},

		// Query Performance Metrics
		{
			ID:          "25",
			Name:        "query_failures_rate",
			Description: "Rate of failed queries per second",
			Query:       fmt.Sprintf(`sum(rate(query_failures_count_%s[30s]))`, nsMetric),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "26",
			Name:        "total_queries_rate",
			Description: "Total query rate (queries per second)",
			Query:       fmt.Sprintf(`sum(rate(query_load_test_%s_count[30s]))`, nsMetric),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "27",
			Name:        "spans_returned_sum",
			Description: "Rate of total spans returned (sum component)",
			Query:       fmt.Sprintf(`sum(rate(query_load_test_spans_returned_%s_sum[30s]))`, nsMetric),
			Category:    "query_performance",
			Type:        "range",
		},
		{
			ID:          "28",
			Name:        "spans_returned_count",
			Description: "Rate of query count for spans returned",
			Query:       fmt.Sprintf(`sum(rate(query_load_test_spans_returned_%s_count[30s]))`, nsMetric),
			Category:    "query_performance",
			Type:        "range",
		},

		// Query Latency Time-Series
		{
			ID:          "29",
			Name:        "query_latency_p90",
			Description: "90th percentile query latency",
			Query:       fmt.Sprintf(`histogram_quantile(0.90, sum(rate(query_load_test_%s_bucket[30s])) by (le))`, nsMetric),
			Category:    "query_latency",
			Type:        "range",
		},
		{
			ID:          "30",
			Name:        "query_latency_p99",
			Description: "99th percentile query latency",
			Query:       fmt.Sprintf(`histogram_quantile(0.99, sum(rate(query_load_test_%s_bucket[30s])) by (le))`, nsMetric),
			Category:    "query_latency",
			Type:        "range",
		},
		{
			ID:          "31",
			Name:        "query_failures_timeseries",
			Description: "Query failure rate over time",
			Query:       fmt.Sprintf(`sum(rate(query_failures_count_%s[30s]))`, nsMetric),
			Category:    "query_latency",
			Type:        "range",
		},

		// Querier Specific Metrics
		{
			ID:          "32",
			Name:        "querier_queue_length",
			Description: "Number of queries waiting in querier queue",
			Query:       fmt.Sprintf(`sum(tempo_querier_queue_length{namespace="%s"}) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
		{
			ID:          "33",
			Name:        "querier_jobs_in_progress",
			Description: "Number of jobs currently being processed by querier",
			Query:       fmt.Sprintf(`sum(tempo_querier_jobs_in_progress{namespace="%s"}) by (pod)`, namespace),
			Category:    "querier",
			Type:        "range",
		},
	}

	return queries
}
