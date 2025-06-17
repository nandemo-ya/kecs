# Admission Webhook for AWS SDK Proxy Sidecar Injection

This package provides a Kubernetes admission webhook that automatically injects an AWS SDK proxy sidecar into pods that need to communicate with LocalStack.

## Architecture

The admission webhook consists of:

1. **SidecarInjector**: Core logic for determining when to inject sidecars and creating the necessary patches
2. **WebhookServer**: HTTPS server that handles admission review requests from Kubernetes
3. **CertificateManager**: Manages TLS certificates for secure webhook communication
4. **WebhookIntegration**: Orchestrates the webhook lifecycle

## Sidecar Injection Logic

The sidecar is injected when:

1. Pod has annotation `kecs.io/inject-aws-proxy: "true"`
2. Pod has AWS environment variables (AWS_*)
3. Container definitions indicate AWS service usage

## Configuration

### Pod Annotations

- `kecs.io/inject-aws-proxy`: Set to "true" to force sidecar injection
- `kecs.io/localstack-endpoint`: Override LocalStack endpoint (default: http://localstack.localstack.svc.cluster.local:4566)
- `kecs.io/proxy-services`: Comma-separated list of services to proxy (default: s3,dynamodb,sqs,sns,ssm,secretsmanager,cloudwatch)

### Namespace Label

For the webhook to process pods in a namespace, the namespace must have the label:
```yaml
kecs.io/localstack-enabled: "true"
```

## Sidecar Container

The injected sidecar:
- Runs the AWS SDK proxy on port 8080
- Forwards AWS API calls to LocalStack
- Has minimal resource requirements (50m CPU, 64Mi memory)
- Includes health checks on /health endpoint

## Environment Variables

The webhook adds these environment variables to main containers:
- `AWS_ENDPOINT_URL`: Points to the local proxy (http://localhost:8080)
- `HTTPS_PROXY`: For AWS SDK v1 compatibility
- `NO_PROXY`: Excludes local traffic from proxying