-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- Disable productCatalogFailure to prevent intentional GetProduct errors on OLJCESPC7Z
-- (was causing sustained span error alerts on productcatalogservice)
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

-- Ensure productCatalogFailure chaos injection is disabled.
-- This flag was found enabled (~1.4% probability) in production, causing
-- GetProduct to return gRPC INTERNAL errors on product OLJCESPC7Z, resulting
-- in HTTP 500s surfaced to end users via the frontend and checkout services.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

-- Disable productCatalogFailure fault injection flag to prevent unintended high error
-- rates on the GetProduct endpoint for product OLJCESPC7Z.
-- When this flag is set to a non-zero value, the productcatalogservice probabilistically
-- returns Internal errors for that product, triggering the "Product Catalog Service High
-- Error Percentage" alert.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

