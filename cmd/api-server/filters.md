# API Server Filters Reference

This document provides a comprehensive reference of all filters available in the EC2 Instance Selector API Server.

## Currently Exposed Filters

The following filters are currently available through the API endpoints (`GET /api/v1/instances` and `POST /api/v1/instances/filter`):

### CPU Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `vcpus` | int | Exact number of vCPUs | `4` |
| `vcpus_min` | int | Minimum vCPUs | `2` |
| `vcpus_max` | int | Maximum vCPUs | `8` |
| `cpu_architecture` | string | CPU architecture | `"x86_64"`, `"arm64"` |

### Memory Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `memory` | string | Exact memory amount | `"4gb"`, `"8 GiB"` |
| `memory_min` | string | Minimum memory | `"8gb"` |
| `memory_max` | string | Maximum memory | `"32gb"` |
| `memory_per_cpu_min` | float | Minimum memory per vCPU ratio (GiB per vCPU) | `2.0` |
| `memory_per_cpu_max` | float | Maximum memory per vCPU ratio (GiB per vCPU) | `8.0` |

**Memory Format Examples:**
- `"4gb"` or `"4 GB"`
- `"8gib"` or `"8 GiB"`
- `"1024mb"` or `"1024 MB"`
- `"2tb"` or `"2 TB"`

**Memory-per-CPU Use Cases:**
- **Compute-optimized** instances: `1-2 GiB per vCPU`
- **General-purpose** instances: `3-4 GiB per vCPU`
- **Memory-optimized** instances: `8+ GiB per vCPU`

### GPU Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `gpus` | int | Exact number of GPUs | `1` |
| `gpus_min` | int | Minimum GPUs | `1` |
| `gpus_max` | int | Maximum GPUs | `4` |

### Network Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `network_performance` | int | Network performance in Gbps | `25` |

### Instance Type Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `instance_types` | array/string | Comma-separated list of specific instance types (exact matches) | `["m5.large", "m5.xlarge"]` or `"m5.large,m5.xlarge"` |
| `allow_list` | string | Regex pattern for allowed instance types | `"m[5-6]\\.*"`, `"c5\\..*"` |
| `deny_list` | string | Regex pattern for denied instance types | `"t[234].*"` |

**Regex Pattern Examples:**

| Pattern | Matches | Example |
|---------|---------|---------|
| `r5d\\..*` | All r5d instances | r5d.large, r5d.xlarge |
| `r5.*` | All r5 family | r5, r5a, r5ad, r5d, r5dn |
| `[rm]5.*` | All r5 and m5 families | r5.large, m5.large |
| `.*\\.large` | All large sizes | c5.large, m5.large |
| `c[5-7].*` | C5, C6, C7 families | c5.large, c6i.large |
| `.*\\.(xl\|2xl)arge` | xlarge and 2xlarge | c5.xlarge, m5.2xlarge |

### Boolean Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `current_generation` | bool | Filter for current generation instances | `true`, `false` |
| `bare_metal` | bool | Bare metal instances only | `true`, `false` |
| `burstable` | bool | Burstable performance instances (T family) | `true`, `false` |
| `free_tier` | bool | Free tier eligible instances | `true`, `false` |
| `nvme` | bool | NVME storage support | `true`, `false` |

### NVMe Storage Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `nvme_instance_storage` | string | Exact NVMe instance storage size | `100gb`, `1tb` |
| `nvme_instance_storage_min` | string | Minimum NVMe instance storage | `500gb` |
| `nvme_instance_storage_max` | string | Maximum NVMe instance storage | `2tb` |

**Note:** These filters only match instances where NVMe support is "required" (true NVMe instances). Storage is filtered on total size across all disks.

### Location Filters

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `region` | string | AWS region to query. If specified, AWS API queries will be executed in this region. If omitted, the default region from AWS_REGION environment variable is used. The server caches selectors per region for performance. | `"us-east-1"`, `"eu-west-1"` |
| `availability_zones` | array/string | Comma-separated list of availability zones | `["us-east-1a", "us-east-1b"]` or `"us-east-1a,us-east-1b"` |

### Usage Class Filter

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `usage_class` | string | Usage class | `"on-demand"`, `"spot"` |

### Result Limit

| Filter | Type | Description | Example |
|--------|------|-------------|---------|
| `max_results` | int | Maximum number of results to return | `20` (default) |

