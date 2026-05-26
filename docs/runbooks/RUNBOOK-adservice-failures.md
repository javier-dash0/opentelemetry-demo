# Runbook: AdService Feature Flag Failures

## Overview
The adservice in the opentelemetry-demo namespace experiences intermittent failures (~30% error rate) caused by the `adServiceFailure` feature flag. When enabled, this feature flag simulates resource exhaustion errors for chaos engineering purposes.

## Alert Details
- **Alert Name**: AdService Feature Flag Failures
- **Trigger Condition**: Error rate > 20% for 5 minutes
- **Service**: adservice
- **Namespace**: opentelemetry-demo

## Symptoms
- Elevated error rate (~30%) for the adservice
- HTTP 500 errors returned to frontend service
- gRPC RESOURCE_EXHAUSTED errors (status code 8)
- Log entries with severity WARN (not ERROR) containing:
  - "adServiceFailure fail feature flag enabled, failing request."
  - "GetAds Failed with status Status{code=RESOURCE_EXHAUSTED, description=null, cause=null}"

## Root Cause
The `adServiceFailure` feature flag is enabled in the featureflagservice, causing the adservice to intentionally fail a percentage of requests. This is a **controlled chaos engineering mechanism**, not a genuine production issue.

## Investigation Steps

### 1. Verify the Alert
Check the dashboard at `adservice-dashboard.yaml` to confirm:
- Current error rate
- Request throughput
- P95 latency trends
- Log volume by severity

### 2. Check Feature Flag Status
Query the featureflagservice to determine if the `adServiceFailure` flag is enabled:
```bash
# Check recent feature flag evaluations in traces
# Look for spans from featureflagservice with operation "EvaluateProbabilityFeatureFlag"
```

### 3. Review Error Logs
Check adservice logs for the characteristic warning messages:
```bash
# Filter logs by service and severity
service.name:adservice AND otel.log.severity.range:WARN
```

Look for the sequence:
1. "checking adServiceFailure feature flag"
2. "adServiceFailure fail feature flag enabled, failing request."
3. "GetAds Failed with status Status{code=RESOURCE_EXHAUSTED, description=null, cause=null}"

### 4. Examine Error Spans
Check spans for:
- `otel.span.status`: ERROR
- `rpc.grpc.status_code`: 8 (RESOURCE_EXHAUSTED)
- `exception.message`: "RESOURCE_EXHAUSTED"
- Service: adservice, namespace: opentelemetry-demo

## Resolution

### If This is a Test/Demo Environment
The feature flag is working as intended for chaos engineering. **No action required** unless:
- The error rate exceeds expected thresholds
- The feature flag should be disabled per test plan

### If This is a Production Environment
**Immediate Action**: Disable the `adServiceFailure` feature flag in the featureflagservice configuration.

1. **Disable the Feature Flag**:
   - Update the featureflagservice configuration to set `adServiceFailure` to `false`
   - Restart the featureflagservice if configuration changes require it

2. **Verify Resolution**:
   - Monitor the error rate for 5-10 minutes
   - Confirm error rate drops to baseline (~0%)
   - Check that logs no longer show feature flag failure messages

3. **Update Monitoring** (if needed):
   - Review why this alert fired in a production environment
   - Consider separate monitoring for test/demo vs production namespaces

## Prevention

### Short Term
- Clearly label demo/test namespaces to avoid confusion with production
- Set up environment-specific alerting thresholds
- Document which feature flags are for testing only

### Long Term
- **Code Improvement**: Upgrade log severity from WARN to ERROR when the service actually fails requests (see code changes in PR)
- Implement feature flag auditing to track when chaos flags are enabled
- Add dashboard annotations when chaos engineering tests are running

## Impact Assessment
- **User Impact**: Users see failed ad requests, falling back to random ads or no ads
- **Service Dependencies**: 
  - Frontend service receives 500 errors from ad service
  - Feature flag service is queried successfully for each request
- **Data Loss**: None - this is a controlled failure that doesn't affect data persistence

## Related Resources
- Dashboard: `adservice-dashboard.yaml`
- Alert Definition: `adservice-alert.json`
- Source Code: `src/adservice/src/main/java/oteldemo/AdService.java` (lines 182-196)
- Trace Example: Look for `service.name:adservice` AND `otel.span.status:ERROR`

## Escalation Path
1. Check if this is a test/demo environment - if yes, no escalation needed
2. If production, contact the team responsible for the featureflagservice
3. If feature flag cannot be disabled quickly, consider:
   - Temporarily routing around the adservice
   - Falling back to random ads for all requests
   - Emergency deployment with the feature flag check disabled

## Post-Incident Actions
- [ ] Document why the feature flag was enabled
- [ ] Review change management process for feature flag toggles
- [ ] Update monitoring to distinguish chaos engineering from real incidents
- [ ] Consider implementing the log severity upgrade (WARN → ERROR) for actual failures
