#!/bin/bash
set -e

# Default values - modify these to match your InfluxDB setup
INFLUXDB_URL="https://monitoring-cluster-brt-nprd.bankingcloud.moodysanalytics.net/influxdb"
INFLUXDB_DATABASE="ec2metrics"
INFLUXDB_JWT="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJpbmZsdXhkYiIsImNsdXN0ZXIiOiJiNzgwNmE3YzNiMmVmMjkzNGJmZGUxYjgyMmU3ZmViMi5ncjcudXMtd2VzdC0yLmVrcy5hbWF6b25hd3MuY29tIiwiZXhwIjoxNzY4NTc1NDY3LCJraWQiOiJtb25pdG9yaW5nLXN0YWNrIiwic2NvcGUiOnt9fQ.qh524sjKH4SFIhrc2uufzUFZu99VNeWXZ/GoWINvhZG3PFzDWW5G24O2Lwuyb3SVyHAA-W0pgir0rcqI7x0wvlbbc+j1UXLNWW1TP7iKG/djfN9Vzi7WTKEn3ASFo9HrxChM1i1cYls0m2jmO3Z8lFzbiTHTe28BoMq4iN0Qib+Y8h3kbzEg/OFCDGzEcW30eBtHwnCRsufomTUzU37naWCBW0gw8a4xoN/pcEoS/tkT7Jbgacgk7ErDCLARcdnE25P7qx2C+M6fLfF3P6YKSnHRxrc4gEpBTk+ZVhTc0QjawPhYClb4WCqboIMOl4dXL+VtPn/nLoIC3l4L3aAsJg"  # Replace with your JWT token
API_SERVER_PORT=8080

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if InfluxDB is accessible
echo -e "${YELLOW}Checking InfluxDB connection...${NC}"
if ! curl -s "${INFLUXDB_URL}/ping" -H "Authorization: Bearer ${INFLUXDB_JWT}" >/dev/null; then
    echo -e "${RED}Error: Cannot connect to InfluxDB at ${INFLUXDB_URL}${NC}"
    echo "Please ensure InfluxDB is running and accessible, and the JWT token is valid"
    exit 1
fi
echo -e "${GREEN}InfluxDB is accessible!${NC}"

# Start the API server with metrics enabled
echo -e "${GREEN}Starting API server with metrics enabled...${NC}"
export INFLUXDB_ENABLED=true
export INFLUXDB_URL="${INFLUXDB_URL}"
export INFLUXDB_DATABASE="${INFLUXDB_DATABASE}"
export INFLUXDB_JWT="${INFLUXDB_JWT}"
export PORT="${API_SERVER_PORT}"

# Run API server in background
./build/api-server.exe &
API_SERVER_PID=$!

# Function to cleanup on script exit
cleanup() {
    echo -e "${YELLOW}Stopping API server...${NC}"
    kill $API_SERVER_PID 2>/dev/null || true
}

# Register cleanup function
trap cleanup EXIT

# Wait for API server to be ready
echo -e "${YELLOW}Waiting for API server to start...${NC}"
for i in {1..30}; do
    if curl -s "http://localhost:${API_SERVER_PORT}/health" >/dev/null; then
        echo -e "${GREEN}API server is ready!${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}Timeout waiting for API server${NC}"
        exit 1
    fi
    sleep 1
done

# Test API endpoints and generate metrics
echo -e "${GREEN}Testing API endpoints...${NC}"

# Test 1: Basic instance filtering
echo -e "${YELLOW}Test 1: Basic instance filtering${NC}"
curl -X POST "http://localhost:${API_SERVER_PORT}/api/v1/instances/filter" \
    -H "Content-Type: application/json" \
    -d '{
        "vcpus": 2,
        "memory": "4GiB",
        "region": "us-east-1"
    }'
echo -e "\n"

# Test 2: Advanced filtering
echo -e "${YELLOW}Test 2: Advanced filtering${NC}"
curl -X POST "http://localhost:${API_SERVER_PORT}/api/v1/instances/filter" \
    -H "Content-Type: application/json" \
    -d '{
        "vcpus_min": 4,
        "vcpus_max": 8,
        "memory_min": "8GiB",
        "memory_max": "16GiB",
        "region": "us-west-2"
    }'
echo -e "\n"

# Wait a moment for metrics to be written
sleep 2

# Query InfluxDB to verify metrics
echo -e "${GREEN}Verifying metrics in InfluxDB...${NC}"
echo -e "${YELLOW}Instance counts by region:${NC}"
curl -G "${INFLUXDB_URL}/query" \
    -H "Authorization: Bearer ${INFLUXDB_JWT}" \
    --data-urlencode "db=${INFLUXDB_DATABASE}" \
    --data-urlencode "q=SELECT COUNT(DISTINCT(\"instance_type\")) FROM \"ec2_instances\" GROUP BY \"region\""
echo -e "\n"

echo -e "${YELLOW}Memory distribution by family:${NC}"
curl -G "${INFLUXDB_URL}/query" \
    -H "Authorization: Bearer ${INFLUXDB_JWT}" \
    --data-urlencode "db=${INFLUXDB_DATABASE}" \
    --data-urlencode "q=SELECT MEAN(\"memory_gb\") FROM \"ec2_instances\" GROUP BY \"family\" LIMIT 5"
echo -e "\n"

echo -e "${GREEN}Test completed successfully!${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the API server${NC}"

# Wait for Ctrl+C
wait $API_SERVER_PID 