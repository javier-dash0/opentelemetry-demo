-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

-- Ensure chaos flags are explicitly disabled to prevent production incidents.
-- productCatalogFailure: was causing GetProduct to fail with gRPC INTERNAL (status 13)
--   for product OLJCESPC7Z, propagating as errors to the frontend on every affected
--   product page request.
-- adServiceFailure: was causing GetAds to throw RESOURCE_EXHAUSTED (gRPC status 8)
--   on ~31% of requests, leading to HTTP 500 responses on /api/data.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';
UPDATE public.featureflags SET enabled = 0 WHERE name = 'adServiceFailure';

