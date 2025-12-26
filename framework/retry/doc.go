// Package retry provides exponential backoff retry functionality for transient failures.
//
// # Basic Usage
//
// Use Do to retry a function until it succeeds or max attempts is reached:
//
//	err := retry.Do(ctx, func(ctx context.Context) error {
//	    return someOperation()
//	}, retry.WithMaxAttempts(5))
//
// # Configuration Options
//
// Customize retry behavior with options:
//
//	err := retry.Do(ctx, fn,
//	    retry.WithMaxAttempts(5),
//	    retry.WithInitialDelay(time.Second),
//	    retry.WithMaxDelay(30*time.Second),
//	    retry.WithMultiplier(2.0),
//	    retry.WithJitter(0.1),
//	)
//
// # Retry Predicates
//
// Control which errors should be retried:
//
//	err := retry.Do(ctx, fn,
//	    retry.WithRetryIf(func(err error) bool {
//	        // Only retry on network errors
//	        return errors.Is(err, io.ErrUnexpectedEOF)
//	    }),
//	)
//
// # Permanent Errors
//
// Mark errors as permanent to stop retrying immediately:
//
//	func operation() error {
//	    if isValidationError {
//	        return retry.Permanent(err) // Won't retry
//	    }
//	    return err // Will retry
//	}
//
// # Retry Callbacks
//
// Get notified before each retry:
//
//	err := retry.Do(ctx, fn,
//	    retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
//	        log.Printf("Attempt %d failed: %v, retrying in %v", attempt, err, delay)
//	    }),
//	)
//
// # Returning Values
//
// Use DoWithData to retry and return a value:
//
//	result, err := retry.DoWithData(ctx, func(ctx context.Context) (string, error) {
//	    return fetchData()
//	})
package retry
