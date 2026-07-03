-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- Disable productCatalogFailure feature flag to stop intentional errors on product OLJCESPC7Z
-- This flag was causing GetProduct requests for OLJCESPC7Z to fail with gRPC INTERNAL (status 13),
-- resulting in elevated error rates on the Product Catalog Service.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

-- Disable productCatalogFailure fault injection (was causing ~1% error rate on GetProduct)
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

