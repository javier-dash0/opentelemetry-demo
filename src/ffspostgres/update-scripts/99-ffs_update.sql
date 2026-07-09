-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

-- Reset productCatalogFailure to 0 (disabled) on startup to prevent
-- accidental fault injection. The flag was found enabled at 0.1 (10%
-- failure rate), causing gRPC INTERNAL errors (code 13) on GetProduct
-- calls for specific SKUs (e.g. OLJCESPC7Z) and HTTP 500s to callers.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