## Usage Examples

### GET Request
```bash
curl "http://localhost:8080/api/v1/instances?vcpus_min=4&memory_min=8gb&current_generation=true&max_results=10"
```

### GET Request - NVMe storage filtering
```bash
# Find i3 family instances with at least 1TB of NVMe storage
curl "http://localhost:8080/api/v1/instances?nvme_instance_storage_min=1tb&current_generation=true&max_results=20"

# Find instances with NVMe storage between 500GB and 2TB
curl "http://localhost:8080/api/v1/instances?nvme_instance_storage_min=500gb&nvme_instance_storage_max=2tb&current_generation=true"
```

### POST Request
```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "vcpus_min": 4,
    "vcpus_max": 8,
    "memory_min": "8gb",
    "memory_per_cpu_min": 2.0,
    "memory_per_cpu_max": 4.0,
    "cpu_architecture": "x86_64",
    "current_generation": true,
    "nvme": true,
    "allow_list": "m[5-6].*",
    "max_results": 10
  }'
```

### POST Request - NVMe storage filtering
```bash
curl -X POST http://localhost:8080/api/v1/instances/filter \
  -H "Content-Type: application/json" \
  -d '{
    "nvme_instance_storage_min": "1tb",
    "nvme_instance_storage_max": "5tb",
    "cpu_architecture": "x86_64",
    "current_generation": true,
    "max_results": 20
  }'
```

## Additional Filters Available in Selector (Not Yet Exposed)

The underlying `selector.Filters` struct supports additional filters that are not currently exposed through the API server. These could be added in future versions:

### CPU & Architecture
- `cpu_manufacturer` - CPU manufacturer filter
- `hypervisor` - Hypervisor type (e.g., xen, nitro)
- `virtualization_type` - Virtualization type (e.g., hvm, paravirtual)

### Memory & Performance
- `vcpus_to_memory_ratio` - Specific vCPUs to memory ratio

### GPU & Accelerators
- `gpu_memory_range` - GPU memory range filter
- `gpu_manufacturer` - GPU manufacturer (e.g., NVIDIA, AMD)
- `gpu_model` - Specific GPU model
- `inference_accelerators_range` - Number of inference accelerators
- `inference_accelerator_manufacturer` - Inference accelerator manufacturer
- `inference_accelerator_model` - Specific inference accelerator model

### Storage
- `instance_storage_range` - Instance storage range filter
- `disk_type` - Disk type (e.g., hdd, ssd)
- `disk_encryption` - Disk encryption support
- `ebs_optimized` - EBS optimized support
- `ebs_optimized_baseline_bandwidth` - EBS baseline bandwidth
- `ebs_optimized_baseline_iops` - EBS baseline IOPS
- `ebs_optimized_baseline_throughput` - EBS baseline throughput

### Network
- `network_interfaces` - Maximum network interfaces count
- `network_encryption` - Network encryption in transit support
- `ipv6` - IPv6 support
- `ena_support` - Elastic Network Adapter support
- `efa_support` - Elastic Fabric Adapter support

### Other Features
- `root_device_type` - Root device type (e.g., ebs, instance-store)
- `hibernation_supported` - Hibernation support
- `fpga` - FPGA support
- `placement_group_strategy` - Placement group strategy support
- `auto_recovery` - Auto recovery support
- `dedicated_hosts` - Dedicated hosts support
- `generation` - Instance generation filter
- `price_per_hour` - Price per hour range filter

## Filter Implementation Details

### Code Location
- **API Request Structure:** `cmd/api-server/main.go` - `FilterRequest` struct (lines 78-104)
- **Filter Parsing:** `cmd/api-server/main.go` - `parseQueryParams()` and `requestToFilters()` functions
- **Selector Filters:** `pkg/selector/types.go` - `Filters` struct
- **Filter Execution:** `pkg/selector/selector.go` - Filter execution logic

### Adding New Filters

To expose additional filters from the selector:

1. Add the filter field to the `FilterRequest` struct in `cmd/api-server/main.go`
2. Add parsing logic in `parseQueryParams()` for GET requests
3. Add conversion logic in `requestToFilters()` to map to selector filters
4. Update this documentation

## See Also

- [API Server README](README.md) - Complete API documentation with examples
- [InfluxDB Metrics](influxdb-metrics.md) - Metrics collection documentation

