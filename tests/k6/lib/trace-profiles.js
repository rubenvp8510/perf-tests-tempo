// Trace Profiles for xk6-tempo
// Adapted from https://github.com/rubenvp8510/xk6-tempo/blob/main/examples/trace-profiles.js
// Each profile represents a different complexity level of distributed traces

// Helper to generate realistic context for traces
function createContext(scale) {
    return {
        propagation: {
            user_id: `user-${Math.floor(Math.random() * 10000)}`,
            session_id: `session-${Date.now()}`,
            correlation_id: `corr-${Math.random().toString(36).substring(7)}`,
            tenant_id: 'tenant-1',
            region: ['us-east-1', 'us-west-2', 'eu-west-1'][Math.floor(Math.random() * 3)],
            request_id: `req-${Math.random().toString(36).substring(7)}`,
        },
    };
}

// Default settings for all profiles
const defaultSettings = {
    attributes: {
        enableSemanticAttributes: true,
        enableTags: true,
        tagDensity: 0.85,
    },
};

// Small Profile: Startup/MVP (~8-15 spans, 5-8 services)
const smallProfile = {
    name: 'small',
    description: 'Startup/MVP - Simple API with basic services',
    spans: { min: 8, max: 15 },
    services: ['frontend', 'api-gateway', 'user-service', 'database', 'cache'],
    settings: defaultSettings,
    rootOperation: {
        name: 'POST /api/orders',
        service: 'api-gateway',
    },
    operations: [
        { name: 'validate-request', service: 'api-gateway', duration: { min: 5, max: 20 } },
        { name: 'authenticate', service: 'user-service', duration: { min: 10, max: 50 } },
        { name: 'process-order', service: 'api-gateway', duration: { min: 20, max: 100 } },
        { name: 'check-inventory', service: 'database', duration: { min: 5, max: 30 } },
        { name: 'cache-lookup', service: 'cache', duration: { min: 1, max: 5 } },
        { name: 'save-order', service: 'database', duration: { min: 10, max: 50 } },
        { name: 'send-notification', service: 'api-gateway', duration: { min: 5, max: 20 } },
    ],
    context: createContext('small'),
};

// Medium Profile: E-commerce/SaaS (~25-40 spans, 15-20 services)
const mediumProfile = {
    name: 'medium',
    description: 'E-commerce/SaaS - Typical production system',
    spans: { min: 25, max: 40 },
    services: [
        'frontend', 'api-gateway', 'user-service', 'order-service',
        'inventory-service', 'payment-service', 'notification-service',
        'shipping-service', 'pricing-service', 'cache', 'database',
        'message-queue', 'search-service', 'recommendation-engine',
    ],
    settings: defaultSettings,
    rootOperation: {
        name: 'POST /api/v1/checkout',
        service: 'api-gateway',
    },
    operations: [
        { name: 'validate-request', service: 'api-gateway', duration: { min: 5, max: 20 } },
        { name: 'authenticate', service: 'user-service', duration: { min: 10, max: 50 } },
        { name: 'authorize', service: 'user-service', duration: { min: 5, max: 20 } },
        { name: 'get-cart', service: 'order-service', duration: { min: 10, max: 40 } },
        { name: 'validate-items', service: 'inventory-service', duration: { min: 20, max: 80 } },
        { name: 'check-stock', service: 'inventory-service', duration: { min: 15, max: 60 } },
        { name: 'reserve-inventory', service: 'inventory-service', duration: { min: 10, max: 50 } },
        { name: 'calculate-pricing', service: 'pricing-service', duration: { min: 15, max: 70 } },
        { name: 'apply-discounts', service: 'pricing-service', duration: { min: 5, max: 30 } },
        { name: 'validate-payment', service: 'payment-service', duration: { min: 50, max: 200 } },
        { name: 'process-payment', service: 'payment-service', duration: { min: 100, max: 500 } },
        { name: 'create-order', service: 'order-service', duration: { min: 20, max: 80 } },
        { name: 'save-order', service: 'database', duration: { min: 10, max: 50 } },
        { name: 'queue-fulfillment', service: 'message-queue', duration: { min: 5, max: 20 } },
        { name: 'calculate-shipping', service: 'shipping-service', duration: { min: 20, max: 100 } },
        { name: 'send-confirmation', service: 'notification-service', duration: { min: 10, max: 50 } },
        { name: 'update-recommendations', service: 'recommendation-engine', duration: { min: 15, max: 60 } },
        { name: 'invalidate-cache', service: 'cache', duration: { min: 2, max: 10 } },
    ],
    context: createContext('medium'),
};

