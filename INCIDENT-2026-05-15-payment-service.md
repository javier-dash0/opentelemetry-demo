# Incident Report: Payment Service Null Pointer Risk

**Date**: 2026-05-15  
**Service**: paymentservice  
**Namespace**: opentelemetry-demo  
**Severity**: Medium  
**Status**: Resolved

## Summary

Investigation of the payment service revealed a potential null pointer error in the graceful shutdown handler. The service attempted to call `close()` on `featureFlagServiceClient` without checking if it was initialized, which could cause crashes during pod termination when the feature flag service address is not configured.

## Root Cause

In `/src/paymentservice/index.js`, the `closeGracefully()` function (line 69-73) unconditionally calls `featureFlagServiceClient.close()` without verifying that the client was successfully initialized. When the environment variable `FEATURE_FLAG_GRPC_SERVICE_ADDR` is not set, `featureFlagServiceClient` is `null`, causing a TypeError during graceful shutdown.

### Code Issue
```javascript
async function closeGracefully(signal) {
  featureFlagServiceClient.close()  // ❌ Potential null pointer
  server.forceShutdown()
  process.kill(process.pid, signal)
}
```

## Impact

- **Potential**: Crash during SIGTERM/SIGINT shutdown sequences
- **Observed**: No errors detected in current telemetry (last hour)
- **Risk**: Affects deployments where feature flag integration is disabled

## Resolution

### Code Fix
Added null check before closing feature flag client:

```javascript
async function closeGracefully(signal) {
  if (featureFlagServiceClient) {
    featureFlagServiceClient.close()
  }
  server.forceShutdown()
  process.kill(process.pid, signal)
}
```

### Alert Creation
Created CheckRule alert "Payment Service Error Rate" to monitor for elevated errors:
- Tracks error spans using `dash0.spans.red` metric
- Threshold: 5 errors in 5-minute window
- Alerts on any error activity exceeding baseline

## Monitoring

Service metrics (last hour):
- **Request Rate**: ~248 requests
- **P95 Latency**: 1.24 seconds
- **Error Rate**: 0% (no errors observed)
- **Health Status**: Healthy

## Recommendations

1. Ensure all optional external service clients include null checks in cleanup code
2. Add integration tests for shutdown sequences with missing environment variables
3. Monitor alert effectiveness over next 7 days

## Related Resources

- Service: paymentservice (opentelemetry-demo namespace)
- Repository: https://github.com/javier-dash0/opentelemetry-demo
- Modified File: src/paymentservice/index.js
