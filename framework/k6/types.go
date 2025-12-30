package k6

import "time"

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

const (
	// DefaultImage is the default xk6-tempo image
	DefaultImage = "quay.io/rvargasp/xk6-tempo:latest"

	// ScriptsConfigMap is the name of the ConfigMap containing k6 scripts
	ScriptsConfigMap = "k6-scripts"

	// JobTimeout is the maximum time to wait for a k6 job to complete
	JobTimeout = 30 * time.Minute
)

// Config holds configuration for k6 test execution
type Config struct {
	// Size is the t-shirt size (small, medium, large, xlarge)
	Size Size

	// Image is the k6 container image (optional, defaults to xk6-tempo image)
	Image string

	// Custom overrides (optional)
	MBPerSecond      float64
	QueriesPerSecond int
	Duration         string
	VUsMin           int
	VUsMax           int
	TraceProfile     string

	// Endpoints (auto-discovered if empty)
	TempoEndpoint      string
	TempoQueryEndpoint string
	TempoTenant        string
	TempoToken         string
}

// Result holds the result of a k6 test execution
type Result struct {
	Success  bool
	Output   string
	Duration time.Duration
	Error    error
}
