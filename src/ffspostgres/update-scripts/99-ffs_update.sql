-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

-- Disable productCatalogFailure to restore normal GetProduct behavior.
-- This flag was found enabled in production causing all GetProduct requests
-- for product OLJCESPC7Z to fail with INTERNAL error (high error rate alert).
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';
