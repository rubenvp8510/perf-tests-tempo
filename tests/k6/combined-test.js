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
import { Counter } from 'k6/metrics';
import { getConfig, getEndpoints, getTLSConfig, THRESHOLDS } from './lib/config.js';
import { getProfile } from './lib/trace-profiles.js';

// Create failure counters - must be initialized before options export
// so the metrics exist even if there are no failures
const ingestionFailures = new Counter('tempo_ingestion_failures_total');
const queryFailures = new Counter('tempo_query_failures_total');

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();
const tlsConfig = getTLSConfig();
const traceProfile = getProfile(config.ingestion.traceProfile);

// Build trace configuration for xk6-tempo
const traceConfig = {
    useTraceTree: true,
    traceTree: traceProfile,
};

// Calculate throughput using xk6-tempo's built-in function
const ingestionVUs = Math.floor(config.vus.min / 2);
const throughput = tempo.calculateThroughput(traceConfig, config.ingestion.bytesPerSecond, ingestionVUs);
const tracesPerSecond = Math.ceil(throughput.totalTracesPerSec);

// k6 options with two concurrent scenarios
export const options = {
    scenarios: {
        ingestion: {
            executor: 'constant-arrival-rate',
            rate: tracesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: ingestionVUs,
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

// Initialize ingestion client (connects to OTel Collector, no TLS needed)
const ingestionClient = tempo.IngestClient({
    endpoint: endpoints.ingestion,
    protocol: 'otlp-grpc',
    timeout: 30,
});

// Build query client configuration (connects to Tempo gateway with TLS)
const queryClientConfig = {
    endpoint: endpoints.query,
    tenant: endpoints.tenant,
    timeout: 30,
};

// Add TLS configuration for Tempo gateway
if (tlsConfig.queryTLSEnabled) {
    queryClientConfig.tls = {
        caFile: tlsConfig.caFile,
        insecureSkipVerify: tlsConfig.insecureSkipVerify,
    };
    if (tlsConfig.tokenFile) {
        queryClientConfig.bearerTokenFile = tlsConfig.tokenFile;
    }
} else if (endpoints.token) {
    queryClientConfig.bearerToken = endpoints.token;
}

// Initialize query client
const queryClient = tempo.QueryClient(queryClientConfig);

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

  INGESTION (via OTel Collector):
    Target Rate:     ${config.ingestion.mbPerSecond} MB/s
    Traces/sec:      ${tracesPerSecond} (${throughput.tracesPerVU.toFixed(2)} per VU)
    Trace Profile:   ${traceProfile.name} (${traceProfile.spans.min}-${traceProfile.spans.max} spans)
    Endpoint:        ${endpoints.ingestion}

  QUERIES (via Tempo Gateway):
    Queries/second:  ${config.query.queriesPerSecond}
    Query Count:     ${queries.length} different queries
    Endpoint:        ${endpoints.query}
    TLS:             ${tlsConfig.queryTLSEnabled ? 'enabled' : 'disabled'}

  GENERAL:
    Duration:        ${config.duration}
    Total VUs:       ${config.vus.min} - ${config.vus.max}
    Tenant:          ${endpoints.tenant || '(default)'}
================================================================================
`);

    return {
        traceConfig: traceProfile,
    };
}

// Ingestion function - called by ingestion scenario
export function ingest(data) {
    // Generate trace using the configured profile
    const trace = tempo.generateTrace({
        useTraceTree: true,
        traceTree: data.traceConfig,
    });

    const err = ingestionClient.push(trace);
    if (err) {
        ingestionFailures.add(1);
        console.error(`Failed to push trace: ${err}`);
    }
}

// Query function - called by queries scenario
export function query() {
    const queryDef = queries[Math.floor(Math.random() * queries.length)];

    const result = queryClient.search(queryDef.query, {
        start: '1h',
        end: 'now',
        limit: queryDef.limit,
    });

    if (!result) {
        queryFailures.add(1);
        console.error('Search failed');
        return;
    }

    if (result.traces && result.traces.length > 0) {
        if (Math.random() < TRACE_FETCH_PROBABILITY) {
            const traceId = result.traces[0].traceID;
            const fullTrace = queryClient.getTrace(traceId);

            if (!fullTrace) {
                queryFailures.add(1);
                console.error(`Trace fetch failed: ${traceId}`);
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
