# Nginx with Log Generator Example

This example demonstrates log generation and viewing in KECS using a multi-container task definition with:
- An nginx web server container
- A sidecar container that continuously sends HTTP requests to nginx every 5 seconds

This setup ensures continuous log generation without needing port forwarding or external access.

## Prerequisites
- KECS instance running
- Default cluster created

## Deployment

1. Register the task definition:
```bash
export AWS_ENDPOINT_URL=http://localhost:5373
aws ecs register-task-definition --cli-input-json file://task_def.json --region us-east-1
```

2. Create the service:
```bash
aws ecs create-service --cli-input-json file://service_def.json --region us-east-1
```

## Viewing Logs

### Using KECS TUI
```bash
kecs
```
Navigate to the task and press 'l' to view logs.

### Using AWS CLI (CloudWatch Logs)
```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

# List log streams to find the stream name
aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1

# Get the log stream name (example output: nginx/nginx-with-log-generator-78cc97f7c-pgrg5)
LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --query 'logStreams[?contains(logStreamName, `nginx/`)].logStreamName' \
  --output text)

# View nginx logs
aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --limit 20

# View log-generator logs
LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --query 'logStreams[?contains(logStreamName, `log-generator/`)].logStreamName' \
  --output text)

aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --limit 20

# Tail logs (using --start-time with recent timestamp)
aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --start-time $(date -u -v-5M +%s)000
```

### Using kubectl
```bash
# Get the pod name
kubectl get pods -n default-us-east-1

# View nginx logs
kubectl logs -n default-us-east-1 <pod-name> nginx

# View log-generator logs
kubectl logs -n default-us-east-1 <pod-name> log-generator
```

## Expected Behavior

The log-generator container will:
1. Send an HTTP request to the nginx container every 5 seconds
2. Log "Sending request to nginx..." before each request
3. Log "Request successful" after a successful request

The nginx container will log access logs for each request:
- Standard nginx access logs showing the internal requests from the log-generator

## Cleanup

Delete the service:
```bash
aws ecs delete-service --cluster default --service nginx-with-log-generator --force --region us-east-1
```