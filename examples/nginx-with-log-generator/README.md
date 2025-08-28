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
aws ecs register-task-definition --cli-input-json file://task_def.json
```

2. Create the service:
```bash
aws ecs create-service --cli-input-json file://service_def.json
```

## Viewing Logs

### Using KECS TUI
```bash
kecs tui --instance <instance-name>
```
Navigate to the task and press 'l' to view logs.

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
aws ecs delete-service --cluster default --service nginx-with-log-generator --force
```