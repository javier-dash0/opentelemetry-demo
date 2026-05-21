# Runbook: Frontend GET /api/products/{productId} Health Monitoring

## Overview

**Service**: `frontend` (namespace: `opentelemetry-demo`)  
**Operation**: `GET /api/products/2ZYFJ3GM2N`  
**Dashboard**: `frontend-get-products-2zyfj3gm2n`  
**Alert**: `Frontend GET /api/products/2ZYFJ3GM2N - High Latency`

This runbook provides guidance for monitoring and troubleshooting the product detail retrieval operation in the OpenTelemetry Demo frontend service.

## Service Context

The `frontend` service handles user-facing HTTP requests and makes downstream calls to the Product Catalog service via gRPC. The analyzed operation retrieves details for a specific product (product ID: 2ZYFJ3GM2N).

**Architecture Flow:**
```
Load Generator → Frontend Proxy (Envoy) → Frontend Service → Product Catalog Service
```

**Historical Performance Baseline:**
- Median latency: 3ms
- P75: 6ms
- P90: 11ms
- P95: 17ms
- Request volume: ~18 requests/second
- Success rate: 88% return HTTP 200

## Dashboard

**Name**: Frontend - GET /api/products/2ZYFJ3GM2N  
**Location**: Dash0 → Dashboards → `frontend-get-products-2zyfj3gm2n`

### Panels

1. **Request Rate** - Total requests per interval (tracks traffic volume)
2. **Error Count** - Count of spans with ERROR status
3. **Latency p50** - Median latency in milliseconds
4. **Latency p90** - 90th percentile latency
5. **Latency p95** - 95th percentile latency (primary SLI)
6. **Latency p99** - 99th percentile latency
7. **HTTP Status Code Breakdown** - Distribution of HTTP response codes

### Key Metrics

**Request Rate:**
```promql
sum (increase({otel_metric_name = "dash0.spans", service_name = "frontend", service_namespace = "opentelemetry-demo", dash0_operation_name = "GET /api/products/2ZYFJ3GM2N"}[$__interval]))
```

**Error Count:**
```promql
sum (increase({otel_metric_name = "dash0.spans", service_name = "frontend", service_namespace = "opentelemetry-demo", dash0_operation_name = "GET /api/products/2ZYFJ3GM2N", otel_status_code = "ERROR"}[$__interval])) or vector(0)
```

**P95 Latency (milliseconds):**
```promql
histogram_quantile(0.95, sum by() (rate({otel_metric_name = "dash0.spans.duration", service_name = "frontend", service_namespace = "opentelemetry-demo", dash0_operation_name = "GET /api/products/2ZYFJ3GM2N"}[$__interval]))) * 1000
```

## Alert Configuration

**Alert Name**: Frontend GET /api/products/2ZYFJ3GM2N - High Latency

### Expression
```promql
histogram_quantile(0.95, sum by() (rate({otel_metric_name = "dash0.spans.duration", service_name = "frontend", service_namespace = "opentelemetry-demo", dash0_operation_name = "GET /api/products/2ZYFJ3GM2N"}[5m]))) > $__threshold
```

### Thresholds

| Severity  | Threshold | Baseline Multiple | When It Fires |
|-----------|-----------|-------------------|---------------|
| Degraded  | 30ms      | 6x baseline       | P95 latency consistently exceeds 30ms for 2 minutes |
| Failed    | 50ms      | 10x baseline      | P95 latency consistently exceeds 50ms for 2 minutes |

### Alert Metadata

**Labels:**
- `service`: `frontend`
- `operation`: `GET /api/products/2ZYFJ3GM2N`

**Annotations:**
- `service_namespace`: `opentelemetry-demo`

**For Duration**: 2 minutes (reduces noise from transient spikes)

## Response Procedures

### Degraded Alert (30ms threshold)

**Severity**: Warning  
**Impact**: User experience degradation; page loads slower than normal

**Immediate Actions:**

1. **Check Dashboard** - Open the dashboard to confirm elevated latency across all percentiles
2. **Verify Traffic Pattern** - Check if request rate has increased significantly
3. **Review HTTP Status Codes** - Confirm most requests still return 200
4. **Check Downstream Dependencies** - Investigate Product Catalog Service health

**Investigation Steps:**

1. Query recent spans with elevated latency:
   ```
   service.name = "frontend" AND 
   dash0.operation.name = "GET /api/products/2ZYFJ3GM2N" AND 
   otel.span.duration > 0.030
   ```

2. Check for patterns:
   - Are all product IDs affected or just specific ones?
   - Does latency correlate with specific time windows?
   - Are there upstream proxy delays visible in traces?

3. Examine Product Catalog Service:
   - Check gRPC call latency to `grpc.oteldemo.ProductCatalogService/GetProduct`
   - Review Product Catalog Service RED metrics
   - Check for errors in Product Catalog Service logs

4. Review infrastructure:
   - Check pod resource utilization (CPU, memory)
   - Verify no pod restarts or crashes
   - Check for Kubernetes node issues

**Escalation Criteria:**
- Latency remains elevated for >10 minutes
- Error rate increases above 5%
- Alert escalates to Failed severity

### Failed Alert (50ms threshold)

**Severity**: Critical  
**Impact**: Severe user experience degradation; potential business impact

**Immediate Actions:**

1. **Page on-call engineer** - This requires immediate attention
2. **Check for ongoing incidents** - Verify if related to a broader outage
3. **Review recent deployments** - Check if correlated with recent changes
4. **Assess blast radius** - Determine how many users are affected

**Investigation Steps:**

