package dashboard

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Generator creates HTML dashboards from CSV metrics
type Generator struct {
	config    DashboardConfig
	templates *template.Template
}

// NewGenerator creates a new dashboard generator
func NewGenerator(config DashboardConfig) (*Generator, error) {
	tmpl, err := template.New("dashboard").
		Funcs(GetTemplateFuncs()).
		ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Generator{
		config:    config,
		templates: tmpl,
	}, nil
}

// GenerateFromCSV reads CSV and generates HTML dashboard
func (g *Generator) GenerateFromCSV(csvPath, outputPath string) error {
	// Parse CSV
	metrics, err := parseCSV(csvPath)
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(metrics) == 0 {
		return fmt.Errorf("no metrics found in CSV file")
	}

	// Build dashboard data
	data := g.buildDashboardData(metrics, "")

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Render template
	if err := g.templates.ExecuteTemplate(file, "dashboard.html", data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}

// GenerateComparison generates a comparison dashboard from multiple CSV files
func (g *Generator) GenerateComparison(csvPaths []string, outputPath string) error {
	if len(csvPaths) < 2 {
		return fmt.Errorf("comparison requires at least 2 CSV files")
	}

	// Update config for comparison mode
	g.config.CompareMode = true
	if len(g.config.RunNames) == 0 {
		// Auto-generate run names from file names
		for _, p := range csvPaths {
			name := strings.TrimSuffix(filepath.Base(p), "-metrics.csv")
			name = strings.TrimSuffix(name, ".csv")
			g.config.RunNames = append(g.config.RunNames, name)
		}
	}

	// Parse all CSVs
	var allMetrics []MetricSeries
	for i, csvPath := range csvPaths {
		metrics, err := parseCSV(csvPath)
		if err != nil {
			return fmt.Errorf("failed to parse CSV %s: %w", csvPath, err)
		}

		runName := g.config.RunNames[i]
		for j := range metrics {
			metrics[j].Labels["_run"] = runName
		}
		allMetrics = append(allMetrics, metrics...)
	}

	if len(allMetrics) == 0 {
		return fmt.Errorf("no metrics found in any CSV file")
	}

	// Build dashboard data
	data := g.buildDashboardData(allMetrics, "")
	data.ComparisonSummary = g.buildComparisonSummary(allMetrics)

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Render template
	if err := g.templates.ExecuteTemplate(file, "dashboard.html", data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}

// parseCSV reads the metrics CSV file
func parseCSV(csvPath string) ([]MetricSeries, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has only headers")
	}

	// Skip header, group by query_id + labels
	metricsMap := make(map[string]*MetricSeries)

	for i, record := range records {
		if i == 0 { // skip header
			continue
		}

		if len(record) < 7 {
			continue // skip malformed rows
		}

		// Parse: query_id, metric_name, category, description, timestamp, value, labels
		ts, err := time.Parse("2006-01-02T15:04:05Z", record[4])
		if err != nil {
			continue // skip rows with invalid timestamps
		}

		val, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			continue // skip rows with invalid values
		}

		// Skip NaN and Inf values (can't be serialized to JSON)
		if math.IsNaN(val) || math.IsInf(val, 0) {
			continue
		}

		labels := parseLabels(record[6])
		key := fmt.Sprintf("%s:%s", record[0], record[6]) // query_id:labels

		if _, exists := metricsMap[key]; !exists {
			metricsMap[key] = &MetricSeries{
				QueryID:     record[0],
				Name:        record[1],
				Category:    record[2],
				Description: record[3],
				Labels:      labels,
				DataPoints:  []DataPoint{},
			}
		}

		metricsMap[key].DataPoints = append(metricsMap[key].DataPoints, DataPoint{
			Timestamp: ts,
			Value:     val,
		})
	}

	// Convert to slice and sort data points
	result := make([]MetricSeries, 0, len(metricsMap))
	for _, m := range metricsMap {
		// Sort data points by timestamp
		sort.Slice(m.DataPoints, func(i, j int) bool {
			return m.DataPoints[i].Timestamp.Before(m.DataPoints[j].Timestamp)
		})
		result = append(result, *m)
	}

	return result, nil
}

// parseLabels parses label string into map
func parseLabels(labelStr string) map[string]string {
	labels := make(map[string]string)
	if labelStr == "" {
		return labels
	}

	// Handle quoted labels (CSV can have quoted strings with commas)
	parts := strings.Split(labelStr, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}

	return labels
}

