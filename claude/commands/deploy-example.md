# Deploy Example Command

This command deploys a KECS example with all required setup steps.

## Usage
```
/deploy-example <example-name>
```

Where `<example-name>` is one of:
- single-task-nginx
- multi-container-webapp
- microservice-with-elb
- service-with-secrets
- batch-job-simple

## What it does

1. Ensures KECS is running
2. Creates the default cluster if not exists
3. Sets up required AWS resources (IAM roles, log groups, etc.)
4. For examples with dependencies (like service-with-secrets), sets up LocalStack and creates secrets
5. Registers the task definition
6. Creates the service (if applicable)
7. Waits for deployment to complete
8. Shows deployment status

## Example

```
/deploy-example single-task-nginx
```

This will:
- Start KECS if needed
- Create cluster, IAM roles, and log group
- Deploy the nginx service
- Show the running tasks

## Implementation

When this command is invoked, execute the following based on the example name:

### For single-task-nginx:
```bash
# Ensure KECS is running
kecs status || kecs start

# Create cluster
aws ecs create-cluster --cluster-name default --endpoint-url http://localhost:8080

# Create log group
aws logs create-log-group --log-group-name /ecs/single-task-nginx --endpoint-url http://localhost:8080

# Create IAM roles
aws iam create-role --role-name ecsTaskExecutionRole --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ecs-tasks.amazonaws.com"},"Action":"sts:AssumeRole"}]}' --endpoint-url http://localhost:8080 || true
aws iam create-role --role-name ecsTaskRole --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ecs-tasks.amazonaws.com"},"Action":"sts:AssumeRole"}]}' --endpoint-url http://localhost:8080 || true

# Deploy using ecspresso
cd examples/single-task-nginx
ecspresso deploy --config ecspresso.yml

# Show status
ecspresso status --config ecspresso.yml
```

### For multi-container-webapp:
```bash
# Same initial setup as above, then:
cd examples/multi-container-webapp
ecspresso deploy --config ecspresso.yml
ecspresso status --config ecspresso.yml
```

### For microservice-with-elb:
```bash
# Initial setup, then:

# Start LocalStack for ALB support
docker run -d --name localstack -p 4566:4566 -e SERVICES=elbv2,iam,logs localstack/localstack || true

# Wait for LocalStack
until curl -s http://localhost:4566/_localstack/health | grep -q '"elbv2": "available"'; do sleep 2; done

# Create ALB resources
VPC_ID="vpc-12345678"  # Default VPC ID
ALB_ARN=$(aws elbv2 create-load-balancer --name microservice-alb --subnets subnet-12345678 subnet-87654321 --endpoint-url http://localhost:8080 --query 'LoadBalancers[0].LoadBalancerArn' --output text)
TG_ARN=$(aws elbv2 create-target-group --name microservice-api-tg --protocol HTTP --port 3000 --vpc-id $VPC_ID --target-type ip --health-check-path /health --endpoint-url http://localhost:8080 --query 'TargetGroups[0].TargetGroupArn' --output text)
aws elbv2 create-listener --load-balancer-arn $ALB_ARN --protocol HTTP --port 80 --default-actions Type=forward,TargetGroupArn=$TG_ARN --endpoint-url http://localhost:8080

# Update service definition with target group ARN
cd examples/microservice-with-elb
sed -i.bak "s|arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/microservice-api-tg/1234567890123456|$TG_ARN|" service_def.json

# Deploy
ecspresso deploy --config ecspresso.yml
ecspresso status --config ecspresso.yml
```

### For service-with-secrets:
```bash
# Initial setup, then:

# Start LocalStack for secrets
docker run -d --name localstack -p 4566:4566 -e SERVICES=secretsmanager,ssm,iam,logs,ecs localstack/localstack || true
until curl -s http://localhost:4566/_localstack/health | grep -q '"ssm": "available"'; do sleep 2; done

# Create SSM parameters
aws ssm put-parameter --name "/myapp/prod/database_url" --value "postgresql://app_user:password@db.example.com:5432/myapp" --type "SecureString" --endpoint-url http://localhost:4566
aws ssm put-parameter --name "/myapp/prod/api_key" --value "sk_live_abcdef123456789" --type "SecureString" --endpoint-url http://localhost:4566
aws ssm put-parameter --name "/myapp/prod/feature_flags" --value '{"new_ui": true, "beta_features": false}' --type "String" --endpoint-url http://localhost:4566

# Create secrets
aws secretsmanager create-secret --name "myapp/prod/db" --secret-string '{"password": "super-secret-db-password"}' --endpoint-url http://localhost:4566
aws secretsmanager create-secret --name "myapp/prod/jwt" --secret-string '{"secret": "jwt-signing-secret-key"}' --endpoint-url http://localhost:4566
aws secretsmanager create-secret --name "myapp/prod/encryption" --secret-string '{"key": "AES256-encryption-key"}' --endpoint-url http://localhost:4566

# Create IAM policy for secrets
aws iam create-policy --policy-name ECSSecretsPolicy --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ssm:GetParameter*","secretsmanager:GetSecretValue","kms:Decrypt"],"Resource":"*"}]}' --endpoint-url http://localhost:4566 || true
aws iam attach-role-policy --role-name ecsTaskExecutionRole --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy --endpoint-url http://localhost:4566 || true

# Deploy
cd examples/service-with-secrets
ecspresso deploy --config ecspresso.yml
ecspresso status --config ecspresso.yml
```

### For batch-job-simple:
```bash
# Initial setup, then:
cd examples/batch-job-simple

# Register task definition
ecspresso register --config ecspresso.yml

# Run the batch job
ecspresso run --config ecspresso.yml

# Show task status
aws ecs list-tasks --cluster default --desired-status RUNNING --endpoint-url http://localhost:8080
```