// Large Profile: Fintech/Enterprise (~50-80 spans, 25-35 services)
const largeProfile = {
    name: 'large',
    description: 'Fintech/Enterprise - Complex distributed system',
    spans: { min: 50, max: 80 },
    services: [
        'edge-gateway', 'api-gateway', 'auth-service', 'user-service',
        'account-service', 'transaction-service', 'ledger-service',
        'fraud-detection', 'risk-engine', 'compliance-service',
        'audit-service', 'notification-service', 'reporting-service',
        'analytics-service', 'cache-layer', 'database-primary',
        'database-replica', 'message-broker', 'event-store',
        'search-cluster', 'ml-scoring', 'rate-limiter',
        'circuit-breaker', 'config-service', 'secret-manager',
    ],
    settings: defaultSettings,
    rootOperation: {
        name: 'POST /api/v2/transactions',
        service: 'edge-gateway',
    },
    operations: [
        { name: 'waf-scan', service: 'edge-gateway', duration: { min: 2, max: 10 } },
        { name: 'rate-limit-check', service: 'rate-limiter', duration: { min: 1, max: 5 } },
        { name: 'authenticate', service: 'auth-service', duration: { min: 20, max: 80 } },
        { name: 'verify-mfa', service: 'auth-service', duration: { min: 50, max: 200 } },
        { name: 'authorize', service: 'auth-service', duration: { min: 10, max: 40 } },
        { name: 'get-user-profile', service: 'user-service', duration: { min: 15, max: 60 } },
        { name: 'get-account', service: 'account-service', duration: { min: 20, max: 80 } },
        { name: 'check-balance', service: 'account-service', duration: { min: 10, max: 40 } },
        { name: 'validate-transaction', service: 'transaction-service', duration: { min: 30, max: 120 } },
        { name: 'fraud-check', service: 'fraud-detection', duration: { min: 50, max: 200 } },
        { name: 'ml-score', service: 'ml-scoring', duration: { min: 30, max: 150 } },
        { name: 'risk-assessment', service: 'risk-engine', duration: { min: 40, max: 160 } },
        { name: 'compliance-check', service: 'compliance-service', duration: { min: 20, max: 100 } },
        { name: 'aml-verification', service: 'compliance-service', duration: { min: 30, max: 150 } },
        { name: 'create-transaction', service: 'transaction-service', duration: { min: 25, max: 100 } },
        { name: 'update-ledger', service: 'ledger-service', duration: { min: 20, max: 80 } },
        { name: 'write-primary', service: 'database-primary', duration: { min: 15, max: 60 } },
        { name: 'replicate', service: 'database-replica', duration: { min: 10, max: 40 } },
        { name: 'publish-event', service: 'event-store', duration: { min: 5, max: 20 } },
        { name: 'audit-log', service: 'audit-service', duration: { min: 10, max: 40 } },
        { name: 'send-notification', service: 'notification-service', duration: { min: 15, max: 60 } },
        { name: 'update-analytics', service: 'analytics-service', duration: { min: 20, max: 80 } },
        { name: 'invalidate-cache', service: 'cache-layer', duration: { min: 2, max: 10 } },
    ],
    context: createContext('large'),
};

