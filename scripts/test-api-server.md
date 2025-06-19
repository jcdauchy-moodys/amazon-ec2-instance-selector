# API Server Test Script

This script (`test-api-server.sh`) helps test the EC2 Instance Selector API server with InfluxDB metrics enabled.

## Prerequisites

- curl command-line tool
- Bash shell environment (Git Bash or WSL on Windows)
- Built API server binary in `build/api-server.exe`
- Running InfluxDB instance with JWT authentication

## Configuration

At the top of the script, configure these values to match your InfluxDB setup:
```bash
INFLUXDB_URL="http://localhost:8086"  # Your InfluxDB URL
INFLUXDB_DATABASE="ec2metrics"        # Your database name
INFLUXDB_JWT="your-jwt-token-here"    # Your JWT token
API_SERVER_PORT=8080                  # Port for the API server
```

## Usage

1. First, build the API server:
```bash
scripts/build.cmd
```

2. Configure your JWT token in the script:
```bash
INFLUXDB_JWT="your-actual-jwt-token"
```

3. Ensure your InfluxDB instance is running and accessible

4. Run the test script:
```bash
# On Windows with Git Bash or WSL:
./scripts/test-api-server.sh

# On Linux/macOS:
scripts/test-api-server.sh
```

## Test Cases

The script runs two test cases:

1. Basic Instance Filtering
   - 2 vCPUs
   - 4 GiB memory
   - us-east-1 region

2. Advanced Filtering
   - 4-8 vCPUs
   - 8-16 GiB memory
   - us-west-2 region

## Metrics Verification

The script verifies metrics by querying InfluxDB for:
1. Instance counts by region
2. Average memory by instance family

All InfluxDB queries are authenticated using the provided JWT token.

## Cleanup

The script automatically stops the API server when you press Ctrl+C.

## Troubleshooting

1. If InfluxDB authentication fails:
   ```
   Error: Cannot connect to InfluxDB at http://localhost:8086
   ```
   Solution: Check your JWT token and ensure it's valid

2. If the API server port is in use:
   ```
   Error: listen tcp :8080: bind: address already in use
   ```
   Solution: Modify API_SERVER_PORT in the script

3. If the API server fails to start:
   ```
   Timeout waiting for API server
   ```
   Solution: Check the API server logs for errors

## Security Note

The JWT token is sensitive information. Consider:
- Not committing the script with your actual token
- Setting the token via environment variable
- Using a configuration file for sensitive values 