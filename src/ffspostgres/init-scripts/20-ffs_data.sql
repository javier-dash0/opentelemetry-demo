-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags created and initialized on startup
INSERT INTO public.featureflags (name, description, enabled)
VALUES
    ('productCatalogFailure', 'Fail product catalog service on a specific product (controlled by PRODUCT_CATALOG_FAILURE_PRODUCT_ID env var, default OLJCESPC7Z). Set to 0 to disable, 0.0-1.0 for probabilistic injection.', 0),
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
