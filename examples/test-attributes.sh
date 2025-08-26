#!/bin/bash

# Test script for ECS Attribute management APIs

ENDPOINT="http://localhost:5373"

echo "=== Testing PutAttributes API ==="
aws ecs put-attributes \
  --endpoint-url $ENDPOINT \
  --attributes name=ecs.instance-type,value=t3.medium,targetType=container-instance \
  --cluster default \
  --region ap-northeast-1

echo -e "\n=== Testing ListAttributes API ==="
aws ecs list-attributes \
  --endpoint-url $ENDPOINT \
  --target-type container-instance \
  --cluster default \
  --region ap-northeast-1

echo -e "\n=== Testing DeleteAttributes API ==="
aws ecs delete-attributes \
  --endpoint-url $ENDPOINT \
  --attributes name=ecs.instance-type,targetType=container-instance \
  --cluster default \
  --region ap-northeast-1