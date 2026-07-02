-- Copyright The OpenTelemetry Authors
-- SPDX-License-Identifier: Apache-2.0

-- Reset adServiceFailure feature flag to disabled state.
-- This flag was found enabled (non-zero probability) causing ~27% of AdService
-- GetAds requests to fail with RESOURCE_EXHAUSTED. It is a chaos-testing flag
-- and must be disabled by default in all environments.
UPDATE public.featureflags
SET enabled = 0
WHERE name = 'adServiceFailure';
