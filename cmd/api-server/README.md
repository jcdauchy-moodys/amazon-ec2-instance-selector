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
- `memory_per_cpu_min` - Minimum memory per vCPU ratio (GiB per vCPU)
- `memory_per_cpu_max` - Maximum memory per vCPU ratio (GiB per vCPU)
- `cpu_architecture` - CPU architecture (x86_64, arm64)
- `instance_types` - Comma-separated list of instance types
- `allow_list` - Regex pattern for allowed instance types
- `deny_list` - Regex pattern for denied instance types
- `current_generation` - Filter for current generation (true/false)
- `bare_metal` - Bare metal instances (true/false)
- `burstable` - Burstable instances (true/false)
- `free_tier` - Free tier eligible (true/false)
- `nvme` - NVME storage support (true/false)
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
  "memory_per_cpu_min": 2.0,
  "memory_per_cpu_max": 8.0,
  "cpu_architecture": "x86_64",
  "current_generation": true,
  "nvme": true,
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

### Example 5: Find memory-optimized instances by memory-per-CPU ratio

```bash
# Find instances with at least 4 GiB of memory per vCPU
curl "http://localhost:8080/api/v1/instances?memory_per_cpu_min=4.0&current_generation=true"
```

```bash
# Find instances with balanced memory (2-4 GiB per vCPU)
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "memory_per_cpu_min": 2.0,
    "memory_per_cpu_max": 4.0,
    "current_generation": true,
    "max_results": 10
  }'
```

### Example 6: Find instances with NVME storage support

```bash
# Find instances that support NVME storage
curl "http://localhost:8080/api/v1/instances?nvme=true&current_generation=true&max_results=5"
```

```bash
# Find NVME instances with specific specs using POST
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "nvme": true,
    "vcpus_min": 4,
    "memory_min": "16gb",
    "current_generation": true,
    "max_results": 10
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

## Memory-per-CPU Filtering

The `memory_per_cpu_min` and `memory_per_cpu_max` parameters allow you to filter instances based on their memory-to-vCPU ratio (GiB of memory per vCPU).

**Examples:**
- **Compute-optimized** instances typically have `1-2 GiB per vCPU`
- **General-purpose** instances typically have `3-4 GiB per vCPU`  
- **Memory-optimized** instances typically have `8+ GiB per vCPU`

**Use Cases:**
```bash
# Find compute-optimized instances (good for CPU-intensive workloads)
curl "http://localhost:8080/api/v1/instances?memory_per_cpu_max=2.0"

# Find memory-optimized instances (good for in-memory databases)
curl "http://localhost:8080/api/v1/instances?memory_per_cpu_min=8.0"

# Find balanced instances (good for web applications)
curl "http://localhost:8080/api/v1/instances?memory_per_cpu_min=3.0&memory_per_cpu_max=5.0"
```

## Instance Family/Model Filtering

Filter by specific instance families or models using `allow_list` (regex patterns) or `instance_types` (exact matches).

### Common Instance Families

**Compute Optimized (C Family):**
```bash
# All current-gen C instances (c5, c5a, c5n, c6i, c6a, c7i, etc.)
curl "http://localhost:8080/api/v1/instances?allow_list=c[5-7].*&current_generation=true"

# Specific C5 instances only
curl "http://localhost:8080/api/v1/instances?allow_list=c5\\..*"

# C6i instances with NVME storage
curl "http://localhost:8080/api/v1/instances?allow_list=c6i\\..*&nvme=true"
```

**Memory Optimized (R Family):**
```bash
# All R5 family instances (r5, r5a, r5ad, r5d, r5dn, r5n)
curl "http://localhost:8080/api/v1/instances?allow_list=r5.*"

# R5d instances only (with NVMe SSD storage)
curl "http://localhost:8080/api/v1/instances?allow_list=r5d\\..*"

# R6i instances with specific memory requirements
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "r6i\\..*",
    "memory_min": "32gb",
    "current_generation": true
  }'
```

**General Purpose (M Family):**
```bash
# All M5/M6 instances
curl "http://localhost:8080/api/v1/instances?allow_list=m[56].*"

# M6i instances with balanced specs
curl "http://localhost:8080/api/v1/instances?allow_list=m6i\\..*&vcpus_min=4&vcpus_max=16"

# M5dn instances (with local NVMe storage and enhanced networking)
curl "http://localhost:8080/api/v1/instances?allow_list=m5dn\\..*"
```

**Storage Optimized (I Family):**
```bash
# All I3 instances with high IOPS
curl "http://localhost:8080/api/v1/instances?allow_list=i3.*&nvme=true"

# I4i instances (latest generation)
curl "http://localhost:8080/api/v1/instances?allow_list=i4i\\..*"
```

**GPU Instances:**
```bash
# All P3/P4 instances (machine learning)
curl "http://localhost:8080/api/v1/instances?allow_list=p[34].*"