// buildDashboardData organizes metrics into dashboard structure
func (g *Generator) buildDashboardData(metrics []MetricSeries, runName string) *DashboardData {
	// Group by category
	categoryMetrics := make(map[string][]MetricSeries)
	for _, m := range metrics {
		categoryMetrics[m.Category] = append(categoryMetrics[m.Category], m)
	}

	// Build category sections with appropriate chart types
	sections := g.buildCategorySections(categoryMetrics, runName)

	// Calculate summary
	summary := g.buildSummary(metrics)

	// Calculate resource statistics
	resourceSummary := g.buildResourceSummary(metrics)

	return &DashboardData{
		Config:          g.config,
		Summary:         summary,
		Categories:      sections,
		ResourceSummary: resourceSummary,
	}
}

// buildSummary calculates summary statistics
func (g *Generator) buildSummary(metrics []MetricSeries) TestSummary {
	summary := TestSummary{}

	var minTime, maxTime time.Time

	for _, m := range metrics {
		summary.TotalMetrics++
		summary.TotalDataPoints += len(m.DataPoints)

		for _, dp := range m.DataPoints {
			if minTime.IsZero() || dp.Timestamp.Before(minTime) {
				minTime = dp.Timestamp
			}
			if maxTime.IsZero() || dp.Timestamp.After(maxTime) {
				maxTime = dp.Timestamp
			}
		}
	}

	summary.TimeRange = TimeRange{
		Start: minTime,
		End:   maxTime,
	}

	// Update config with test duration if not set
	if g.config.TestDuration == 0 && !minTime.IsZero() && !maxTime.IsZero() {
		g.config.TestDuration = maxTime.Sub(minTime)
	}

	return summary
}

// buildCategorySections builds the category sections for the dashboard
func (g *Generator) buildCategorySections(categoryMetrics map[string][]MetricSeries, runName string) []CategorySection {
	configs := GetCategoryChartConfigs()
	order := GetCategoryOrder()

	var sections []CategorySection
	chartID := 0

	for _, categoryName := range order {
		catConfig, ok := configs[categoryName]
		if !ok {
			continue
		}

		metrics, hasData := categoryMetrics[categoryName]

		section := CategorySection{
			Name:        categoryName,
			Title:       catConfig.Title,
			Description: catConfig.Description,
			Charts:      []ChartConfig{},
		}

		for _, chartDef := range catConfig.Charts {
			chartID++
			chart := ChartConfig{
				ID:          fmt.Sprintf("%s-%d", categoryName, chartID),
				Title:       chartDef.Title,
				Description: chartDef.Description,
				Type:        chartDef.Type,
				Options:     chartDef.Options,
				Series:      []SeriesData{},
				MetricInfo:  []MetricQueryInfo{},
			}

			// Add metric query info for each metric in this chart
			for _, metricName := range chartDef.MetricNames {
				query := GetMetricQuery(metricName)
				chart.MetricInfo = append(chart.MetricInfo, MetricQueryInfo{
					Name:  metricName,
					Query: query,
				})
			}

			if hasData {
				// Find matching metrics for this chart
				for _, metricName := range chartDef.MetricNames {
					for _, m := range metrics {
						if m.Name == metricName {
							series := SeriesData{
								Name:    m.Name,
								Labels:  m.Labels,
								Data:    m.DataPoints,
								RunName: runName,
							}

							// Use run name from labels if in comparison mode
							if g.config.CompareMode {
								if rn, ok := m.Labels["_run"]; ok {
									series.RunName = rn
								}
							}

							chart.Series = append(chart.Series, series)
						}
					}
				}
			}

			section.Charts = append(section.Charts, chart)
		}

		sections = append(sections, section)
	}

	return sections
}

