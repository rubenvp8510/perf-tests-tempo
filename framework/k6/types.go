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
}

// Result holds the result of a k6 test execution
type Result struct {
	Success  bool
	Output   string
	Duration time.Duration
	Error    error
}
