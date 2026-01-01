// T-Shirt Size Configuration for k6 Performance Tests
// This module defines load profiles for ingestion and query tests
//
// Ingestion rate is specified in MB/s and converted to bytes/sec for xk6-tempo.
// The actual traces/sec rate is calculated by tempo.calculateThroughput() based on
// the trace profile complexity.

export const SIZES = {
    small: {
        name: 'small',
        description: 'Light load - suitable for development and CI',
        ingestion: {
            mbPerSecond: 0.1,
            traceProfile: 'small',  // 8-15 spans per trace
        },
        query: {
            queriesPerSecond: 5,
        },
        duration: '5m',
        vus: {
            min: 5,
            max: 20,
        },
    },
    medium: {
        name: 'medium',
        description: 'Moderate load - typical production baseline',
        ingestion: {
            mbPerSecond: 1,
            traceProfile: 'medium',  // 25-40 spans per trace
        },
        query: {
            queriesPerSecond: 25,
        },
        duration: '5m',
        vus: {
            min: 10,
            max: 50,
        },
    },
    large: {
        name: 'large',
        description: 'Heavy load - stress testing',
        ingestion: {
            mbPerSecond: 5,
            traceProfile: 'large',  // 50-80 spans per trace
        },
        query: {
            queriesPerSecond: 50,
        },
        duration: '5m',
        vus: {
            min: 20,
            max: 100,
        },
    },
    xlarge: {
        name: 'xlarge',
        description: 'Extreme load - capacity testing',
        ingestion: {
            mbPerSecond: 20,
            traceProfile: 'xlarge',  // 100-150 spans per trace
        },
        query: {
            queriesPerSecond: 100,
        },
        duration: '5m',
        vus: {
            min: 50,
            max: 200,
        },
    },
};

// Get configuration for a specific size
// Supports environment variable override
export function getConfig(sizeOverride) {
    const size = sizeOverride || __ENV.SIZE || 'medium';
    const config = SIZES[size.toLowerCase()];

    if (!config) {
        throw new Error(`Unknown size: ${size}. Valid sizes: ${Object.keys(SIZES).join(', ')}`);
    }

    // Allow environment variable overrides
    const mbPerSecond = parseFloat(__ENV.MB_PER_SECOND) || config.ingestion.mbPerSecond;
    const traceProfile = __ENV.TRACE_PROFILE || config.ingestion.traceProfile;

    // Convert MB/s to bytes/s for xk6-tempo's calculateThroughput()
    const bytesPerSecond = Math.floor(mbPerSecond * 1024 * 1024);

    return {
        ...config,
        ingestion: {
            ...config.ingestion,
            mbPerSecond: mbPerSecond,
            traceProfile: traceProfile,
            bytesPerSecond: bytesPerSecond,  // For tempo.calculateThroughput()
        },
        query: {
            ...config.query,
            queriesPerSecond: parseInt(__ENV.QUERIES_PER_SECOND) || config.query.queriesPerSecond,
        },
        duration: __ENV.DURATION || config.duration,
        vus: {
            min: parseInt(__ENV.VUS_MIN) || config.vus.min,
            max: parseInt(__ENV.VUS_MAX) || config.vus.max,
        },
    };
}

// Get Tempo endpoints from environment
export function getEndpoints() {
    return {
        ingestion: __ENV.TEMPO_ENDPOINT || 'http://localhost:4317',
        query: __ENV.TEMPO_QUERY_ENDPOINT || 'http://localhost:3200',
        tenant: __ENV.TEMPO_TENANT || '',
        token: __ENV.TEMPO_TOKEN || '',
    };
}

// Get TLS configuration from environment
// Query TLS is for Tempo gateway, ingestion goes through OTel Collector (no TLS)
export function getTLSConfig() {
    const queryTLSEnabled = __ENV.TEMPO_QUERY_TLS_ENABLED === 'true';
    return {
        // Query endpoint TLS (Tempo gateway)
        queryTLSEnabled: queryTLSEnabled,
        caFile: __ENV.TEMPO_TLS_CA_FILE || '',
        tokenFile: __ENV.TEMPO_TOKEN_FILE || '',
        insecureSkipVerify: __ENV.TEMPO_TLS_INSECURE === 'true',
    };
}

// Thresholds for test validation
// Note: tempo_query_* metrics are only recorded by QueryWorkload, not direct API calls
// For direct API usage, we rely on k6's built-in iteration metrics and custom counters
export const THRESHOLDS = {
    ingestion: {
        'tempo_ingestion_bytes_total': ['rate>0'],
        'tempo_ingestion_traces_total': ['rate>0'],
        'tempo_ingestion_failures_total': ['rate<1'],
    },
    query: {
        // Using custom failure counter instead of tempo_query_* metrics
        // since direct Search API doesn't record those metrics
        'tempo_query_failures_total': ['rate<1'],
    },
    combined: {
        'tempo_ingestion_bytes_total': ['rate>0'],
        'tempo_ingestion_traces_total': ['rate>0'],
        'tempo_ingestion_failures_total': ['rate<1'],
        'tempo_query_failures_total': ['rate<1'],
    },
};

export default { SIZES, getConfig, getEndpoints, getTLSConfig, THRESHOLDS };