# G4 instances (graphics workloads)
curl "http://localhost:8080/api/v1/instances?allow_list=g4.*"

# G5 instances with specific GPU requirements
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "g5\\..*",
    "gpus_min": 1,
    "current_generation": true
  }'
```

### Size-Specific Filtering

```bash
# All large instances across families
curl "http://localhost:8080/api/v1/instances?allow_list=.*\\.large"

# All xlarge and 2xlarge instances
curl "http://localhost:8080/api/v1/instances?allow_list=.*\\.(xl|2xl)arge"

# Metal instances only
curl "http://localhost:8080/api/v1/instances?allow_list=.*\\.metal"
```

### Practical Use Cases

**High-Performance Computing:**
```bash
# C5n instances with enhanced networking
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "c5n\\..*",
    "network_performance": 25,
    "current_generation": true
  }'
```

**Database Workloads:**
```bash
# R5d or R6i instances with high memory and NVME
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "r(5d|6i)\\..*",
    "memory_min": "64gb",
    "nvme": true,
    "current_generation": true
  }'
```

**Web Applications:**
```bash
# M5/M6 instances with balanced resources
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "m[56].*",
    "vcpus_min": 2,
    "vcpus_max": 8,
    "memory_per_cpu_min": 3.0,
    "memory_per_cpu_max": 5.0,
    "current_generation": true
  }'
```

**Big Data/Analytics:**
```bash
# I3/I4 instances with high local storage
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "allow_list": "i[34].*",
    "instance_storage_min": "1tb",
    "nvme": true
  }'
```

### Exclude Specific Families

```bash
# All instances except T instances (exclude burstable)
curl "http://localhost:8080/api/v1/instances?deny_list=t[234].*&current_generation=true"

# Modern instances only (exclude older generations)
curl "http://localhost:8080/api/v1/instances?deny_list=[acimr][1-4].*&current_generation=true"
```

### Regex Pattern Reference

| Pattern | Matches | Example |
|---------|---------|---------|
| `r5d\\..*` | All r5d instances | r5d.large, r5d.xlarge |
| `r5.*` | All r5 family | r5, r5a, r5ad, r5d, r5dn |
| `[rm]5.*` | All r5 and m5 families | r5.large, m5.large |
| `.*\\.large` | All large sizes | c5.large, m5.large |
| `c[5-7].*` | C5, C6, C7 families | c5.large, c6i.large |
| `.*\\.(xl\\|2xl)arge` | xlarge and 2xlarge | c5.xlarge, m5.2xlarge |

## Configuration

### AWS Credentials

The server uses AWS SDK default configuration for credentials:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM roles (for EC2 instances)

You can also set the AWS region:
```bash
export AWS_REGION=us-west-2
```

### Server Configuration

The API server can be configured using environment variables:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `8080` | Server port |
| `EC2_INSTANCE_SELECTOR_CACHE_TTL` | `24h` | Cache time-to-live for pricing data. Examples: `1h`, `30m`, `24h`, `0` (disables cache) |
| `EC2_INSTANCE_SELECTOR_CACHE_DIR` | `~/.ec2-instance-selector/` | Directory for cache files |
| `EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT` | `false` | Skip pricing cache initialization on startup for faster startup |

### Pricing Data Configuration

**Default Behavior (Recommended):**
```bash
# Uses 24-hour cache, initializes pricing on startup
./ec2-api-server
```

**Custom Cache Configuration:**
```bash
export EC2_INSTANCE_SELECTOR_CACHE_TTL=12h
export EC2_INSTANCE_SELECTOR_CACHE_DIR=/tmp/ec2-cache/
export PORT=3000
./ec2-api-server
```

**Fast Startup (No Pricing Data):**
```bash
# Starts quickly but OndemandPricePerHour and SpotPrice will be null
export EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT=true
./ec2-api-server
```

**Disable Pricing Cache:**
```bash
# No pricing data cached or returned
export EC2_INSTANCE_SELECTOR_CACHE_TTL=0
./ec2-api-server
```

### Understanding Pricing Data

- **First startup** with pricing enabled takes 30-60 seconds to fetch pricing data from AWS
- **Subsequent startups** are fast since pricing data is cached
- **OndemandPricePerHour** and **SpotPrice** fields will only be populated when pricing cache is enabled and initialized
- **Spot prices** are 30-day averages across availability zones

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
docker run -p 8080:8080 \
  -e AWS_ACCESS_KEY_ID=xxx \
  -e AWS_SECRET_ACCESS_KEY=yyy \
  -e EC2_INSTANCE_SELECTOR_CACHE_TTL=12h \
  ec2-instance-selector-api
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
