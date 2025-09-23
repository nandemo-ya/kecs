# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with KECS.

## Diagnostic Tools

### Health Checks

Check KECS component health:

```bash
# API server health
curl http://localhost:8081/health

# Detailed health status
curl http://localhost:8081/health/detailed

# Kubernetes connectivity
kubectl cluster-info
```

### Logs

View KECS logs:

```bash
# If running directly
kecs server 2>&1 | tee kecs.log

# If running in Docker
docker logs kecs-container

# If running in Kubernetes
kubectl logs -n kecs-system deployment/kecs-control-plane
```

### Debug Mode

Enable debug logging:

```bash
# Via command line
kecs server --log-level debug

# Via environment variable
export KECS_LOG_LEVEL=debug
kecs server
```

## Common Issues

### Installation Issues

#### Problem: Build Fails

**Symptoms:**
```
go: cannot find main module
```

**Solution:**
```bash
# Ensure you're in the correct directory
cd /path/to/kecs

# Clean and rebuild
make clean
make deps
make build
```

#### Problem: Missing Dependencies

**Symptoms:**
```
package github.com/... is not in GOROOT
```

**Solution:**
```bash
# Update dependencies
go mod download
go mod tidy

# Verify Go version
go version  # Should be 1.21+
```

### Startup Issues

#### Problem: Port Already in Use

**Symptoms:**
```
listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use different port
kecs server --api-port 9080
```

#### Problem: Cannot Connect to Kubernetes

**Symptoms:**
```
failed to get kubernetes config: stat /home/user/.kube/config: no such file or directory
```

**Solution:**
```bash
# Check kubeconfig exists
ls ~/.kube/config

# Set kubeconfig explicitly
kecs server --kubeconfig /path/to/kubeconfig

# Or use in-cluster config
kubectl apply -f deploy/kubernetes/rbac.yaml
```

### Cluster Operations

#### Problem: Cluster Creation Fails

**Symptoms:**
```
failed to create kind cluster: exit status 1
```

**Solution:**
```bash
# Check Docker is running
docker ps

# Check Kind is installed
kind version

# Create cluster manually
kind create cluster --name kecs-cluster

# Verify cluster
kubectl get nodes
```

#### Problem: Cluster Already Exists

**Symptoms:**
```
cluster already exists
```

**Solution:**
```bash
# List existing clusters
aws ecs list-clusters --endpoint-url http://localhost:8080

# Delete and recreate
aws ecs delete-cluster --cluster <name> --endpoint-url http://localhost:8080
```

### Service Deployment Issues

#### Problem: Service Won't Start

**Symptoms:**
- Service stuck in PENDING
- No running tasks

**Solution:**
1. Check task definition:
   ```bash
   aws ecs describe-task-definition \
     --task-definition <family:revision> \
     --endpoint-url http://localhost:8080
   ```

2. Check cluster resources:
   ```bash
   kubectl top nodes
   kubectl describe nodes
   ```

3. Review service events:
   ```bash
   aws ecs describe-services \
     --cluster <cluster> \
     --services <service> \
     --endpoint-url http://localhost:8080
   ```

#### Problem: Tasks Keep Stopping

**Symptoms:**
- Tasks transition to STOPPED
- Service can't maintain desired count

**Solution:**
1. Check task stop reason:
   ```bash
   aws ecs describe-tasks \
     --cluster <cluster> \
     --tasks <task-arn> \
     --endpoint-url http://localhost:8080 \
     | jq '.tasks[0].stoppedReason'
   ```

2. View container logs:
   ```bash
   kubectl logs -n <cluster-name> <pod-name>
   ```

3. Common causes:
   - Image pull failures
   - Health check failures
   - Resource constraints
   - Application errors

### Task Issues

#### Problem: Image Pull Error

**Symptoms:**
```
CannotPullContainerError: Error response from daemon: pull access denied
```

**Solution:**
1. Verify image exists:
   ```bash
   docker pull <image-name>
   ```

2. Check image registry credentials:
   ```bash
   # For private registries
   kubectl create secret docker-registry regcred \
     --docker-server=<registry> \
     --docker-username=<username> \
     --docker-password=<password> \
     -n <cluster-name>
   ```

3. Update task definition with credentials:
   ```json
   {
     "containerDefinitions": [{
       "repositoryCredentials": {
         "credentialsParameter": "arn:aws:secretsmanager:region:account:secret:name"
       }
     }]
   }
   ```

#### Problem: Out of Memory

**Symptoms:**
```
OutOfMemoryError: Container killed due to memory limit
```

**Solution:**
1. Increase memory limits:
   ```json
   {
     "memory": "1024",
     "memoryReservation": "512"
   }
   ```

2. Check memory usage:
   ```bash
   kubectl top pod -n <cluster-name>
   ```

3. Optimize application memory usage

### Networking Issues

#### Problem: Service Discovery Not Working

**Symptoms:**
- Services can't communicate
- DNS resolution fails

**Solution:**
1. Check service registration:
   ```bash
   aws servicediscovery list-services \
     --endpoint-url http://localhost:4566
   ```

2. Test DNS resolution:
   ```bash
   kubectl exec -n <namespace> <pod> -- nslookup <service-name>
   ```

3. Verify network policies:
   ```bash
   kubectl get networkpolicies -n <namespace>
   ```

#### Problem: Load Balancer Not Working

