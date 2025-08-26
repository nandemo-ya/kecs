#!/bin/bash

# Test script for ECS Container Instance APIs

ENDPOINT="http://localhost:5373"

echo "=== Testing RegisterContainerInstance API ==="
aws ecs register-container-instance \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1 \
  --instance-identity-document "mock-document" \
  --instance-identity-document-signature "mock-signature" \
  --total-resources name=CPU,type=INTEGER,integerValue=2048 name=MEMORY,type=INTEGER,integerValue=4096 \
  --version-info agentVersion=1.51.0,agentHash=4023248,dockerVersion=20.10.7

echo -e "\n=== Testing ListContainerInstances API ==="
aws ecs list-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1

echo -e "\n=== Testing ListContainerInstances API with status filter ==="
aws ecs list-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --status ACTIVE \
  --region ap-northeast-1

echo -e "\n=== Testing DescribeContainerInstances API ==="
# First get the container instance ARN from list
INSTANCE_ARN=$(aws ecs list-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1 \
  --query 'containerInstanceArns[0]' \
  --output text)

echo "Using container instance ARN: $INSTANCE_ARN"

aws ecs describe-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --container-instances "$INSTANCE_ARN" \
  --region ap-northeast-1

echo -e "\n=== Testing DeregisterContainerInstance API ==="
aws ecs deregister-container-instance \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --container-instance "$INSTANCE_ARN" \
  --force \
  --region ap-northeast-1

echo -e "\n=== Testing ListContainerInstances API after deregistration ==="
aws ecs list-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1