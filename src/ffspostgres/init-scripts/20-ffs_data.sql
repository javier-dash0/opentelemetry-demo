-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup.
-- NOTE: Failure-injection flags (adServiceFailure, cartServiceFailure, productCatalogFailure)
-- use ON CONFLICT DO UPDATE to reset them to 0 on every container restart.
-- This prevents a previously enabled failure flag from persisting across restarts and
-- causing unintended production-like error rates (e.g. RESOURCE_EXHAUSTED on adservice).
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
    ON CONFLICT (name) DO UPDATE SET enabled = CASE
        -- Reset failure-injection flags to 0 on restart to prevent persistent error injection
        WHEN EXCLUDED.name IN ('adServiceFailure', 'cartServiceFailure', 'productCatalogFailure')
        THEN 0
        -- Preserve user-configured values for non-failure flags (e.g. slowness bounds)
        ELSE public.featureflags.enabled
    END;
