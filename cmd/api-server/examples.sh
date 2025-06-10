#!/bin/bash

# Example usage of the EC2 Instance Selector REST API

echo "==============================================="
echo "EC2 Instance Selector REST API Examples"
echo "==============================================="
echo ""

# Make sure the server is running
echo "Make sure to start the server first with:"
echo "go run cmd/api-server/main.go"
echo ""

BASE_URL="http://localhost:8080"

echo "1. Health Check"
echo "curl $BASE_URL/health"
curl -s "$BASE_URL/health" | jq '.'
echo ""
echo ""

echo "2. Find instances with 4 vCPUs and 8GB memory (GET)"
echo "curl \"$BASE_URL/api/v1/instances?vcpus=4&memory=8gb\""
curl -s "$BASE_URL/api/v1/instances?vcpus=4&memory=8gb" | jq '.'
echo ""
echo ""

echo "3. Find instances with 4 vCPUs and 8GB memory (POST)"
echo "curl -X POST $BASE_URL/api/v1/instances/filter \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"vcpus\": 4, \"memory\": \"8gb\", \"current_generation\": true}'"
curl -s -X POST "$BASE_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{"vcpus": 4, "memory": "8gb", "current_generation": true}' | jq '.'
echo ""
echo ""

echo "4. Find GPU instances"
echo "curl -X POST $BASE_URL/api/v1/instances/filter \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"gpus_min\": 1, \"max_results\": 5}'"
curl -s -X POST "$BASE_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{"gpus_min": 1, "max_results": 5}' | jq '.'
echo ""
echo ""

echo "5. Find instances in specific availability zones"
echo "curl \"$BASE_URL/api/v1/instances?availability_zones=eu-west-1a,eu-west-1b&vcpus_min=2&vcpus_max=8&max_results=5\""
curl -s "$BASE_URL/api/v1/instances?availability_zones=eu-west-1a,eu-west-1b&vcpus_min=2&vcpus_max=8&max_results=5" | jq '.'
echo ""
echo ""

echo "6. Find instances with regex patterns"
echo "curl -X POST $BASE_URL/api/v1/instances/filter \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"allow_list\": \"m[5-6]\\\\.*\", \"current_generation\": true, \"max_results\": 10}'"
curl -s -X POST "$BASE_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{"allow_list": "m[5-6]\\.*", "current_generation": true, "max_results": 10}' | jq '.'
echo ""
echo ""

echo "7. Find ARM-based instances"
echo "curl \"$BASE_URL/api/v1/instances?cpu_architecture=arm64&max_results=5\""
curl -s "$BASE_URL/api/v1/instances?cpu_architecture=arm64&max_results=5" | jq '.'
echo ""
echo ""

echo "8. Find burstable instances"
echo "curl -X POST $BASE_URL/api/v1/instances/filter \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"burstable\": true, \"max_results\": 5}'"
curl -s -X POST "$BASE_URL/api/v1/instances/filter" \
  -H "Content-Type: application/json" \
  -d '{"burstable": true, "max_results": 5}' | jq '.'
echo ""
echo ""

echo "Examples completed!"
echo ""
echo "Note: If you don't have 'jq' installed for JSON formatting, you can:"
echo "  - Install it: sudo apt-get install jq (Linux) or brew install jq (Mac)"
echo "  - Or remove '| jq \".\"' from the commands above" 