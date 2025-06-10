# EC2 Instance Selector REST API

A REST API server built on top of the Amazon EC2 Instance Selector library that provides JSON endpoints for filtering EC2 instance types.

## Features

- **RESTful endpoints** for filtering EC2 instance types
- **JSON input/output** for easy integration
- **Query parameter support** for GET requests
- **JSON body support** for POST requests
- **AWS SDK integration** with automatic credential discovery
- **CORS support** for web applications
- **Health check endpoint**

## Quick Start

### Prerequisites

- Go 1.19+ installed
- AWS credentials configured (AWS CLI, environment variables, or IAM roles)
- Access to AWS EC2 API

### Running the Server

```bash
# Build and run the API server
go run cmd/api-server/main.go
```

The server will start on port 8080 by default.

## API Endpoints

### Health Check
```
GET /health
```

Returns server health status:
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### Filter Instances (GET)
```
GET /api/v1/instances?vcpus=4&memory=8gb&cpu_architecture=x86_64
```

**Query Parameters:**
- `vcpus` - Number of vCPUs (exact match)
- `vcpus_min` - Minimum vCPUs
- `vcpus_max` - Maximum vCPUs
- `memory` - Memory amount (e.g., "4gb", "8 GiB")
- `memory_min` - Minimum memory
- `memory_max` - Maximum memory
- `cpu_architecture` - CPU architecture (x86_64, arm64)
- `instance_types` - Comma-separated list of instance types
- `allow_list` - Regex pattern for allowed instance types
- `deny_list` - Regex pattern for denied instance types
- `current_generation` - Filter for current generation (true/false)
- `bare_metal` - Bare metal instances (true/false)
- `burstable` - Burstable instances (true/false)
- `free_tier` - Free tier eligible (true/false)
- `max_results` - Maximum number of results (default: 20)
- `availability_zones` - Comma-separated list of AZs
- `usage_class` - Usage class (on-demand, spot)
- `gpus` - Number of GPUs (exact match)
- `gpus_min` - Minimum GPUs
- `gpus_max` - Maximum GPUs
- `network_performance` - Network performance in Gbps

### Filter Instances (POST)
```
POST /api/v1/instances/filter
Content-Type: application/json
```

**Request Body:**
```json
{
  "vcpus": 4,
  "memory": "8gb",
  "cpu_architecture": "x86_64",
  "current_generation": true,
  "max_results": 10
}
```

**Response:**
```json
{
  "success": true,
  "instance_types": [
    {
      "InstanceType": "m5.xlarge",
      "VCpuInfo": {
        "DefaultVCpus": 4,
        "DefaultCores": 2,
        "DefaultThreadsPerCore": 2
      },
      "MemoryInfo": {
        "SizeInMiB": 16384
      },
      "ProcessorInfo": {
        "SupportedArchitectures": ["x86_64"]
      },
      "CurrentGeneration": true,
      "OndemandPricePerHour": 0.192,
      "SpotPrice": 0.058
    }
  ],
  "count": 1
}
```

## Usage Examples

### Example 1: Find instances with 4 vCPUs and 8GB memory

**GET Request:**
```bash
curl "http://localhost:8080/api/v1/instances?vcpus=4&memory=8gb"
```

**POST Request:**
```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus": 4,
    "memory": "8gb"
  }'
```

### Example 2: Find GPU instances

```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "gpus_min": 1,
    "current_generation": true,
    "max_results": 5
  }'
```

### Example 3: Find instances in specific availability zones

```bash
curl "http://localhost:8080/api/v1/instances?availability_zones=us-east-1a,us-east-1b&vcpus_min=2&vcpus_max=8"
```

### Example 4: Find instances with regex patterns

```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "m[5-6]\\.*",
    "current_generation": true,
    "max_results": 15
  }'
```

## Response Format

All successful responses follow this format:

```json
{
  "success": true,
  "instance_types": [...],
  "count": 5
}
```

Error responses:

```json
{
  "success": false,
  "message": "Invalid filter parameters: invalid memory format",
  "count": 0
}
```

## Memory Format

Memory values can be specified in several formats:
- `"4gb"` or `"4 GB"`
- `"8gib"` or `"8 GiB"`
- `"1024mb"` or `"1024 MB"`
- `"2tb"` or `"2 TB"`

## Configuration

The server uses AWS SDK default configuration for credentials:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM roles (for EC2 instances)

You can also set the AWS region:
```bash
export AWS_REGION=us-west-2
```

## Building

To build a standalone binary:

```bash
go build -o ec2-api-server cmd/api-server/main.go
./ec2-api-server
```

## Docker

You can also containerize the API server:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o ec2-api-server cmd/api-server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ec2-api-server .
EXPOSE 8080
CMD ["./ec2-api-server"]
```

Build and run:
```bash
docker build -t ec2-instance-selector-api .
docker run -p 8080:8080 -e AWS_ACCESS_KEY_ID=xxx -e AWS_SECRET_ACCESS_KEY=yyy ec2-instance-selector-api
```

## Integration

This API can be easily integrated into:
- Web applications (React, Vue, Angular)
- Mobile applications
- Infrastructure automation tools
- CI/CD pipelines
- Cost optimization tools
- Instance recommendation systems

The JSON responses contain all the detailed instance information including pricing, specifications, and capabilities, making it perfect for building instance selection UIs or automated infrastructure provisioning tools. 