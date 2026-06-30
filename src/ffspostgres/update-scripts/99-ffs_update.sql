-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- Disable productCatalogFailure feature flag to stop intentional error injection on product OLJCESPC7Z.
-- When enabled, productcatalogservice/main.go calls FeatureFlagService/EvaluateProbabilityFeatureFlag for
-- every GetProduct(OLJCESPC7Z) request and, if the flag is true, returns gRPC INTERNAL (status 13) instead
-- of the product data. This causes elevated span error counts and triggers the
-- "Product Catalog Service High Error Percentage" alert.
UPDATE public.featureflags SET enabled = 0 WHERE name = 'productCatalogFailure';

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

