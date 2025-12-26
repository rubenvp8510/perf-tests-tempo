package metrics

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewExporter_AutoDetectCSV(t *testing.T) {
	exp := NewExporter("output.csv", "")
	if _, ok := exp.(*CSVExporter); !ok {
		t.Error("expected CSVExporter for .csv extension")
	}
}

func TestNewExporter_AutoDetectJSON(t *testing.T) {
	exp := NewExporter("output.json", "")
	if _, ok := exp.(*JSONExporter); !ok {
		t.Error("expected JSONExporter for .json extension")
	}
}

func TestNewExporter_ExplicitFormat(t *testing.T) {
	exp := NewExporter("output.txt", FormatJSON)
	if _, ok := exp.(*JSONExporter); !ok {
		t.Error("expected JSONExporter when FormatJSON is specified")
	}
}

func TestCSVExporter_Export(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "metrics.csv")

	exporter := NewCSVExporter(outputPath)

	now := time.Now()
	results := []MetricResult{
		{
			QueryID:     "query1",
			MetricName:  "test_metric",
			Category:    "test",
			Description: "A test metric",
			Labels:      map[string]string{"pod": "pod-1", "namespace": "default"},
			DataPoints: []DataPoint{
				{Timestamp: now, Value: 1.5},
				{Timestamp: now.Add(time.Minute), Value: 2.5},
			},
		},
		{
			QueryID:     "query2",
			MetricName:  "error_metric",
			Category:    "test",
			Description: "A metric with error",
			Error:       errors.New("query failed"),
		},
	}

	err := exporter.Export(results)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file was created
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	// Should have header + 2 data rows (error result is skipped)
	if len(records) != 3 {
		t.Errorf("expected 3 rows (header + 2 data points), got %d", len(records))
	}

	// Check header
	expectedHeader := []string{"query_id", "metric_name", "category", "description", "timestamp", "value", "labels"}
	for i, h := range expectedHeader {
		if records[0][i] != h {
			t.Errorf("expected header[%d] = %q, got %q", i, h, records[0][i])
		}
	}

	// Check first data row
	if records[1][0] != "query1" {
		t.Errorf("expected query_id 'query1', got %q", records[1][0])
	}
	if records[1][1] != "test_metric" {
		t.Errorf("expected metric_name 'test_metric', got %q", records[1][1])
	}
}

func TestJSONExporter_Export(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "metrics.json")

	exporter := NewJSONExporter(outputPath)

	now := time.Now()
	results := []MetricResult{
		{
			QueryID:     "query1",
			MetricName:  "test_metric",
			Category:    "ingestion",
			Description: "A test metric",
			Labels:      map[string]string{"pod": "pod-1"},
			DataPoints: []DataPoint{
				{Timestamp: now, Value: 1.5},
				{Timestamp: now.Add(time.Minute), Value: 2.5},
			},
		},
		{
			QueryID:     "query2",
			MetricName:  "error_metric",
			Category:    "query",
			Description: "A metric with error",
			Error:       errors.New("query failed"),
		},
	}

	err := exporter.Export(results)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file was created and is valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var report JSONExportReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify report contents
	if report.TotalMetrics != 2 {
		t.Errorf("expected TotalMetrics 2, got %d", report.TotalMetrics)
	}
	if report.TotalPoints != 2 {
		t.Errorf("expected TotalPoints 2, got %d", report.TotalPoints)
	}
	if report.Errors != 1 {
		t.Errorf("expected Errors 1, got %d", report.Errors)
	}

	// Verify summary
	if report.Summary == nil {
		t.Fatal("expected Summary to be non-nil")
	}
	if report.Summary.ByCategory["ingestion"].MetricCount != 1 {
		t.Errorf("expected ingestion category to have 1 metric")
	}
	if report.Summary.ByCategory["query"].ErrorCount != 1 {
		t.Errorf("expected query category to have 1 error")
	}

	// Verify first metric
	if len(report.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(report.Metrics))
	}
	if report.Metrics[0].QueryID != "query1" {
		t.Errorf("expected first metric query_id 'query1', got %q", report.Metrics[0].QueryID)
	}
	if len(report.Metrics[0].DataPoints) != 2 {
		t.Errorf("expected 2 data points, got %d", len(report.Metrics[0].DataPoints))
	}

	// Verify error metric
	if report.Metrics[1].Error != "query failed" {
		t.Errorf("expected error message 'query failed', got %q", report.Metrics[1].Error)
	}
}

func TestJSONExporter_WithPrettyPrint(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with pretty print
	prettyPath := filepath.Join(tmpDir, "pretty.json")
	prettyExporter := NewJSONExporter(prettyPath).WithPrettyPrint(true)

	results := []MetricResult{{
		QueryID:    "test",
		MetricName: "test",
		DataPoints: []DataPoint{{Timestamp: time.Now(), Value: 1}},
	}}

	if err := prettyExporter.Export(results); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	prettyData, _ := os.ReadFile(prettyPath)

	// Test without pretty print
	compactPath := filepath.Join(tmpDir, "compact.json")
	compactExporter := NewJSONExporter(compactPath).WithPrettyPrint(false)

	if err := compactExporter.Export(results); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	compactData, _ := os.ReadFile(compactPath)

	// Pretty printed should be longer due to indentation
	if len(prettyData) <= len(compactData) {
		t.Error("expected pretty printed JSON to be longer than compact")
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "single",
			labels:   map[string]string{"key": "value"},
			expected: "key=value",
		},
		{
			name:     "multiple sorted",
			labels:   map[string]string{"z": "3", "a": "1", "m": "2"},
			expected: "a=1,m=2,z=3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabels(tt.labels)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCSVExporter_EmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.csv")

	exporter := NewCSVExporter(outputPath)
	err := exporter.Export([]MetricResult{})

	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should have just the header
	file, _ := os.Open(outputPath)
	defer file.Close()
	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	if len(records) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(records))
	}
}

func TestJSONExporter_EmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.json")

	exporter := NewJSONExporter(outputPath)
	err := exporter.Export([]MetricResult{})

	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, _ := os.ReadFile(outputPath)
	var report JSONExportReport
	json.Unmarshal(data, &report)

	if report.TotalMetrics != 0 {
		t.Errorf("expected TotalMetrics 0, got %d", report.TotalMetrics)
	}
	if len(report.Metrics) != 0 {
		t.Errorf("expected empty Metrics, got %d", len(report.Metrics))
	}
}
