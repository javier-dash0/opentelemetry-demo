# Adservice Runbook

## Service Overview
The adservice is a Java-based gRPC service responsible for serving targeted advertisements based on product categories. It integrates with the feature flag service to control ad serving behavior.

## Alert: Adservice High Error Rate

### Symptom
Error count exceeds 30 errors per 5 minutes.

### Impact
- Users may not see advertisements on product pages
- Revenue loss from missing ad impressions
- Degraded user experience

### Root Causes

#### 1. Feature Flag Service Unavailability (Most Common - 30.9% error rate observed)
**Symptoms:**
- `RESOURCE_EXHAUSTED` errors in traces
- Feature flag check failures
- Errors spike when feature flag service is slow or unavailable

**Investigation:**
```bash
# Check feature flag service health
kubectl get pods -n otel-demo | grep featureflag

# Check adservice logs
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-adservice --tail=100

# Check for feature flag connection errors
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-adservice | grep "Feature Flag"
```

**Resolution:**
1. Check feature flag service status:
   ```bash
   kubectl describe pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-featureflagservice
   ```
2. Restart feature flag service if unhealthy:
   ```bash
   kubectl rollout restart deployment -n otel-demo opentelemetry-demo-featureflagservice
   ```
3. Scale up feature flag service if overloaded:
   ```bash
   kubectl scale deployment -n otel-demo opentelemetry-demo-featureflagservice --replicas=3
   ```

#### 2. Resource Exhaustion
**Symptoms:**
- High CPU/memory usage
- Slow response times
- Connection timeouts

**Investigation:**
```bash
# Check resource usage
kubectl top pods -n otel-demo | grep adservice

# Check resource limits
kubectl describe pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-adservice | grep -A 5 "Limits"
```

**Resolution:**
1. Increase resource limits:
   ```yaml
   resources:
     requests:
       memory: "256Mi"
       cpu: "250m"
     limits:
       memory: "512Mi"
       cpu: "500m"
   ```
2. Scale horizontally:
   ```bash
   kubectl scale deployment -n otel-demo opentelemetry-demo-adservice --replicas=5
   ```

#### 3. Network Issues
**Symptoms:**
- Connection refused errors
- DNS resolution failures
- Timeout errors

**Investigation:**
```bash
# Test connectivity to feature flag service
kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-adservice -o jsonpath='{.items[0].metadata.name}') -- nc -zv opentelemetry-demo-featureflagservice 8081

# Check network policies
kubectl get networkpolicies -n otel-demo
```

**Resolution:**
1. Verify service endpoints:
   ```bash
   kubectl get endpoints -n otel-demo opentelemetry-demo-featureflagservice
   ```
2. Check DNS resolution:
   ```bash
   kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-adservice -o jsonpath='{.items[0].metadata.name}') -- nslookup opentelemetry-demo-featureflagservice
   ```

### Code Improvements Implemented
See PR #3 for timeout and error handling improvements:
- Added 2-second timeout for feature flag requests
- Added keepalive configuration (30s interval, 5s timeout)
- Wrapped feature flag checks in try-catch with graceful degradation
- Default to not failing when feature flag service is unreachable

### Prevention
1. **Monitor feature flag service health** - Set up alerts for feature flag service availability
2. **Implement circuit breaker** - Use Resilience4j circuit breaker (implemented in PR #3)
3. **Add request timeouts** - Prevent hanging requests (implemented in PR #3)
4. **Enable horizontal pod autoscaling**:
   ```yaml
   apiVersion: autoscaling/v2
   kind: HorizontalPodAutoscaler
   metadata:
     name: adservice-hpa
   spec:
     scaleTargetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: opentelemetry-demo-adservice
     minReplicas: 2
     maxReplicas: 10
     metrics:
     - type: Resource
       resource:
         name: cpu
         target:
           type: Utilization
           averageUtilization: 70
   ```

### Escalation
- **Severity: High**
- **On-call team:** Platform Engineering
- **Slack channel:** #otel-demo-alerts
- **Escalation path:**
  1. Check recent deployments and rollback if needed
  2. Engage Platform SRE team if infrastructure issues suspected
  3. Contact OpenTelemetry demo maintainers if persistent issues

### Related Dashboards
- [Adservice Service Overview](https://dash0.com/services/adservice)
- [Feature Flag Service Health](https://dash0.com/services/featureflagservice)
- [OpenTelemetry Demo RED Metrics](https://dash0.com/dashboards/otel-demo-red)

### Related Alerts
- **Featureflagservice High Error Rate** - Check if feature flag service is the root cause
- **Adservice High Latency** - May indicate resource contention
- **Frontend High Error Rate** - May be caused by adservice failures

### References
- [OpenTelemetry Demo Documentation](https://github.com/opentelemetry/opentelemetry-demo)
- [PR #3: Improve adservice resilience](https://github.com/javier-dash0/opentelemetry-demo/pull/3)