1. **Identify failing requests:**
   ```
   service.name = "frontend" AND 
   dash0.operation.name = "GET /api/products/2ZYFJ3GM2N" AND 
   (otel.span.duration > 0.050 OR otel.span.status.code = "ERROR")
   ```

2. **Analyze full trace trees:**
   - Select spans with highest latency
   - Use Dash0 Trace Explorer to view complete request flow
   - Identify bottleneck in trace hierarchy (frontend → proxy → product catalog)

3. **Check for exceptions:**
   - Look for span events named `exception`
   - Review exception types and stack traces
   - Check for database timeouts, connection pool exhaustion, or downstream 5xx errors

4. **Correlate with logs:**
   - Filter logs by `otel.trace.id` from failing traces
   - Check for ERROR or WARN level logs
   - Look for specific error messages or patterns

5. **Review infrastructure metrics:**
   - Pod CPU/memory utilization
   - Kubernetes events (OOMKilled, CrashLoopBackOff)
   - Network connectivity issues

**Remediation Actions:**

- **If CPU/memory constrained**: Scale up frontend pods or increase resource limits
- **If downstream dependency failing**: Investigate Product Catalog Service; consider circuit breaker
- **If recent deployment**: Consider rollback
- **If isolated to specific product ID**: Check product data integrity

**Communication:**
- Update status page if customer-facing
- Notify stakeholders of impact and ETA
- Document findings in incident tracker

## Common Failure Modes

### 1. Downstream gRPC Timeout

**Symptoms:**
- P95/P99 latency spikes
- Traces show long duration on `grpc.oteldemo.ProductCatalogService/GetProduct` child span
- No ERROR status on frontend span, but slow response

**Root Cause:**
- Product Catalog Service overloaded or slow
- Database query slow in Product Catalog
- Network latency between services

**Resolution:**
- Investigate Product Catalog Service
- Check database query performance
- Verify network connectivity

### 2. High Request Volume

**Symptoms:**
- Request rate significantly higher than baseline (~18 req/s)
- Latency increases across all percentiles
- Pod CPU utilization elevated

**Root Cause:**
- Traffic spike (legitimate or attack)
- Load generator misconfiguration
- Retry storms

**Resolution:**
- Scale frontend pods horizontally
- Implement rate limiting if attack suspected
- Check load generator configuration

### 3. Non-200 HTTP Responses

**Symptoms:**
- HTTP Status Code Breakdown shows increase in 404, 500, or other non-200 codes
- May or may not trigger latency alert
- Error Count panel shows increase

**Root Cause:**
- Invalid product IDs requested
- Product Catalog Service returning errors
- Frontend application bug

**Resolution:**
- Check `http.response.status_code` distribution
- Review error logs for specific error messages
- Verify Product Catalog Service health

### 4. Cross-Trace Causal Links

**Symptoms:**
- Spans contain links to other traces
- Synthetic requests marked with `app.synthetic_request: true`
- Elevated traffic from load generator

**Root Cause:**
- Load generator triggering synthetic requests based on real traffic
- May amplify load during high-traffic periods

**Resolution:**
- Review load generator configuration
- Ensure synthetic traffic doesn't compound real load
- Consider disabling synthetic tests during incidents

## Troubleshooting Queries

### Find Recent Slow Spans
```
service.name = "frontend" AND 
service.namespace = "opentelemetry-demo" AND 
dash0.operation.name = "GET /api/products/2ZYFJ3GM2N" AND 
otel.span.duration > 0.030
```

### Find Error Spans
```
service.name = "frontend" AND 
service.namespace = "opentelemetry-demo" AND 
dash0.operation.name = "GET /api/products/2ZYFJ3GM2N" AND 
otel.span.status.code = "ERROR"
```

### Find Non-200 Responses
```
service.name = "frontend" AND 
service.namespace = "opentelemetry-demo" AND 
dash0.operation.name = "GET /api/products/2ZYFJ3GM2N" AND 
http.response.status_code != 200
```

### Check gRPC Downstream Calls
```
service.name = "frontend" AND 
service.namespace = "opentelemetry-demo" AND 
otel.span.name = "grpc.oteldemo.ProductCatalogService/GetProduct"
```

## Historical Context

**Performance Characteristics:**
- This operation typically performs in the fastest quartile
- Historical p95 is ~17ms; alert fires at 50ms (3x degradation)
- 88% of requests return HTTP 200 (12% variance likely 404s or cache hits with 304)

**Trace Analysis Findings:**
- Root span: `ingress` (CLIENT) - 3ms
- Intermediate: `router frontend egress` (PRODUCER) - 2ms
- Operation span: `GET` (CLIENT) - 1-2ms
- Downstream: gRPC call to Product Catalog - typically <1ms

**Known Issues:**
- Occasional span links to synthetic request traces (load testing)
- Some traces show `otel.span.status.code: UNSET` instead of explicit OK

## Related Services

- **Frontend Proxy**: `frontendproxy` (namespace: `opentelemetry-demo`)
- **Product Catalog Service**: `productcatalogservice` (namespace: `opentelemetry-demo`)
- **Load Generator**: `loadgenerator` (namespace: `opentelemetry-demo`)

## References

- [OpenTelemetry Demo Repository](https://github.com/open-telemetry/opentelemetry-demo)
- Dash0 Dashboard: `frontend-get-products-2zyfj3gm2n`
- Alert Rule: `Frontend GET /api/products/2ZYFJ3GM2N - High Latency`

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2026-05-21 | Agent0 (Dash0) | Initial runbook creation based on trace analysis and alert configuration |

---

**Last Updated**: 2026-05-21  
**Maintained By**: Platform/SRE Team  
**Escalation Path**: On-call SRE → Platform Engineering Lead
