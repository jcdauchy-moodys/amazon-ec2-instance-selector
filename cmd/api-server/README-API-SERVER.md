
### REST API

The EC2 Instance Selector also provides a REST API server for integration into web applications and automation tools.

**Start the API server:**
```bash
# Build and run the API server
go run cmd/api-server/main.go

# Or build a binary
go build -o ec2-api-server cmd/api-server/main.go
./ec2-api-server
```

**Find instances with memory-per-CPU filtering:**
```bash
# Find memory-optimized instances (8+ GiB per vCPU)
curl "http://localhost:8080/api/v1/instances?memory_per_cpu_min=8.0&current_generation=true&max_results=5"

# Find instances with NVME storage support
curl "http://localhost:8080/api/v1/instances?nvme=true&current_generation=true&max_results=5"

# Find balanced instances with POST request
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "memory_per_cpu_min": 2.0,
    "memory_per_cpu_max": 4.0,
    "nvme": true,
    "current_generation": true,
    "max_results": 5
  }'
```

**Sample JSON Response:**
```json
{
  "success": true,
  "instance_types": [
    {
      "InstanceType": "r5.large",
      "VCpuInfo": {
        "DefaultVCpus": 2
      },
      "MemoryInfo": {
        "SizeInMiB": 16384
      },
      "OndemandPricePerHour": 0.126,
      "SpotPrice": 0.045
    }
  ],
  "count": 1
}
```

For detailed API documentation, see [cmd/api-server/README.md](README.md).