package config

import (
	"os"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.CRDeletionTimeout != DefaultCRDeletionTimeout {
		t.Errorf("expected CRDeletionTimeout %v, got %v", DefaultCRDeletionTimeout, cfg.CRDeletionTimeout)
	}
	if cfg.PodReadyTimeout != DefaultPodReadyTimeout {
		t.Errorf("expected PodReadyTimeout %v, got %v", DefaultPodReadyTimeout, cfg.PodReadyTimeout)
	}
	if cfg.JobTimeout != DefaultJobTimeout {
		t.Errorf("expected JobTimeout %v, got %v", DefaultJobTimeout, cfg.JobTimeout)
	}
	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("expected HTTPTimeout %v, got %v", DefaultHTTPTimeout, cfg.HTTPTimeout)
	}
	if cfg.MaxConcurrentQueries != DefaultMaxConcurrentQueries {
		t.Errorf("expected MaxConcurrentQueries %d, got %d", DefaultMaxConcurrentQueries, cfg.MaxConcurrentQueries)
	}
}

func TestFromEnv_Defaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv(EnvCRDeletionTimeout)
	os.Unsetenv(EnvPodReadyTimeout)
	os.Unsetenv(EnvJobTimeout)
	os.Unsetenv(EnvHTTPTimeout)
	os.Unsetenv(EnvMaxConcurrentQuery)

	cfg := FromEnv()

	if cfg.CRDeletionTimeout != DefaultCRDeletionTimeout {
		t.Errorf("expected CRDeletionTimeout %v, got %v", DefaultCRDeletionTimeout, cfg.CRDeletionTimeout)
	}
}

func TestFromEnv_CustomValues(t *testing.T) {
	// Set custom env vars
	os.Setenv(EnvCRDeletionTimeout, "5m")
	os.Setenv(EnvPodReadyTimeout, "3m")
	os.Setenv(EnvJobTimeout, "1h")
	os.Setenv(EnvHTTPTimeout, "2m")
	os.Setenv(EnvMaxConcurrentQuery, "10")
	defer func() {
		os.Unsetenv(EnvCRDeletionTimeout)
		os.Unsetenv(EnvPodReadyTimeout)
		os.Unsetenv(EnvJobTimeout)
		os.Unsetenv(EnvHTTPTimeout)
		os.Unsetenv(EnvMaxConcurrentQuery)
	}()

	cfg := FromEnv()

	if cfg.CRDeletionTimeout != 5*time.Minute {
		t.Errorf("expected CRDeletionTimeout 5m, got %v", cfg.CRDeletionTimeout)
	}
	if cfg.PodReadyTimeout != 3*time.Minute {
		t.Errorf("expected PodReadyTimeout 3m, got %v", cfg.PodReadyTimeout)
	}
	if cfg.JobTimeout != 1*time.Hour {
		t.Errorf("expected JobTimeout 1h, got %v", cfg.JobTimeout)
	}
	if cfg.HTTPTimeout != 2*time.Minute {
		t.Errorf("expected HTTPTimeout 2m, got %v", cfg.HTTPTimeout)
	}
	if cfg.MaxConcurrentQueries != 10 {
		t.Errorf("expected MaxConcurrentQueries 10, got %d", cfg.MaxConcurrentQueries)
	}
}

func TestFromEnv_InvalidValues(t *testing.T) {
	// Set invalid env vars - should fall back to defaults
	os.Setenv(EnvCRDeletionTimeout, "invalid")
	os.Setenv(EnvMaxConcurrentQuery, "not-a-number")
	defer func() {
		os.Unsetenv(EnvCRDeletionTimeout)
		os.Unsetenv(EnvMaxConcurrentQuery)
	}()

	cfg := FromEnv()

	// Should fall back to defaults for invalid values
	if cfg.CRDeletionTimeout != DefaultCRDeletionTimeout {
		t.Errorf("expected default CRDeletionTimeout, got %v", cfg.CRDeletionTimeout)
	}
	if cfg.MaxConcurrentQueries != DefaultMaxConcurrentQueries {
		t.Errorf("expected default MaxConcurrentQueries, got %d", cfg.MaxConcurrentQueries)
	}
}

func TestWithCRDeletionTimeout(t *testing.T) {
	cfg := Default()
	newTimeout := 10 * time.Minute
	newCfg := cfg.WithCRDeletionTimeout(newTimeout)

	// Original should be unchanged
	if cfg.CRDeletionTimeout != DefaultCRDeletionTimeout {
		t.Error("original config was modified")
	}

	// New config should have new value
	if newCfg.CRDeletionTimeout != newTimeout {
		t.Errorf("expected CRDeletionTimeout %v, got %v", newTimeout, newCfg.CRDeletionTimeout)
	}
}

func TestWithPodReadyTimeout(t *testing.T) {
	cfg := Default()
	newTimeout := 5 * time.Minute
	newCfg := cfg.WithPodReadyTimeout(newTimeout)

	if cfg.PodReadyTimeout != DefaultPodReadyTimeout {
		t.Error("original config was modified")
	}
	if newCfg.PodReadyTimeout != newTimeout {
		t.Errorf("expected PodReadyTimeout %v, got %v", newTimeout, newCfg.PodReadyTimeout)
	}
}

func TestWithJobTimeout(t *testing.T) {
	cfg := Default()
	newTimeout := 1 * time.Hour
	newCfg := cfg.WithJobTimeout(newTimeout)

	if cfg.JobTimeout != DefaultJobTimeout {
		t.Error("original config was modified")
	}
	if newCfg.JobTimeout != newTimeout {
		t.Errorf("expected JobTimeout %v, got %v", newTimeout, newCfg.JobTimeout)
	}
}

func TestWithHTTPTimeout(t *testing.T) {
	cfg := Default()
	newTimeout := 2 * time.Minute
	newCfg := cfg.WithHTTPTimeout(newTimeout)

	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Error("original config was modified")
	}
	if newCfg.HTTPTimeout != newTimeout {
		t.Errorf("expected HTTPTimeout %v, got %v", newTimeout, newCfg.HTTPTimeout)
	}
}

func TestWithMaxConcurrentQueries(t *testing.T) {
	cfg := Default()
	newMax := 20
	newCfg := cfg.WithMaxConcurrentQueries(newMax)

	if cfg.MaxConcurrentQueries != DefaultMaxConcurrentQueries {
		t.Error("original config was modified")
	}
	if newCfg.MaxConcurrentQueries != newMax {
		t.Errorf("expected MaxConcurrentQueries %d, got %d", newMax, newCfg.MaxConcurrentQueries)
	}
}

func TestChainedWith(t *testing.T) {
	cfg := Default().
		WithCRDeletionTimeout(5 * time.Minute).
		WithPodReadyTimeout(3 * time.Minute).
		WithJobTimeout(1 * time.Hour).
		WithMaxConcurrentQueries(10)

	if cfg.CRDeletionTimeout != 5*time.Minute {
		t.Errorf("expected CRDeletionTimeout 5m, got %v", cfg.CRDeletionTimeout)
	}
	if cfg.PodReadyTimeout != 3*time.Minute {
		t.Errorf("expected PodReadyTimeout 3m, got %v", cfg.PodReadyTimeout)
	}
	if cfg.JobTimeout != 1*time.Hour {
		t.Errorf("expected JobTimeout 1h, got %v", cfg.JobTimeout)
	}
	if cfg.MaxConcurrentQueries != 10 {
		t.Errorf("expected MaxConcurrentQueries 10, got %d", cfg.MaxConcurrentQueries)
	}
}
