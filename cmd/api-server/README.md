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
- **AWS region configured** (required)
- Access to AWS EC2 API

### Running the Server

```bash
# Set AWS region (required)
export AWS_REGION=us-east-1

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

### Readiness Probe
```
GET /ready
```

Returns readiness status (for Kubernetes probes). Returns HTTP 200 when pricing data is available, HTTP 503 otherwise:

**Ready (HTTP 200):**
```json
{
  "status": "ready",
  "version": "1.0.0"
}
```

**Not Ready (HTTP 503):**
```json
{
  "status": "not ready - pricing data not yet available",
  "version": "1.0.0"
}
```

This endpoint is intended for Kubernetes readiness probes and will only return success once the pricing caches have been initialized.

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
- `nvme_instance_storage` - Exact NVMe instance storage size (e.g., "1tb")
- `nvme_instance_storage_min` - Minimum NVMe instance storage (e.g., "500gb")
- `nvme_instance_storage_max` - Maximum NVMe instance storage (e.g., "5tb")
- `max_results` - Maximum number of results (default: 20)
- `availability_zones` - Comma-separated list of AZs
- `usage_class` - Usage class (on-demand, spot)
- `gpus` - Number of GPUs (exact match)
- `gpus_min` - Minimum GPUs
- `gpus_max` - Maximum GPUs
- `network_performance` - Network performance in Gbps
- `sort_by` - Field to sort by (see Sorting section below)
- `sort_direction` - Sort direction: "ascending", "asc", "descending", "desc" (default: "ascending")

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

### Get Instance Price (GET)
```
GET /api/v1/instances/price?instance_type=m5.xlarge&region=us-east-1&usage_class=on-demand
```

Retrieves the latest cached pricing information for a specific instance type.

**Query Parameters:**
- `instance_type` (required) - The EC2 instance type (e.g., "m5.xlarge", "t3.medium")
- `region` (optional) - AWS region (defaults to server's default region)
- `usage_class` (optional) - Pricing type: "on-demand" or "spot" (default: "on-demand")

**Response:**
```json
{
  "success": true,
  "region": "us-east-1",
  "instance_type": "m5.xlarge",
  "usage_class": "on-demand",
  "price_per_hour": 0.192,
  "currency": "USD",
  "last_updated": "2025-10-22T10:30:00Z",
  "cache_expiration": "2025-10-23T10:30:00Z"
}
```

**Response Fields:**
- `success` - Whether the request was successful
- `region` - AWS region for the pricing data
- `instance_type` - The requested instance type
- `usage_class` - The pricing type (on-demand or spot)
- `price_per_hour` - The price per hour in USD
- `currency` - Currency code (always "USD")
- `last_updated` - When the pricing data was last fetched/cached
- `cache_expiration` - When the cached pricing data will expire

**Notes:**
- For spot pricing, the endpoint returns a 30-day average price
- Pricing data is cached based on the `EC2_INSTANCE_SELECTOR_CACHE_TTL` setting (default: 24 hours)
- If pricing data is not in cache, it will be fetched from AWS APIs

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

### Example 6: Get pricing for a specific instance type

```bash
# Get on-demand pricing for m5.xlarge in us-east-1
curl "http://localhost:8080/api/v1/instances/price?instance_type=m5.xlarge&region=us-east-1&usage_class=on-demand"

# Get spot pricing for t3.medium (uses default region)
curl "http://localhost:8080/api/v1/instances/price?instance_type=t3.medium&usage_class=spot"

# Get on-demand pricing for c5.2xlarge (defaults to on-demand if usage_class not specified)
curl "http://localhost:8080/api/v1/instances/price?instance_type=c5.2xlarge"
```

**Example Response:**
```json
{
  "success": true,
  "region": "us-east-1",
  "instance_type": "m5.xlarge",
  "usage_class": "on-demand",
  "price_per_hour": 0.192,
  "currency": "USD",
  "last_updated": "2025-10-22T10:30:00Z",
  "cache_expiration": "2025-10-23T10:30:00Z"
}
```

### Example 7: Find instances with NVME storage support

```bash
# Find instances that support NVME storage
curl "http://localhost:8080/api/v1/instances?nvme=true&current_generation=true&max_results=5"

# Find instances with at least 1TB of NVMe instance storage
curl "http://localhost:8080/api/v1/instances?nvme_instance_storage_min=1tb&current_generation=true&max_results=10"

# Find instances with NVMe storage between 500GB and 2TB
curl "http://localhost:8080/api/v1/instances?nvme_instance_storage_min=500gb&nvme_instance_storage_max=2tb&current_generation=true"
```

```bash
# Find NVME instances with specific specs using POST
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "nvme": true,
    "nvme_instance_storage_min": "1tb",
    "vcpus_min": 4,
    "memory_min": "16gb",
    "current_generation": true,
    "max_results": 10
  }'
