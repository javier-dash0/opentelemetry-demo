# Loadgenerator Runbook

## Service Overview
The loadgenerator is a Python-based service that uses Locust to generate synthetic traffic for the OpenTelemetry demo. It simulates user behavior including browsing products, adding items to cart, and completing checkouts.

## Alert: Loadgenerator High Error Rate

### Symptom
Error count exceeds 25 errors per 5 minutes.

### Impact
- Indicates downstream service health issues
- May mask real production issues if errors are expected
- Poor test coverage if errors are silently swallowed
- Degraded load testing accuracy

### Root Causes

#### 1. Downstream Service Errors (Most Common - 9% error rate observed)
**Symptoms:**
- Errors from frontendproxy, adservice, productcatalogservice
- HTTP 5xx responses
- Connection timeouts

**Investigation:**
```bash
# Check loadgenerator logs for error patterns
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator --tail=200

# Check which services are returning errors
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator | grep "HTTP 5"

# Check frontendproxy health
kubectl get pods -n otel-demo | grep frontendproxy
```

**Resolution:**
1. This is **expected behavior** - the loadgenerator is designed to trigger errors in downstream services
2. Check if error rate is within acceptable range:
   - < 10% = Normal for demo environment
   - 10-20% = Investigate downstream services
   - > 20% = Incident - multiple services likely degraded

3. If errors are unexpected, investigate the affected services:
   ```bash
   # Check service health
   kubectl get pods -n otel-demo
   
   # Check specific service logs
   kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-frontend --tail=100
   ```

#### 2. Bare Exception Handlers Hiding Issues
**Symptoms:**
- No detailed error logs
- Silent failures in browser automation tasks
- Difficulty debugging Playwright failures

**Investigation:**
```bash
# Check for Python tracebacks
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator | grep "Traceback"

# Check for bare except blocks catching errors
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator | grep "except:"

# Look for missing error details
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator --tail=500 | grep -A 5 "Error"
```

**Resolution:**
Implemented in PR #5:
- Replaced bare `except: pass` blocks with proper exception logging
- Added OpenTelemetry error tracing with span events
- Added structured logging with exception details
- Allow graceful failure instead of crashing tasks

#### 3. Browser Automation Failures
**Symptoms:**
- Playwright timeout errors
- Browser crashes
- Headless Chrome issues

**Investigation:**
```bash
# Check for Playwright errors
kubectl logs -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator | grep "playwright"

# Check resource usage (browsers can be memory intensive)
kubectl top pods -n otel-demo | grep loadgenerator

# Check if browser traffic is enabled
kubectl get deployment -n otel-demo opentelemetry-demo-loadgenerator -o yaml | grep LOCUST_BROWSER_TRAFFIC_ENABLED
```

**Resolution:**
1. If browser traffic is causing issues, disable it temporarily:
   ```bash
   kubectl set env deployment/opentelemetry-demo-loadgenerator -n otel-demo LOCUST_BROWSER_TRAFFIC_ENABLED=false
   ```
2. Increase memory limits if browsers are crashing:
   ```yaml
   resources:
     requests:
       memory: "512Mi"
       cpu: "250m"
     limits:
       memory: "1Gi"
       cpu: "500m"
   ```

#### 4. Connection Pool Exhaustion
**Symptoms:**
- "Too many open files" errors
- Connection refused errors
- Socket timeout errors

**Investigation:**
```bash
# Check open connections
kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator -o jsonpath='{.items[0].metadata.name}') -- netstat -an | wc -l

# Check file descriptor limits
kubectl exec -n otel-demo -it $(kubectl get pod -n otel-demo -l app.kubernetes.io/name=opentelemetry-demo-loadgenerator -o jsonpath='{.items[0].metadata.name}') -- ulimit -n
```

**Resolution:**
1. Reduce concurrent users in Locust:
   ```bash
   # Adjust via environment variables or Locust configuration
   kubectl set env deployment/opentelemetry-demo-loadgenerator -n otel-demo LOCUST_USERS=100
   ```
2. Increase file descriptor limits in the container:
   ```yaml
   securityContext:
     allowPrivilegeEscalation: false
     capabilities:
       add:
       - SYS_RESOURCE
   ```

### Code Improvements Implemented
See PR #5 for error handling improvements:
- Replaced bare exception handlers with proper logging
- Added OpenTelemetry error tracing
- Added structured error logs with exception details
- Allow graceful task failure instead of crashing

### Prevention
1. **Proper error handling** - Implemented in PR #5
2. **Monitor downstream services** - Set up alerts for services called by loadgenerator
3. **Adjust load test parameters** - Tune user count and spawn rate based on environment capacity
4. **Add request retry logic** - Implement exponential backoff for transient failures (suggested in PR #5 comments)
5. **Resource limits**:
   ```yaml
   resources:
     requests:
       memory: "512Mi"
       cpu: "250m"
     limits:
       memory: "1Gi"
       cpu: "500m"
   ```

### Escalation
- **Severity: Medium** (Load testing tool - errors are partially expected)
- **On-call team:** Platform Engineering
- **Slack channel:** #otel-demo-alerts
- **Escalation path:**
  1. Check if error rate is within acceptable range (< 10% is normal)
  2. If errors are unexpected, investigate downstream services
  3. Disable browser traffic if it's causing resource issues
  4. Engage Platform SRE team if infrastructure issues suspected

### Related Dashboards
- [Loadgenerator Service Overview](https://dash0.com/services/loadgenerator)
- [Frontend Service Health](https://dash0.com/services/frontend)
- [Frontendproxy Service Health](https://dash0.com/services/frontendproxy)
- [OpenTelemetry Demo RED Metrics](https://dash0.com/dashboards/otel-demo-red)

### Related Alerts
- **Frontend High Error Rate** - Check if frontend is the root cause
- **Adservice High Error Rate** - Loadgenerator calls adservice via frontend
- **Productcatalogservice High Error Rate** - Loadgenerator calls productcatalog via frontend

### References
- [OpenTelemetry Demo Documentation](https://github.com/opentelemetry/opentelemetry-demo)
- [Locust Documentation](https://docs.locust.io/)
- [PR #5: Improve loadgenerator error handling](https://github.com/javier-dash0/opentelemetry-demo/pull/5)
