package profile

// Profile represents a complete test profile configuration
type Profile struct {
	// Name is the unique identifier for this profile
	Name string `yaml:"name"`

	// Description provides human-readable details about the profile
	Description string `yaml:"description"`

	// Tempo contains Tempo deployment configuration
	Tempo TempoConfig `yaml:"tempo"`

	// K6 contains k6 load test configuration
	K6 K6Config `yaml:"k6"`

	// Storage contains storage configuration (optional)
	Storage *StorageConfig `yaml:"storage,omitempty"`
}

// StorageConfig defines storage settings for the test
type StorageConfig struct {
	// MinioSize is the PVC size for MinIO (e.g., "10Gi")
	// Default: "2Gi"
	MinioSize string `yaml:"minioSize,omitempty"`
}

// TempoConfig defines Tempo deployment settings
type TempoConfig struct {
	// Variant is the deployment type: "monolithic" or "stack"
	Variant string `yaml:"variant"`

	// ReplicationFactor determines how many ingesters must acknowledge data
	// before accepting a span. Only applies to TempoStack (not monolithic).
	// If not set, uses operator default (typically 1).
	ReplicationFactor *int `yaml:"replicationFactor,omitempty"`

	// Resources defines CPU and memory for Tempo pods (optional)
	// If not specified, Tempo will use operator defaults
	Resources *ResourceSpec `yaml:"resources,omitempty"`

	// Overrides defines Tempo overrides configuration (optional)
	Overrides *TempoOverrides `yaml:"overrides,omitempty"`
}

// TempoOverrides defines Tempo limits and overrides
type TempoOverrides struct {
	// MaxTracesPerUser limits the number of active traces per user.
	// Set to 0 for unlimited (prevents "max live traces reached" errors).
	// If nil/not set, uses Tempo's default (which may cause rejections under load).
	MaxTracesPerUser *int `yaml:"maxTracesPerUser,omitempty"`

	// Ingester contains ingester-specific tuning parameters
	Ingester *IngesterConfig `yaml:"ingester,omitempty"`
}

// IngesterConfig defines ingester tuning parameters for performance testing
type IngesterConfig struct {
	// FlushCheckPeriod is the interval for checking flush readiness (e.g., "10s")
	// Default: "10s"
	FlushCheckPeriod string `yaml:"flushCheckPeriod,omitempty"`

	// TraceIdlePeriod is the time before flushing an idle trace to WAL (e.g., "5s")
	// Lower values = faster flush, less memory, but more fractured traces
	// Default: "5s"
	TraceIdlePeriod string `yaml:"traceIdlePeriod,omitempty"`

	// MaxBlockDuration is the maximum time before cutting a block (e.g., "30m")
	// Lower values = more frequent cuts, less memory, more compaction work
	// Default: "30m"
	MaxBlockDuration string `yaml:"maxBlockDuration,omitempty"`

	// ConcurrentFlushes is the number of parallel flush operations
	// Higher values = faster write throughput, more resource usage
	// Default: 4
	ConcurrentFlushes *int `yaml:"concurrentFlushes,omitempty"`
}

// HasResources returns true if custom resources are configured
func (t *TempoConfig) HasResources() bool {
	return t.Resources != nil && (t.Resources.Memory != "" || t.Resources.CPU != "")
}

// ResourceSpec defines Kubernetes resource requirements
type ResourceSpec struct {
	// Memory limit and request (e.g., "8Gi")
	Memory string `yaml:"memory"`

	// CPU limit and request (e.g., "1000m")
	CPU string `yaml:"cpu"`
}

// K6Config defines k6 load test settings
type K6Config struct {
	// Duration of the test (e.g., "5m")
	Duration string `yaml:"duration"`

	// VUs defines virtual user counts
	VUs VUsConfig `yaml:"vus"`

	// Ingestion contains trace ingestion settings
	Ingestion IngestionConfig `yaml:"ingestion"`

	// Query contains query test settings
	Query QueryConfig `yaml:"query"`
}

// VUsConfig defines virtual user range
type VUsConfig struct {
	// Min is the minimum number of VUs
	Min int `yaml:"min"`

	// Max is the maximum number of VUs
	Max int `yaml:"max"`
}

// IngestionConfig defines trace ingestion parameters
type IngestionConfig struct {
	// MBPerSecond is the target ingestion rate in megabytes per second
	MBPerSecond float64 `yaml:"mbPerSecond"`

	// TraceProfile determines trace complexity (small, medium, large, xlarge)
	TraceProfile string `yaml:"traceProfile"`
}

// QueryConfig defines query test parameters
type QueryConfig struct {
	// QueriesPerSecond is the target query rate
	QueriesPerSecond int `yaml:"queriesPerSecond"`
}
