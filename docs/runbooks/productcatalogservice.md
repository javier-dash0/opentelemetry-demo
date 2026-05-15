# Productcatalogservice Runbook

## Service Overview
The productcatalogservice is a Go-based gRPC service that manages the product catalog. It provides product listings, search, and retrieval operations. It integrates with the feature flag service for controlled failure injection.

## Alert: Productcatalogservice High Error Rate

### Symptom
Error count exceeds 15 errors per 5 minutes.

### Impact
- Users cannot browse products
- Product pages fail to load
- Checkout process may be blocked
- Critical service (marked as CRITICAL health status)

### Root Causes

#### 1. Feature Flag Induced Failures (Most Common - 1% error rate observed)
**Symptoms:**
- `INTERNAL` errors for specific product ID (OLJCESPC7Z)
- Error message: "Product Id Lookup Failed: OLJCESPC7Z"
- Feature flag `productCatalogFailure` is enabled

**Investigation:**
```bash
# Check recent error logs
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice --tail=100 | grep "Product Id Lookup Failed"

# Check feature flag service
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-featureflagservice --tail=50

# Check if specific product ID is causing issues
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice | grep "OLJCESPC7Z"
```

**Resolution:**
1. Disable the feature flag:
   ```bash
   # Check feature flag service configuration
   kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-featureflagservice -o jsonpath='{.items[0].metadata.name}') -- cat /etc/ffs/config.yaml
   ```
2. If feature flag is misconfigured, restart feature flag service:
   ```bash
   kubectl rollout restart deployment -n otel-demo opentelemetry-demo-featureflagservice
   ```

#### 2. Linear Search Performance Issues
**Symptoms:**
- Slow response times as catalog grows
- High CPU usage
- Increased latency for GetProduct calls

**Investigation:**
```bash
# Check CPU usage
kubectl top pods -n otel-demo | grep productcatalog

# Check trace data for slow spans
# Look for spans with duration > 100ms in Dash0 UI

# Check catalog size
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice | grep "Loaded.*products"
```

**Resolution:**
Implemented in PR #4:
- HashMap index for O(1) product lookups instead of O(n) linear search
- Feature flag caching with 5-second TTL
- Connection timeout for feature flag service (2 seconds)

#### 3. Missing Product Data
**Symptoms:**
- `NOT_FOUND` errors for specific product IDs
- Error message: "Product Id Not Found: <ID>"

**Investigation:**
```bash
# Check loaded products
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice | grep "Loaded"

# List product JSON files
kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice -o jsonpath='{.items[0].metadata.name}') -- ls -la /products/
```

**Resolution:**
1. Verify product JSON files are mounted correctly:
   ```bash
   kubectl describe pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice | grep -A 10 "Mounts"
   ```
2. Check ConfigMap or volume for product data:
   ```bash
   kubectl get configmap -n otel-demo | grep product
   ```
3. If data is missing, restart the pod:
   ```bash
   kubectl delete pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice
   ```

#### 4. Feature Flag Service Overload
**Symptoms:**
- Repeated gRPC calls to feature flag service
- High connection count
- Feature flag service high latency

**Investigation:**
```bash
# Check connection pool
kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-productcatalogservice -o jsonpath='{.items[0].metadata.name}') -- netstat -an | grep 8081 | wc -l

# Check feature flag service metrics
# Look for high request rate from productcatalogservice in Dash0 UI
```

**Resolution:**
Implemented in PR #4:
- Feature flag caching reduces calls by ~99%
- 5-second TTL prevents stale data
- Connection pooling for feature flag gRPC client

### Code Improvements Implemented
See PR #4 for performance optimizations:
- Replaced O(n) linear search with O(1) HashMap index
- Added feature flag caching with 5-second TTL
- Added 2-second timeout for feature flag requests
- Implemented connection pooling for gRPC client

### Prevention
1. **Add caching layer** - Implemented in PR #4
2. **Optimize data structures** - HashMap index implemented in PR #4
3. **Monitor feature flag service** - Set up alerts for feature flag service health
4. **Enable horizontal pod autoscaling**:
   ```yaml
   apiVersion: autoscaling/v2
   kind: HorizontalPodAutoscaler
   metadata:
     name: productcatalogservice-hpa
   spec:
     scaleTargetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: opentelemetry-demo-productcatalogservice
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
- **Severity: High** (Service marked as CRITICAL)
- **On-call team:** Platform Engineering
- **Slack channel:** #otel-demo-alerts
- **Escalation path:**
  1. Check recent deployments and rollback if needed
  2. Disable problematic feature flags
  3. Engage Platform SRE team if infrastructure issues suspected
  4. Contact OpenTelemetry demo maintainers if persistent issues

### Related Dashboards
- [Productcatalogservice Service Overview](https://dash0.com/services/productcatalogservice)
- [Feature Flag Service Health](https://dash0.com/services/featureflagservice)
- [OpenTelemetry Demo RED Metrics](https://dash0.com/dashboards/otel-demo-red)

### Related Alerts
- **Featureflagservice High Error Rate** - Check if feature flag service is the root cause
- **Frontend High Error Rate** - Frontend depends on productcatalogservice
- **Checkoutservice High Error Rate** - Checkout depends on productcatalogservice
- **Recommendationservice High Error Rate** - Recommendations depend on productcatalogservice

### References
- [OpenTelemetry Demo Documentation](https://github.com/opentelemetry/opentelemetry-demo)
- [PR #4: Optimize productcatalogservice](https://github.com/javier-dash0/opentelemetry-demo/pull/4)