**Symptoms:**
- Can't access service externally
- Health checks failing

**Solution:**
1. Check service type:
   ```bash
   kubectl get svc -n <namespace>
   ```

2. Verify target health:
   ```bash
   aws elbv2 describe-target-health \
     --target-group-arn <arn> \
     --endpoint-url http://localhost:4566
   ```

3. Check security groups and ports

### LocalStack Integration Issues

#### Problem: LocalStack Connection Failed

**Symptoms:**
```
Could not connect to the endpoint URL: "http://localhost:4566/"
```

**Solution:**
1. Verify LocalStack is running:
   ```bash
   docker ps | grep localstack
   curl http://localhost:4566/_localstack/health
   ```

2. Check KECS configuration:
   ```yaml
   localstack:
     enabled: true
     endpoint: http://localhost:4566
   ```

3. Restart both services:
   ```bash
   docker-compose restart
   ```

#### Problem: AWS SDK Not Using LocalStack

**Symptoms:**
- Requests going to real AWS
- Authentication errors

**Solution:**
1. Check sidecar injection:
   ```bash
   kubectl describe pod <pod> -n <namespace> | grep localstack-proxy
   ```

2. Set AWS endpoint explicitly:
   ```python
   boto3.client('s3', endpoint_url='http://localhost:4566')
   ```

3. Verify environment variables:
   ```bash
   kubectl exec <pod> -n <namespace> -- env | grep AWS
   ```

### Performance Issues

#### Problem: Slow API Responses

**Solution:**
1. Check resource usage:
   ```bash
   # KECS server
   top -p $(pgrep kecs)
   
   # Database
   ls -la ~/.kecs/data/kecs.db
   ```

2. Enable performance metrics:
   ```bash
   curl http://localhost:8081/metrics
   ```

3. Optimize database:
   ```bash
   # Vacuum database
   sqlite3 ~/.kecs/data/kecs.db "VACUUM;"
   ```

#### Problem: High Memory Usage

**Solution:**
1. Check for memory leaks:
   ```bash
   go tool pprof http://localhost:8081/debug/pprof/heap
   ```

2. Limit concurrent operations:
   ```yaml
   server:
     maxConcurrentRequests: 100
   ```

3. Adjust cache settings:
   ```yaml
   cache:
     maxSize: 1000
     ttl: 5m
   ```

## Advanced Debugging

### Enable Verbose Logging

```bash
# All components
export KECS_LOG_LEVEL=trace

# Specific components
export KECS_API_LOG_LEVEL=debug
export KECS_STORAGE_LOG_LEVEL=trace
export KECS_K8S_LOG_LEVEL=debug
```

### Trace Requests

```bash
# Enable request tracing
curl -H "X-Debug-Trace: true" \
  -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

### Database Inspection

```bash
# Open database
sqlite3 ~/.kecs/data/kecs.db

# List tables
.tables

# Check clusters
SELECT * FROM clusters;

# Check services
SELECT * FROM services WHERE cluster_arn = 'arn:...';
```

### Kubernetes Debugging

```bash
# Get all resources in namespace
kubectl get all -n <cluster-name>

# Describe problematic pod
kubectl describe pod <pod-name> -n <cluster-name>

# Get pod events
kubectl get events -n <cluster-name> --sort-by='.lastTimestamp'

# Debug container
kubectl debug -it <pod-name> -n <cluster-name> --image=busybox
```

## Getting Help

### Collect Diagnostic Information

Run the diagnostic script:
```bash
./scripts/collect-diagnostics.sh
```

This collects:
- KECS logs
- Configuration files
- Kubernetes cluster state
- System information

### Report Issues

When reporting issues, include:

1. **Environment Details**
   - KECS version: `kecs version`
   - OS: `uname -a`
   - Kubernetes version: `kubectl version`
   - Docker version: `docker version`

2. **Steps to Reproduce**
   - Exact commands run
   - Configuration files used
   - Expected vs actual behavior

3. **Logs and Errors**
   - KECS server logs
   - Relevant Kubernetes events
   - Error messages

4. **Diagnostic Bundle**
   - Output from diagnostic script

### Community Support

- GitHub Issues: [github.com/nandemo-ya/kecs/issues](https://github.com/nandemo-ya/kecs/issues)

## Prevention Tips

### Regular Maintenance

1. **Update Regularly**
   ```bash
   git pull origin main
   make build
   ```

2. **Monitor Resources**
   - Set up alerts for disk space
   - Monitor memory usage
   - Track API response times

3. **Backup Data**
   ```bash
   # Backup database
   cp ~/.kecs/data/kecs.db ~/.kecs/data/kecs.db.backup
   ```

4. **Clean Up Resources**
   ```bash
   # Remove stopped tasks
   kubectl delete pods -n <namespace> --field-selector=status.phase=Succeeded
   
   # Prune unused images
   docker image prune -a
   ```

### Best Practices

1. **Use Resource Limits**
   - Set appropriate CPU/memory limits
   - Monitor actual usage
   - Leave headroom for spikes

2. **Enable Health Checks**
   - Configure liveness probes
   - Set readiness probes
   - Monitor health metrics

3. **Plan for Failures**
   - Test failure scenarios
   - Document recovery procedures
   - Keep backups current

4. **Stay Informed**
   - Read release notes
   - Follow security advisories
   - Join community discussions