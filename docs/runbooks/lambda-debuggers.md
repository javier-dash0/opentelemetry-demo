# Lambda Debugger Functions Runbook

## Service Overview
The Lambda Debugger functions (`OtelSqsDebuggerLambda` and `XRaySqsDebuggerLambda`) are AWS Lambda functions that demonstrate OpenTelemetry and X-Ray tracing in serverless environments. They consume messages from SQS queues and propagate trace context.

## Alert: Lambda Debugger Functions High Error Rate

### Symptom
Error count exceeds 15 errors per 5 minutes for either Lambda function.

### Impact
- Trace propagation may be broken
- Demo functionality degraded
- Monitoring and observability gaps
- Higher AWS Lambda costs (errors still incur charges)

### Root Causes

#### 1. Lambda Execution Environment Errors (Most Common - 44.5% OtelSqsDebuggerLambda, 50% XRaySqsDebuggerLambda)
**Symptoms:**
- Errors in Lambda execution environment
- Cold start failures
- Runtime initialization errors
- Handler execution failures

**Investigation:**
```bash
# Check CloudWatch Logs for Lambda errors
aws logs tail /aws/lambda/OtelSqsDebuggerLambda --follow --region eu-west-1

aws logs tail /aws/lambda/XRaySqsDebuggerLambda --follow --region eu-west-1

# Check Lambda metrics in CloudWatch
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Errors \
  --dimensions Name=FunctionName,Value=OtelSqsDebuggerLambda \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Sum \
  --region eu-west-1

# Check Lambda configuration
aws lambda get-function-configuration \
  --function-name OtelSqsDebuggerLambda \
  --region eu-west-1
```

**Resolution:**
1. Check Lambda function logs for specific error messages:
   ```bash
   # Search for error patterns
   aws logs filter-log-events \
     --log-group-name /aws/lambda/OtelSqsDebuggerLambda \
     --filter-pattern "ERROR" \
     --start-time $(date -u -d '1 hour ago' +%s)000 \
     --region eu-west-1
   ```

2. Check Lambda timeout configuration:
   ```bash
   aws lambda update-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --timeout 30 \
     --region eu-west-1
   ```

3. Check memory allocation:
   ```bash
   # Increase memory if Lambda is running out
   aws lambda update-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --memory-size 512 \
     --region eu-west-1
   ```

4. Check environment variables:
   ```bash
   aws lambda get-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --query 'Environment.Variables' \
     --region eu-west-1
   ```

#### 2. SQS Queue Issues
**Symptoms:**
- Messages not being processed
- Dead letter queue (DLQ) filling up
- Message visibility timeout errors

**Investigation:**
```bash
# Check SQS queue metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/SQS \
  --metric-name ApproximateNumberOfMessagesVisible \
  --dimensions Name=QueueName,Value=OtelPropagationQueue \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average \
  --region eu-west-1

# Check DLQ
aws sqs get-queue-attributes \
  --queue-url $(aws sqs get-queue-url --queue-name OtelPropagationQueue-DLQ --region eu-west-1 --output text) \
  --attribute-names ApproximateNumberOfMessages \
  --region eu-west-1

# Inspect messages in DLQ
aws sqs receive-message \
  --queue-url $(aws sqs get-queue-url --queue-name OtelPropagationQueue-DLQ --region eu-west-1 --output text) \
  --max-number-of-messages 10 \
  --region eu-west-1
```

**Resolution:**
1. Purge DLQ if it's full of old messages:
   ```bash
   aws sqs purge-queue \
     --queue-url $(aws sqs get-queue-url --queue-name OtelPropagationQueue-DLQ --region eu-west-1 --output text) \
     --region eu-west-1
   ```

2. Adjust message visibility timeout:
   ```bash
   aws sqs set-queue-attributes \
     --queue-url $(aws sqs get-queue-url --queue-name OtelPropagationQueue --region eu-west-1 --output text) \
     --attributes VisibilityTimeout=60 \
     --region eu-west-1
   ```

3. Increase batch size if Lambda can handle more:
   ```bash
   aws lambda update-event-source-mapping \
     --uuid <event-source-mapping-id> \
     --batch-size 10 \
     --region eu-west-1
   ```

#### 3. IAM Permission Issues
**Symptoms:**
- Access denied errors
- Cannot read from SQS
- Cannot write to CloudWatch Logs

**Investigation:**
```bash
# Check Lambda execution role
aws lambda get-function \
  --function-name OtelSqsDebuggerLambda \
  --query 'Configuration.Role' \
  --region eu-west-1

# List attached policies
aws iam list-attached-role-policies \
  --role-name <role-name>

# Check inline policies
aws iam list-role-policies \
  --role-name <role-name>
```

