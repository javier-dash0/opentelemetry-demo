-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Feature Flags updated for startup
--     'enabled' is a decimal value between 0 and 1 (inclusive)
--     0.0 is always disabled
--     1.0 is always enabled
--     All values between set a percentage chance on each request
--     example: 0.55 is enabled 55% of the time

-- UPDATE public.featureflags SET enabled = 0.55 WHERE name = 'cartServiceFailure';
UPDATE public.featureflags SET enabled = 0 WHERE name = 'adServiceFailure';

-- Disable adServiceFailure flag to prevent intentional RESOURCE_EXHAUSTED errors on GetAds.
-- This flag was causing ~27-31% of adservice/GetAds requests to fail.
-- See: https://runbooks.example.com/adservice-errors
UPDATE public.featureflags SET enabled = 0 WHERE name = 'adServiceFailure';

