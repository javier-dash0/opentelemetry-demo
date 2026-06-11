-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup.
-- WARNING: Setting 'adServiceFailure', 'cartServiceFailure', or 'productCatalogFailure' to any
-- non-zero value intentionally causes that service to return errors for demo/observability purposes.
-- These are seeded to 0 (disabled). Only enable them temporarily via the FeatureFlagService UI.
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
    ON CONFLICT DO NOTHING;

-- Reset fault-injection flags to disabled on every deploy to prevent accidental persistent errors.
-- Remove or comment out the lines below if you want to preserve live flag state across redeploys.
UPDATE public.featureflags SET enabled = 0 WHERE name IN (
    'adServiceFailure',
    'cartServiceFailure',
    'productCatalogFailure'
);
