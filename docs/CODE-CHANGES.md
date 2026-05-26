# Code Changes: Upgrade Log Severity for AdService Failures

## Overview
This change upgrades the log severity from `WARN` to `ERROR` when the adservice actually fails requests. Currently, when a request fails due to the feature flag, the logs use WARN severity even though the service returns a 500 error to clients.

## Rationale
When a service genuinely fails a request and returns an error to its caller:
- The span status is set to ERROR
- The response observer returns an error
- HTTP 500 is propagated to the frontend
- But the logs only show WARN severity

This inconsistency makes it harder to:
- Query for actual failures using log severity filters
- Correlate ERROR spans with ERROR logs
- Set up proper log-based alerting

## Changes
Two log statements are upgraded from WARN to ERROR:

### Change 1: Feature Flag Trigger (Line 183)
**Before:**
```java
logger.warn(ADSERVICE_FAIL_FEATURE_FLAG + " fail feature flag enabled, failing request.");
```

**After:**
```java
logger.error(ADSERVICE_FAIL_FEATURE_FLAG + " fail feature flag enabled, failing request.");
```

### Change 2: Exception Handler (Line 195)
**Before:**
```java
logger.log(Level.WARN, "GetAds Failed with status {}", e.getStatus());
```

**After:**
```java
logger.log(Level.ERROR, "GetAds Failed with status {}", e.getStatus());
```

## Impact
- **Logs**: Failed requests will now appear with ERROR severity instead of WARN
- **Monitoring**: Error-based log alerts will now catch these failures
- **Consistency**: Log severity now matches span status (ERROR) and HTTP status (500)
- **Backward Compatibility**: No API changes; only internal logging behavior

## Testing
After deployment, verify:
1. Trigger the feature flag failure by enabling `adServiceFailure`
2. Check logs show ERROR severity: `otel.log.severity.range:ERROR AND service.name:adservice`
3. Confirm error logs include the expected messages:
   - "adServiceFailure fail feature flag enabled, failing request."
   - "GetAds Failed with status Status{code=RESOURCE_EXHAUSTED, description=null, cause=null}"
4. Verify span status remains ERROR
5. Confirm no regression in successful request flows

## Patch File
See `adservice-log-severity-fix.patch` for the unified diff.
