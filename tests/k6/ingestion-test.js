// Ingestion Performance Test for Tempo
// Tests trace ingestion throughput using xk6-tempo
//
// Usage:
//   k6 run ingestion-test.js                           # Default: medium size
//   k6 run -e SIZE=small ingestion-test.js             # Small load
//   k6 run -e SIZE=large ingestion-test.js             # Large load
//   k6 run -e SIZE=xlarge ingestion-test.js            # Extreme load
//   k6 run -e TRACES_PER_SECOND=200 ingestion-test.js  # Custom rate

import tempo from 'k6/x/tempo';
import { getConfig, getEndpoints, THRESHOLDS } from './lib/config.js';
import { getProfile } from './lib/trace-profiles.js';

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();
const traceProfile = getProfile(config.ingestion.traceProfile);

// k6 options
export const options = {
    scenarios: {
        ingestion: {
            executor: 'constant-arrival-rate',
            rate: config.ingestion.tracesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: config.vus.min,
            maxVUs: config.vus.max,
        },
    },
    thresholds: THRESHOLDS.ingestion,
};

// Initialize ingestion client
const client = tempo.IngestionClient({
    endpoint: endpoints.ingestion,
    protocol: 'grpc',  // OTLP gRPC
    tenant: endpoints.tenant,
    timeout: 30,
    tls: {
        insecure: true,  // Skip TLS verification for internal clusters
    },
});

// Setup function - runs once before the test
export function setup() {
    console.log(`
================================================================================
  TEMPO INGESTION PERFORMANCE TEST
================================================================================
  Size:              ${config.name}
  Description:       ${config.description}
  Traces/second:     ${config.ingestion.tracesPerSecond}
  Trace Profile:     ${traceProfile.name} (${traceProfile.spans.min}-${traceProfile.spans.max} spans)
  Duration:          ${config.duration}
  VUs:               ${config.vus.min} - ${config.vus.max}
  Endpoint:          ${endpoints.ingestion}
  Tenant:            ${endpoints.tenant || '(default)'}
================================================================================
`);

    return {
        profile: traceProfile,
    };
}

// Main test function - runs for each iteration
export default function(data) {
    // Generate trace using the configured profile
    const trace = tempo.generateTrace(data.profile);

    // Push trace to Tempo
    const response = client.push(trace);

    if (response.error) {
        console.error(`Failed to push trace: ${response.error}`);
    }
}

// Teardown function - runs once after the test
export function teardown(data) {
    console.log(`
================================================================================
  TEST COMPLETE
================================================================================
  Check the k6 summary above for detailed metrics:
  - tempo_ingestion_bytes_total: Total bytes ingested
  - tempo_ingestion_traces_total: Total traces sent
  - tempo_ingestion_spans_total: Total spans sent
  - tempo_ingestion_failures_total: Failed ingestion attempts
================================================================================
`);
}