// XLarge Profile: FAANG-scale (~100-150 spans, 40-50 services)
const xlargeProfile = {
    name: 'xlarge',
    description: 'FAANG-scale - Ultra-distributed global platform',
    spans: { min: 100, max: 150 },
    services: [
        'edge-cdn', 'edge-gateway', 'global-lb', 'regional-lb',
        'api-gateway', 'graphql-gateway', 'auth-service', 'identity-provider',
        'session-manager', 'user-service', 'profile-service', 'preferences-service',
        'order-service', 'cart-service', 'inventory-service', 'warehouse-service',
        'fulfillment-service', 'shipping-service', 'tracking-service',
        'payment-gateway', 'payment-processor-1', 'payment-processor-2',
        'fraud-detection', 'risk-engine', 'ml-platform', 'feature-store',
        'recommendation-engine', 'personalization-service', 'search-service',
        'ranking-service', 'pricing-engine', 'promotion-service',
        'notification-hub', 'email-service', 'sms-service', 'push-service',
        'analytics-collector', 'metrics-aggregator', 'logging-service',
        'audit-service', 'compliance-engine', 'cache-l1', 'cache-l2',
        'database-shard-1', 'database-shard-2', 'database-shard-3',
        'message-broker', 'event-streaming', 'config-service', 'secret-vault',
    ],
    settings: defaultSettings,
    rootOperation: {
        name: 'POST /api/v3/orders/create',
        service: 'edge-cdn',
    },
    operations: [
        { name: 'cdn-edge', service: 'edge-cdn', duration: { min: 1, max: 5 } },
        { name: 'edge-routing', service: 'edge-gateway', duration: { min: 2, max: 8 } },
        { name: 'global-lb', service: 'global-lb', duration: { min: 1, max: 5 } },
        { name: 'regional-lb', service: 'regional-lb', duration: { min: 1, max: 5 } },
        { name: 'api-routing', service: 'api-gateway', duration: { min: 3, max: 12 } },
        { name: 'graphql-parse', service: 'graphql-gateway', duration: { min: 5, max: 20 } },
        { name: 'authenticate', service: 'auth-service', duration: { min: 15, max: 60 } },
        { name: 'verify-identity', service: 'identity-provider', duration: { min: 20, max: 80 } },
        { name: 'session-validate', service: 'session-manager', duration: { min: 5, max: 20 } },
        { name: 'get-user', service: 'user-service', duration: { min: 10, max: 40 } },
        { name: 'get-profile', service: 'profile-service', duration: { min: 10, max: 40 } },
        { name: 'get-preferences', service: 'preferences-service', duration: { min: 8, max: 32 } },
        { name: 'get-cart', service: 'cart-service', duration: { min: 12, max: 48 } },
        { name: 'validate-cart', service: 'order-service', duration: { min: 15, max: 60 } },
        { name: 'check-inventory', service: 'inventory-service', duration: { min: 20, max: 80 } },
        { name: 'warehouse-query', service: 'warehouse-service', duration: { min: 15, max: 60 } },
        { name: 'calculate-pricing', service: 'pricing-engine', duration: { min: 25, max: 100 } },
        { name: 'apply-promotions', service: 'promotion-service', duration: { min: 15, max: 60 } },
        { name: 'personalize', service: 'personalization-service', duration: { min: 20, max: 80 } },
        { name: 'get-recommendations', service: 'recommendation-engine', duration: { min: 30, max: 120 } },
        { name: 'ml-predict', service: 'ml-platform', duration: { min: 25, max: 100 } },
        { name: 'feature-lookup', service: 'feature-store', duration: { min: 10, max: 40 } },
        { name: 'fraud-check', service: 'fraud-detection', duration: { min: 40, max: 160 } },
        { name: 'risk-score', service: 'risk-engine', duration: { min: 35, max: 140 } },
        { name: 'payment-route', service: 'payment-gateway', duration: { min: 10, max: 40 } },
        { name: 'process-payment-1', service: 'payment-processor-1', duration: { min: 100, max: 400 } },
        { name: 'process-payment-2', service: 'payment-processor-2', duration: { min: 100, max: 400 } },
        { name: 'create-order', service: 'order-service', duration: { min: 30, max: 120 } },
        { name: 'write-shard-1', service: 'database-shard-1', duration: { min: 15, max: 60 } },
        { name: 'write-shard-2', service: 'database-shard-2', duration: { min: 15, max: 60 } },
        { name: 'write-shard-3', service: 'database-shard-3', duration: { min: 15, max: 60 } },
        { name: 'cache-l1-set', service: 'cache-l1', duration: { min: 2, max: 8 } },
        { name: 'cache-l2-set', service: 'cache-l2', duration: { min: 3, max: 12 } },
        { name: 'publish-event', service: 'event-streaming', duration: { min: 5, max: 20 } },
        { name: 'queue-fulfillment', service: 'fulfillment-service', duration: { min: 10, max: 40 } },
        { name: 'calculate-shipping', service: 'shipping-service', duration: { min: 20, max: 80 } },
        { name: 'init-tracking', service: 'tracking-service', duration: { min: 10, max: 40 } },
        { name: 'notify-hub', service: 'notification-hub', duration: { min: 8, max: 32 } },
        { name: 'send-email', service: 'email-service', duration: { min: 15, max: 60 } },
        { name: 'send-sms', service: 'sms-service', duration: { min: 20, max: 80 } },
        { name: 'send-push', service: 'push-service', duration: { min: 10, max: 40 } },
        { name: 'collect-analytics', service: 'analytics-collector', duration: { min: 5, max: 20 } },
        { name: 'aggregate-metrics', service: 'metrics-aggregator', duration: { min: 8, max: 32 } },
        { name: 'audit-log', service: 'audit-service', duration: { min: 10, max: 40 } },
        { name: 'compliance-check', service: 'compliance-engine', duration: { min: 20, max: 80 } },
        { name: 'log-event', service: 'logging-service', duration: { min: 3, max: 12 } },
    ],
    context: createContext('xlarge'),
};

// Export all profiles
export const TRACE_PROFILES = {
    small: smallProfile,
    medium: mediumProfile,
    large: largeProfile,
    xlarge: xlargeProfile,
};

// Get profile by name
export function getProfile(name) {
    const profile = TRACE_PROFILES[name.toLowerCase()];
    if (!profile) {
        throw new Error(`Unknown profile: ${name}. Valid profiles: ${Object.keys(TRACE_PROFILES).join(', ')}`);
    }
    return profile;
}

export default TRACE_PROFILES;
