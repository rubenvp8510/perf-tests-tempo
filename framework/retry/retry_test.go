package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_Success(t *testing.T) {
	callCount := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestDo_RetryOnError(t *testing.T) {
	callCount := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("transient error")
		}
		return nil
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond))

	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	callCount := 0
	testErr := errors.New("persistent error")
	err := Do(context.Background(), func(ctx context.Context) error {
		callCount++
		return testErr
	}, WithMaxAttempts(3), WithInitialDelay(1*time.Millisecond))

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_PermanentError(t *testing.T) {
	callCount := 0
	testErr := errors.New("permanent error")
	err := Do(context.Background(), func(ctx context.Context) error {
		callCount++
		return Permanent(testErr)
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond))

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries for permanent error), got %d", callCount)
	}
}

func TestDo_RetryIfPredicate(t *testing.T) {
	callCount := 0
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	// First test: retryable error should be retried
	callCount = 0
	err := Do(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return retryableErr
		}
		return nil
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond),
		WithRetryIf(func(e error) bool { return errors.Is(e, retryableErr) }))

	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	// Second test: non-retryable error should not be retried
	callCount = 0
	err = Do(context.Background(), func(ctx context.Context) error {
		callCount++
		return nonRetryableErr
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond),
		WithRetryIf(func(e error) bool { return errors.Is(e, retryableErr) }))

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries for non-retryable error), got %d", callCount)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, func(ctx context.Context) error {
		callCount++
		return errors.New("error")
	}, WithMaxAttempts(100), WithInitialDelay(20*time.Millisecond))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestDo_OnRetryCallback(t *testing.T) {
	retryCount := 0
	Do(context.Background(), func(ctx context.Context) error {
		if retryCount < 2 {
			return errors.New("error")
		}
		return nil
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond),
		WithOnRetry(func(attempt int, err error, delay time.Duration) {
			retryCount++
		}))

	if retryCount != 2 {
		t.Errorf("expected OnRetry called 2 times, got %d", retryCount)
	}
}

func TestDoWithData_Success(t *testing.T) {
	result, err := DoWithData(context.Background(), func(ctx context.Context) (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got '%s'", result)
	}
}

func TestDoWithData_RetryAndSuccess(t *testing.T) {
	callCount := 0
	result, err := DoWithData(context.Background(), func(ctx context.Context) (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("error")
		}
		return 42, nil
	}, WithMaxAttempts(5), WithInitialDelay(1*time.Millisecond))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestRetryable(t *testing.T) {
	err := errors.New("test error")
	retryableErr := Retryable(err)

	if !IsRetryable(retryableErr) {
		t.Error("expected error to be retryable")
	}

	if IsRetryable(err) {
		t.Error("expected plain error to not be retryable")
	}

	if Retryable(nil) != nil {
		t.Error("expected Retryable(nil) to return nil")
	}
}

func TestPermanent(t *testing.T) {
	err := errors.New("test error")
	permanentErr := Permanent(err)

	if !IsPermanent(permanentErr) {
		t.Error("expected error to be permanent")
	}

	if IsPermanent(err) {
		t.Error("expected plain error to not be permanent")
	}

	if Permanent(nil) != nil {
		t.Error("expected Permanent(nil) to return nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != DefaultMaxAttempts {
		t.Errorf("expected MaxAttempts %d, got %d", DefaultMaxAttempts, cfg.MaxAttempts)
	}
	if cfg.InitialDelay != DefaultInitialDelay {
		t.Errorf("expected InitialDelay %v, got %v", DefaultInitialDelay, cfg.InitialDelay)
	}
	if cfg.MaxDelay != DefaultMaxDelay {
		t.Errorf("expected MaxDelay %v, got %v", DefaultMaxDelay, cfg.MaxDelay)
	}
	if cfg.Multiplier != DefaultMultiplier {
		t.Errorf("expected Multiplier %v, got %v", DefaultMultiplier, cfg.Multiplier)
	}
}