**Resolution:**
1. Ensure Lambda has necessary permissions:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "sqs:ReceiveMessage",
           "sqs:DeleteMessage",
           "sqs:GetQueueAttributes"
         ],
         "Resource": "arn:aws:sqs:*:*:OtelPropagationQueue"
       },
       {
         "Effect": "Allow",
         "Action": [
           "logs:CreateLogGroup",
           "logs:CreateLogStream",
           "logs:PutLogEvents"
         ],
         "Resource": "arn:aws:logs:*:*:*"
       },
       {
         "Effect": "Allow",
         "Action": [
           "xray:PutTraceSegments",
           "xray:PutTelemetryRecords"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

2. Update the IAM role:
   ```bash
   aws iam put-role-policy \
     --role-name <role-name> \
     --policy-name LambdaExecutionPolicy \
     --policy-document file://policy.json
   ```

#### 4. Lambda Cold Start Issues
**Symptoms:**
- Timeout errors during initialization
- High latency on first invocation
- Initialization failures

**Investigation:**
```bash
# Check init duration metrics
aws logs filter-log-events \
  --log-group-name /aws/lambda/OtelSqsDebuggerLambda \
  --filter-pattern "Init Duration" \
  --start-time $(date -u -d '1 hour ago' +%s)000 \
  --region eu-west-1

# Check for timeout errors
aws logs filter-log-events \
  --log-group-name /aws/lambda/OtelSqsDebuggerLambda \
  --filter-pattern "Task timed out" \
  --start-time $(date -u -d '1 hour ago' +%s)000 \
  --region eu-west-1
```

**Resolution:**
1. Enable provisioned concurrency:
   ```bash
   aws lambda put-provisioned-concurrency-config \
     --function-name OtelSqsDebuggerLambda \
     --provisioned-concurrent-executions 2 \
     --qualifier <version-or-alias> \
     --region eu-west-1
   ```

2. Increase memory (improves cold start time):
   ```bash
   aws lambda update-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --memory-size 1024 \
     --region eu-west-1
   ```

3. Use reserved concurrency to prevent throttling:
   ```bash
   aws lambda put-function-concurrency \
     --function-name OtelSqsDebuggerLambda \
     --reserved-concurrent-executions 10 \
     --region eu-west-1
   ```

#### 5. OpenTelemetry/X-Ray Configuration Issues
**Symptoms:**
- Trace data not appearing in Dash0/X-Ray
- Instrumentation errors
- Context propagation failures

**Investigation:**
```bash
# Check environment variables for OTEL configuration
aws lambda get-function-configuration \
  --function-name OtelSqsDebuggerLambda \
  --query 'Environment.Variables' \
  --region eu-west-1 | grep OTEL

# Check X-Ray configuration
aws lambda get-function-configuration \
  --function-name XRaySqsDebuggerLambda \
  --query 'TracingConfig' \
  --region eu-west-1
```

**Resolution:**
1. Enable X-Ray tracing:
   ```bash
   aws lambda update-function-configuration \
     --function-name XRaySqsDebuggerLambda \
     --tracing-config Mode=Active \
     --region eu-west-1
   ```

2. Check OTEL collector endpoint:
   ```bash
   aws lambda update-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --environment "Variables={OTEL_EXPORTER_OTLP_ENDPOINT=https://your-otel-collector:4317}" \
     --region eu-west-1
   ```

### Prevention
1. **Set up CloudWatch alarms**:
   ```bash
   aws cloudwatch put-metric-alarm \
     --alarm-name OtelSqsDebuggerLambda-Errors \
     --alarm-description "Alert when Lambda error rate is high" \
     --metric-name Errors \
     --namespace AWS/Lambda \
     --statistic Sum \
     --period 300 \
     --threshold 15 \
     --comparison-operator GreaterThanThreshold \
     --dimensions Name=FunctionName,Value=OtelSqsDebuggerLambda \
     --evaluation-periods 1 \
     --region eu-west-1
   ```

2. **Enable Lambda Insights**:
   ```bash
   aws lambda update-function-configuration \
     --function-name OtelSqsDebuggerLambda \
     --layers arn:aws:lambda:eu-west-1:580247275435:layer:LambdaInsightsExtension:14 \
     --region eu-west-1
   ```

3. **Set up DLQ monitoring**:
   ```bash
   aws cloudwatch put-metric-alarm \
     --alarm-name OtelPropagationQueue-DLQ-Messages \
     --alarm-description "Alert when DLQ has messages" \
     --metric-name ApproximateNumberOfMessagesVisible \
     --namespace AWS/SQS \
     --statistic Sum \
     --period 300 \
     --threshold 1 \
     --comparison-operator GreaterThanThreshold \
     --dimensions Name=QueueName,Value=OtelPropagationQueue-DLQ \
     --evaluation-periods 1 \
     --region eu-west-1
   ```

4. **Implement proper error handling in Lambda code**:
   - Add try-catch blocks around message processing
   - Log detailed error information
   - Return appropriate responses to SQS
   - Implement retry logic with exponential backoff

### Escalation
- **Severity: Critical** (High error rates indicate infrastructure issues)
- **On-call team:** Platform Engineering / AWS Team
- **Slack channel:** #otel-demo-alerts
- **Escalation path:**
  1. Check CloudWatch Logs for specific errors
  2. Verify AWS service health (Lambda, SQS, IAM)
  3. Engage AWS Support if infrastructure issues suspected
  4. Contact OpenTelemetry demo maintainers if code issues suspected

### Related Dashboards
- [OtelSqsDebuggerLambda Service Overview](https://dash0.com/services/OtelSqsDebuggerLambda)
- [XRaySqsDebuggerLambda Service Overview](https://dash0.com/services/XRaySqsDebuggerLambda)
- [AWS Lambda Metrics Dashboard](https://console.aws.amazon.com/lambda)
- [AWS X-Ray Console](https://console.aws.amazon.com/xray)

### Related Alerts
- **SQS Queue Depth** - Check if messages are backing up
- **Lambda Throttling** - Check if Lambda is being throttled
- **Lambda Duration** - Check if Lambda is timing out

### References
- [OpenTelemetry Demo Documentation](https://github.com/opentelemetry/opentelemetry-demo)
- [AWS Lambda Best Practices](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
- [AWS Lambda Error Handling](https://docs.aws.amazon.com/lambda/latest/dg/invocation-retries.html)
- [OpenTelemetry Lambda Layer](https://aws-otel.github.io/docs/getting-started/lambda)
