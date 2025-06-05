#!/bin/bash

# Test script for ECS Capacity Provider APIs

ENDPOINT="http://localhost:8080"

echo "=== Testing DescribeCapacityProviders API (list default providers) ==="
aws ecs describe-capacity-providers \
  --endpoint-url $ENDPOINT \
  --region ap-northeast-1

echo -e "\n=== Testing CreateCapacityProvider API ==="
aws ecs create-capacity-provider \
  --endpoint-url $ENDPOINT \
  --name my-capacity-provider \
  --region ap-northeast-1 \
  --auto-scaling-group-provider file:///dev/stdin <<EOF
{
  "autoScalingGroupArn": "arn:aws:autoscaling:ap-northeast-1:123456789012:autoScalingGroup:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg",
  "managedScaling": {
    "status": "ENABLED",
    "targetCapacity": 100,
    "minimumScalingStepSize": 1,
    "maximumScalingStepSize": 10000
  }
}
EOF

echo -e "\n=== Testing DescribeCapacityProviders API (specific provider) ==="
aws ecs describe-capacity-providers \
  --endpoint-url $ENDPOINT \
  --capacity-providers my-capacity-provider \
  --region ap-northeast-1

echo -e "\n=== Testing UpdateCapacityProvider API ==="
aws ecs update-capacity-provider \
  --endpoint-url $ENDPOINT \
  --name my-capacity-provider \
  --region ap-northeast-1 \
  --auto-scaling-group-provider file:///dev/stdin <<EOF
{
  "autoScalingGroupArn": "arn:aws:autoscaling:ap-northeast-1:123456789012:autoScalingGroup:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg",
  "managedScaling": {
    "status": "ENABLED",
    "targetCapacity": 150,
    "minimumScalingStepSize": 1,
    "maximumScalingStepSize": 10000
  }
}
EOF

echo -e "\n=== Testing DeleteCapacityProvider API ==="
aws ecs delete-capacity-provider \
  --endpoint-url $ENDPOINT \
  --capacity-provider my-capacity-provider \
  --region ap-northeast-1