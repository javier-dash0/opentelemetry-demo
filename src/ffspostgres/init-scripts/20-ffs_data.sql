-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup.
-- productCatalogFailure is explicitly reset to 0 on every startup (ON CONFLICT ... DO UPDATE)
-- to prevent previously-enabled fault injection from persisting across redeployments.
INSERT INTO public.featureflags (name, description, enabled)
VALUES
    ('productCatalogFailure', 'Fail product catalog service on a specific product', 0)
    ON CONFLICT (name) DO UPDATE SET enabled = EXCLUDED.enabled;

INSERT INTO public.featureflags (name, description, enabled)
VALUES
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
