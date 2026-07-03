-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup.
-- All fault-injection flags (productCatalogFailure, adServiceFailure, cartServiceFailure,
-- paymentServiceSimulateSlowness, shippingServiceSimulateSlowness) default to 0 (disabled).
-- Enabling these flags in a production-like environment will cause real service errors and
-- alert the on-call team. Only enable them deliberately for chaos/fault injection testing.
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

-- Ensure fault-injection flags are reset to disabled on every startup to prevent
-- accidentally leaving them enabled from a previous testing session.
UPDATE public.featureflags SET enabled = 0
WHERE name IN ('productCatalogFailure', 'adServiceFailure', 'cartServiceFailure',
               'paymentServiceSimulateSlowness', 'shippingServiceSimulateSlowness');
