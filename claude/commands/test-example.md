# Test Example Command

This command runs verification tests for a deployed KECS example.

## Usage
```
/test-example <example-name>
```

Where `<example-name>` is one of:
- single-task-nginx
- multi-container-webapp
- microservice-with-elb
- service-with-secrets
- batch-job-simple

## What it does

1. Verifies the service/task is running
2. Performs endpoint tests specific to each example
3. Checks health status
4. Validates expected behavior
5. Shows test results

## Example

```
/test-example single-task-nginx
```

This will:
- Check if the nginx service is running
- Port forward to access the nginx server
- Test the HTTP endpoint
- Show the test results

## Implementation

When this command is invoked, execute the following based on the example name:

### For single-task-nginx:
```bash
# Check service status
aws ecs describe-services --cluster default --services single-task-nginx --endpoint-url http://localhost:8080 --query 'services[0].{Status:status,Running:runningCount}'

# Get a running pod
POD_NAME=$(kubectl get pods -n default -l app=single-task-nginx -o jsonpath='{.items[0].metadata.name}')

if [ -z "$POD_NAME" ]; then
  echo "❌ No running pods found for single-task-nginx"
  exit 1
fi

# Port forward and test
kubectl port-forward -n default $POD_NAME 8888:80 &
PF_PID=$!
sleep 3

# Test nginx
echo "Testing nginx endpoint..."
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" http://localhost:8888/

# Check response
if curl -s http://localhost:8888/ | grep -q "Welcome to nginx"; then
  echo "✅ Nginx is serving the default page correctly"
else
  echo "❌ Nginx response not as expected"
fi

# Cleanup
kill $PF_PID 2>/dev/null

# Check logs
echo -e "\nRecent logs:"
kubectl logs -n default $POD_NAME --tail=10
```

### For multi-container-webapp:
```bash
# Check service
aws ecs describe-services --cluster default --services multi-container-webapp --endpoint-url http://localhost:8080 --query 'services[0].{Status:status,Running:runningCount}'

# Get pod
POD_NAME=$(kubectl get pods -n default -l app=multi-container-webapp -o jsonpath='{.items[0].metadata.name}')

# Test all containers are running
echo "Checking container statuses:"
kubectl get pod -n default $POD_NAME -o json | jq '.status.containerStatuses[] | {name: .name, ready: .ready, started: .started}'

# Port forward to both services
kubectl port-forward -n default $POD_NAME 8080:80 &
PF1=$!
kubectl port-forward -n default $POD_NAME 3000:3000 &
PF2=$!
sleep 3

# Test backend API
echo -e "\nTesting backend API..."
curl -s http://localhost:3000/ | jq

# Test frontend can reach backend
echo -e "\nTesting inter-container communication..."
kubectl exec -n default $POD_NAME -c frontend-nginx -- wget -q -O - http://localhost:3000

# Check shared volume
echo -e "\nChecking shared volume:"
kubectl exec -n default $POD_NAME -c backend-api -- ls -la /data/
kubectl exec -n default $POD_NAME -c sidecar-logger -- tail -n 3 /data/health.log

# Cleanup
kill $PF1 $PF2 2>/dev/null

echo -e "\n✅ Multi-container webapp test completed"
```

### For microservice-with-elb:
```bash
# Check service and target health
aws ecs describe-services --cluster default --services microservice-api --endpoint-url http://localhost:8080 --query 'services[0].{Status:status,Running:runningCount}'

# Get target group ARN from service
TG_ARN=$(aws ecs describe-services --cluster default --services microservice-api --endpoint-url http://localhost:8080 --query 'services[0].loadBalancers[0].targetGroupArn' --output text)

# Check target health
echo "Checking target health:"
aws elbv2 describe-target-health --target-group-arn $TG_ARN --endpoint-url http://localhost:8080

# Port forward to Traefik (ALB)
kubectl port-forward -n kecs-system svc/traefik 8888:80 &
PF_PID=$!
sleep 3

# Test all API endpoints
echo -e "\nTesting /health endpoint:"
curl -s -H "Host: microservice-alb" http://localhost:8888/health | jq

echo -e "\nTesting /api/users endpoint:"
curl -s -H "Host: microservice-alb" http://localhost:8888/api/users | jq

echo -e "\nTesting /api/products endpoint:"
curl -s -H "Host: microservice-alb" http://localhost:8888/api/products | jq

echo -e "\nTesting /api/info endpoint:"
curl -s -H "Host: microservice-alb" http://localhost:8888/api/info | jq

# Test load balancing
echo -e "\nTesting load distribution (10 requests):"
for i in {1..10}; do
  INSTANCE=$(curl -s -H "Host: microservice-alb" http://localhost:8888/api/info | jq -r '.instance' | cut -d'-' -f5)
  echo "Request $i routed to task: $INSTANCE"
done | sort | uniq -c

# Cleanup
kill $PF_PID 2>/dev/null

echo -e "\n✅ Microservice with ELB test completed"
```

