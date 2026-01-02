package k6

import (
	"encoding/json"
	"strings"
	"time"
)

// TestType represents the type of k6 test to run
type TestType string

const (
	TestIngestion TestType = "ingestion"
	TestQuery     TestType = "query"
	TestCombined  TestType = "combined"
)

// Size represents t-shirt sizes for k6 tests
type Size string

const (
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
	SizeXLarge Size = "xlarge"
)

// TempoVariant represents the type of Tempo deployment
type TempoVariant string

const (
	// TempoMonolithic is the single-pod Tempo deployment
	TempoMonolithic TempoVariant = "monolithic"
	// TempoStack is the distributed Tempo deployment
	TempoStack TempoVariant = "stack"
)

// CR names used by the framework
const (
	// MonolithicCRName is the name of the TempoMonolithic CR created by the framework
	MonolithicCRName = "simplest"
	// StackCRName is the name of the TempoStack CR created by the framework
	StackCRName = "tempostack"
)

const (
	// DefaultImage is the default xk6-tempo image
	DefaultImage = "quay.io/rvargasp/xk6-tempo:latest"

	// ScriptsConfigMap is the name of the ConfigMap containing k6 scripts
	ScriptsConfigMap = "k6-scripts"

	// JobTimeout is the maximum time to wait for a k6 job to complete
	JobTimeout = 30 * time.Minute

	// DefaultTenant is the default tenant ID for multitenancy mode
	DefaultTenant = "tenant-1"

	// TLS paths for service account credentials (OpenShift)
	ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	ServiceAccountCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

// Config holds configuration for k6 test execution
type Config struct {
	// Size is the t-shirt size (small, medium, large, xlarge)
	Size Size

	// TempoVariant is the Tempo deployment type (monolithic or stack)
	// Used to auto-discover endpoints if not explicitly set
	TempoVariant TempoVariant

	// Image is the k6 container image (optional, defaults to xk6-tempo image)
	Image string

	// Custom overrides (optional)
	MBPerSecond      float64
	QueriesPerSecond int
	Duration         string
	VUsMin           int
	VUsMax           int
	TraceProfile     string

	// Endpoints (auto-discovered based on TempoVariant if empty)
	TempoEndpoint      string
	TempoQueryEndpoint string
	TempoTenant        string
	TempoToken         string

	// Prometheus metrics export configuration
	// If set, k6 will export metrics to Prometheus via remote write
	PrometheusRWURL string
}

// Result holds the result of a k6 test execution
type Result struct {
	Success  bool
	Output   string
	Duration time.Duration
	Error    error
	Metrics  *K6Metrics
}

// K6Metrics holds parsed metrics from k6 JSON summary output
type K6Metrics struct {
	// Query metrics from xk6-tempo
	QueryRequestsTotal   float64
	QueryFailuresTotal   float64
	QuerySpansReturned   MetricStats
	QueryDurationSeconds MetricStats

	// Ingestion metrics from xk6-tempo
	IngestionBytesTotal  float64
	IngestionTracesTotal float64
	IngestionRateBPS     float64
	IngestionDuration    MetricStats
}

// MetricStats holds statistical values for a metric
type MetricStats struct {
	Avg float64
	Min float64
	Med float64
	Max float64
	P90 float64
	P95 float64
	P99 float64
}

// k6SummaryJSON represents the structure of k6's --summary-export JSON output
type k6SummaryJSON struct {
	Metrics map[string]k6MetricData `json:"metrics"`
}

type k6MetricData struct {
	Type       string          `json:"type"`
	Contains   string          `json:"contains"`
	Values     k6MetricValues  `json:"values"`
	Thresholds map[string]bool `json:"thresholds,omitempty"`
}

type k6MetricValues struct {
	Count float64 `json:"count,omitempty"`
	Rate  float64 `json:"rate,omitempty"`
	Value float64 `json:"value,omitempty"`
	Avg   float64 `json:"avg,omitempty"`
	Min   float64 `json:"min,omitempty"`
	Med   float64 `json:"med,omitempty"`
	Max   float64 `json:"max,omitempty"`
	P90   float64 `json:"p(90),omitempty"`
	P95   float64 `json:"p(95),omitempty"`
	P99   float64 `json:"p(99),omitempty"`
}

// ParseK6Metrics extracts k6 metrics from the output containing the JSON summary
func ParseK6Metrics(output string) *K6Metrics {
	// Find the JSON between markers
	startMarker := "===K6_SUMMARY_JSON_START==="
	endMarker := "===K6_SUMMARY_JSON_END==="

	startIdx := strings.Index(output, startMarker)
	endIdx := strings.Index(output, endMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil
	}

	jsonStr := strings.TrimSpace(output[startIdx+len(startMarker) : endIdx])
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}

	var summary k6SummaryJSON
	if err := json.Unmarshal([]byte(jsonStr), &summary); err != nil {
		return nil
	}

	metrics := &K6Metrics{}

	// Extract query metrics
	if m, ok := summary.Metrics["tempo_query_requests_total"]; ok {
		metrics.QueryRequestsTotal = m.Values.Count
	}
	if m, ok := summary.Metrics["tempo_query_failures_total"]; ok {
		metrics.QueryFailuresTotal = m.Values.Count
	}
	if m, ok := summary.Metrics["tempo_query_spans_returned"]; ok {
		metrics.QuerySpansReturned = MetricStats{
			Avg: m.Values.Avg,
			Min: m.Values.Min,
			Med: m.Values.Med,
			Max: m.Values.Max,
			P90: m.Values.P90,
			P95: m.Values.P95,
			P99: m.Values.P99,
		}
	}
	if m, ok := summary.Metrics["tempo_query_duration_seconds"]; ok {
		metrics.QueryDurationSeconds = MetricStats{
			Avg: m.Values.Avg,
			Min: m.Values.Min,
			Med: m.Values.Med,
			Max: m.Values.Max,
			P90: m.Values.P90,
			P95: m.Values.P95,
			P99: m.Values.P99,
		}
	}

	// Extract ingestion metrics
	if m, ok := summary.Metrics["tempo_ingestion_bytes_total"]; ok {
		metrics.IngestionBytesTotal = m.Values.Count
	}
	if m, ok := summary.Metrics["tempo_ingestion_traces_total"]; ok {
		metrics.IngestionTracesTotal = m.Values.Count
	}
	if m, ok := summary.Metrics["tempo_ingestion_rate_bytes_per_sec"]; ok {
		metrics.IngestionRateBPS = m.Values.Value
	}
	if m, ok := summary.Metrics["tempo_ingestion_duration_seconds"]; ok {
		metrics.IngestionDuration = MetricStats{
			Avg: m.Values.Avg,
			Min: m.Values.Min,
			Med: m.Values.Med,
			Max: m.Values.Max,
			P90: m.Values.P90,
			P95: m.Values.P95,
			P99: m.Values.P99,
		}
	}

	return metrics
}
