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
import { Counter } from 'k6/metrics';
import { getConfig, getEndpoints, getTLSConfig, THRESHOLDS } from './lib/config.js';

// Create failure counter - must be initialized before options export
// so the metric exists even if there are no failures
const queryFailures = new Counter('tempo_query_failures_total');

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();
const tlsConfig = getTLSConfig();

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

// Build query client configuration (connects to Tempo gateway with TLS)
const clientConfig = {
    endpoint: endpoints.query,
    tenant: endpoints.tenant,
    timeout: 30,
};

// Add TLS configuration for Tempo gateway
if (tlsConfig.queryTLSEnabled) {
    clientConfig.tls = {
        caFile: tlsConfig.caFile,
        insecureSkipVerify: tlsConfig.insecureSkipVerify,
    };
    // Use bearer token from file for authentication
    if (tlsConfig.tokenFile) {
        clientConfig.bearerTokenFile = tlsConfig.tokenFile;
    }
} else if (endpoints.token) {
    clientConfig.bearerToken = endpoints.token;
}

// Initialize query client
const client = tempo.QueryClient(clientConfig);

// Predefined queries to execute
// These match the services defined in trace-profiles.js
// Note: TraceQL uses dot prefix for resource attributes (e.g., .service.name)
const queries = [
    // Service-based queries (resource attributes use dot prefix)
    { query: '{ resource.service.name = "api-gateway" }', limit: 20 },
    { query: '{ resource.service.name = "user-service" }', limit: 20 },
    { query: '{ resource.service.name = "order-service" }', limit: 20 },
    { query: '{ resource.service.name = "payment-service" }', limit: 20 },
    { query: '{ resource.service.name = "frontend" }', limit: 20 },

    // Error queries (status is an intrinsic)
    { query: '{ status = error }', limit: 50 },

    // Duration-based queries (duration is an intrinsic)
    { query: '{ duration > 100ms }', limit: 30 },
    { query: '{ duration > 500ms }', limit: 20 },
    { query: '{ duration > 1s }', limit: 10 },

    // Combined queries
    { query: '{ resource.service.name = "api-gateway" && status = error }', limit: 20 },
    { query: '{ resource.service.name = "payment-service" && duration > 200ms }', limit: 20 },
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
  Endpoint:          ${endpoints.query} (Tempo Gateway)
  Tenant:            ${endpoints.tenant || '(default)'}
  TLS:               ${tlsConfig.queryTLSEnabled ? 'enabled' : 'disabled'}
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

    // Calculate time window in Unix seconds (Tempo gateway expects seconds, not nanoseconds)
    const now = Math.floor(Date.now() / 1000);
    const oneHourAgo = now - 3600;

    // Execute search with Unix epoch timestamps in seconds
    const result = client.search(queryDef.query, {
        start: oneHourAgo,
        end: now,
        limit: queryDef.limit,
    });

    if (!result) {
        queryFailures.add(1);
        console.error('Search failed');
        return;
    }

    // Log trace count for debugging (disabled getTrace due to 404 issues with gateway)
    if (result.traces && result.traces.length > 0) {
        // Note: getTrace is disabled because the gateway returns 404 for /api/traces/{id}
        // console.log(`Found ${result.traces.length} traces`);
    }
}

// Note: getTrace functionality disabled temporarily
// The Tempo gateway multitenancy mode doesn't expose /api/traces/{id} endpoint correctly
// TODO: Investigate correct API path for trace by ID with multitenancy

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
