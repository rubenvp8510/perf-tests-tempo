// T-Shirt Size Configuration for k6 Performance Tests
// This module defines load profiles for ingestion and query tests

export const SIZES = {
    small: {
        name: 'small',
        description: 'Light load - suitable for development and CI',
        ingestion: {
            tracesPerSecond: 10,
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
            tracesPerSecond: 50,
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
            tracesPerSecond: 100,
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
            tracesPerSecond: 500,
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
    return {
        ...config,
        ingestion: {
            ...config.ingestion,
            tracesPerSecond: parseInt(__ENV.TRACES_PER_SECOND) || config.ingestion.tracesPerSecond,
        },
        query: {
            ...config.query,
            queriesPerSecond: parseInt(__ENV.QUERIES_PER_SECOND) || config.query.queriesPerSecond,
        },
        duration: __ENV.DURATION || config.duration,
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

// Thresholds for test validation
export const THRESHOLDS = {
    ingestion: {
        'tempo_ingestion_bytes_total': ['rate>0'],
        'tempo_ingestion_traces_total': ['rate>0'],
        'tempo_ingestion_failures_total': ['rate<1'],
    },
    query: {
        'tempo_query_duration_seconds': ['p(95)<2'],
        'tempo_query_requests_total': ['rate>0'],
        'tempo_query_failures_total': ['rate<1'],
    },
    combined: {
        'tempo_ingestion_bytes_total': ['rate>0'],
        'tempo_ingestion_traces_total': ['rate>0'],
        'tempo_ingestion_failures_total': ['rate<1'],
        'tempo_query_duration_seconds': ['p(95)<2'],
        'tempo_query_requests_total': ['rate>0'],
        'tempo_query_failures_total': ['rate<1'],
    },
};

export default { SIZES, getConfig, getEndpoints, THRESHOLDS };