```

### Example 8: Sort results to find best value instances

```bash
# Find 5 cheapest current-generation instances with at least 4 vCPUs
curl "http://localhost:8080/api/v1/instances?vcpus_min=4&current_generation=true&sort_by=on-demand-price&sort_direction=asc&max_results=5"

# Find instances with best memory-per-vCPU ratio (memory-optimized)
curl "http://localhost:8080/api/v1/instances?vcpus_min=4&current_generation=true&sort_by=memory-per-cpu&sort_direction=desc&max_results=10"

# Find cheapest GPU instances sorted by spot price
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "gpus_min": 1,
    "current_generation": true,
    "sort_by": "spot-price",
    "sort_direction": "ascending",
    "max_results": 5
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

## Sorting Results

Results can be sorted by various fields before applying the `max_results` limit. This allows you to get the top N instances based on specific criteria.

### Sort Parameters

- `sort_by` - The field to sort by (see available fields below)
- `sort_direction` - Direction: `ascending`, `asc`, `descending`, `desc` (default: `ascending`)

### Available Sort Fields

**Shorthand fields:**
- `vcpus` - Number of vCPUs
- `memory` - Memory size in MiB
- `memory-per-cpu` - Memory per vCPU ratio (GiB per vCPU)
- `gpus` - Number of GPUs
- `gpu-memory-total` - Total GPU memory
- `network-interfaces` - Maximum network interfaces
- `spot-price` - Spot price (30-day average)
- `on-demand-price` - On-demand price per hour
- `instance-storage` - Total instance storage in GB
- `ebs-optimized-baseline-bandwidth` - EBS bandwidth in Mbps
- `ebs-optimized-baseline-throughput` - EBS throughput in MBps
- `ebs-optimized-baseline-iops` - EBS baseline IOPS
- `inference-accelerators` - Number of inference accelerators

**JSON paths:** You can also use JSON paths to sort by any field in the instance details (e.g., `.MemoryInfo.SizeInMiB`, `.NetworkInfo.NetworkPerformance`)

### Sorting Examples

**Example 1: Find cheapest on-demand instances**
```bash
curl "http://localhost:8080/api/v1/instances?vcpus_min=2&current_generation=true&sort_by=on-demand-price&sort_direction=asc&max_results=5"
```

**Example 2: Find largest memory instances**
```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "current_generation": true,
    "sort_by": "memory",
    "sort_direction": "desc",
    "max_results": 10
  }'
```

**Example 3: Find most cost-effective GPU instances**
```bash
curl "http://localhost:8080/api/v1/instances?gpus_min=1&sort_by=spot-price&sort_direction=asc&max_results=5"
```

**Example 4: Find instances with best memory-per-vCPU ratio**
```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus_min": 4,
    "current_generation": true,
    "sort_by": "memory-per-cpu",
    "sort_direction": "descending",
    "max_results": 10
  }'
```

**Example 5: Find instances with most vCPUs**
```bash
curl "http://localhost:8080/api/v1/instances?current_generation=true&sort_by=vcpus&sort_direction=desc&max_results=10"
```

