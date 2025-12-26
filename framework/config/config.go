package config

import (
	"os"
	"strconv"
	"time"
)

// Default timeouts used throughout the framework
const (
	// DefaultCRDeletionTimeout is the default timeout for waiting for CR deletion
	DefaultCRDeletionTimeout = 120 * time.Second

	// DefaultCRDeletionPollInterval is the default interval for polling CR deletion status
	DefaultCRDeletionPollInterval = 2 * time.Second

	// DefaultPodReadyTimeout is the default timeout for waiting for pods to be ready
	DefaultPodReadyTimeout = 120 * time.Second

	// DefaultPodReadyPollInterval is the default interval for polling pod readiness
	DefaultPodReadyPollInterval = 5 * time.Second

	// DefaultNamespaceTimeout is the default timeout for namespace operations
	DefaultNamespaceTimeout = 120 * time.Second

	// DefaultNamespacePollInterval is the default interval for polling namespace status
	DefaultNamespacePollInterval = 2 * time.Second

	// DefaultJobTimeout is the default timeout for k6 job completion
	DefaultJobTimeout = 30 * time.Minute

	// DefaultJobPollInterval is the default interval for polling job status
	DefaultJobPollInterval = 5 * time.Second

	// DefaultHTTPTimeout is the default timeout for HTTP requests
	DefaultHTTPTimeout = 60 * time.Second

	// DefaultMetricsQueryStep is the default step for Prometheus range queries
	DefaultMetricsQueryStep = 15 * time.Second

	// DefaultMaxConcurrentQueries is the default max concurrent Prometheus queries
	DefaultMaxConcurrentQueries = 5
)

// Environment variable names for configuration overrides
const (
	EnvCRDeletionTimeout  = "TEMPO_PERF_CR_DELETION_TIMEOUT"
	EnvPodReadyTimeout    = "TEMPO_PERF_POD_READY_TIMEOUT"
	EnvJobTimeout         = "TEMPO_PERF_JOB_TIMEOUT"
	EnvHTTPTimeout        = "TEMPO_PERF_HTTP_TIMEOUT"
	EnvMaxConcurrentQuery = "TEMPO_PERF_MAX_CONCURRENT_QUERIES"
)

// Config holds framework configuration with optional overrides
type Config struct {
	// Timeouts
	CRDeletionTimeout      time.Duration
	CRDeletionPollInterval time.Duration
	PodReadyTimeout        time.Duration
	PodReadyPollInterval   time.Duration
	NamespaceTimeout       time.Duration
	NamespacePollInterval  time.Duration
	JobTimeout             time.Duration
	JobPollInterval        time.Duration
	HTTPTimeout            time.Duration

	// Metrics
	MetricsQueryStep     time.Duration
	MaxConcurrentQueries int
}

// Default returns a Config with all default values
func Default() *Config {
	return &Config{
		CRDeletionTimeout:      DefaultCRDeletionTimeout,
		CRDeletionPollInterval: DefaultCRDeletionPollInterval,
		PodReadyTimeout:        DefaultPodReadyTimeout,
		PodReadyPollInterval:   DefaultPodReadyPollInterval,
		NamespaceTimeout:       DefaultNamespaceTimeout,
		NamespacePollInterval:  DefaultNamespacePollInterval,
		JobTimeout:             DefaultJobTimeout,
		JobPollInterval:        DefaultJobPollInterval,
		HTTPTimeout:            DefaultHTTPTimeout,
		MetricsQueryStep:       DefaultMetricsQueryStep,
		MaxConcurrentQueries:   DefaultMaxConcurrentQueries,
	}
}

// FromEnv returns a Config with values from environment variables, falling back to defaults
func FromEnv() *Config {
	cfg := Default()

	if v := os.Getenv(EnvCRDeletionTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.CRDeletionTimeout = d
		}
	}

	if v := os.Getenv(EnvPodReadyTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.PodReadyTimeout = d
		}
	}

	if v := os.Getenv(EnvJobTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JobTimeout = d
		}
	}

	if v := os.Getenv(EnvHTTPTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.HTTPTimeout = d
		}
	}

	if v := os.Getenv(EnvMaxConcurrentQuery); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxConcurrentQueries = n
		}
	}

	return cfg
}

// WithCRDeletionTimeout returns a copy with updated CR deletion timeout
func (c *Config) WithCRDeletionTimeout(d time.Duration) *Config {
	cp := *c
	cp.CRDeletionTimeout = d
	return &cp
}

// WithPodReadyTimeout returns a copy with updated pod ready timeout
func (c *Config) WithPodReadyTimeout(d time.Duration) *Config {
	cp := *c
	cp.PodReadyTimeout = d
	return &cp
}

// WithJobTimeout returns a copy with updated job timeout
func (c *Config) WithJobTimeout(d time.Duration) *Config {
	cp := *c
	cp.JobTimeout = d
	return &cp
}

// WithHTTPTimeout returns a copy with updated HTTP timeout
func (c *Config) WithHTTPTimeout(d time.Duration) *Config {
	cp := *c
	cp.HTTPTimeout = d
	return &cp
}

// WithMaxConcurrentQueries returns a copy with updated max concurrent queries
func (c *Config) WithMaxConcurrentQueries(n int) *Config {
	cp := *c
	cp.MaxConcurrentQueries = n
	return &cp
}
