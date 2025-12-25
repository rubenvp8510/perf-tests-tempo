package metrics

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	// MaxConcurrentQueries limits parallel Prometheus queries to avoid overwhelming the server
	MaxConcurrentQueries = 5
)

// DataPoint represents a single time-series data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// MetricResult holds the results for a single metric query
type MetricResult struct {
	QueryID     string
	MetricName  string
	Description string
	Category    string
	Labels      map[string]string
	DataPoints  []DataPoint
	Error       error
}

// CollectAllMetrics collects all metrics for the given time range using concurrent queries
func (c *Client) CollectAllMetrics(ctx context.Context, start, end time.Time) ([]MetricResult, error) {
	queries := GetAllQueries(c.config.Namespace)
	step := 60 * time.Second // 1-minute intervals

	fmt.Printf("📈 Collecting %d metrics (concurrency: %d)...\n\n", len(queries), MaxConcurrentQueries)

	var (
		results   []MetricResult
		mu        sync.Mutex
		wg        sync.WaitGroup
		sem       = make(chan struct{}, MaxConcurrentQueries)
		completed int32
	)

	for _, query := range queries {
		wg.Add(1)
		go func(q MetricQuery) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}

			metricResults, err := c.collectMetric(ctx, q, start, end, step)

			mu.Lock()
			defer mu.Unlock()

			completed++
			if err != nil {
				fmt.Printf("[%d/%d] ⚠️  %s: %v\n", completed, len(queries), q.Name, err)
				results = append(results, MetricResult{
					QueryID:     q.ID,
					MetricName:  q.Name,
					Description: q.Description,
					Category:    q.Category,
					Labels:      map[string]string{},
					DataPoints:  []DataPoint{},
					Error:       err,
				})
				return
			}

			results = append(results, metricResults...)
			fmt.Printf("[%d/%d] ✅ %s: %d series, %d points\n",
				completed, len(queries), q.Name, len(metricResults), countDataPoints(metricResults))
		}(query)
	}

	wg.Wait()

	fmt.Println()
	return results, nil
}

// collectMetric collects a single metric using range query
func (c *Client) collectMetric(ctx context.Context, query MetricQuery, start, end time.Time, step time.Duration) ([]MetricResult, error) {
	resp, err := c.QueryRange(ctx, query.Query, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if len(resp.Data.Result) == 0 {
		return nil, fmt.Errorf("no data returned (metric may not exist)")
	}

	results := make([]MetricResult, 0, len(resp.Data.Result))

	for _, result := range resp.Data.Result {
		dataPoints := make([]DataPoint, 0, len(result.Values))

		for _, value := range result.Values {
			if len(value) != 2 {
				continue
			}

			timestamp, ok := value[0].(float64)
			if !ok {
				continue
			}

			valueStr, ok := value[1].(string)
			if !ok {
				continue
			}

			floatValue, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			dataPoints = append(dataPoints, DataPoint{
				Timestamp: time.Unix(int64(timestamp), 0),
				Value:     floatValue,
			})
		}

		results = append(results, MetricResult{
			QueryID:     query.ID,
			MetricName:  query.Name,
			Description: query.Description,
			Category:    query.Category,
			Labels:      result.Metric,
			DataPoints:  dataPoints,
		})
	}

	return results, nil
}

// countDataPoints counts total data points across all metric results
func countDataPoints(results []MetricResult) int {
	total := 0
	for _, r := range results {
		total += len(r.DataPoints)
	}
	return total
}

