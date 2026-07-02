-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup.
-- ON CONFLICT UPDATE ensures failure-injection flags are always reset to 0 (disabled) on
-- container restart or re-deploy. This prevents a runaway enabled state from persisting
-- across deployments and causing a sustained high-error-count alert (e.g. adServiceFailure
-- triggering gRPC RESOURCE_EXHAUSTED on every GetAds call).
INSERT INTO public.featureflags (name, description, enabled)
VALUES
    ('productCatalogFailure', 'Fail product catalog service on a specific product', 0),
    ('recommendationCache', 'Cache recommendations', 0),
    ('adServiceFailure', 'Fail ad service requests', 0),
    ('cartServiceFailure', 'Fail cart service requests', 0),
    ('paymentServiceSimulateSlowness', 'Simulate slow response times in the payment service', 0),
    ('paymentServiceSimulateSlownessLowerBound', 'Minimum simulated delay in milliseconds in payment service, if enabled', 200),
    ('paymentServiceSimulateSlownessUpperBound', 'Maximum simulated delay in milliseconds in payment service, if enabled', 600),
    ('shippingServiceSimulateSlowness', 'Simulate slow response times in the shipping service', 0),
    ('shippingServiceSimulateSlownessLowerBound', 'Minimum simulated delay in milliseconds in shipping service, if enabled', 250),
    ('shippingServiceSimulateSlownessUpperBound', 'Maximum simulated delay in milliseconds in shipping service, if enabled', 400)
    ON CONFLICT (name) DO UPDATE SET enabled = EXCLUDED.enabled
        WHERE featureflags.name IN (
            'adServiceFailure',
            'productCatalogFailure',
            'cartServiceFailure',
            'paymentServiceSimulateSlowness',
            'shippingServiceSimulateSlowness'
        );
