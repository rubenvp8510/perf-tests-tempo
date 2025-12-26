package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Default retry configuration values
const (
	DefaultMaxAttempts  = 3
	DefaultInitialDelay = 1 * time.Second
	DefaultMaxDelay     = 30 * time.Second
	DefaultMultiplier   = 2.0
	DefaultJitter       = 0.1
)

// Config holds retry configuration
type Config struct {
	// MaxAttempts is the maximum number of attempts (including the first)
	MaxAttempts int

	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases after each retry
	Multiplier float64

	// Jitter adds randomness to the delay (0.0-1.0, as a fraction of delay)
	Jitter float64

	// RetryIf is a function that determines if an error should be retried
	// If nil, all errors are retried
	RetryIf func(error) bool

	// OnRetry is called before each retry with the attempt number and error
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:  DefaultMaxAttempts,
		InitialDelay: DefaultInitialDelay,
		MaxDelay:     DefaultMaxDelay,
		Multiplier:   DefaultMultiplier,
		Jitter:       DefaultJitter,
	}
}

// Option is a function that modifies Config
type Option func(*Config)

// WithMaxAttempts sets the maximum number of attempts
func WithMaxAttempts(n int) Option {
	return func(c *Config) {
		c.MaxAttempts = n
	}
}

// WithInitialDelay sets the initial delay
func WithInitialDelay(d time.Duration) Option {
	return func(c *Config) {
		c.InitialDelay = d
	}
}

// WithMaxDelay sets the maximum delay
func WithMaxDelay(d time.Duration) Option {
	return func(c *Config) {
		c.MaxDelay = d
	}
}

// WithMultiplier sets the backoff multiplier
func WithMultiplier(m float64) Option {
	return func(c *Config) {
		c.Multiplier = m
	}
}

// WithJitter sets the jitter factor
func WithJitter(j float64) Option {
	return func(c *Config) {
		c.Jitter = j
	}
}

// WithRetryIf sets the retry predicate function
func WithRetryIf(fn func(error) bool) Option {
	return func(c *Config) {
		c.RetryIf = fn
	}
}

// WithOnRetry sets the retry callback function
func WithOnRetry(fn func(attempt int, err error, delay time.Duration)) Option {
	return func(c *Config) {
		c.OnRetry = fn
	}
}

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// Retryable wraps an error to mark it as retryable
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// IsRetryable returns true if the error is marked as retryable
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}

// PermanentError wraps an error to indicate it should not be retried
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Permanent wraps an error to mark it as permanent (non-retryable)
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// IsPermanent returns true if the error is marked as permanent
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// Do executes the function with retries according to the configuration
func Do(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Check if error is permanent
		if IsPermanent(lastErr) {
			var pe *PermanentError
			if errors.As(lastErr, &pe) {
				return pe.Err
			}
			return lastErr
		}

		// Check if we should retry this error
		if cfg.RetryIf != nil && !cfg.RetryIf(lastErr) {
			return lastErr
		}

		// Check if we've exhausted all attempts
		if attempt >= cfg.MaxAttempts {
			break
		}

		// Calculate delay with jitter
		actualDelay := delay
		if cfg.Jitter > 0 {
			jitterRange := float64(delay) * cfg.Jitter
			actualDelay = time.Duration(float64(delay) + (rand.Float64()*2-1)*jitterRange)
		}

		// Call retry callback if configured
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, lastErr, actualDelay)
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(actualDelay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return lastErr
}

// DoWithData executes the function with retries and returns a result
func DoWithData[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	var result T
	err := Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}, opts...)
	return result, err
}

// calculateBackoff calculates the delay for a given attempt using exponential backoff
func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	if attempt <= 1 {
		return initialDelay
	}

	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	return time.Duration(delay)
}
