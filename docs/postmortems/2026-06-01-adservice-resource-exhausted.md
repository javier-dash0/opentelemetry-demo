# Incident Postmortem — adservice RESOURCE_EXHAUSTED (2026-06-01)

**Status:** Resolved  
**Severity:** High  
**Duration:** Ongoing at time of detection (feature flag was left enabled)  
**Detected:** 2026-06-01 ~13:00 UTC  
**Resolved:** 2026-06-01 15:00 UTC (fix PR opened; flag reset required in live DB)

---

## Summary

The `adservice` (`opentelemetry-demo` namespace) experienced a **29.1% error rate** on the `oteldemo.AdService/GetAds` operation (26.5% on `oteldemo.AdService GetAds` specifically). Every failing request returned gRPC status code **8 — RESOURCE_EXHAUSTED**. The root cause was the `adServiceFailure` feature flag being set to a non-zero probability (~0.29) in the `featureflagservice` PostgreSQL database, causing approximately 1 in 3 ad requests to be deliberately failed by the service itself.

---

## Timeline

| Time (UTC) | Event |
|---|---|
| Unknown | `adServiceFailure` feature flag enabled in live DB with probability ~0.29 |
| 2026-06-01 ~12:00 | Error rate of 29.1% detected on adservice |
| 2026-06-01 13:01 | Trace `297d451cee1d5cf0a03320ce18c4b8ad` sampled and analysed — root cause confirmed |
| 2026-06-01 15:00 | Fix PR opened to reset `adServiceFailure` to 0 and strengthen ON CONFLICT behaviour |

---

## Root Cause

The `adServiceFailure` feature flag controls a **probabilistic error injection** path in `AdService.java` (lines 181–185):

```java
logger.debug("checking adServiceFailure feature flag");
if (checkAdFailure()) {
  logger.warn(ADSERVICE_FAIL_FEATURE_FLAG + " fail feature flag enabled, failing request.");
  throw new StatusRuntimeException(Status.RESOURCE_EXHAUSTED);
}
```

The `featureflagservice` Erlang gRPC handler (`ffs_service.erl`) evaluates the flag by comparing `rand:uniform()` against the stored `enabled` float value. A value of `0.29` means ~29% of calls return `enabled: true`, which causes adservice to discard a fully assembled ad response and throw `RESOURCE_EXHAUSTED` instead.

The PostgreSQL seed script (`20-ffs_data.sql`) uses `ON CONFLICT DO NOTHING`, which means that if the flag row already exists (from a previous deployment) with a non-zero value, the seed does **not** reset it to 0. This allowed the flag to persist at 0.29 across redeployments.

---

## Impact

- ~29% of all `GetAds` gRPC calls returned `RESOURCE_EXHAUSTED`
- Downstream: `frontend` HTTP `GET /api/data` requests failed with HTTP 500 where ads were requested
- Affected pods: `adservice-7bc574b9f9-vmnn5`, `adservice-7bc574b9f9-5wcg2`, `adservice-7bc574b9f9-njjh4`
- No data loss; only ad display was affected

---

## Observed Evidence

**Log sequence on each failing request (trace `297d451cee1d5cf0a03320ce18c4b8ad`):**
1. `DEBUG: received getAds request`
2. `INFO: Targeted ad request received for [binoculars]`
3. `DEBUG: checking adServiceFailure feature flag`
4. `WARN: adServiceFailure fail feature flag enabled, failing request.`
5. `WARN: GetAds Failed with status Status{code=RESOURCE_EXHAUSTED, description=null, cause=null}`

**Span attributes on failing `oteldemo.AdService/GetAds` (CLIENT span):**
- `rpc.grpc.status_code: 8`
- `grpc.error_message: 8 RESOURCE_EXHAUSTED:`
- `app.ads.ad_request_type: TARGETED`
- `app.ads.ad_response_type: TARGETED`

The `FeatureFlagService/EvaluateProbabilityFeatureFlag` call completed successfully (status 0 / OK) on every request — confirming the flag evaluation itself works correctly; the problem was purely the stored value.

---

## Fix

**Immediate:** Reset `adServiceFailure` to `0` in the live PostgreSQL database:

```sql
UPDATE public.featureflags SET enabled = 0 WHERE name = 'adServiceFailure';
```

**Long-term (this PR):** Change `ON CONFLICT DO NOTHING` to `ON CONFLICT (name) DO UPDATE SET enabled = EXCLUDED.enabled` in `20-ffs_data.sql` so that future redeployments always reset feature flags to their declared seed values, preventing flag state from leaking across deployments.

---

## Contributing Factors

1. **Seed script used `ON CONFLICT DO NOTHING`** — allowed non-zero flag values to persist silently across redeployments.
2. **No alerting on `adServiceFailure` flag** — the flag state change was not audited or alerted on.
3. **No automatic rollback** — once the flag probability is set to non-zero there is no mechanism to detect and auto-remediate it.

---

## Action Items

| Action | Owner | Priority |
|---|---|---|
| Reset `adServiceFailure` to 0 in live DB | On-call engineer | Immediate |
| Change seed script `ON CONFLICT DO NOTHING` → `DO UPDATE` | This PR | High |
| Add alerting on adservice error rate > 5% | SRE | High |
| Add audit logging when feature flags are changed | Platform | Medium |
| Consider moving feature flag management to a dedicated config tool with change history | Platform | Low |

---

## Lessons Learned

- Feature flags that inject errors are powerful observability tools, but their seed/reset semantics must be explicit. `DO NOTHING` silently preserves whatever value is in the DB.
- The probabilistic injection pattern (float probability rather than boolean) makes it easy to accidentally leave a non-zero value that is hard to notice in normal operation but visible under load.
- Distributed traces were the fastest path to diagnosis: the log sequence within a single trace made the cause unambiguous within minutes.
