package metrics

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

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
