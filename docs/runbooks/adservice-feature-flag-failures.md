# AdService Feature Flag Failure Runbook

## Overview
This runbook provides guidance for responding to alerts related to AdService feature flag failures, specifically the `adServiceFailure` feature flag.

## Alert Details
- **Service**: adservice (opentelemetry-demo namespace)
- **Issue**: Feature flag `adServiceFailure` causing intentional request failures
- **Error Type**: RESOURCE_EXHAUSTED (gRPC status)
- **Typical Error Rate**: ~30% when feature flag is enabled

## Problem Description
The AdService includes a feature flag called `adServiceFailure` that intentionally fails requests for testing and chaos engineering purposes. When this flag is enabled via the Feature Flag Service, the AdService throws a `StatusRuntimeException` with `Status.RESOURCE_EXHAUSTED`, causing a significant error rate.

### Code Location
The failure is triggered in `/src/adservice/src/main/java/oteldemo/AdService.java`:
```java
logger.debug("checking adServiceFailure feature flag");
if (checkAdFailure()) {
  logger.warn(ADSERVICE_FAIL_FEATURE_FLAG + " fail feature flag enabled, failing request.");
  throw new StatusRuntimeException(Status.RESOURCE_EXHAUSTED);
}
```

## Diagnosis Steps

### 1. Verify Alert Trigger
Check the alert details to confirm:
- Service: adservice
- Namespace: opentelemetry-demo
- Metric: Error rate or request volume exceeds threshold

### 2. Check Feature Flag Status
Query the Feature Flag Service to determine if `adServiceFailure` is enabled:
```bash
# Check feature flag service logs
kubectl logs -n opentelemetry-demo deployment/featureflagservice
```

### 3. Review Recent Error Logs
Check AdService logs for feature flag warnings:
```bash
kubectl logs -n opentelemetry-demo deployment/adservice | grep "adServiceFailure"
```

Expected log message when flag is enabled:
```
WARN adServiceFailure fail feature flag enabled, failing request.
```

### 4. Analyze Error Rate
Review the AdService monitoring dashboard to:
- Confirm current error rate percentage
- Check if errors correlate with specific operations
- Verify error type is RESOURCE_EXHAUSTED

Dashboard: [AdService Feature Flag Monitoring](/dashboards/ba64f722-1a9b-448a-80d7-3d19c8c1465f)

### 5. Check Span Data
Query for failing spans:
```promql
sum (rate({otel_metric_name = "dash0.spans", service_name = "adservice", service_namespace = "opentelemetry-demo", otel_status = "ERROR"}[5m]))
```

## Resolution

### If Feature Flag Is Intentionally Enabled (Testing/Chaos Engineering)
1. Acknowledge the alert
2. Document the testing activity
3. Monitor the duration and impact
4. Disable the flag when testing is complete

### If Feature Flag Is Unintentionally Enabled
1. **Immediate Action**: Disable the `adServiceFailure` feature flag in the Feature Flag Service
2. Verify error rate drops to normal levels (typically <1%)
3. Check for any cascading effects on dependent services:
   - Frontend service
   - Load generator
4. Monitor for 10-15 minutes to ensure stability

### How to Disable the Feature Flag
```bash
# Option 1: Via Feature Flag Service API (if available)
curl -X POST http://featureflagservice:8081/feature-flags/adServiceFailure/disable

# Option 2: Update feature flag configuration
kubectl edit configmap featureflag-config -n opentelemetry-demo
```

## Prevention

### For Production Environments
- Ensure feature flag state is properly managed through configuration management
- Implement approval workflows for enabling failure injection flags
- Set up separate feature flag configurations for dev/staging/prod environments
- Consider removing or permanently disabling chaos engineering flags in production

### Monitoring Improvements
- Alert on feature flag state changes
- Add dashboard panels showing feature flag status alongside metrics
- Track feature flag enable/disable events as annotations on metrics

## Related Resources
- [AdService Source Code](https://github.com/javier-dash0/opentelemetry-demo/blob/main/src/adservice/src/main/java/oteldemo/AdService.java)
- [AdService Feature Flag Monitoring Dashboard](/dashboards/ba64f722-1a9b-448a-80d7-3d19c8c1465f)
- [Feature Flag Service Documentation](../services/featureflagservice.md)

## Post-Incident Actions
1. Document the incident timeline
2. Review why the feature flag was enabled
3. Update feature flag management procedures if needed
4. Consider improving log severity for feature flag failures (see issue #XXX)
5. Review other services for similar chaos engineering feature flags

## Escalation
If disabling the feature flag does not resolve the issue or if you suspect a different root cause:
1. Check for actual resource exhaustion issues (CPU, memory, connections)
2. Review AdService deployment for recent changes
3. Contact the platform team via #ops-alerts channel
4. Escalate to on-call SRE if impact is severe

---

**Last Updated**: 2026-05-26  
**Maintained by**: Platform Team  
**Related Alerts**: AdService Feature Flag Failure Alert
