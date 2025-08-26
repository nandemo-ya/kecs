#!/bin/bash

# Test script for ECS Tag Management APIs

ENDPOINT="http://localhost:5373"

echo "=== Create a sample cluster for testing ==="
aws ecs create-cluster \
  --endpoint-url $ENDPOINT \
  --cluster-name test-cluster \
  --region ap-northeast-1

CLUSTER_ARN="arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster"

echo -e "\n=== Testing TagResource API on cluster ==="
aws ecs tag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$CLUSTER_ARN" \
  --tags key=Environment,value=Development key=Team,value=Platform key=Project,value=KECS \
  --region ap-northeast-1

echo -e "\n=== Testing ListTagsForResource API on cluster ==="
aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$CLUSTER_ARN" \
  --region ap-northeast-1

echo -e "\n=== Create a service for testing ==="
aws ecs create-service \
  --endpoint-url $ENDPOINT \
  --cluster test-cluster \
  --service-name test-service \
  --task-definition test-app:1 \
  --desired-count 3 \
  --region ap-northeast-1

SERVICE_ARN="arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/test-service"

echo -e "\n=== Testing TagResource API on service ==="
aws ecs tag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$SERVICE_ARN" \
  --tags key=Application,value=WebApp key=Version,value=1.0.0 key=Owner,value=TeamA \
  --region ap-northeast-1

echo -e "\n=== Testing ListTagsForResource API on service ==="
aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$SERVICE_ARN" \
  --region ap-northeast-1

echo -e "\n=== Testing UntagResource API on service ==="
aws ecs untag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$SERVICE_ARN" \
  --tag-keys Owner \
  --region ap-northeast-1

echo -e "\n=== Verify tag removal ==="
aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$SERVICE_ARN" \
  --region ap-northeast-1

echo -e "\n=== Test with task definition ARN ==="
TASK_DEF_ARN="arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-app:1"

aws ecs tag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$TASK_DEF_ARN" \
  --tags key=Component,value=Backend key=Language,value=Go \
  --region ap-northeast-1

aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$TASK_DEF_ARN" \
  --region ap-northeast-1

echo -e "\n=== Test with container instance ARN ==="
INSTANCE_ARN="arn:aws:ecs:ap-northeast-1:123456789012:container-instance/test-cluster/i-1234567890abcdef0"

aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$INSTANCE_ARN" \
  --region ap-northeast-1

echo -e "\n=== Test with capacity provider ARN ==="
CAPACITY_PROVIDER_ARN="arn:aws:ecs:ap-northeast-1:123456789012:capacity-provider/MyCapacityProvider"

aws ecs list-tags-for-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$CAPACITY_PROVIDER_ARN" \
  --region ap-northeast-1

echo -e "\n=== Test error case: Invalid ARN format ==="
aws ecs tag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "invalid-arn" \
  --tags key=Test,value=Value \
  --region ap-northeast-1 || echo "Expected error for invalid ARN"

echo -e "\n=== Test error case: Empty tags ==="
aws ecs tag-resource \
  --endpoint-url $ENDPOINT \
  --resource-arn "$CLUSTER_ARN" \
  --region ap-northeast-1 || echo "Expected error for empty tags"