**Example 6: Find cheapest instances with NVMe storage**
```bash
curl "http://localhost:8080/api/v1/instances?nvme=true&sort_by=on-demand-price&sort_direction=asc&max_results=10"
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

### AWS Credentials and Region

The server uses AWS SDK default configuration for credentials:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM roles (for EC2 instances)

**AWS region is required** and can be set via:
1. Environment variable:
   ```bash
   export AWS_REGION=us-west-2
   # OR
   export AWS_DEFAULT_REGION=us-west-2
   ```
2. AWS config file (`~/.aws/config`):
   ```ini
   [default]
   region = us-west-2
   ```

If no region is configured, the server will fail to start with an error message.

### Server Configuration

The API server can be configured using environment variables:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `AWS_REGION` or `AWS_DEFAULT_REGION` | *(required)* | AWS region to use for API calls |
| `PORT` | `8080` | Server port |
| `EC2_INSTANCE_SELECTOR_CACHE_TTL` | `24h` | Cache time-to-live for pricing data. Examples: `1h`, `30m`, `24h`, `0` (disables cache) |
| `EC2_INSTANCE_SELECTOR_CACHE_DIR` | `~/.ec2-instance-selector/` | Directory for cache files |
| `EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT` | `false` | Skip pricing cache initialization on startup for faster startup |
| `EC2_INSTANCE_SELECTOR_VERBOSE` | `false` | Enable verbose/debug logging to see detailed AWS API calls and timing |

### Pricing Data Configuration

**Default Behavior (Recommended):**
```bash
# Uses 24-hour cache, initializes pricing on startup
export AWS_REGION=us-east-1
./ec2-api-server
```

**Custom Cache Configuration:**
```bash
export AWS_REGION=us-west-2
export EC2_INSTANCE_SELECTOR_CACHE_TTL=12h
export EC2_INSTANCE_SELECTOR_CACHE_DIR=/tmp/ec2-cache/
export PORT=3000
./ec2-api-server
```

**Fast Startup (No Pricing Data):**
```bash
# Starts quickly but OndemandPricePerHour and SpotPrice will be null
export AWS_REGION=us-east-1
export EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT=true
./ec2-api-server
```

**Disable Pricing Cache:**
```bash
# No pricing data cached or returned
export AWS_REGION=us-east-1
export EC2_INSTANCE_SELECTOR_CACHE_TTL=0
./ec2-api-server
```

### Understanding Pricing Data

- **First startup** with pricing enabled takes 30-60 seconds to fetch pricing data from AWS
- **Subsequent startups** are fast since pricing data is cached
- **OndemandPricePerHour** and **SpotPrice** fields will only be populated when pricing cache is enabled and initialized
- **Spot prices** are 30-day averages across availability zones

### Verbose Logging

Enable verbose logging to see detailed information about:
- AWS API calls and their timing
- Pricing cache operations
- Filter execution performance
- Request processing details

```bash
export AWS_REGION=us-east-1
export EC2_INSTANCE_SELECTOR_VERBOSE=true
./ec2-api-server
```

**Normal mode logs:**
```
2025/10/21 10:34:10 Starting EC2 Instance Selector API Server...
2025/10/21 10:34:10 Server successfully initialized!
2025/10/21 10:34:15 Filter executed in 125ms, found 15 instances
```

**Verbose mode logs:**
```
2025/10/21 10:34:10 Starting EC2 Instance Selector API Server...
2025/10/21 10:34:10 Verbose logging enabled
2025/10/21 10:34:10 Took 301ms and 1 calls to collect OD pricing
2025/10/21 10:34:10 Server successfully initialized!
2025/10/21 10:34:15 Filter executed in 125ms, found 15 instances
... (detailed AWS SDK logs)
```

Verbose mode is useful for:
- **Debugging** performance issues
- **Troubleshooting** API call failures
- **Understanding** cache behavior
- **Monitoring** AWS API usage

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
  -e AWS_REGION=us-east-1 \
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

## Environment Variables

### Core Configuration
- `AWS_REGION` or `AWS_DEFAULT_REGION` - AWS region (required)
- `EC2_INSTANCE_SELECTOR_CACHE_TTL` - Cache time-to-live for pricing data (default: 24h)
  - Examples: "1h", "30m", "24h", "0" (disables cache)
- `EC2_INSTANCE_SELECTOR_CACHE_DIR` - Directory for cache files (default: ~/.ec2-instance-selector/)
- `EC2_INSTANCE_SELECTOR_SKIP_PRICING_CACHE_INIT` - Skip pricing cache initialization on startup (default: false)
- `EC2_INSTANCE_SELECTOR_VERBOSE` - Enable verbose/debug logging (default: false)
- `PORT` - Server port (default: 8080)

### InfluxDB Metrics Configuration (v1.x)
- `INFLUXDB_ENABLED` - Enable InfluxDB metrics collection (default: false)
- `INFLUXDB_URL` - InfluxDB server URL (required if enabled)
- `INFLUXDB_DATABASE` - InfluxDB database name (required if enabled)
- `INFLUXDB_JWT` - JWT token for InfluxDB authentication (optional, adds Authorization Bearer header)

## Metrics Collection

When InfluxDB metrics collection is enabled, the server will record instance metrics with the following schema:

### ec2_instances
Tags:
- region
- instance_type
- family (e.g., "m5")
- size (e.g., "large")
- architecture

Fields:
- vcpus (integer)
- memory_gb (float)

## Example Usage

### Basic Server Start
```bash
export AWS_REGION=us-east-1
./api-server
```

### With InfluxDB Metrics
```bash
export AWS_REGION=us-east-1
export INFLUXDB_ENABLED=true
export INFLUXDB_URL="https://influxdb.example.com"
export INFLUXDB_DATABASE="ec2metrics"
export INFLUXDB_JWT="your-jwt-token-here"  # Optional, for authenticated InfluxDB
./api-server
```

### Custom Configuration
```bash
export AWS_REGION=us-east-1
export EC2_INSTANCE_SELECTOR_CACHE_TTL=12h
export EC2_INSTANCE_SELECTOR_CACHE_DIR=/tmp/ec2-cache/
export PORT=3000
export INFLUXDB_ENABLED=true
export INFLUXDB_URL="https://influxdb.example.com"
export INFLUXDB_DATABASE="ec2metrics"
export INFLUXDB_JWT="your-jwt-token-here"  # Optional, for authenticated InfluxDB
./api-server
```

## API Endpoints

### GET /health
Health check endpoint.

### GET /ready
Readiness probe endpoint. Returns HTTP 200 when pricing data is available, HTTP 503 otherwise.

### GET /api/v1/instances
Filter instances using query parameters.

### POST /api/v1/instances/filter
Filter instances using JSON request body.

### GET /api/v1/instances/price
Get pricing information for a specific instance type.

For detailed API documentation and examples, see the examples.sh file.