// buildComparisonSummary builds comparison summary for multi-run dashboards
func (g *Generator) buildComparisonSummary(metrics []MetricSeries) *ComparisonSummary {
	if !g.config.CompareMode {
		return nil
	}

	// Key metrics to compare
	keyMetricNames := []string{
		"memory_usage_total",
		"cpu_usage_total",
		"accepted_spans_rate",
		"query_latency_p99",
	}

	summary := &ComparisonSummary{
		RunCount: len(g.config.RunNames),
		RunNames: g.config.RunNames,
	}

	// Group metrics by name and run
	metricsByNameAndRun := make(map[string]map[string][]float64)
	for _, m := range metrics {
		runName := m.Labels["_run"]
		if runName == "" {
			continue
		}

		if _, ok := metricsByNameAndRun[m.Name]; !ok {
			metricsByNameAndRun[m.Name] = make(map[string][]float64)
		}

		for _, dp := range m.DataPoints {
			metricsByNameAndRun[m.Name][runName] = append(metricsByNameAndRun[m.Name][runName], dp.Value)
		}
	}

	// Calculate averages for key metrics
	for _, metricName := range keyMetricNames {
		runData, ok := metricsByNameAndRun[metricName]
		if !ok {
			continue
		}

		cm := ComparisonMetric{
			Name: metricName,
			Unit: GetMetricUnit(metricName),
		}

		var firstAvg float64
		for i, runName := range g.config.RunNames {
			values := runData[runName]
			if len(values) == 0 {
				continue
			}

			// Calculate average
			var sum float64
			for _, v := range values {
				sum += v
			}
			avg := sum / float64(len(values))

			if i == 0 {
				firstAvg = avg
			}

			change := 0.0
			if firstAvg > 0 && i > 0 {
				change = ((avg - firstAvg) / firstAvg) * 100
			}

			cm.Values = append(cm.Values, ComparisonValue{
				RunName: runName,
				Value:   avg,
				Change:  change,
			})
		}

		if len(cm.Values) > 0 {
			summary.KeyMetrics = append(summary.KeyMetrics, cm)
		}
	}

	return summary
}

// Generate is a convenience function that creates a generator and produces a dashboard
func Generate(csvPath, outputPath string, config DashboardConfig) error {
	gen, err := NewGenerator(config)
	if err != nil {
		return err
	}
	return gen.GenerateFromCSV(csvPath, outputPath)
}

// GenerateComparison is a convenience function for comparison dashboards
func GenerateComparison(csvPaths []string, outputPath string, config DashboardConfig) error {
	gen, err := NewGenerator(config)
	if err != nil {
		return err
	}
	return gen.GenerateComparison(csvPaths, outputPath)
}

// buildResourceSummary calculates statistics for resource metrics
func (g *Generator) buildResourceSummary(metrics []MetricSeries) *ResourceSummary {
	summary := &ResourceSummary{
		Memory: []ComponentStats{},
		CPU:    []ComponentStats{},
	}

	// Collect values by component for memory and CPU
	memoryByComponent := make(map[string][]float64)
	cpuByComponent := make(map[string][]float64)

	for _, m := range metrics {
		// Handle memory_usage_by_component
		if m.Name == "memory_usage_by_component" {
			component := m.Labels["component"]
			if component == "" {
				component = "total"
			}
			for _, dp := range m.DataPoints {
				memoryByComponent[component] = append(memoryByComponent[component], dp.Value)
			}
		}

		// Handle cpu_usage_by_component
		if m.Name == "cpu_usage_by_component" {
			component := m.Labels["component"]
			if component == "" {
				component = "total"
			}
			for _, dp := range m.DataPoints {
				cpuByComponent[component] = append(cpuByComponent[component], dp.Value)
			}
		}

		// Handle totals
		if m.Name == "memory_usage_total" {
			for _, dp := range m.DataPoints {
				memoryByComponent["total"] = append(memoryByComponent["total"], dp.Value)
			}
		}
		if m.Name == "cpu_usage_total" {
			for _, dp := range m.DataPoints {
				cpuByComponent["total"] = append(cpuByComponent["total"], dp.Value)
			}
		}
	}

	// Calculate stats for memory
	for component, values := range memoryByComponent {
		if len(values) == 0 {
			continue
		}
		stats := calculateStats(values)
		stats.Component = component
		stats.Unit = "bytes"
		summary.Memory = append(summary.Memory, stats)
	}

	// Calculate stats for CPU
	for component, values := range cpuByComponent {
		if len(values) == 0 {
			continue
		}
		stats := calculateStats(values)
		stats.Component = component
		stats.Unit = "cores"
		summary.CPU = append(summary.CPU, stats)
	}

	// Sort by component name (total first, then alphabetical)
	sortStats := func(stats []ComponentStats) {
		sort.Slice(stats, func(i, j int) bool {
			if stats[i].Component == "total" {
				return true
			}
			if stats[j].Component == "total" {
				return false
			}
			return stats[i].Component < stats[j].Component
		})
	}
	sortStats(summary.Memory)
	sortStats(summary.CPU)

	return summary
}

// calculateStats computes avg, max, min, P95, P99 from a slice of values
func calculateStats(values []float64) ComponentStats {
	if len(values) == 0 {
		return ComponentStats{}
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate sum for average
	var sum float64
	for _, v := range sorted {
		sum += v
	}

	stats := ComponentStats{
		Avg: sum / float64(len(sorted)),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
		P95: percentile(sorted, 0.95),
		P99: percentile(sorted, 0.99),
	}

	return stats
}

// percentile calculates the p-th percentile of a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	// Use linear interpolation
	index := p * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}
