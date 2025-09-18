#!/bin/bash
# Test script for TUI Backend API

set -e

API_URL="${KECS_API_ENDPOINT:-http://localhost:8081}"

echo "Testing TUI Backend API at $API_URL"
echo "=================================="

# Test 1: List instances
echo -e "\n1. Testing GET /api/instances"
curl -s "$API_URL/api/instances" | jq .

# Test 2: Get default instance
echo -e "\n2. Testing GET /api/instances/default"
curl -s "$API_URL/api/instances/default" | jq .

# Test 3: Instance health check
echo -e "\n3. Testing GET /api/instances/default/health"
curl -s "$API_URL/api/instances/default/health" | jq .

# Test 4: List clusters via proxy
echo -e "\n4. Testing POST /api/instances/default/clusters (ListClusters)"
curl -s -X POST "$API_URL/api/instances/default/clusters" \
  -H "Content-Type: application/json" \
  -d '{}' | jq .

# Test 5: Describe clusters
echo -e "\n5. Testing POST /api/instances/default/clusters/describe (DescribeClusters)"
curl -s -X POST "$API_URL/api/instances/default/clusters/describe" \
  -H "Content-Type: application/json" \
  -d '{"clusters":["default"]}' | jq .

# Test 6: List services
echo -e "\n6. Testing POST /api/instances/default/services (ListServices)"
curl -s -X POST "$API_URL/api/instances/default/services" \
  -H "Content-Type: application/json" \
  -d '{"cluster":"default"}' | jq .

# Test 7: CORS preflight
echo -e "\n7. Testing CORS preflight request"
curl -s -X OPTIONS "$API_URL/api/instances" \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v 2>&1 | grep -E "(Access-Control|< HTTP)"

echo -e "\nAll tests completed!"