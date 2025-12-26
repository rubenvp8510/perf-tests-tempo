package concurrent

import (
	"context"
	"errors"
	"sync"
)

// ForEach executes fn for each item in items concurrently.
// Returns the first error encountered, or nil if all succeeded.
// All goroutines are waited for even if one fails.
func ForEach[T any](items []T, fn func(T) error) error {
	if len(items) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(items))

	for _, item := range items {
		wg.Add(1)
		go func(item T) {
			defer wg.Done()
			if err := fn(item); err != nil {
				errCh <- err
			}
		}(item)
	}

	wg.Wait()
	close(errCh)

	// Collect all errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ForEachWithContext executes fn for each item in items concurrently with context support.
// Cancels remaining work if context is cancelled.
func ForEachWithContext[T any](ctx context.Context, items []T, fn func(context.Context, T) error) error {
	if len(items) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(items))

	for _, item := range items {
		wg.Add(1)
		go func(item T) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}
			if err := fn(ctx, item); err != nil {
				errCh <- err
			}
		}(item)
	}

	wg.Wait()
	close(errCh)

	// Collect all errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ForEachWithLimit executes fn for each item with a concurrency limit.
func ForEachWithLimit[T any](ctx context.Context, items []T, limit int, fn func(context.Context, T) error) error {
	if len(items) == 0 {
		return nil
	}

	if limit <= 0 {
		limit = 1
	}

	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	errCh := make(chan error, len(items))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, item := range items {
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			if err := fn(ctx, item); err != nil {
				errCh <- err
			}
		}(item)
	}

	wg.Wait()
	close(errCh)

	// Collect all errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Map applies fn to each item concurrently and returns the results.
// Order of results matches order of items.
func Map[T, R any](items []T, fn func(T) (R, error)) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]R, len(items))
	errs := make([]error, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(i int, item T) {
			defer wg.Done()
			results[i], errs[i] = fn(item)
		}(i, item)
	}

	wg.Wait()

	// Collect errors
	var allErrs []error
	for _, err := range errs {
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return results, errors.Join(allErrs...)
	}

	return results, nil
}

// MapWithLimit applies fn to each item with a concurrency limit.
// Order of results matches order of items.
func MapWithLimit[T, R any](ctx context.Context, items []T, limit int, fn func(context.Context, T) (R, error)) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	if limit <= 0 {
		limit = 1
	}

	results := make([]R, len(items))
	errs := make([]error, len(items))
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, item := range items {
		select {
		case <-ctx.Done():
			errs[i] = ctx.Err()
			continue
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				errs[i] = ctx.Err()
				return
			default:
			}

			results[i], errs[i] = fn(ctx, item)
		}(i, item)
	}

	wg.Wait()

	// Collect errors
	var allErrs []error
	for _, err := range errs {
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return results, errors.Join(allErrs...)
	}

	return results, nil
}

// Filter returns items for which fn returns true, processing concurrently.
// Order of results matches order of input items.
func Filter[T any](items []T, fn func(T) bool) []T {
	if len(items) == 0 {
		return nil
	}

	// Store keep decisions for each index
	keep := make([]bool, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(i int, item T) {
			defer wg.Done()
			keep[i] = fn(item)
		}(i, item)
	}

	wg.Wait()

	// Collect results in original order
	results := make([]T, 0, len(items))
	for i, item := range items {
		if keep[i] {
			results = append(results, item)
		}
	}

	return results
}

// Collect gathers results from multiple concurrent operations.
// Returns all results and any errors encountered.
type Collector[T any] struct {
	mu      sync.Mutex
	results []T
	errs    []error
	wg      sync.WaitGroup
}

// NewCollector creates a new Collector
func NewCollector[T any]() *Collector[T] {
	return &Collector[T]{}
}

// Go runs fn in a goroutine and collects its result
func (c *Collector[T]) Go(fn func() (T, error)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		result, err := fn()
		c.mu.Lock()
		defer c.mu.Unlock()
		if err != nil {
			c.errs = append(c.errs, err)
		} else {
			c.results = append(c.results, result)
		}
	}()
}

// Wait waits for all operations to complete and returns results and errors
func (c *Collector[T]) Wait() ([]T, error) {
	c.wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.errs) > 0 {
		return c.results, errors.Join(c.errs...)
	}
	return c.results, nil
}

// Results returns the collected results (must be called after Wait)
func (c *Collector[T]) Results() []T {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.results
}

// Errors returns the collected errors (must be called after Wait)
func (c *Collector[T]) Errors() []error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.errs
}