### For service-with-secrets:
```bash
# Check service
aws ecs describe-services --cluster default --services service-with-secrets --endpoint-url http://localhost:8080 --query 'services[0].{Status:status,Running:runningCount}'

# Get pod
POD_NAME=$(kubectl get pods -n default -l app=service-with-secrets -o jsonpath='{.items[0].metadata.name}')

# Port forward
kubectl port-forward -n default $POD_NAME 8080:8080 &
PF_PID=$!
sleep 3

# Test endpoints
echo "Testing /health endpoint:"
curl -s http://localhost:8080/health | jq

echo -e "\nTesting /config endpoint (non-secrets):"
curl -s http://localhost:8080/config | jq

echo -e "\nTesting /secrets endpoint (verification only):"
curl -s http://localhost:8080/secrets | jq

# Verify all secrets are loaded
SECRETS_LOADED=$(curl -s http://localhost:8080/secrets | jq -r 'to_entries | map(select(.value == true)) | length')
if [ "$SECRETS_LOADED" -eq "3" ]; then
  echo "✅ All 3 secrets are loaded successfully"
else
  echo "❌ Only $SECRETS_LOADED out of 3 secrets are loaded"
fi

# Check environment variables are set (without showing values)
echo -e "\nVerifying environment variables:"
for var in DATABASE_URL API_KEY DB_PASSWORD JWT_SECRET ENCRYPTION_KEY; do
  if kubectl exec -n default $POD_NAME -- sh -c "[ -n \"\$$var\" ] && echo '✅ $var is set' || echo '❌ $var is NOT set'"; then
    :
  fi
done

# Cleanup
kill $PF_PID 2>/dev/null

echo -e "\n✅ Service with secrets test completed"
```

### For batch-job-simple:
```bash
# List recent tasks
echo "Recent batch tasks:"
aws ecs list-tasks --cluster default --desired-status STOPPED --endpoint-url http://localhost:8080 --query 'taskArns' --output table

# Get the most recent task
TASK_ARN=$(aws ecs list-tasks --cluster default --endpoint-url http://localhost:8080 --query 'taskArns[0]' --output text)

if [ "$TASK_ARN" != "None" ] && [ -n "$TASK_ARN" ]; then
  # Check task details
  echo -e "\nTask details:"
  aws ecs describe-tasks --cluster default --tasks $TASK_ARN --endpoint-url http://localhost:8080 --query 'tasks[0].{Status:lastStatus,StartedAt:startedAt,StoppedAt:stoppedAt,ExitCode:containers[0].exitCode,StoppedReason:stoppedReason}'
  
  # Check exit code
  EXIT_CODE=$(aws ecs describe-tasks --cluster default --tasks $TASK_ARN --endpoint-url http://localhost:8080 --query 'tasks[0].containers[0].exitCode' --output text)
  
  if [ "$EXIT_CODE" = "0" ]; then
    echo "✅ Batch job completed successfully (exit code: 0)"
  else
    echo "❌ Batch job failed (exit code: $EXIT_CODE)"
  fi
fi

# Check logs
echo -e "\nRecent batch job logs:"
aws logs tail /ecs/batch-job-simple --endpoint-url http://localhost:8080 --since 5m

# Run a new test job
echo -e "\nRunning a new test batch job..."
NEW_TASK=$(aws ecs run-task --cluster default --task-definition batch-job-simple --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" --overrides '{"containerOverrides":[{"name":"batch-processor","environment":[{"name":"JOB_TYPE","value":"test-run"}]}]}' --endpoint-url http://localhost:8080 --query 'tasks[0].taskArn' --output text)

echo "Started task: $NEW_TASK"
echo "Waiting for completion..."

# Wait for task to complete (max 30 seconds)
for i in {1..30}; do
  STATUS=$(aws ecs describe-tasks --cluster default --tasks $NEW_TASK --endpoint-url http://localhost:8080 --query 'tasks[0].lastStatus' --output text)
  if [ "$STATUS" = "STOPPED" ]; then
    break
  fi
  echo -n "."
  sleep 1
done

echo -e "\n✅ Batch job test completed"
```

## Common Test Patterns

All tests follow these patterns:
1. Verify service/task is running
2. Set up port forwarding for HTTP testing
3. Test specific endpoints
4. Validate responses
5. Clean up resources
6. Report results with ✅ or ❌ indicators