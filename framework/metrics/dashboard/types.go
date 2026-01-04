package dashboard

import (
	"time"
)

// ChartType represents the type of chart to render
type ChartType string

const (
	ChartTypeLine  ChartType = "line"
	ChartTypeArea  ChartType = "area"
	ChartTypeBar   ChartType = "bar"
	ChartTypeStat  ChartType = "stat"
	ChartTypeGauge ChartType = "gauge"
)

// DashboardConfig configures dashboard generation
type DashboardConfig struct {
	Title        string
	ProfileName  string
	TestType     string
	GeneratedAt  time.Time
	TestDuration time.Duration
	// Comparison mode settings
	CompareMode bool
	RunNames    []string // Names for each run in comparison mode
}

// DashboardData holds all data for rendering the dashboard
type DashboardData struct {
	Config     DashboardConfig
	Summary    TestSummary
	Categories []CategorySection
	// For comparison mode
	ComparisonSummary *ComparisonSummary
	// Resource statistics (avg, max, P95, P99)
	ResourceSummary *ResourceSummary
}

// TestSummary provides high-level test information
type TestSummary struct {
	TotalMetrics    int
	TotalDataPoints int
	TimeRange       TimeRange
	Errors          int
}

// TimeRange represents the test time window
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ComparisonSummary contains summary data for multi-run comparisons
type ComparisonSummary struct {
	RunCount   int
	RunNames   []string
	KeyMetrics []ComparisonMetric
}

// ComparisonMetric shows a single metric across multiple runs
type ComparisonMetric struct {
	Name   string
	Unit   string
	Values []ComparisonValue
}

// ComparisonValue represents a value from one run
type ComparisonValue struct {
	RunName string
	Value   float64
	Change  float64 // Percentage change from first run
}

// CategorySection groups charts by category for display
type CategorySection struct {
	Name        string
	Title       string
	Description string
	Charts      []ChartConfig
}

// ChartConfig defines a single chart
type ChartConfig struct {
	ID          string
	Title       string
	Description string
	Type        ChartType
	Series      []SeriesData
	Options     ChartOptions
	// MetricInfo contains the Prometheus metric names and queries used
	MetricInfo []MetricQueryInfo
}

// MetricQueryInfo holds the metric name and PromQL query for display
type MetricQueryInfo struct {
	Name  string // e.g., "accepted_spans_rate"
	Query string // The PromQL query used
}

// SeriesData represents a single data series for a chart
type SeriesData struct {
	Name    string
	Labels  map[string]string
	Data    []DataPoint
	RunName string // For comparison mode
}

// DataPoint is a timestamp-value pair
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// ChartOptions contains chart-specific configuration
type ChartOptions struct {
	YAxisLabel  string
	YAxisUnit   string // bytes, seconds, percent, count
	Stacked     bool
	ShowLegend  bool
	ShowGrid    bool
	ColorScheme string // default, red, blue, green
}

// MetricSeries represents a single metric time-series from CSV
type MetricSeries struct {
	QueryID     string
	Name        string
	Category    string
	Description string
	Labels      map[string]string
	DataPoints  []DataPoint
}

// CSVRecord represents a single row from the metrics CSV
type CSVRecord struct {
	QueryID     string
	MetricName  string
	Category    string
	Description string
	Timestamp   time.Time
	Value       float64
	Labels      map[string]string
}

// ResourceSummary contains aggregated statistics for resource metrics
type ResourceSummary struct {
	Memory []ComponentStats
	CPU    []ComponentStats
}

// ComponentStats contains statistics for a single component
type ComponentStats struct {
	Component string
	Avg       float64
	Max       float64
	Min       float64
	P95       float64
	P99       float64
	Unit      string
}
