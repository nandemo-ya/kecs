#!/bin/bash

echo "Initializing LocalStack resources..."

# Create DynamoDB tables
awslocal dynamodb create-table \
  --table-name users \
  --attribute-definitions \
    AttributeName=id,AttributeType=S \
    AttributeName=email,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --global-secondary-indexes \
    IndexName=email-index,Keys=[{AttributeName=email,KeyType=HASH}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5} \
  --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal dynamodb create-table \
  --table-name orders \
  --attribute-definitions \
    AttributeName=id,AttributeType=S \
    AttributeName=userId,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --global-secondary-indexes \
    IndexName=userId-index,Keys=[{AttributeName=userId,KeyType=HASH}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5} \
  --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

# Create S3 bucket
awslocal s3 mb s3://microservices-storage

# Create SQS queues
awslocal sqs create-queue --queue-name notification-queue
awslocal sqs create-queue --queue-name notification-dlq

# Create SNS topics
awslocal sns create-topic --name order-events
awslocal sns create-topic --name user-events

# Subscribe SQS to SNS
QUEUE_URL=$(awslocal sqs get-queue-url --queue-name notification-queue --query 'QueueUrl' --output text)
TOPIC_ARN=$(awslocal sns list-topics --query "Topics[?contains(TopicArn, 'order-events')].TopicArn" --output text)
awslocal sns subscribe --topic-arn $TOPIC_ARN --protocol sqs --notification-endpoint $QUEUE_URL

# Create CloudWatch log groups
awslocal logs create-log-group --log-group-name /ecs/api-gateway
awslocal logs create-log-group --log-group-name /ecs/user-service
awslocal logs create-log-group --log-group-name /ecs/order-service
awslocal logs create-log-group --log-group-name /ecs/storage-service
awslocal logs create-log-group --log-group-name /ecs/notification-service

# Create Cloud Map namespace and services
awslocal servicediscovery create-private-dns-namespace \
  --name microservices.local \
  --vpc vpc-default

NAMESPACE_ID=$(awslocal servicediscovery list-namespaces --query "Namespaces[?Name=='microservices.local'].Id" --output text)

# Create services in Cloud Map
awslocal servicediscovery create-service \
  --name user-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]"

awslocal servicediscovery create-service \
  --name order-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]"

awslocal servicediscovery create-service \
  --name storage-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]"

awslocal servicediscovery create-service \
  --name notification-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]"

echo "LocalStack initialization complete"