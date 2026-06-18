-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- Disable adServiceFailure to stop RESOURCE_EXHAUSTED errors on GetAds requests
UPDATE public.featureflags SET enabled = 0 WHERE name = 'adServiceFailure';

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';

