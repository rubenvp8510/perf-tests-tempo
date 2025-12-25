// Combined Performance Test for Tempo
// Tests both ingestion and query simultaneously using xk6-tempo
// This simulates realistic production load patterns
//
// Usage:
//   k6 run combined-test.js                           # Default: medium size
//   k6 run -e SIZE=small combined-test.js             # Small load
//   k6 run -e SIZE=large combined-test.js             # Large load
//   k6 run -e SIZE=xlarge combined-test.js            # Extreme load

import tempo from 'k6/x/tempo';
import { getConfig, getEndpoints, THRESHOLDS } from './lib/config.js';
import { getProfile } from './lib/trace-profiles.js';

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();
const traceProfile = getProfile(config.ingestion.traceProfile);

// k6 options with two concurrent scenarios
export const options = {
    scenarios: {
        ingestion: {
            executor: 'constant-arrival-rate',
            rate: config.ingestion.tracesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: Math.floor(config.vus.min / 2),
            maxVUs: Math.floor(config.vus.max / 2),
            exec: 'ingest',
        },
        queries: {
            executor: 'constant-arrival-rate',
            rate: config.query.queriesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: Math.floor(config.vus.min / 2),
            maxVUs: Math.floor(config.vus.max / 2),
            exec: 'query',
        },
    },
    thresholds: THRESHOLDS.combined,
};

// Initialize clients
const ingestionClient = tempo.IngestionClient({
    endpoint: endpoints.ingestion,
    protocol: 'grpc',
    tenant: endpoints.tenant,
    timeout: 30,
    tls: {
        insecure: true,
    },
});

const queryClient = tempo.QueryClient({
    endpoint: endpoints.query,
    tenant: endpoints.tenant,
    bearerToken: endpoints.token,
    timeout: 30,
});

// Predefined queries
const queries = [
    { query: '{ service.name="api-gateway" }', limit: 20 },
    { query: '{ service.name="user-service" }', limit: 20 },
    { query: '{ service.name="order-service" }', limit: 20 },
    { query: '{ status=error }', limit: 50 },
    { query: '{ duration>100ms }', limit: 30 },
    { query: '{ service.name="payment-service" && duration>200ms }', limit: 20 },
];

const TRACE_FETCH_PROBABILITY = 0.1;

// Setup function
export function setup() {
    console.log(`
================================================================================
  TEMPO COMBINED PERFORMANCE TEST
================================================================================
  Size:              ${config.name}
  Description:       ${config.description}

  INGESTION:
    Traces/second:   ${config.ingestion.tracesPerSecond}
    Trace Profile:   ${traceProfile.name} (${traceProfile.spans.min}-${traceProfile.spans.max} spans)
    Endpoint:        ${endpoints.ingestion}

  QUERIES:
    Queries/second:  ${config.query.queriesPerSecond}
    Query Count:     ${queries.length} different queries
    Endpoint:        ${endpoints.query}

  GENERAL:
    Duration:        ${config.duration}
    Total VUs:       ${config.vus.min} - ${config.vus.max}
    Tenant:          ${endpoints.tenant || '(default)'}
================================================================================
`);

    return {
        profile: traceProfile,
    };
}

// Ingestion function - called by ingestion scenario
export function ingest(data) {
    const trace = tempo.generateTrace(data.profile);
    const response = ingestionClient.push(trace);

    if (response.error) {
        console.error(`Failed to push trace: ${response.error}`);
    }
}

// Query function - called by queries scenario
export function query() {
    const queryDef = queries[Math.floor(Math.random() * queries.length)];

    const end = Date.now();
    const start = end - (60 * 60 * 1000);

    const searchResult = queryClient.search({
        query: queryDef.query,
        start: start,
        end: end,
        limit: queryDef.limit,
    });

    if (searchResult.error) {
        console.error(`Search failed: ${searchResult.error}`);
        return;
    }

    if (searchResult.traces && searchResult.traces.length > 0) {
        if (Math.random() < TRACE_FETCH_PROBABILITY) {
            const traceId = searchResult.traces[0].traceId;
            const traceResult = queryClient.getTrace(traceId);

            if (traceResult.error) {
                console.error(`Trace fetch failed: ${traceResult.error}`);
            }
        }
    }
}

// Teardown function
export function teardown(data) {
    console.log(`
================================================================================
  TEST COMPLETE
================================================================================
  Check the k6 summary above for detailed metrics:

  INGESTION METRICS:
  - tempo_ingestion_bytes_total: Total bytes ingested
  - tempo_ingestion_traces_total: Total traces sent
  - tempo_ingestion_failures_total: Failed ingestion attempts

  QUERY METRICS:
  - tempo_query_duration_seconds: Query latency histogram
  - tempo_query_requests_total: Total queries executed
  - tempo_query_failures_total: Failed queries
================================================================================
`);
}

// Default export required by k6 (won't be used since we have named functions)
export default function() {}
