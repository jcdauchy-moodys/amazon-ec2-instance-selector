#!/bin/bash

# Test script for filter cache functionality
# This script demonstrates cache hits and misses

set -e

export API_URL="https://monitoring-cluster-brt-nprd.bankingcloud.moodysanalytics.net/aws"

API_URL="${API_URL:-http://localhost:8080}"

echo "=========================================="
echo "EC2 Instance Selector Filter Cache Test"
echo "=========================================="
echo ""
echo "API URL: $API_URL"
echo ""

# Test 1: Cache MISS (first request)
echo "Test 1: First request (expect cache MISS)"
echo "Requesting: 4 vCPUs, 16GB memory, current generation"
echo ""
time curl -s -X POST "$API_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 4,
    "memory": "16gb",
    "current_generation": true,
    "max_results": 5
  }' | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "---"
echo ""

# Wait a moment
sleep 1

# Test 2: Cache HIT (identical request)
echo "Test 2: Identical request (expect cache HIT - much faster)"
echo "Requesting: 4 vCPUs, 16GB memory, current generation"
echo ""
time curl -s -X POST "$API_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 4,
    "memory": "16gb",
    "current_generation": true,
    "max_results": 5
  }' | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "---"
echo ""

# Wait a moment
sleep 1

# Test 3: Cache HIT with different sort (reuses same cache)
echo "Test 3: Same filters, different sort (expect cache HIT)"
echo "Requesting: 4 vCPUs, 16GB memory, sorted by memory"
echo ""
time curl -s -X POST "$API_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 4,
    "memory": "16gb",
    "current_generation": true,
    "sort_by": "memory",
    "sort_direction": "desc",
    "max_results": 3
  }' | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "---"
echo ""

# Wait a moment
sleep 1

# Test 4: Cache MISS (different parameters)
echo "Test 4: Different parameters (expect cache MISS)"
echo "Requesting: 8 vCPUs, 32GB memory, current generation"
echo ""
time curl -s -X POST "$API_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 8,
    "memory": "32gb",
    "current_generation": true,
    "max_results": 5
  }' | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "---"
echo ""

# Wait a moment
sleep 1

# Test 5: Cache HIT (GET request with same params as Test 1)
echo "Test 5: GET request with same filters as Test 1 (expect cache HIT)"
echo "Requesting: 4 vCPUs, 16GB memory via GET"
echo ""
time curl -s "$API_URL/api/v1/instances?vcpus=4&memory=16gb&current_generation=true&max_results=5" \
  | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "---"
echo ""

# Wait a moment
sleep 1

# Test 6: Cache MISS (different region)
echo "Test 6: Same parameters, different region (expect cache MISS)"
echo "Requesting: 4 vCPUs, 16GB memory in eu-central-1"
echo ""
time curl -s -X POST "$API_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 4,
    "memory": "16gb",
    "current_generation": true,
    "region": "eu-central-1",
    "max_results": 5
  }' | jq -r '.count, .instance_types[0].InstanceType' 2>/dev/null || echo "Request failed"
echo ""
echo "=========================================="
echo "Test Complete!"
echo "=========================================="
echo ""
echo "Check server logs for cache HIT/MISS messages:"
echo "  - 'Cache HIT' means data was served from cache (fast)"
echo "  - 'Cache MISS' means data was fetched from AWS (slower)"
echo ""
echo "You should see:"
echo "  Test 1: Cache MISS (first time)"
echo "  Test 2: Cache HIT (identical request)"
echo "  Test 3: Cache HIT (different sort uses same cache)"
echo "  Test 4: Cache MISS (different parameters)"
echo "  Test 5: Cache HIT (GET request matches POST from Test 1)"
echo "  Test 6: Cache MISS (different region)"
echo ""

