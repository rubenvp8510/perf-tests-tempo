// Ingestion Performance Test for Tempo
// Tests trace ingestion throughput using xk6-tempo
//
// Usage:
//   k6 run ingestion-test.js                        # Default: medium size
//   k6 run -e SIZE=small ingestion-test.js          # Small load
//   k6 run -e SIZE=large ingestion-test.js          # Large load
//   k6 run -e SIZE=xlarge ingestion-test.js         # Extreme load
//   k6 run -e MB_PER_SECOND=5 ingestion-test.js     # Custom rate (MB/s)

import tempo from 'k6/x/tempo';
import { Counter } from 'k6/metrics';
import { getConfig, getEndpoints, THRESHOLDS } from './lib/config.js';
import { getProfile } from './lib/trace-profiles.js';

// Create failure counter - must be initialized before options export
// so the metric exists even if there are no failures
const ingestionFailures = new Counter('tempo_ingestion_failures_total');

// Get configuration based on SIZE environment variable
const config = getConfig();
const endpoints = getEndpoints();
const traceProfile = getProfile(config.ingestion.traceProfile);

// Build trace configuration for xk6-tempo
const traceConfig = {
    useTraceTree: true,
    traceTree: traceProfile,
};

// Calculate throughput using xk6-tempo's built-in function
const throughput = tempo.calculateThroughput(traceConfig, config.ingestion.bytesPerSecond, config.vus.min);
const tracesPerSecond = Math.ceil(throughput.totalTracesPerSec);

// k6 options - rate calculated by xk6-tempo based on trace profile and target MB/s
export const options = {
    scenarios: {
        ingestion: {
            executor: 'constant-arrival-rate',
            rate: tracesPerSecond,
            timeUnit: '1s',
            duration: config.duration,
            preAllocatedVUs: config.vus.min,
            maxVUs: config.vus.max,
        },
    },
    thresholds: THRESHOLDS.ingestion,
};

// Initialize ingestion client (connects to OTel Collector, no TLS needed)
const client = tempo.IngestClient({
    endpoint: endpoints.ingestion,
    protocol: 'otlp-grpc',
    timeout: 30,
});

// Setup function - runs once before the test
export function setup() {
    console.log(`
================================================================================
  TEMPO INGESTION PERFORMANCE TEST
================================================================================
  Size:              ${config.name}
  Description:       ${config.description}
  Target Rate:       ${config.ingestion.mbPerSecond} MB/s
  Traces/sec:        ${tracesPerSecond} (${throughput.tracesPerVU.toFixed(2)} per VU)
  Trace Profile:     ${traceProfile.name} (${traceProfile.spans.min}-${traceProfile.spans.max} spans)
  Duration:          ${config.duration}
  VUs:               ${config.vus.min} - ${config.vus.max}
  Endpoint:          ${endpoints.ingestion} (OTel Collector)
================================================================================
`);

    return {
        traceConfig: traceProfile,
    };
}

// Main test function - runs for each iteration
export default function(data) {
    // Generate trace using the configured profile
    const trace = tempo.generateTrace({
        useTraceTree: true,
        traceTree: data.traceConfig,
    });

    // Push trace to Tempo
    const err = client.push(trace);
    if (err) {
        ingestionFailures.add(1);
        console.error(`Failed to push trace: ${err}`);
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
