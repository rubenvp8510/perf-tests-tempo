package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExportFormat represents the output format for metrics export
type ExportFormat string

const (
	// FormatCSV exports metrics in CSV format
	FormatCSV ExportFormat = "csv"
	// FormatJSON exports metrics in JSON format
	FormatJSON ExportFormat = "json"
)

// Exporter is the interface for metric exporters
type Exporter interface {
	Export(results []MetricResult) error
}

// NewExporter creates an exporter based on the file extension or specified format
func NewExporter(outputPath string, format ExportFormat) Exporter {
	if format == "" {
		// Auto-detect format from file extension
		ext := strings.ToLower(filepath.Ext(outputPath))
		switch ext {
		case ".json":
			format = FormatJSON
		default:
			format = FormatCSV
		}
	}

	switch format {
	case FormatJSON:
		return NewJSONExporter(outputPath)
	default:
		return NewCSVExporter(outputPath)
	}
}

// CSVExporter handles exporting metrics to CSV format
type CSVExporter struct {
	outputPath string
}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter(outputPath string) *CSVExporter {
	return &CSVExporter{
		outputPath: outputPath,
	}
}

// Export exports metric results to CSV
func (e *CSVExporter) Export(results []MetricResult) error {
	file, err := os.Create(e.outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"query_id",
		"metric_name",
		"category",
		"description",
		"timestamp",
		"value",
		"labels",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	rowCount := 0
	for _, result := range results {
		// Skip results with errors
		if result.Error != nil {
			continue
		}

		// Format labels as key=value pairs
		labelStr := formatLabels(result.Labels)

		for _, dp := range result.DataPoints {
			row := []string{
				result.QueryID,
				result.MetricName,
				result.Category,
				result.Description,
				dp.Timestamp.Format("2006-01-02T15:04:05Z"),
				fmt.Sprintf("%.6f", dp.Value),
				labelStr,
			}

			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
			rowCount++
		}
	}

	fmt.Printf("📝 Wrote %d data points to CSV\n", rowCount)

	return nil
}

// JSONExporter handles exporting metrics to JSON format
type JSONExporter struct {
	outputPath string
	pretty     bool
}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter(outputPath string) *JSONExporter {
	return &JSONExporter{
		outputPath: outputPath,
		pretty:     true,
	}
}

// WithPrettyPrint sets whether to use indented JSON output
func (e *JSONExporter) WithPrettyPrint(pretty bool) *JSONExporter {
	e.pretty = pretty
	return e
}

// JSONMetricResult is the JSON-serializable version of MetricResult
type JSONMetricResult struct {
	QueryID     string            `json:"query_id"`
	MetricName  string            `json:"metric_name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Labels      map[string]string `json:"labels,omitempty"`
	DataPoints  []JSONDataPoint   `json:"data_points"`
	Error       string            `json:"error,omitempty"`
}

// JSONDataPoint is the JSON-serializable version of DataPoint
type JSONDataPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// JSONExportReport is the top-level JSON export structure
type JSONExportReport struct {
	ExportedAt   string             `json:"exported_at"`
	TotalMetrics int                `json:"total_metrics"`
	TotalPoints  int                `json:"total_points"`
	Errors       int                `json:"errors"`
	Summary      *JSONExportSummary `json:"summary,omitempty"`
	Metrics      []JSONMetricResult `json:"metrics"`
}

// JSONExportSummary contains statistical summary of the metrics
type JSONExportSummary struct {
	ByCategory map[string]CategorySummary `json:"by_category"`
}

// CategorySummary contains summary for a single category
type CategorySummary struct {
	MetricCount int `json:"metric_count"`
	PointCount  int `json:"point_count"`
	ErrorCount  int `json:"error_count"`
}

// Export exports metric results to JSON
func (e *JSONExporter) Export(results []MetricResult) error {
	file, err := os.Create(e.outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Build the report
	report := JSONExportReport{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Metrics:    make([]JSONMetricResult, 0, len(results)),
		Summary: &JSONExportSummary{
			ByCategory: make(map[string]CategorySummary),
		},
	}

	for _, result := range results {
		jsonResult := JSONMetricResult{
			QueryID:     result.QueryID,
			MetricName:  result.MetricName,
			Description: result.Description,
			Category:    result.Category,
			Labels:      result.Labels,
			DataPoints:  make([]JSONDataPoint, 0, len(result.DataPoints)),
		}

		if result.Error != nil {
			jsonResult.Error = result.Error.Error()
			report.Errors++
		}

		for _, dp := range result.DataPoints {
			jsonResult.DataPoints = append(jsonResult.DataPoints, JSONDataPoint{
				Timestamp: dp.Timestamp.Format(time.RFC3339),
				Value:     dp.Value,
			})
			report.TotalPoints++
		}

		report.Metrics = append(report.Metrics, jsonResult)
		report.TotalMetrics++

		// Update category summary
		cat := result.Category
		if cat == "" {
			cat = "uncategorized"
		}
		summary := report.Summary.ByCategory[cat]
		summary.MetricCount++
		summary.PointCount += len(result.DataPoints)
		if result.Error != nil {
			summary.ErrorCount++
		}
		report.Summary.ByCategory[cat] = summary
	}

	// Encode to JSON
	encoder := json.NewEncoder(file)
	if e.pretty {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Printf("📝 Wrote %d metrics with %d data points to JSON\n", report.TotalMetrics, report.TotalPoints)

	return nil
}

// formatLabels formats label map as comma-separated key=value pairs
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(labels))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, labels[k]))
	}

	return strings.Join(pairs, ",")
}
