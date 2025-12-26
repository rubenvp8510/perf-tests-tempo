package concurrent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestForEach_Success(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	var sum int64

	err := ForEach(items, func(item int) error {
		atomic.AddInt64(&sum, int64(item))
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if sum != 15 {
		t.Errorf("expected sum 15, got %d", sum)
	}
}

func TestForEach_EmptySlice(t *testing.T) {
	err := ForEach([]int{}, func(item int) error {
		t.Error("should not be called")
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestForEach_WithErrors(t *testing.T) {
	items := []int{1, 2, 3}
	testErr := errors.New("test error")

	err := ForEach(items, func(item int) error {
		if item == 2 {
			return testErr
		}
		return nil
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestForEachWithContext_Success(t *testing.T) {
	items := []string{"a", "b", "c"}
	var count int64

	err := ForEachWithContext(context.Background(), items, func(ctx context.Context, item string) error {
		atomic.AddInt64(&count, 1)
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestForEachWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	items := []int{1, 2, 3, 4, 5}
	var started int64

	// Cancel immediately
	cancel()

	err := ForEachWithContext(ctx, items, func(ctx context.Context, item int) error {
		atomic.AddInt64(&started, 1)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	if err == nil {
		t.Error("expected error due to cancellation")
	}
}

func TestForEachWithLimit_Concurrency(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var maxConcurrent int64
	var current int64

	err := ForEachWithLimit(context.Background(), items, 3, func(ctx context.Context, item int) error {
		c := atomic.AddInt64(&current, 1)
		if c > atomic.LoadInt64(&maxConcurrent) {
			atomic.StoreInt64(&maxConcurrent, c)
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if maxConcurrent > 3 {
		t.Errorf("expected max concurrency 3, got %d", maxConcurrent)
	}
}

func TestMap_Success(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	results, err := Map(items, func(item int) (int, error) {
		return item * 2, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expected := []int{2, 4, 6, 8, 10}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("expected results[%d] = %d, got %d", i, expected[i], v)
		}
	}
}

func TestMap_EmptySlice(t *testing.T) {
	results, err := Map([]int{}, func(item int) (int, error) {
		t.Error("should not be called")
		return 0, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestMap_PreservesOrder(t *testing.T) {
	items := []int{5, 4, 3, 2, 1}

	results, err := Map(items, func(item int) (int, error) {
		// Add some jitter to test ordering
		time.Sleep(time.Duration(item) * time.Millisecond)
		return item * 10, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expected := []int{50, 40, 30, 20, 10}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("expected results[%d] = %d, got %d", i, expected[i], v)
		}
	}
}

func TestMap_WithErrors(t *testing.T) {
	items := []int{1, 2, 3}
	testErr := errors.New("test error")

	results, err := Map(items, func(item int) (int, error) {
		if item == 2 {
			return 0, testErr
		}
		return item * 2, nil
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	// Results should still have successful values
	if results[0] != 2 || results[2] != 6 {
		t.Errorf("expected partial results, got %v", results)
	}
}

func TestMapWithLimit_Concurrency(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	var maxConcurrent int64
	var current int64

	results, err := MapWithLimit(context.Background(), items, 2, func(ctx context.Context, item int) (int, error) {
		c := atomic.AddInt64(&current, 1)
		if c > atomic.LoadInt64(&maxConcurrent) {
			atomic.StoreInt64(&maxConcurrent, c)
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		return item * 2, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if maxConcurrent > 2 {
		t.Errorf("expected max concurrency 2, got %d", maxConcurrent)
	}
	if len(results) != 6 {
		t.Errorf("expected 6 results, got %d", len(results))
	}
}

func TestFilter_Success(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	results := Filter(items, func(item int) bool {
		return item%2 == 0
	})

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestFilter_EmptySlice(t *testing.T) {
	results := Filter([]int{}, func(item int) bool {
		t.Error("should not be called")
		return true
	})

	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestCollector_Success(t *testing.T) {
	collector := NewCollector[int]()

	collector.Go(func() (int, error) { return 1, nil })
	collector.Go(func() (int, error) { return 2, nil })
	collector.Go(func() (int, error) { return 3, nil })

	results, err := collector.Wait()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Sum should be 6
	sum := 0
	for _, v := range results {
		sum += v
	}
	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestCollector_WithErrors(t *testing.T) {
	collector := NewCollector[int]()
	testErr := errors.New("test error")

	collector.Go(func() (int, error) { return 1, nil })
	collector.Go(func() (int, error) { return 0, testErr })
	collector.Go(func() (int, error) { return 3, nil })

	results, err := collector.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}
	// Should still have successful results
	if len(results) != 2 {
		t.Errorf("expected 2 successful results, got %d", len(results))
	}
	// Should have 1 error
	errs := collector.Errors()
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}
