# LocalStack Service Availability by Tier

## Overview

LocalStack provides AWS-compatible services for local development.
Services are available across different pricing tiers:

- **Free (Community)**: Open source, no cost
- **Base**: $39/month
- **Ultimate**: $89/month

## Service Tier Matrix

### Core Storage & Database Services

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| S3 | ✅ | ✅ | ✅ | Full API support |
| DynamoDB | ✅ | ✅ | ✅ | Full API support |
| SecretsManager | ✅ | ✅ | ✅ | Secret storage and rotation |

### Messaging & Streaming

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| SQS | ✅ | ✅ | ✅ | Standard and FIFO queues |
| SNS | ✅ | ✅ | ✅ | Topics and subscriptions |

### Identity & Security

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| IAM | ✅ | ✅ | ✅ | Users, roles, policies |
| STS | ✅ | ✅ | ✅ | Temporary credentials |
| KMS | ✅ | ✅ | ✅ | Key management |
| ACM | ✅ | ✅ | ✅ | Certificate management |
| SSM | ✅ | ✅ | ✅ | Parameter Store |

### Monitoring & Logging

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| CloudWatch | ✅ | ✅ | ✅ | Advanced log queries in Pro |
| CloudWatch Logs | ✅ | ✅ | ✅ | Filter patterns in Pro only |

### Compute

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| Lambda | ✅ | ✅ | ✅ | Kafka event sources & full Layers in Pro |
| EC2 | ⚠️ | ✅ | ✅ | Mock VM in Free, Docker VM in Base+, Libvirt in Ultimate |

### Container Services

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| **ECS** | ❌ | ✅ | ✅ | **NOT free - requires Base tier** |
| **ECR** | ❌ | ✅ | ✅ | **NOT free - requires Base tier** |

### Load Balancing

| Service | Free | Base | Ultimate | Notes |
|---------|------|------|----------|-------|
| **ELB/ALB** | ❌ | ✅ | ✅ | **NOT free - requires Base tier** |

## KECS Development Strategy

### Why KECS Exists

LocalStack's ECS service requires a paid tier ($39/mo minimum).
KECS provides a **free, local ECS-compatible environment** by:

1. Implementing ECS APIs on Kubernetes (k3d)
2. Leveraging LocalStack's **free tier services** for integrations

### Recommended Architecture

```
KECS (Free)                  LocalStack Free Tier
┌─────────────────┐          ┌─────────────────┐
│ ECS APIs        │          │ S3              │
│ - Clusters      │◄────────►│ SecretsManager  │
│ - Services      │          │ SSM             │
│ - Tasks         │          │ CloudWatch Logs │
│ - Task Defs     │          │ SQS/SNS         │
└─────────────────┘          │ DynamoDB        │
        │                    │ IAM/STS         │
        ▼                    └─────────────────┘
┌─────────────────┐
│ k3d/Kubernetes  │
└─────────────────┘
```

### Integration Points

KECS integrates with LocalStack for:

1. **SecretsManager** - Task definition secrets
2. **SSM Parameter Store** - Configuration values
3. **S3** - Environment files for containers
4. **CloudWatch Logs** - Container logging
5. **IAM/STS** - Credential simulation

### LocalStack Configuration for KECS

```bash
# Start LocalStack with free tier services
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=s3,secretsmanager,ssm,sqs,sns,dynamodb,iam,sts,logs \
  localstack/localstack

# Configure AWS CLI to use LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566
```

## Feature Limitations in Free Tier

### Lambda
- Basic function creation and invocation: ✅ Free
- Container image support: ✅ Free
- Event source mappings (SQS, DynamoDB, Kinesis): ✅ Free
- **Kafka event source mappings**: ❌ Pro only
- **Lambda Layers (full functionality)**: ❌ Pro only (creation works, but not applied on invoke)

### CloudWatch
- Log groups and streams: ✅ Free
- Basic log storage: ✅ Free
- **Advanced log queries/filters**: ❌ Pro only

### EC2
- Mock VM mode: ✅ Free (limited functionality)
- **Docker VM mode**: ❌ Base+ only
- **Libvirt VM mode**: ❌ Ultimate only

## References

- LocalStack Documentation: https://docs.localstack.cloud/
- Service Coverage: https://docs.localstack.cloud/aws/services/
- Pricing: https://www.localstack.cloud/pricing
