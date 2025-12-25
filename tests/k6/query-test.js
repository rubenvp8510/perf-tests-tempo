// Query Performance Test for Tempo
// Tests trace query throughput using xk6-tempo
//
// Usage:
//   k6 run query-test.js                              # Default: medium size
//   k6 run -e SIZE=small query-test.js                # Small load (5 QPS)
//   k6 run -e SIZE=large query-test.js                # Large load (50 QPS)
//   k6 run -e SIZE=xlarge query-test.js               # Extreme load (100 QPS)
//   k6 run -e QUERIES_PER_SECOND=30 query-test.js     # Custom rate

import tempo from 'k6/x/tempo';
import { getConfig, getEndpoints, THRESHOLDS } from './lib/config.js';

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();

// k6 options
export const options = {
    scenarios: {
        queries: {
            executor: 'constant-arrival-rate',
            rate: config.query.queriesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: config.vus.min,
            maxVUs: config.vus.max,
        },
    },
    thresholds: THRESHOLDS.query,
};

// Initialize query client
const client = tempo.QueryClient({
    endpoint: endpoints.query,
    tenant: endpoints.tenant,
    bearerToken: endpoints.token,
    timeout: 30,
});

// Predefined queries to execute
// These match the services defined in trace-profiles.js
const queries = [
    // Service-based queries
    { query: '{ service.name="api-gateway" }', limit: 20 },
    { query: '{ service.name="user-service" }', limit: 20 },
    { query: '{ service.name="order-service" }', limit: 20 },
    { query: '{ service.name="payment-service" }', limit: 20 },
    { query: '{ service.name="frontend" }', limit: 20 },

    // Error queries
    { query: '{ status=error }', limit: 50 },
    { query: '{ status.code=2 }', limit: 50 },

    // Duration-based queries
    { query: '{ duration>100ms }', limit: 30 },
    { query: '{ duration>500ms }', limit: 20 },
    { query: '{ duration>1s }', limit: 10 },

    // Combined queries
    { query: '{ service.name="api-gateway" && status=error }', limit: 20 },
    { query: '{ service.name="payment-service" && duration>200ms }', limit: 20 },
];

// Probability of fetching full trace details after a search
const TRACE_FETCH_PROBABILITY = 0.1;

// Setup function - runs once before the test
export function setup() {
    console.log(`
================================================================================
  TEMPO QUERY PERFORMANCE TEST
================================================================================
  Size:              ${config.name}
  Description:       ${config.description}
  Queries/second:    ${config.query.queriesPerSecond}
  Duration:          ${config.duration}
  VUs:               ${config.vus.min} - ${config.vus.max}
  Endpoint:          ${endpoints.query}
  Tenant:            ${endpoints.tenant || '(default)'}
  Query Count:       ${queries.length} different queries
  Trace Fetch Prob:  ${TRACE_FETCH_PROBABILITY * 100}%
================================================================================
`);

    return {};
}

// Main test function - runs for each iteration
export default function() {
    // Select a random query
    const queryDef = queries[Math.floor(Math.random() * queries.length)];

    // Calculate time window (last 1 hour)
    const end = Date.now();
    const start = end - (60 * 60 * 1000);  // 1 hour ago

    // Execute search
    const searchResult = client.search({
        query: queryDef.query,
        start: start,
        end: end,
        limit: queryDef.limit,
    });

    if (searchResult.error) {
        console.error(`Search failed: ${searchResult.error}`);
        return;
    }

    // Optionally fetch full trace details (simulates real user behavior)
    if (searchResult.traces && searchResult.traces.length > 0) {
        if (Math.random() < TRACE_FETCH_PROBABILITY) {
            const traceId = searchResult.traces[0].traceId;
            const traceResult = client.getTrace(traceId);

            if (traceResult.error) {
                console.error(`Trace fetch failed: ${traceResult.error}`);
            }
        }
    }
}

// Teardown function - runs once after the test
export function teardown(data) {
    console.log(`
================================================================================
  TEST COMPLETE
================================================================================
  Check the k6 summary above for detailed metrics:
  - tempo_query_duration_seconds: Query latency histogram
  - tempo_query_requests_total: Total queries executed
  - tempo_query_failures_total: Failed queries
  - tempo_query_traces_returned: Traces returned per query
================================================================================
`);
}
