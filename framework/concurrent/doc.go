// Package concurrent provides utilities for concurrent execution with error handling.
//
// # ForEach
//
// Execute a function for each item in parallel:
//
//	items := []string{"a", "b", "c"}
//	err := concurrent.ForEach(items, func(item string) error {
//	    return processItem(item)
//	})
//
// # ForEachWithLimit
//
// Control concurrency with a limit:
//
//	err := concurrent.ForEachWithLimit(ctx, items, 5, func(ctx context.Context, item string) error {
//	    return processItem(ctx, item)
//	})
//
// # Map
//
// Transform items concurrently while preserving order:
//
//	results, err := concurrent.Map(numbers, func(n int) (int, error) {
//	    return n * 2, nil
//	})
//
// # MapWithLimit
//
// Transform with concurrency limit:
//
//	results, err := concurrent.MapWithLimit(ctx, items, 3, func(ctx context.Context, item Item) (Result, error) {
//	    return process(ctx, item)
//	})
//
// # Filter
//
// Filter items concurrently:
//
//	results := concurrent.Filter(items, func(item int) bool {
//	    return item > 0
//	})
//
// # Collector
//
// Collect results from multiple concurrent operations:
//
//	collector := concurrent.NewCollector[Result]()
//	collector.Go(func() (Result, error) { return operation1() })
//	collector.Go(func() (Result, error) { return operation2() })
//	results, err := collector.Wait()
//
// # Error Handling
//
// All functions aggregate errors using errors.Join and continue processing
// all items even if some fail. This allows you to see all failures rather
// than just the first one.
package concurrent
