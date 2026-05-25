# High Service Error Rate

## Overview
This alert fires when a service's error rate exceeds acceptable thresholds. An elevated error rate indicates that a significant percentage of requests to the service are failing, which directly impacts user experience and dependent services.

## Alert Details
- **Severity:** Critical
- **Threshold:** Error rate > 5% over 5 minutes
- **Evaluation Interval:** 1 minute
- **Metric:** `dash0.spans` with `otel.status_code = "ERROR"`

## Impact
- **Users:** Failed requests, degraded functionality, potential data loss
- **Dependent Services:** Cascading failures, increased latency, retry storms
- **Business:** Service unavailability, customer complaints, potential revenue loss

## Triage Steps

### 1. Verify the Alert
- Check Dash0 dashboard to confirm current error rate
- Verify alert is not a false positive (check if there's actual traffic)
- Note the time window when error rate increased

### 2. Identify Affected Service
- Determine which service(s) are experiencing elevated errors
- Check if multiple services are affected (indicates broader issue)
- Identify the service namespace and environment

### 3. Check Service Health
```promql
# Current error rate
sum(rate({otel_metric_name="dash0.spans", otel_status_code="ERROR"}[5m])) by (service_name)

# Total request rate
sum(rate({otel_metric_name="dash0.spans"}[5m])) by (service_name)

# Error percentage
(sum(rate({otel_metric_name="dash0.spans", otel_status_code="ERROR"}[5m])) by (service_name) 
/ 
sum(rate({otel_metric_name="dash0.spans"}[5m])) by (service_name)) * 100
```

## Diagnosis

### Step 1: Examine Error Patterns
1. **Filter spans by error status:**
   - Filter: `otel.status_code = "ERROR"`
   - Time range: Last 15-30 minutes
   - Group by: `service.name`, `http.status_code`, `error.type`

2. **Check error distribution:**
   - Are errors concentrated in specific endpoints? (check `http.target` or `rpc.method`)
   - Are errors from specific clients? (check `http.user_agent` or `peer.service`)
   - Do errors correlate with specific input patterns?

### Step 2: Analyze Error Logs
1. **Query error logs:**
   ```
   Filter: service.name = "<affected_service>" AND severity >= "ERROR"
   Time range: Last 30 minutes
   ```

2. **Look for patterns:**
   - Exception types and stack traces
   - Error messages indicating root cause
   - Correlation with deployment or configuration changes

### Step 3: Inspect Failing Traces
1. **Find representative error traces:**
   - Sort spans by error status
   - Select 5-10 recent error spans
   - Examine full trace context

2. **Check trace details:**
   - Which operation is failing? (span name)
   - What is the error message? (span events, `exception.message`)
   - Where in the request flow does failure occur? (span hierarchy)
   - Is there a common dependency failing? (downstream spans)

### Step 4: Check Dependencies
1. **Identify downstream failures:**
   - Review spans called by the affected service
   - Check if errors originate from external dependencies (databases, APIs, message queues)
   - Verify downstream service health

2. **Common dependency issues:**
   - Database connection pool exhaustion
   - External API timeouts or rate limiting
   - Message queue unavailability
   - DNS resolution failures

### Step 5: Review Recent Changes
1. **Check deployment history:**
   - Was there a recent deployment? (check `service.version` attribute)
   - Did error rate spike after the deployment?
   - Compare error rates before/after deployment time

2. **Check configuration changes:**
   - Environment variable updates
   - Feature flag changes
   - Infrastructure changes (scaling, migrations)

## Common Root Causes

### 1. Code Defects
- **Symptoms:** Consistent error messages, specific endpoint failures
- **Validation:** Check recent code changes, review exception stack traces
- **Resolution:** Rollback to previous version, deploy hotfix

### 2. Dependency Failures
- **Symptoms:** Timeout errors, connection refused, 5xx responses from downstream
- **Validation:** Check downstream service health, test connectivity
- **Resolution:** Contact dependency team, implement circuit breaker, increase timeout

### 3. Resource Exhaustion
- **Symptoms:** Out of memory errors, connection pool exhaustion, thread starvation
- **Validation:** Check CPU/memory metrics, connection pool metrics
- **Resolution:** Scale horizontally, increase resource limits, optimize resource usage

### 4. External API Issues
- **Symptoms:** 4xx/5xx errors from external APIs, rate limit errors
- **Validation:** Check external API status pages, test API directly
- **Resolution:** Implement retry with backoff, contact external vendor, use cached data

### 5. Data Quality Issues
- **Symptoms:** Validation errors, parsing errors, null pointer exceptions
- **Validation:** Examine failing request payloads, check data format
- **Resolution:** Add input validation, handle edge cases, fix data pipeline

### 6. Infrastructure Problems
- **Symptoms:** Pod crashes, OOM kills, network connectivity issues
- **Validation:** Check Kubernetes pod status, node health, network policies
- **Resolution:** Fix pod configuration, scale resources, resolve network issues

## Resolution Steps

### Immediate Actions (0-15 minutes)
1. **Assess severity:**
   - Is the service completely down or partially degraded?
   - How many users are affected?
   - Is there data loss or data corruption risk?

2. **Communication:**
   - Notify on-call team and service owners
   - Update incident status page (if applicable)
   - Create incident channel for coordination

3. **Quick wins:**
   - Restart unhealthy pods/containers
   - Scale up if resource constrained
   - Enable circuit breaker for failing dependency

### Short-term Mitigation (15-60 minutes)
1. **Rollback if applicable:**
   - If error spike correlates with recent deployment, rollback
   - Verify error rate returns to normal after rollback
   - Document what was rolled back

2. **Isolate the problem:**
   - Route traffic away from failing instances
   - Disable problematic feature flag
   - Use canary deployment to test fixes

3. **Apply configuration fixes:**
   - Increase timeout values
   - Adjust rate limits
   - Update environment variables

### Long-term Resolution (hours to days)
1. **Root cause fix:**
   - Deploy code fix addressing the root cause
   - Test fix in staging environment first
   - Use canary or gradual rollout strategy

2. **Validate fix:**
   - Monitor error rate after deployment
   - Check that error rate returns to baseline
   - Verify no new issues introduced

3. **Post-incident:**
   - Document root cause and resolution
   - Create bug ticket for permanent fix (if mitigation was temporary)
   - Update runbook based on learnings

## Escalation Criteria
Escalate to next tier if:
- Error rate exceeds 50% (service critically impacted)
- Issue persists > 30 minutes without clear diagnosis
- Multiple services affected (potential platform issue)
- Data loss or security concerns identified
- Requires infrastructure or platform team intervention

## Escalation Contacts
- **Service Owner:** [Team/Engineer responsible for the service]
- **Platform Team:** [On-call contact for infrastructure issues]
- **Database Team:** [If database-related issues]
- **Security Team:** [If potential security incident]

## Validation Checklist
After resolution, verify:
- [ ] Error rate returned to < 1% baseline
- [ ] No new errors introduced
- [ ] Dependent services recovered
- [ ] User-facing functionality restored
- [ ] Monitoring alerts cleared
- [ ] Post-incident review scheduled

## Related Resources
- **Dashboards:**
  - [Service Overview Dashboard](#)
  - [RED Metrics Dashboard](#)
  - [Error Analysis Dashboard](#)

- **Logs Explorer:**
  - [Filter: Error logs for this service](#)

- **Trace Explorer:**
  - [Filter: Error spans for this service](#)

- **Documentation:**
  - [Service Architecture](#)
  - [Deployment Process](#)
  - [Circuit Breaker Configuration](#)

## Prevention Measures
- Implement comprehensive error handling and retry logic
- Add circuit breakers for external dependencies
- Set up pre-production testing (staging, canary)
- Monitor error rates and set up alerts
- Regular load testing and chaos engineering
- Keep dependencies updated and monitored
- Document common failure modes and resolutions

## Revision History
- **2026-05-25:** Initial runbook creation
